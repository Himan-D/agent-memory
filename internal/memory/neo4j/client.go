package neo4j

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
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

// ValidateRelationType exports the validation function for testing
func ValidateRelationType(relType string) error {
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
	if err := ValidateRelationType(relType); err != nil {
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
	if err := ValidateRelationType(relType); err != nil {
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

// ==================== Memory Operations ====================

func (c *Client) CreateMemory(mem *types.Memory) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	session := c.driver.NewSession(ctx, neo4jdriver.SessionConfig{
		AccessMode: neo4jdriver.AccessModeWrite,
	})
	defer session.Close(ctx)

	query := `
		CREATE (m:Memory {
			id: $id,
			tenant_id: $tenant_id,
			user_id: $user_id,
			org_id: $org_id,
			agent_id: $agent_id,
			session_id: $session_id,
			type: $type,
			content: $content,
			category: $category,
			metadata: $metadata,
			status: $status,
			immutable: $immutable,
			expiration_date: $expiration_date,
			feedback_score: $feedback_score,
			created_at: datetime($created_at),
			updated_at: datetime($updated_at)
		})
		RETURN m.id
	`

	expirationDate := ""
	if mem.ExpirationDate != nil {
		expirationDate = mem.ExpirationDate.Format(time.RFC3339)
	}

	metadataJSON, _ := json.Marshal(mem.Metadata)
	result, err := session.Run(ctx, query, map[string]interface{}{
		"id":              mem.ID,
		"tenant_id":       mem.TenantID,
		"user_id":         mem.UserID,
		"org_id":          mem.OrgID,
		"agent_id":        mem.AgentID,
		"session_id":      mem.SessionID,
		"type":            string(mem.Type),
		"content":         mem.Content,
		"category":        mem.Category,
		"metadata":        string(metadataJSON),
		"status":          string(mem.Status),
		"immutable":       mem.Immutable,
		"expiration_date": expirationDate,
		"feedback_score":  string(mem.FeedbackScore),
		"created_at":      mem.CreatedAt.Format(time.RFC3339),
		"updated_at":      mem.UpdatedAt.Format(time.RFC3339),
	})
	if err != nil {
		return fmt.Errorf("create memory: %w", err)
	}
	_, err = result.Consume(ctx)
	return err
}

func (c *Client) GetMemory(id string) (*types.Memory, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	session := c.driver.NewSession(ctx, neo4jdriver.SessionConfig{
		AccessMode: neo4jdriver.AccessModeRead,
	})
	defer session.Close(ctx)

	query := `
		MATCH (m:Memory {id: $id})
		RETURN m.id, m.tenant_id, m.user_id, m.org_id, m.agent_id, m.session_id,
		       m.type, m.content, m.category, m.metadata, m.status, m.immutable,
		       m.expiration_date, m.feedback_score, m.created_at, m.updated_at, m.last_accessed
	`

	result, err := session.Run(ctx, query, map[string]interface{}{"id": id})
	if err != nil {
		return nil, fmt.Errorf("get memory: %w", err)
	}

	if result.Next(ctx) {
		rec := result.Record()
		return c.recordToMemory(rec)
	}
	return nil, fmt.Errorf("memory not found: %s", id)
}

func (c *Client) UpdateMemory(mem *types.Memory) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	session := c.driver.NewSession(ctx, neo4jdriver.SessionConfig{
		AccessMode: neo4jdriver.AccessModeWrite,
	})
	defer session.Close(ctx)

	query := `
		MATCH (m:Memory {id: $id})
		SET m.content = $content,
		    m.category = $category,
		    m.metadata = $metadata,
		    m.status = $status,
		    m.immutable = $immutable,
		    m.expiration_date = $expiration_date,
		    m.feedback_score = $feedback_score,
		    m.updated_at = datetime($updated_at)
		RETURN m.id
	`

	expirationDate := ""
	if mem.ExpirationDate != nil {
		expirationDate = mem.ExpirationDate.Format(time.RFC3339)
	}

	metadataJSON, _ := json.Marshal(mem.Metadata)
	_, err := session.Run(ctx, query, map[string]interface{}{
		"id":              mem.ID,
		"content":         mem.Content,
		"category":        mem.Category,
		"metadata":        string(metadataJSON),
		"status":          string(mem.Status),
		"immutable":       mem.Immutable,
		"expiration_date": expirationDate,
		"feedback_score":  string(mem.FeedbackScore),
		"updated_at":      mem.UpdatedAt.Format(time.RFC3339),
	})
	return err
}

func (c *Client) DeleteMemory(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	session := c.driver.NewSession(ctx, neo4jdriver.SessionConfig{
		AccessMode: neo4jdriver.AccessModeWrite,
	})
	defer session.Close(ctx)

	query := `
		MATCH (m:Memory {id: $id})
		DETACH DELETE m
	`

	_, err := session.Run(ctx, query, map[string]interface{}{"id": id})
	return err
}

func (c *Client) UpdateMemoryAccess(id string, accessedAt time.Time) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	session := c.driver.NewSession(ctx, neo4jdriver.SessionConfig{
		AccessMode: neo4jdriver.AccessModeWrite,
	})
	defer session.Close(ctx)

	query := `
		MATCH (m:Memory {id: $id})
		SET m.last_accessed = datetime($accessed_at)
	`

	_, err := session.Run(ctx, query, map[string]interface{}{
		"id":          id,
		"accessed_at": accessedAt.Format(time.RFC3339),
	})
	return err
}

