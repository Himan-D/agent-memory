package neo4j

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/google/uuid"
	neo4jdriver "github.com/neo4j/neo4j-go-driver/v6/neo4j"

	"agent-memory/internal/config"
	"agent-memory/internal/memory/types"
)

var (
	validRelTypeRegex = regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)
	allowedRelTypes   = map[string]bool{
		"KNOWS":      true,
		"HAS":        true,
		"RELATED_TO": true,
		"DEPENDS_ON": true,
		"USES":       true,
		"CREATED_BY": true,
		"PART_OF":    true,
		"IMPROVES":   true,
		"CONFLICTS":  true,
		"FOLLOWS":    true,
		"LIKES":      true,
		"DISLIKES":   true,
		"SUBSCRIBED": true,
	}
)

func validateRelationType(relType string) error {
	if !validRelTypeRegex.MatchString(relType) {
		return fmt.Errorf("invalid relation type: must be uppercase alphanumeric with underscores, got %q", relType)
	}
	if !allowedRelTypes[relType] {
		return fmt.Errorf("relation type %q not allowed; allowed: KNOWS, HAS, RELATED_TO, etc.", relType)
	}
	return nil
}

type Client struct {
	driver   neo4jdriver.Driver
	config   config.Neo4jConfig
	pool     chan neo4jdriver.SessionWithContext
	maxConns int
}