func (c *Client) UpdateMemoryFeedbackScore(id string, score types.FeedbackType) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	session := c.driver.NewSession(ctx, neo4jdriver.SessionConfig{
		AccessMode: neo4jdriver.AccessModeWrite,
	})
	defer session.Close(ctx)

	query := `
		MATCH (m:Memory {id: $id})
		SET m.feedback_score = $score,
		    m.updated_at = datetime($updated_at)
	`

	_, err := session.Run(ctx, query, map[string]interface{}{
		"id":         id,
		"score":      string(score),
		"updated_at": time.Now().Format(time.RFC3339),
	})
	return err
}

func (c *Client) GetMemoriesByUser(userID string) ([]*types.Memory, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	session := c.driver.NewSession(ctx, neo4jdriver.SessionConfig{
		AccessMode: neo4jdriver.AccessModeRead,
	})
	defer session.Close(ctx)

	query := `
		MATCH (m:Memory {user_id: $user_id, status: 'active'})
		RETURN m.id, m.tenant_id, m.user_id, m.org_id, m.agent_id, m.session_id,
		       m.type, m.content, m.category, m.metadata, m.status, m.immutable,
		       m.expiration_date, m.feedback_score, m.created_at, m.updated_at, m.last_accessed
		ORDER BY m.created_at DESC
		LIMIT 1000
	`

	result, err := session.Run(ctx, query, map[string]interface{}{"user_id": userID})
	if err != nil {
		return nil, err
	}

	var memories []*types.Memory
	for result.Next(ctx) {
		if mem, err := c.recordToMemoryPtr(result.Record()); err == nil {
			memories = append(memories, mem)
		}
	}
	return memories, nil
}

func (c *Client) GetMemoriesByOrg(orgID string) ([]*types.Memory, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	session := c.driver.NewSession(ctx, neo4jdriver.SessionConfig{
		AccessMode: neo4jdriver.AccessModeRead,
	})
	defer session.Close(ctx)

	query := `
		MATCH (m:Memory {org_id: $org_id, status: 'active'})
		RETURN m.id, m.tenant_id, m.user_id, m.org_id, m.agent_id, m.session_id,
		       m.type, m.content, m.category, m.metadata, m.status, m.immutable,
		       m.expiration_date, m.feedback_score, m.created_at, m.updated_at, m.last_accessed
		ORDER BY m.created_at DESC
		LIMIT 1000
	`

	result, err := session.Run(ctx, query, map[string]interface{}{"org_id": orgID})
	if err != nil {
		return nil, err
	}

	var memories []*types.Memory
	for result.Next(ctx) {
		if mem, err := c.recordToMemoryPtr(result.Record()); err == nil {
			memories = append(memories, mem)
		}
	}
	return memories, nil
}

func (c *Client) GetExpiredMemories() ([]*types.Memory, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	session := c.driver.NewSession(ctx, neo4jdriver.SessionConfig{
		AccessMode: neo4jdriver.AccessModeRead,
	})
	defer session.Close(ctx)

	now := time.Now().Format(time.RFC3339)

	query := `
		MATCH (m:Memory)
		WHERE m.expiration_date IS NOT NULL AND m.expiration_date < datetime($now)
		       AND m.status = 'active'
		RETURN m.id, m.tenant_id, m.user_id, m.org_id, m.agent_id, m.session_id,
		       m.type, m.content, m.category, m.metadata, m.status, m.immutable,
		       m.expiration_date, m.feedback_score, m.created_at, m.updated_at, m.last_accessed
	`

	result, err := session.Run(ctx, query, map[string]interface{}{"now": now})
	if err != nil {
		return nil, err
	}

	var memories []*types.Memory
	for result.Next(ctx) {
		if mem, err := c.recordToMemoryPtr(result.Record()); err == nil {
			memories = append(memories, mem)
		}
	}
	return memories, nil
}

func (c *Client) BulkDeleteByFilter(userID, orgID, category string) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	session := c.driver.NewSession(ctx, neo4jdriver.SessionConfig{
		AccessMode: neo4jdriver.AccessModeWrite,
	})
	defer session.Close(ctx)

	var query string
	params := map[string]interface{}{}

	if userID != "" {
		query = `
			MATCH (m:Memory {user_id: $user_id})
			DETACH DELETE m
			RETURN count(m) as deleted
		`
		params["user_id"] = userID
	} else if orgID != "" {
		query = `
			MATCH (m:Memory {org_id: $org_id})
			DETACH DELETE m
			RETURN count(m) as deleted
		`
		params["org_id"] = orgID
	} else if category != "" {
		query = `
			MATCH (m:Memory {category: $category})
			DETACH DELETE m
			RETURN count(m) as deleted
		`
		params["category"] = category
	} else {
		return 0, fmt.Errorf("at least one filter (user_id, org_id, or category) is required")
	}

	result, err := session.Run(ctx, query, params)
	if err != nil {
		return 0, err
	}

	if result.Next(ctx) {
		rec := result.Record()
		if count, ok := rec.Values[0].(int64); ok {
			return int(count), nil
		}
	}
	return 0, nil
}

func (c *Client) LinkMemoryEntity(memoryID, entityID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	session := c.driver.NewSession(ctx, neo4jdriver.SessionConfig{
		AccessMode: neo4jdriver.AccessModeWrite,
	})
	defer session.Close(ctx)

	query := `
		MATCH (m:Memory {id: $memory_id}), (e:Entity {id: $entity_id})
		MERGE (e)-[:MEMORY_OF]->(m)
	`

	_, err := session.Run(ctx, query, map[string]interface{}{
		"memory_id": memoryID,
		"entity_id": entityID,
	})
	return err
}