func NewClient(cfg config.Neo4jConfig) (*Client, error) {
	driver, err := neo4jdriver.NewDriverWithContext(
		cfg.URI,
		neo4jdriver.BasicAuth(cfg.User, cfg.Password, ""),
	)
	if err != nil {
		return nil, fmt.Errorf("neo4j driver init: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := driver.VerifyConnectivity(ctx); err != nil {
		return nil, fmt.Errorf("neo4j connectivity: %w", err)
	}

	maxConns := cfg.MaxConns
	if maxConns <= 0 {
		maxConns = 50
	}

	pool := make(chan neo4jdriver.SessionWithContext, maxConns)

	for i := 0; i < maxConns; i++ {
		session := driver.NewSession(ctx, neo4jdriver.SessionConfig{
			AccessMode: neo4jdriver.AccessModeWrite,
		})
		pool <- session
	}

	c := &Client{
		driver:   driver,
		config:   cfg,
		pool:     pool,
		maxConns: maxConns,
	}

	if err := c.ensureIndexes(ctx); err != nil {
		return nil, fmt.Errorf("ensure indexes: %w", err)
	}

	return c, nil
}

func (c *Client) AcquireSession(ctx context.Context) (neo4jdriver.SessionWithContext, func(), error) {
	select {
	case session := <-c.pool:
		return session, func() { c.pool <- session }, nil
	default:
		if c.pool == nil || cap(c.pool) == 0 {
			return c.driver.NewSession(ctx, neo4jdriver.SessionConfig{
				AccessMode: neo4jdriver.AccessModeWrite,
			}), func() {}, nil
		}
		newSession := c.driver.NewSession(ctx, neo4jdriver.SessionConfig{
			AccessMode: neo4jdriver.AccessModeWrite,
		})
		return newSession, func() { c.pool <- newSession }, nil
	}
}

func (c *Client) Session() neo4jdriver.SessionWithContext {
	return c.driver.NewSession(context.Background(), neo4jdriver.SessionConfig{
		AccessMode: neo4jdriver.AccessModeWrite,
	})
}

func (c *Client) Close() error {
	close(c.pool)
	for session := range c.pool {
		session.Close(context.Background())
	}
	return c.driver.Close(context.Background())
}

func (c *Client) Ping(ctx context.Context) error {
	session := c.driver.NewSession(ctx, neo4jdriver.SessionConfig{
		AccessMode: neo4jdriver.AccessModeRead,
	})
	defer session.Close(ctx)

	_, err := session.Run(ctx, "RETURN 1", nil)
	return err
}

func (c *Client) ensureIndexes(ctx context.Context) error {
	indexes := []string{
		"CREATE INDEX entity_id_idx IF NOT EXISTS FOR (e:Entity) ON (e.id)",
		"CREATE INDEX session_id_idx IF NOT EXISTS FOR (s:Session) ON (s.id)",
		"CREATE INDEX message_session_idx IF NOT EXISTS FOR (m:Message) ON (m.session_id)",
	}

	for _, idx := range indexes {
		session := c.driver.NewSession(ctx, neo4jdriver.SessionConfig{
			AccessMode: neo4jdriver.AccessModeWrite,
		})
		defer session.Close(ctx)

		_, err := session.Run(ctx, idx, nil)
		if err != nil {
			return fmt.Errorf("create index: %w", err)
		}
	}
	return nil
}

// ==================== Short-term Memory ====================

func (c *Client) CreateSession(agentID string, metadata map[string]interface{}) (*types.Session, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	session := c.driver.NewSession(ctx, neo4jdriver.SessionConfig{
		AccessMode: neo4jdriver.AccessModeWrite,
	})
	defer session.Close(ctx)

	sessionID := uuid.New().String()

	query := `
		CREATE (s:Session {
			id: $sessionID,
			agent_id: $agentID
		})
		RETURN s.id, s.agent_id
	`

	params := map[string]interface{}{
		"sessionID": sessionID,
		"agentID":   agentID,
	}

	result, err := session.Run(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	if result.Next(ctx) {
		rec := result.Record()
		_, err = result.Consume(ctx)
		if err != nil {
			return nil, fmt.Errorf("create session consume: %w", err)
		}
		return &types.Session{
			ID:        rec.Values[0].(string),
			AgentID:   rec.Values[1].(string),
			Metadata:  metadata,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}, nil
	}
	_, err = result.Consume(ctx)
	if err != nil {
		return nil, fmt.Errorf("create session consume: %w", err)
	}
	return nil, fmt.Errorf("create session: no result")
}

func (c *Client) AddMessage(sessionID string, msg types.Message) error {
	ctx := context.Background()
	session := c.Session()
	defer session.Close(ctx)

	msgID := uuid.New().String()

	query := `
		MATCH (s:Session {id: $sessionID})
		CREATE (m:Message {
			id: $msgID,
			role: $role,
			content: $content,
			timestamp: datetime($timestamp)
		})
		CREATE (s)-[:HAS_MESSAGE]->(m)
		RETURN m.id
	`

	_, err := session.Run(ctx, query, map[string]interface{}{
		"sessionID": sessionID,
		"msgID":     msgID,
		"role":      msg.Role,
		"content":   msg.Content,
		"timestamp": time.Now().Format(time.RFC3339),
	})
	if err != nil {
		return fmt.Errorf("add message: %w", err)
	}
	return nil
}

func (c *Client) GetMessages(sessionID string, limit int) ([]types.Message, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	session := c.driver.NewSession(ctx, neo4jdriver.SessionConfig{
		AccessMode: neo4jdriver.AccessModeRead,
	})
	defer session.Close(ctx)

	query := `
		MATCH (s:Session {id: $sessionID})-[:HAS_MESSAGE]->(m:Message)
		RETURN m.id, m.role, m.content, m.timestamp
		ORDER BY m.timestamp DESC
		LIMIT $limit
	`

	result, err := session.Run(ctx, query, map[string]interface{}{
		"sessionID": sessionID,
		"limit":     int64(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("get messages: %w", err)
	}

	keys, _ := result.Keys()

	var messages []types.Message
	for result.Next(ctx) {
		rec := result.Record()
		msg := types.Message{
			SessionID: sessionID,
		}
		for i, key := range keys {
			val := rec.Values[i]
			switch key {
			case "m.id":
				msg.ID = val.(string)
			case "m.role":
				msg.Role = val.(string)
			case "m.content":
				msg.Content = val.(string)
			case "m.timestamp":
				msg.Timestamp = val.(time.Time)
			}
		}
		messages = append(messages, msg)
	}

	return messages, nil
}

func (c *Client) ClearMessages(sessionID string) error {
	ctx := context.Background()
	session := c.Session()
	defer session.Close(ctx)

	query := `
		MATCH (s:Session {id: $sessionID})-[:HAS_MESSAGE]->(m:Message)
		DETACH DELETE m
	`

	_, err := session.Run(ctx, query, map[string]interface{}{
		"sessionID": sessionID,
	})
	if err != nil {
		return fmt.Errorf("clear messages: %w", err)
	}
	return nil
}

// ==================== Knowledge Graph ====================

func (c *Client) AddEntity(entity types.Entity) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	session := c.driver.NewSession(ctx, neo4jdriver.SessionConfig{
		AccessMode: neo4jdriver.AccessModeWrite,
	})
	defer session.Close(ctx)

	if entity.ID == "" {
		entity.ID = uuid.New().String()
	}

	query := `
		MERGE (e:Entity {id: $id})
		SET e.type = $type,
			e.name = $name,
			e.tenant_id = $tenant_id,
			e.created_at = datetime($createdAt),
			e.updated_at = datetime($updatedAt)
		RETURN e.id
	`

	tenantID := "default"
	if entity.TenantID != "" {
		tenantID = entity.TenantID
	}

	result, err := session.Run(ctx, query, map[string]interface{}{
		"id":        entity.ID,
		"type":      entity.Type,
		"name":      entity.Name,
		"tenant_id": tenantID,
		"createdAt": time.Now().Format(time.RFC3339),
		"updatedAt": time.Now().Format(time.RFC3339),
	})
	if err != nil {
		return fmt.Errorf("add entity: %w", err)
	}

	_, err = result.Consume(ctx)
	if err != nil {
		return fmt.Errorf("add entity consume: %w", err)
	}
	return nil
}

func (c *Client) UpdateEntitySyncTime(entityID string) error {
	ctx := context.Background()
	session := c.Session()
	defer session.Close(ctx)

	query := `
		MATCH (e:Entity {id: $id})
		SET e.last_synced = datetime($lastSynced)
	`

	_, err := session.Run(ctx, query, map[string]interface{}{
		"id":         entityID,
		"lastSynced": time.Now().Format(time.RFC3339),
	})
	if err != nil {
		return fmt.Errorf("update entity sync time: %w", err)
	}
	return nil
}

func (c *Client) GetEntity(id string) (*types.Entity, error) {
	ctx := context.Background()
	session := c.Session()
	defer session.Close(ctx)

	query := `
		MATCH (e:Entity {id: $id})
		RETURN e.id, e.type, e.name, e.properties, e.created_at, e.updated_at, e.last_synced
	`

	result, err := session.Run(ctx, query, map[string]interface{}{"id": id})
	if err != nil {
		return nil, fmt.Errorf("get entity: %w", err)
	}

	if result.Next(ctx) {
		rec := result.Record()
		props := map[string]interface{}{}
		if rec.Values[3] != nil {
			props = rec.Values[3].(map[string]interface{})
		}
		entity := &types.Entity{
			ID:         rec.Values[0].(string),
			Type:       rec.Values[1].(string),
			Name:       rec.Values[2].(string),
			Properties: props,
			CreatedAt:  rec.Values[4].(time.Time),
			UpdatedAt:  rec.Values[5].(time.Time),
		}
		if rec.Values[6] != nil {
			lastSynced := rec.Values[6].(time.Time)
			entity.LastSynced = &lastSynced
		}
		return entity, nil
	}
	return nil, fmt.Errorf("entity not found: %s", id)
}

func (c *Client) BatchUpdateSyncTime(entityIDs []string) error {
	if len(entityIDs) == 0 {
		return nil
	}

	ctx := context.Background()
	session := c.Session()
	defer session.Close(ctx)

	now := time.Now().Format(time.RFC3339)

	query := `
		MATCH (e:Entity)
		WHERE e.id IN $ids
		SET e.last_synced = datetime($lastSynced)
	`

	_, err := session.Run(ctx, query, map[string]interface{}{
		"ids":        entityIDs,
		"lastSynced": now,
	})
	if err != nil {
		return fmt.Errorf("batch update sync time: %w", err)
	}
	return nil
}

func (c *Client) AddRelation(fromID, toID, relType string, props map[string]interface{}) error {
	if err := validateRelationType(relType); err != nil {
		return fmt.Errorf("invalid relation type: %w", err)
	}

	ctx := context.Background()
	session := c.Session()
	defer session.Close(ctx)

	relID := uuid.New().String()
	weight := 1.0
	if w, ok := props["weight"].(float64); ok {
		weight = w
	}

	query := fmt.Sprintf(`
		MATCH (a:Entity {id: $fromID}), (b:Entity {id: $toID})
		MERGE (a)-[r:%s {id: $relID}]->(b)
		SET r.weight = $weight,
			r.metadata = $metadata
		RETURN r.id
	`, relType)

	_, err := session.Run(ctx, query, map[string]interface{}{
		"fromID":   fromID,
		"toID":     toID,
		"relID":    relID,
		"weight":   weight,
		"metadata": props,
	})
	if err != nil {
		return fmt.Errorf("add relation: %w", err)
	}
	return nil
}

func (c *Client) QueryGraph(cypher string, params map[string]interface{}) ([]map[string]interface{}, error) {
	ctx := context.Background()
	session := c.Session()
	defer session.Close(ctx)

	result, err := session.Run(ctx, cypher, params)
	if err != nil {
		return nil, fmt.Errorf("query graph: %w", err)
	}

	keys, err := result.Keys()
	if err != nil {
		return nil, fmt.Errorf("get keys: %w", err)
	}

	var records []map[string]interface{}
	for result.Next(ctx) {
		rec := result.Record()
		record := map[string]interface{}{}
		for i, key := range keys {
			record[key] = rec.Values[i]
		}
		records = append(records, record)
	}
	return records, nil
}

func (c *Client) Traverse(fromEntityID string, depth int) ([]types.Path, error) {
	ctx := context.Background()
	session := c.Session()
	defer session.Close(ctx)

	query := `
		MATCH path = (start:Entity {id: $id})-[*1..$depth]-(end:Entity)
		RETURN nodes(path) as nodes, relationships(path) as edges
		LIMIT 100
	`

	result, err := session.Run(ctx, query, map[string]interface{}{
		"id":    fromEntityID,
		"depth": int64(depth),
	})
	if err != nil {
		return nil, fmt.Errorf("traverse: %w", err)
	}

	var paths []types.Path
	for result.Next(ctx) {
		rec := result.Record()
		nodesRaw := rec.Values[0].([]interface{})
		edgesRaw := rec.Values[1].([]interface{})

		var nodes []types.Entity
		for _, n := range nodesRaw {
			node := n.(neo4jdriver.Node)
			props := node.Props
			nodes = append(nodes, types.Entity{
				ID:         props["id"].(string),
				Type:       props["type"].(string),
				Name:       props["name"].(string),
				Properties: props,
			})
		}

		var edges []types.Relation
		for _, e := range edgesRaw {
			edge := e.(neo4jdriver.Relationship)
			props := edge.Props
			edges = append(edges, types.Relation{
				ID:     props["id"].(string),
				FromID: fmt.Sprintf("%d", edge.StartId),
				ToID:   fmt.Sprintf("%d", edge.EndId),
				Type:   edge.Type,
			})
		}

		paths = append(paths, types.Path{Nodes: nodes, Edges: edges})
	}
	return paths, nil
}

func (c *Client) GetEntityRelations(entityID string, relType string) ([]types.Relation, error) {
	if err := validateRelationType(relType); err != nil {
		return nil, fmt.Errorf("invalid relation type: %w", err)
	}

	ctx := context.Background()
	session := c.Session()
	defer session.Close(ctx)

	query := fmt.Sprintf(`
		MATCH (a:Entity {id: $id})-[r:%s]->(b:Entity)
		RETURN r.id, b.id, r.weight, r.metadata
	`, relType)

	result, err := session.Run(ctx, query, map[string]interface{}{"id": entityID})
	if err != nil {
		return nil, fmt.Errorf("get relations: %w", err)
	}

	var relations []types.Relation
	for result.Next(ctx) {
		rec := result.Record()
		relations = append(relations, types.Relation{
			ID:     rec.Values[0].(string),
			FromID: entityID,
			ToID:   rec.Values[1].(string),
			Type:   relType,
		})
	}
	return relations, nil
}