func (c *Client) GetMemoryIDsByEntity(entityID string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	session := c.driver.NewSession(ctx, neo4jdriver.SessionConfig{
		AccessMode: neo4jdriver.AccessModeRead,
	})
	defer session.Close(ctx)

	query := `
		MATCH (e:Entity {id: $entity_id})-[:MEMORY_OF]->(m:Memory)
		RETURN m.id
	`

	result, err := session.Run(ctx, query, map[string]interface{}{"entity_id": entityID})
	if err != nil {
		return nil, err
	}

	var ids []string
	for result.Next(ctx) {
		rec := result.Record()
		if id, ok := rec.Values[0].(string); ok {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

// ==================== Feedback Operations ====================

func (c *Client) CreateFeedback(feedback *types.Feedback) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	session := c.driver.NewSession(ctx, neo4jdriver.SessionConfig{
		AccessMode: neo4jdriver.AccessModeWrite,
	})
	defer session.Close(ctx)

	query := `
		CREATE (f:Feedback {
			id: $id,
			memory_id: $memory_id,
			type: $type,
			comment: $comment,
			session_id: $session_id,
			user_id: $user_id,
			created_at: datetime($created_at)
		})
		RETURN f.id
	`

	_, err := session.Run(ctx, query, map[string]interface{}{
		"id":         feedback.ID,
		"memory_id":  feedback.MemoryID,
		"type":       string(feedback.Type),
		"comment":    feedback.Comment,
		"session_id": feedback.SessionID,
		"user_id":    feedback.UserID,
		"created_at": feedback.CreatedAt.Format(time.RFC3339),
	})
	return err
}

func (c *Client) GetFeedbackByType(feedbackType types.FeedbackType, limit int) ([]*types.Feedback, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	session := c.driver.NewSession(ctx, neo4jdriver.SessionConfig{
		AccessMode: neo4jdriver.AccessModeRead,
	})
	defer session.Close(ctx)

	if limit <= 0 {
		limit = 100
	}

	query := `
		MATCH (f:Feedback {type: $type})
		RETURN f.id, f.memory_id, f.type, f.comment, f.session_id, f.user_id, f.created_at
		ORDER BY f.created_at DESC
		LIMIT $limit
	`

	result, err := session.Run(ctx, query, map[string]interface{}{
		"type":  string(feedbackType),
		"limit": int64(limit),
	})
	if err != nil {
		return nil, err
	}

	var feedbacks []*types.Feedback
	for result.Next(ctx) {
		rec := result.Record()
		feedbacks = append(feedbacks, &types.Feedback{
			ID:        rec.Values[0].(string),
			MemoryID:  rec.Values[1].(string),
			Type:      types.FeedbackType(rec.Values[2].(string)),
			Comment:   rec.Values[3].(string),
			SessionID: rec.Values[4].(string),
			UserID:    rec.Values[5].(string),
			CreatedAt: rec.Values[6].(time.Time),
		})
	}
	return feedbacks, nil
}

// ==================== History Operations ====================

func (c *Client) RecordHistory(memoryID, action, oldValue, newValue, changedBy, reason string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	session := c.driver.NewSession(ctx, neo4jdriver.SessionConfig{
		AccessMode: neo4jdriver.AccessModeWrite,
	})
	defer session.Close(ctx)

	historyID := uuid.New().String()
	metadata := map[string]interface{}{
		"reason": reason,
	}
	metadataJSON, _ := json.Marshal(metadata)

	query := `
		MATCH (m:Memory {id: $memory_id})
		CREATE (h:MemoryHistory {
			id: $id,
			memory_id: $memory_id,
			action: $action,
			old_value: $old_value,
			new_value: $new_value,
			changed_by: $changed_by,
			metadata: $metadata,
			created_at: datetime($created_at)
		})
		CREATE (m)-[:HAS_HISTORY]->(h)
	`

	_, err := session.Run(ctx, query, map[string]interface{}{
		"id":         historyID,
		"memory_id":  memoryID,
		"action":     action,
		"old_value":  oldValue,
		"new_value":  newValue,
		"changed_by": changedBy,
		"metadata":   string(metadataJSON),
		"created_at": time.Now().Format(time.RFC3339),
	})
	return err
}

func (c *Client) GetMemoryHistory(memoryID string) ([]types.MemoryHistory, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	session := c.driver.NewSession(ctx, neo4jdriver.SessionConfig{
		AccessMode: neo4jdriver.AccessModeRead,
	})
	defer session.Close(ctx)

	query := `
		MATCH (m:Memory {id: $memory_id})-[:HAS_HISTORY]->(h:MemoryHistory)
		RETURN h.id, h.memory_id, h.action, h.old_value, h.new_value, h.changed_by, h.metadata, h.created_at
		ORDER BY h.created_at DESC
	`

	result, err := session.Run(ctx, query, map[string]interface{}{"memory_id": memoryID})
	if err != nil {
		return nil, err
	}

	var history []types.MemoryHistory
	for result.Next(ctx) {
		rec := result.Record()
		metadata := make(map[string]interface{})
		if rec.Values[6] != nil {
			if metaStr, ok := rec.Values[6].(string); ok {
				_ = json.Unmarshal([]byte(metaStr), &metadata)
			}
		}
		history = append(history, types.MemoryHistory{
			ID:        rec.Values[0].(string),
			MemoryID:  rec.Values[1].(string),
			Action:    types.HistoryAction(rec.Values[2].(string)),
			OldValue:  rec.Values[3].(string),
			NewValue:  rec.Values[4].(string),
			ChangedBy: rec.Values[5].(string),
			Metadata:  metadata,
			CreatedAt: rec.Values[7].(time.Time),
		})
	}
	return history, nil
}

// ==================== Advanced Search ====================

func (c *Client) AdvancedSearch(filters *types.SearchFilters) ([]*types.Memory, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	session := c.driver.NewSession(ctx, neo4jdriver.SessionConfig{
		AccessMode: neo4jdriver.AccessModeRead,
	})
	defer session.Close(ctx)

	whereClause, params := c.buildWhereClause(filters, 0)

	query := fmt.Sprintf(`
		MATCH (m:Memory)
		WHERE %s
		RETURN m.id, m.tenant_id, m.user_id, m.org_id, m.agent_id, m.session_id,
		       m.type, m.content, m.category, m.metadata, m.status, m.immutable,
		       m.expiration_date, m.feedback_score, m.created_at, m.updated_at, m.last_accessed
		ORDER BY m.created_at DESC
		LIMIT 1000
	`, whereClause)

	result, err := session.Run(ctx, query, params)
	if err != nil {
		return nil, err
	}

	var memories []*types.Memory
	for result.Next(ctx) {
		if mem, err := c.recordToMemoryPtr(result.Record()); err == nil {
			memories = append(memories, mem)
		}
	}
	return memories, nil
}

func (c *Client) buildWhereClause(filters *types.SearchFilters, depth int) (string, map[string]interface{}) {
	conditions := []string{}
	params := map[string]interface{}{}
	paramIndex := 0

	for _, rule := range filters.Rules {
		paramName := fmt.Sprintf("p%d", paramIndex)
		paramIndex++

		switch rule.Operator {
		case "eq", "==", "=":
			conditions = append(conditions, fmt.Sprintf("m.%s = $%s", rule.Field, paramName))
			params[paramName] = rule.Value
		case "ne", "!=":
			conditions = append(conditions, fmt.Sprintf("m.%s <> $%s", rule.Field, paramName))
			params[paramName] = rule.Value
		case "gt", ">":
			conditions = append(conditions, fmt.Sprintf("m.%s > $%s", rule.Field, paramName))
			params[paramName] = rule.Value
		case "gte", ">=":
			conditions = append(conditions, fmt.Sprintf("m.%s >= $%s", rule.Field, paramName))
			params[paramName] = rule.Value
		case "lt", "<":
			conditions = append(conditions, fmt.Sprintf("m.%s < $%s", rule.Field, paramName))
			params[paramName] = rule.Value
		case "lte", "<=":
			conditions = append(conditions, fmt.Sprintf("m.%s <= $%s", rule.Field, paramName))
			params[paramName] = rule.Value
		case "contains":
			conditions = append(conditions, fmt.Sprintf("m.%s CONTAINS $%s", rule.Field, paramName))
			params[paramName] = rule.Value
		case "icontains":
			conditions = append(conditions, fmt.Sprintf("toLower(m.%s) CONTAINS toLower($%s)", rule.Field, paramName))
			params[paramName] = rule.Value
		case "in":
			if values, ok := rule.Value.([]interface{}); ok {
				conditions = append(conditions, fmt.Sprintf("m.%s IN $%s", rule.Field, paramName))
				params[paramName] = values
			}
		case "starts_with":
			conditions = append(conditions, fmt.Sprintf("m.%s STARTS WITH $%s", rule.Field, paramName))
			params[paramName] = rule.Value
		case "ends_with":
			conditions = append(conditions, fmt.Sprintf("m.%s ENDS WITH $%s", rule.Field, paramName))
			params[paramName] = rule.Value
		}
	}

	logic := "AND"
	if filters.Logic == types.FilterLogicOr {
		logic = "OR"
	} else if filters.Logic == types.FilterLogicNot {
		logic = "NOT"
	}

	if len(conditions) > 0 {
		return strings.Join(conditions, " "+logic+" "), params
	}
	return "true", params
}

// ==================== Helper Methods ====================

func (c *Client) recordToMemory(rec *neo4jdriver.Record) (*types.Memory, error) {
	metadata := make(map[string]interface{})
	if rec.Values[9] != nil {
		if metaStr, ok := rec.Values[9].(string); ok {
			_ = json.Unmarshal([]byte(metaStr), &metadata)
		}
	}

	expirationDate := parseTime(rec.Values[12])
	lastAccessed := parseTime(rec.Values[16])

	return &types.Memory{
		ID:             rec.Values[0].(string),
		TenantID:       getString(rec.Values[1]),
		UserID:         getString(rec.Values[2]),
		OrgID:          getString(rec.Values[3]),
		AgentID:        getString(rec.Values[4]),
		SessionID:      getString(rec.Values[5]),
		Type:           types.MemoryType(getString(rec.Values[6])),
		Content:        getString(rec.Values[7]),
		Category:       getString(rec.Values[8]),
		Metadata:       metadata,
		Status:         types.MemoryStatus(getString(rec.Values[10])),
		Immutable:      getBool(rec.Values[11]),
		ExpirationDate: expirationDate,
		FeedbackScore:  types.FeedbackType(getString(rec.Values[13])),
		CreatedAt:      rec.Values[14].(time.Time),
		UpdatedAt:      rec.Values[15].(time.Time),
		LastAccessed:   lastAccessed,
	}, nil
}

func (c *Client) recordToMemoryPtr(rec *neo4jdriver.Record) (*types.Memory, error) {
	mem, err := c.recordToMemory(rec)
	return mem, err
}

func getString(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func getBool(v interface{}) bool {
	if v == nil {
		return false
	}
	if b, ok := v.(bool); ok {
		return b
	}
	return false
}

func parseTime(v interface{}) *time.Time {
	if v == nil {
		return nil
	}
	if t, ok := v.(time.Time); ok {
		return &t
	}
	return nil
}
