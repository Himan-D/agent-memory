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
			tags: $tags,
			importance: $importance,
			metadata: $metadata,
			status: $status,
			immutable: $immutable,
			expiration_date: $expiration_date,
			feedback_score: $feedback_score,
			parent_memory_id: $parent_memory_id,
			related_memory_ids: $related_memory_ids,
			version: $version,
			access_count: $access_count,
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
		"id":                 mem.ID,
		"tenant_id":          mem.TenantID,
		"user_id":            mem.UserID,
		"org_id":             mem.OrgID,
		"agent_id":           mem.AgentID,
		"session_id":         mem.SessionID,
		"type":               string(mem.Type),
		"content":            mem.Content,
		"category":           mem.Category,
		"tags":               mem.Tags,
		"importance":         string(mem.Importance),
		"metadata":           string(metadataJSON),
		"status":             string(mem.Status),
		"immutable":          mem.Immutable,
		"expiration_date":    expirationDate,
		"feedback_score":     string(mem.FeedbackScore),
		"parent_memory_id":   mem.ParentMemoryID,
		"related_memory_ids": mem.RelatedMemoryIDs,
		"version":            mem.Version,
		"access_count":       mem.AccessCount,
		"created_at":         mem.CreatedAt.Format(time.RFC3339),
		"updated_at":         mem.UpdatedAt.Format(time.RFC3339),
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
		       m.type, m.content, m.category, m.tags, m.importance, m.metadata, m.status, m.immutable,
		       m.expiration_date, m.feedback_score, m.parent_memory_id, m.related_memory_ids,
		       m.version, m.access_count, m.created_at, m.updated_at, m.last_accessed
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
		       m.type, m.content, m.category, m.tags, m.importance, m.metadata, m.status, m.immutable,
		       m.expiration_date, m.feedback_score, m.parent_memory_id, m.related_memory_ids,
		       m.version, m.access_count, m.created_at, m.updated_at, m.last_accessed
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

// ==================== Memory Links ====================

func (c *Client) CreateMemoryLink(link *types.MemoryLink) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	session := c.driver.NewSession(ctx, neo4jdriver.SessionConfig{
		AccessMode: neo4jdriver.AccessModeWrite,
	})
	defer session.Close(ctx)

	query := `
		MATCH (m:Memory {id: $from_id}), (n:Memory {id: $to_id})
		CREATE (m)-[r:MEMORY_LINK {id: $id, type: $type, weight: $weight}]->(n)
		SET r += $metadata
	`

	metadataStr := "{}"
	if link.Metadata != nil {
		if data, err := json.Marshal(link.Metadata); err == nil {
			metadataStr = string(data)
		}
	}

	_, err := session.Run(ctx, query, map[string]interface{}{
		"id":       link.ID,
		"from_id":  link.FromID,
		"to_id":    link.ToID,
		"type":     string(link.Type),
		"weight":   link.Weight,
		"metadata": metadataStr,
	})

	return err
}

func (c *Client) GetMemoryLinks(memoryID string) ([]types.MemoryLink, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	session := c.driver.NewSession(ctx, neo4jdriver.SessionConfig{
		AccessMode: neo4jdriver.AccessModeRead,
	})
	defer session.Close(ctx)

	query := `
		MATCH (m:Memory {id: $memory_id})-[r:MEMORY_LINK]-(related:Memory)
		RETURN r.id, r.from_id, r.to_id, r.type, r.weight, r.metadata
	`

	result, err := session.Run(ctx, query, map[string]interface{}{
		"memory_id": memoryID,
	})
	if err != nil {
		return nil, err
	}

	var links []types.MemoryLink
	for result.Next(ctx) {
		rec := result.Record()
		metadata := make(map[string]interface{})
		if rec.Values[5] != nil {
			if metaStr, ok := rec.Values[5].(string); ok {
				_ = json.Unmarshal([]byte(metaStr), &metadata)
			}
		}

		links = append(links, types.MemoryLink{
			ID:       rec.Values[0].(string),
			FromID:   rec.Values[1].(string),
			ToID:     rec.Values[2].(string),
			Type:     types.MemoryLinkType(rec.Values[3].(string)),
			Weight:   rec.Values[4].(float64),
			Metadata: metadata,
		})
	}

	return links, nil
}

func (c *Client) DeleteMemoryLink(linkID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	session := c.driver.NewSession(ctx, neo4jdriver.SessionConfig{
		AccessMode: neo4jdriver.AccessModeWrite,
	})
	defer session.Close(ctx)

	query := `
		MATCH ()-[r:MEMORY_LINK {id: $link_id}]-()
		DELETE r
	`

	_, err := session.Run(ctx, query, map[string]interface{}{
		"link_id": linkID,
	})

	return err
}

// ==================== Memory Versions ====================

func (c *Client) CreateMemoryVersion(version *types.MemoryVersion) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	session := c.driver.NewSession(ctx, neo4jdriver.SessionConfig{
		AccessMode: neo4jdriver.AccessModeWrite,
	})
	defer session.Close(ctx)

	query := `
		MATCH (m:Memory {id: $memory_id})
		CREATE (m)-[:HAS_VERSION]->(v:MemoryVersion)
		SET v.id = $id,
		    v.version = $version,
		    v.content = $content,
		    v.created_by = $created_by,
		    v.created_at = $created_at
	`

	metadataStr := "{}"
	if version.Metadata != nil {
		if data, err := json.Marshal(version.Metadata); err == nil {
			metadataStr = string(data)
		}
	}

	_, err := session.Run(ctx, query, map[string]interface{}{
		"id":         version.ID,
		"memory_id":  version.MemoryID,
		"version":    version.Version,
		"content":    version.Content,
		"metadata":   metadataStr,
		"created_by": version.CreatedBy,
		"created_at": version.CreatedAt,
	})

	return err
}

func (c *Client) GetMemoryVersions(memoryID string) ([]types.MemoryVersion, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	session := c.driver.NewSession(ctx, neo4jdriver.SessionConfig{
		AccessMode: neo4jdriver.AccessModeRead,
	})
	defer session.Close(ctx)

	query := `
		MATCH (m:Memory {id: $memory_id})-[:HAS_VERSION]->(v:MemoryVersion)
		RETURN v.id, v.memory_id, v.version, v.content, v.metadata, v.created_by, v.created_at
		ORDER BY v.version DESC
	`

	result, err := session.Run(ctx, query, map[string]interface{}{
		"memory_id": memoryID,
	})
	if err != nil {
		return nil, err
	}

	var versions []types.MemoryVersion
	for result.Next(ctx) {
		rec := result.Record()
		metadata := make(map[string]interface{})
		if rec.Values[4] != nil {
			if metaStr, ok := rec.Values[4].(string); ok {
				_ = json.Unmarshal([]byte(metaStr), &metadata)
			}
		}

		versions = append(versions, types.MemoryVersion{
			ID:        rec.Values[0].(string),
			MemoryID:  rec.Values[1].(string),
			Version:   int(rec.Values[2].(int64)),
			Content:   rec.Values[3].(string),
			Metadata:  metadata,
			CreatedBy: getString(rec.Values[5]),
			CreatedAt: rec.Values[6].(time.Time),
		})
	}

	return versions, nil
}

func (c *Client) recordToMemory(rec *neo4jdriver.Record) (*types.Memory, error) {
	metadata := make(map[string]interface{})
	if rec.Values[11] != nil {
		if metaStr, ok := rec.Values[11].(string); ok {
			_ = json.Unmarshal([]byte(metaStr), &metadata)
		}
	}

	expirationDate := parseTime(rec.Values[17])
	lastAccessed := parseTime(rec.Values[23])

	return &types.Memory{
		ID:               rec.Values[0].(string),
		TenantID:         getString(rec.Values[1]),
		UserID:           getString(rec.Values[2]),
		OrgID:            getString(rec.Values[3]),
		AgentID:          getString(rec.Values[4]),
		SessionID:        getString(rec.Values[5]),
		Type:             types.MemoryType(getString(rec.Values[6])),
		Content:          getString(rec.Values[7]),
		Category:         getString(rec.Values[8]),
		Tags:             getStringSlice(rec.Values[9]),
		Importance:       types.ImportanceLevel(getString(rec.Values[10])),
		Metadata:         metadata,
		Status:           types.MemoryStatus(getString(rec.Values[12])),
		Immutable:        getBool(rec.Values[13]),
		ExpirationDate:   expirationDate,
		FeedbackScore:    types.FeedbackType(getString(rec.Values[14])),
		ParentMemoryID:   getString(rec.Values[15]),
		RelatedMemoryIDs: getStringSlice(rec.Values[16]),
		Version:          getInt(rec.Values[18]),
		AccessCount:      getInt64(rec.Values[19]),
		CreatedAt:        rec.Values[20].(time.Time),
		UpdatedAt:        rec.Values[21].(time.Time),
		LastAccessed:     lastAccessed,
	}, nil
}

func (c *Client) recordToMemoryPtr(rec *neo4jdriver.Record) (*types.Memory, error) {
	mem, err := c.recordToMemory(rec)
	return mem, err
}

// ==================== Skill Methods ====================

func (c *Client) CreateSkill(ctx context.Context, skill *types.Skill) error {
	if skill.ID == "" {
		skill.ID = uuid.New().String()
	}
	skill.CreatedAt = time.Now()
	skill.UpdatedAt = time.Now()
	skill.UsageCount = 0

	query := `
		CREATE (s:Skill {
			id: $id,
			tenant_id: $tenant_id,
			group_id: $group_id,
			name: $name,
			domain: $domain,
			trigger: $trigger,
			action: $action,
			confidence: $confidence,
			usage_count: $usage_count,
			source_memory: $source_memory,
			created_by: $created_by,
			verified: $verified,
			human_reviewed: $human_reviewed,
			version: $version,
			tags: $tags,
			examples: $examples,
			metadata: $metadata,
			created_at: datetime($created_at),
			updated_at: datetime($updated_at)
		})
		RETURN s.id`

	tags, _ := json.Marshal(skill.Tags)
	examples, _ := json.Marshal(skill.Examples)
	metadata, _ := json.Marshal(skill.Metadata)

	session, release, err := c.AcquireSession(ctx)
	if err != nil {
		return err
	}
	defer release()

	_, err = session.Run(ctx, query, map[string]interface{}{
		"id":             skill.ID,
		"tenant_id":      skill.TenantID,
		"group_id":       skill.GroupID,
		"name":           skill.Name,
		"domain":         skill.Domain,
		"trigger":        skill.Trigger,
		"action":         skill.Action,
		"confidence":     skill.Confidence,
		"usage_count":    skill.UsageCount,
		"source_memory":  skill.SourceMemory,
		"created_by":     skill.CreatedBy,
		"verified":       skill.Verified,
		"human_reviewed": skill.HumanReviewed,
		"version":        skill.Version,
		"tags":           string(tags),
		"examples":       string(examples),
		"metadata":       string(metadata),
		"created_at":     skill.CreatedAt.Format(time.RFC3339),
		"updated_at":     skill.UpdatedAt.Format(time.RFC3339),
	})
	return err
}

func (c *Client) GetSkill(ctx context.Context, skillID string) (*types.Skill, error) {
	query := `
		MATCH (s:Skill {id: $id})
		RETURN s.id, s.tenant_id, s.group_id, s.name, s.domain, s.trigger, s.action,
		       s.confidence, s.usage_count, s.source_memory, s.created_by, s.verified,
		       s.human_reviewed, s.version, s.tags, s.examples, s.metadata,
		       s.created_at, s.updated_at, s.last_used`

	session, release, err := c.AcquireSession(ctx)
	if err != nil {
		return nil, err
	}
	defer release()

	rec, err := session.Run(ctx, query, map[string]interface{}{"id": skillID})
	if err != nil {
		return nil, err
	}

	if !rec.Next(ctx) {
		return nil, fmt.Errorf("skill not found: %s", skillID)
	}

	return c.recordToSkill(rec.Record())
}

func (c *Client) recordToSkill(rec *neo4jdriver.Record) (*types.Skill, error) {
	var tags, examples, metadata []string
	json.Unmarshal([]byte(getString(rec.Values[13])), &tags)
	json.Unmarshal([]byte(getString(rec.Values[14])), &examples)
	json.Unmarshal([]byte(getString(rec.Values[15])), &metadata)

	var metaMap map[string]interface{}
	json.Unmarshal([]byte(getString(rec.Values[15])), &metaMap)

	return &types.Skill{
		ID:            getString(rec.Values[0]),
		TenantID:      getString(rec.Values[1]),
		GroupID:       getString(rec.Values[2]),
		Name:          getString(rec.Values[3]),
		Domain:        getString(rec.Values[4]),
		Trigger:       getString(rec.Values[5]),
		Action:        getString(rec.Values[6]),
		Confidence:    getFloat32(rec.Values[7]),
		UsageCount:    getInt64(rec.Values[8]),
		SourceMemory:  getString(rec.Values[9]),
		CreatedBy:     getString(rec.Values[10]),
		Verified:      getBool(rec.Values[11]),
		HumanReviewed: getBool(rec.Values[12]),
		Version:       getInt(rec.Values[13]),
		Tags:          tags,
		Examples:      examples,
		Metadata:      metaMap,
		CreatedAt:     getTime(rec.Values[16]),
		UpdatedAt:     getTime(rec.Values[17]),
		LastUsed:      parseTime(rec.Values[18]),
	}, nil
}

func (c *Client) GetSkillsByTrigger(ctx context.Context, trigger string, limit int) ([]*types.Skill, error) {
	query := `
		MATCH (s:Skill)
		WHERE s.trigger CONTAINS $trigger OR $trigger CONTAINS s.trigger
		RETURN s.id, s.tenant_id, s.group_id, s.name, s.domain, s.trigger, s.action,
		       s.confidence, s.usage_count, s.source_memory, s.created_by, s.verified,
		       s.human_reviewed, s.version, s.tags, s.examples, s.metadata,
		       s.created_at, s.updated_at, s.last_used
		ORDER BY s.confidence DESC, s.usage_count DESC
		LIMIT $limit`

	session, release, err := c.AcquireSession(ctx)
	if err != nil {
		return nil, err
	}
	defer release()

	rec, err := session.Run(ctx, query, map[string]interface{}{
		"trigger": trigger,
		"limit":   limit,
	})
	if err != nil {
		return nil, err
	}

	var skills []*types.Skill
	for rec.Next(ctx) {
		skill, err := c.recordToSkill(rec.Record())
		if err != nil {
			continue
		}
		skills = append(skills, skill)
	}

	return skills, nil
}

func (c *Client) GetSkillsByDomain(ctx context.Context, domain string, limit int) ([]*types.Skill, error) {
	query := `
		MATCH (s:Skill {domain: $domain})
		RETURN s.id, s.tenant_id, s.group_id, s.name, s.domain, s.trigger, s.action,
		       s.confidence, s.usage_count, s.source_memory, s.created_by, s.verified,
		       s.human_reviewed, s.version, s.tags, s.examples, s.metadata,
		       s.created_at, s.updated_at, s.last_used
		ORDER BY s.usage_count DESC
		LIMIT $limit`

	session, release, err := c.AcquireSession(ctx)
	if err != nil {
		return nil, err
	}
	defer release()

	rec, err := session.Run(ctx, query, map[string]interface{}{
		"domain": domain,
		"limit":  limit,
	})
	if err != nil {
		return nil, err
	}

	var skills []*types.Skill
	for rec.Next(ctx) {
		skill, err := c.recordToSkill(rec.Record())
		if err != nil {
			continue
		}
		skills = append(skills, skill)
	}

	return skills, nil
}

func (c *Client) IncrementSkillUsage(ctx context.Context, skillID string) error {
	query := `
		MATCH (s:Skill {id: $id})
		SET s.usage_count = s.usage_count + 1,
		    s.last_used = datetime(),
		    s.updated_at = datetime()
		RETURN s.usage_count`

	session, release, err := c.AcquireSession(ctx)
	if err != nil {
		return err
	}
	defer release()

	_, err = session.Run(ctx, query, map[string]interface{}{"id": skillID})
	return err
}

func (c *Client) UpdateSkill(ctx context.Context, skill *types.Skill) error {
	skill.UpdatedAt = time.Now()

	query := `
		MATCH (s:Skill {id: $id})
		SET s.name = $name,
		    s.domain = $domain,
		    s.trigger = $trigger,
		    s.action = $action,
		    s.confidence = $confidence,
		    s.verified = $verified,
		    s.human_reviewed = $human_reviewed,
		    s.version = s.version + 1,
		    s.tags = $tags,
		    s.examples = $examples,
		    s.metadata = $metadata,
		    s.updated_at = datetime()`

	tags, _ := json.Marshal(skill.Tags)
	examples, _ := json.Marshal(skill.Examples)
	metadata, _ := json.Marshal(skill.Metadata)

	session, release, err := c.AcquireSession(ctx)
	if err != nil {
		return err
	}
	defer release()

	_, err = session.Run(ctx, query, map[string]interface{}{
		"id":             skill.ID,
		"name":           skill.Name,
		"domain":         skill.Domain,
		"trigger":        skill.Trigger,
		"action":         skill.Action,
		"confidence":     skill.Confidence,
		"verified":       skill.Verified,
		"human_reviewed": skill.HumanReviewed,
		"tags":           string(tags),
		"examples":       string(examples),
		"metadata":       string(metadata),
	})
	return err
}

func (c *Client) DeleteSkill(ctx context.Context, skillID string) error {
	query := `MATCH (s:Skill {id: $id}) DELETE s`

	session, release, err := c.AcquireSession(ctx)
	if err != nil {
		return err
	}
	defer release()

	_, err = session.Run(ctx, query, map[string]interface{}{"id": skillID})
	return err
}

func (c *Client) GetSimilarSkills(ctx context.Context, skillID string, limit int) ([]*types.Skill, error) {
	query := `
		MATCH (s1:Skill {id: $id}), (s2:Skill)
		WHERE s1 <> s2
		  AND (s1.domain = s2.domain OR s1.trigger = s2.trigger)
		RETURN s2.id, s2.tenant_id, s2.group_id, s2.name, s2.domain, s2.trigger, s2.action,
		       s2.confidence, s2.usage_count, s2.source_memory, s2.created_by, s2.verified,
		       s2.human_reviewed, s2.version, s2.tags, s2.examples, s2.metadata,
		       s2.created_at, s2.updated_at, s2.last_used
		ORDER BY s2.confidence DESC
		LIMIT $limit`

	session, release, err := c.AcquireSession(ctx)
	if err != nil {
		return nil, err
	}
	defer release()

	rec, err := session.Run(ctx, query, map[string]interface{}{
		"id":    skillID,
		"limit": limit,
	})
	if err != nil {
		return nil, err
	}

	var skills []*types.Skill
	for rec.Next(ctx) {
		skill, err := c.recordToSkill(rec.Record())
		if err != nil {
			continue
		}
		skills = append(skills, skill)
	}

	return skills, nil
}

// ==================== Agent Methods ====================

func (c *Client) CreateAgent(ctx context.Context, agent *types.Agent) error {
	if agent.ID == "" {
		agent.ID = uuid.New().String()
	}
	agent.CreatedAt = time.Now()
	agent.UpdatedAt = time.Now()
	agent.Status = types.AgentStatusActive

	query := `
		CREATE (a:Agent {
			id: $id,
			tenant_id: $tenant_id,
			name: $name,
			description: $description,
			status: $status,
			groups: $groups,
			metadata: $metadata,
			created_at: datetime($created_at),
			updated_at: datetime($updated_at)
		})
		RETURN a.id`

	groups, _ := json.Marshal(agent.Groups)
	metadata, _ := json.Marshal(agent.Metadata)

	session, release, err := c.AcquireSession(ctx)
	if err != nil {
		return err
	}
	defer release()

	_, err = session.Run(ctx, query, map[string]interface{}{
		"id":          agent.ID,
		"tenant_id":   agent.TenantID,
		"name":        agent.Name,
		"description": agent.Description,
		"status":      string(agent.Status),
		"groups":      string(groups),
		"metadata":    string(metadata),
		"created_at":  agent.CreatedAt.Format(time.RFC3339),
		"updated_at":  agent.UpdatedAt.Format(time.RFC3339),
	})
	return err
}

func (c *Client) GetAgent(ctx context.Context, agentID string) (*types.Agent, error) {
	query := `
		MATCH (a:Agent {id: $id})
		RETURN a.id, a.tenant_id, a.name, a.description, a.status, a.groups, a.metadata,
		       a.created_at, a.updated_at, a.last_active`

	session, release, err := c.AcquireSession(ctx)
	if err != nil {
		return nil, err
	}
	defer release()

	rec, err := session.Run(ctx, query, map[string]interface{}{"id": agentID})
	if err != nil {
		return nil, err
	}
	if !rec.Next(ctx) {
		return nil, fmt.Errorf("agent not found: %s", agentID)
	}

	return c.recordToAgent(rec.Record())
}

func (c *Client) recordToAgent(rec *neo4jdriver.Record) (*types.Agent, error) {
	var groups []string
	var metadata map[string]interface{}
	json.Unmarshal([]byte(getString(rec.Values[5])), &groups)
	json.Unmarshal([]byte(getString(rec.Values[6])), &metadata)

	return &types.Agent{
		ID:          getString(rec.Values[0]),
		TenantID:    getString(rec.Values[1]),
		Name:        getString(rec.Values[2]),
		Description: getString(rec.Values[3]),
		Status:      types.AgentStatus(getString(rec.Values[4])),
		Groups:      groups,
		Metadata:    metadata,
		CreatedAt:   getTime(rec.Values[7]),
		UpdatedAt:   getTime(rec.Values[8]),
		LastActive:  parseTime(rec.Values[9]),
	}, nil
}

func (c *Client) UpdateAgent(ctx context.Context, agent *types.Agent) error {
	agent.UpdatedAt = time.Now()

	query := `
		MATCH (a:Agent {id: $id})
		SET a.name = $name,
		    a.description = $description,
		    a.status = $status,
		    a.groups = $groups,
		    a.metadata = $metadata,
		    a.updated_at = datetime()`

	groups, _ := json.Marshal(agent.Groups)
	metadata, _ := json.Marshal(agent.Metadata)

	session, release, err := c.AcquireSession(ctx)
	if err != nil {
		return err
	}
	defer release()

	_, err = session.Run(ctx, query, map[string]interface{}{
		"id":          agent.ID,
		"name":        agent.Name,
		"description": agent.Description,
		"status":      string(agent.Status),
		"groups":      string(groups),
		"metadata":    string(metadata),
	})
	return err
}

func (c *Client) DeleteAgent(ctx context.Context, agentID string) error {
	query := `
		MATCH (a:Agent {id: $id})
		SET a.status = $status,
		    a.updated_at = datetime()`

	session, release, err := c.AcquireSession(ctx)
	if err != nil {
		return err
	}
	defer release()

	_, err = session.Run(ctx, query, map[string]interface{}{
		"id":     agentID,
		"status": string(types.AgentStatusInactive),
	})
	return err
}

func (c *Client) ListAgents(ctx context.Context, tenantID string, limit, offset int) ([]*types.Agent, int64, error) {
	countQuery := `MATCH (a:Agent) WHERE a.tenant_id = $tenant_id AND a.status <> $inactive RETURN count(a)`
	listQuery := `
		MATCH (a:Agent)
		WHERE a.tenant_id = $tenant_id AND a.status <> $inactive
		RETURN a.id, a.tenant_id, a.name, a.description, a.status, a.groups, a.metadata,
		       a.created_at, a.updated_at, a.last_active
		ORDER BY a.created_at DESC
		SKIP $offset LIMIT $limit`

	session, release, err := c.AcquireSession(ctx)
	if err != nil {
		return nil, 0, err
	}
	defer release()

	countRec, err := session.Run(ctx, countQuery, map[string]interface{}{
		"tenant_id": tenantID,
		"inactive":  string(types.AgentStatusInactive),
	})
	if err != nil {
		return nil, 0, err
	}

	var total int64
	if countRec.Next(ctx) {
		total = getInt64(countRec.Record().Values[0])
	}

	rec, err := session.Run(ctx, listQuery, map[string]interface{}{
		"tenant_id": tenantID,
		"inactive":  string(types.AgentStatusInactive),
		"offset":    offset,
		"limit":     limit,
	})
	if err != nil {
		return nil, 0, err
	}

	var agents []*types.Agent
	for rec.Next(ctx) {
		agent, err := c.recordToAgent(rec.Record())
		if err != nil {
			continue
		}
		agents = append(agents, agent)
	}

	return agents, total, nil
}

// ==================== Agent Group Methods ====================

func (c *Client) CreateAgentGroup(ctx context.Context, group *types.AgentGroup) error {
	if group.ID == "" {
		group.ID = uuid.New().String()
	}
	group.CreatedAt = time.Now()
	group.UpdatedAt = time.Now()

	query := `
		CREATE (g:AgentGroup {
			id: $id,
			tenant_id: $tenant_id,
			name: $name,
			description: $description,
			domain: $domain,
			policy: $policy,
			memory_pool_id: $memory_pool_id,
			metadata: $metadata,
			created_at: datetime($created_at),
			updated_at: datetime($updated_at)
		})
		RETURN g.id`

	policy, _ := json.Marshal(group.Policy)
	metadata, _ := json.Marshal(group.Metadata)

	session, release, err := c.AcquireSession(ctx)
	if err != nil {
		return err
	}
	defer release()

	_, err = session.Run(ctx, query, map[string]interface{}{
		"id":             group.ID,
		"tenant_id":      group.TenantID,
		"name":           group.Name,
		"description":    group.Description,
		"domain":         group.Domain,
		"policy":         string(policy),
		"memory_pool_id": group.MemoryPoolID,
		"metadata":       string(metadata),
		"created_at":     group.CreatedAt.Format(time.RFC3339),
		"updated_at":     group.UpdatedAt.Format(time.RFC3339),
	})
	if err != nil {
		return err
	}

	for _, member := range group.Members {
		if err := c.AddAgentToGroup(ctx, member.AgentID, group.ID, member.Role); err != nil {
			return err
		}
	}

	return nil
}

func (c *Client) GetAgentGroup(ctx context.Context, groupID string) (*types.AgentGroup, error) {
	query := `
		MATCH (g:AgentGroup {id: $id})
		OPTIONAL MATCH (a:Agent)-[r:MEMBER_OF]->(g)
		RETURN g.id, g.tenant_id, g.name, g.description, g.domain, g.policy,
		       g.memory_pool_id, g.metadata, g.created_at, g.updated_at,
		       collect(CASE WHEN a IS NOT NULL THEN {agent_id: a.id, role: r.role, joined_at: r.joined_at} END) as members`

	session, release, err := c.AcquireSession(ctx)
	if err != nil {
		return nil, err
	}
	defer release()

	rec, err := session.Run(ctx, query, map[string]interface{}{"id": groupID})
	if err != nil {
		return nil, err
	}
	if !rec.Next(ctx) {
		return nil, fmt.Errorf("group not found: %s", groupID)
	}

	return c.recordToAgentGroup(rec.Record())
}

func (c *Client) recordToAgentGroup(rec *neo4jdriver.Record) (*types.AgentGroup, error) {
	var policy map[string]interface{}
	var metadata map[string]interface{}
	var members []types.AgentMember

	json.Unmarshal([]byte(getString(rec.Values[5])), &policy)
	json.Unmarshal([]byte(getString(rec.Values[7])), &metadata)

	membersData, ok := rec.Values[9].([]interface{})
	if ok {
		for _, m := range membersData {
			if m == nil {
				continue
			}
			if memberMap, ok := m.(map[string]interface{}); ok {
				members = append(members, types.AgentMember{
					AgentID:  getString(memberMap["agent_id"]),
					Role:     types.MemberRole(getString(memberMap["role"])),
					JoinedAt: getTime(memberMap["joined_at"]),
				})
			}
		}
	}

	return &types.AgentGroup{
		ID:           getString(rec.Values[0]),
		TenantID:     getString(rec.Values[1]),
		Name:         getString(rec.Values[2]),
		Description:  getString(rec.Values[3]),
		Domain:       getString(rec.Values[4]),
		Policy:       types.GroupPolicy{},
		MemoryPoolID: getString(rec.Values[6]),
		Metadata:     metadata,
		Members:      members,
		CreatedAt:    getTime(rec.Values[8]),
		UpdatedAt:    getTime(rec.Values[9]),
	}, nil
}

func (c *Client) AddAgentToGroup(ctx context.Context, agentID, groupID string, role types.MemberRole) error {
	query := `
		MATCH (a:Agent {id: $agent_id})
		MATCH (g:AgentGroup {id: $group_id})
		CREATE (a)-[r:MEMBER_OF {role: $role, joined_at: datetime()}]->(g)
		SET a.groups = [g.id] + COALESCE(a.groups, [])`

	session, release, err := c.AcquireSession(ctx)
	if err != nil {
		return err
	}
	defer release()

	_, err = session.Run(ctx, query, map[string]interface{}{
		"agent_id": agentID,
		"group_id": groupID,
		"role":     string(role),
	})
	return err
}

func (c *Client) RemoveAgentFromGroup(ctx context.Context, agentID, groupID string) error {
	query := `
		MATCH (a:Agent {id: $agent_id})-[r:MEMBER_OF]->(g:AgentGroup {id: $group_id})
		DELETE r
		SET a.groups = [x IN a.groups WHERE x <> $group_id]`

	session, release, err := c.AcquireSession(ctx)
	if err != nil {
		return err
	}
	defer release()

	_, err = session.Run(ctx, query, map[string]interface{}{
		"agent_id": agentID,
		"group_id": groupID,
	})
	return err
}

func (c *Client) ListAgentGroups(ctx context.Context, tenantID string, limit, offset int) ([]*types.AgentGroup, int64, error) {
	countQuery := `MATCH (g:AgentGroup) WHERE g.tenant_id = $tenant_id RETURN count(g)`
	listQuery := `
		MATCH (g:AgentGroup)
		WHERE g.tenant_id = $tenant_id
		RETURN g.id, g.tenant_id, g.name, g.description, g.domain, g.policy,
		       g.memory_pool_id, g.metadata, g.created_at, g.updated_at
		ORDER BY g.created_at DESC
		SKIP $offset LIMIT $limit`

	session, release, err := c.AcquireSession(ctx)
	if err != nil {
		return nil, 0, err
	}
	defer release()

	countRec, err := session.Run(ctx, countQuery, map[string]interface{}{
		"tenant_id": tenantID,
	})
	if err != nil {
		return nil, 0, err
	}

	var total int64
	if countRec.Next(ctx) {
		total = getInt64(countRec.Record().Values[0])
	}

	rec, err := session.Run(ctx, listQuery, map[string]interface{}{
		"tenant_id": tenantID,
		"offset":    offset,
		"limit":     limit,
	})
	if err != nil {
		return nil, 0, err
	}

	var groups []*types.AgentGroup
	for rec.Next(ctx) {
		group, err := c.recordToAgentGroupSimple(rec.Record())
		if err != nil {
			continue
		}
		groups = append(groups, group)
	}

	return groups, total, nil
}

func (c *Client) recordToAgentGroupSimple(rec *neo4jdriver.Record) (*types.AgentGroup, error) {
	var policy map[string]interface{}
	var metadata map[string]interface{}

	json.Unmarshal([]byte(getString(rec.Values[5])), &policy)
	json.Unmarshal([]byte(getString(rec.Values[7])), &metadata)

	return &types.AgentGroup{
		ID:           getString(rec.Values[0]),
		TenantID:     getString(rec.Values[1]),
		Name:         getString(rec.Values[2]),
		Description:  getString(rec.Values[3]),
		Domain:       getString(rec.Values[4]),
		Policy:       types.GroupPolicy{},
		MemoryPoolID: getString(rec.Values[6]),
		Metadata:     metadata,
		CreatedAt:    getTime(rec.Values[8]),
		UpdatedAt:    getTime(rec.Values[9]),
	}, nil
}

func (c *Client) ListSkills(ctx context.Context, tenantID, domain string, limit, offset int) ([]*types.Skill, error) {
	var query string
	params := map[string]interface{}{
		"tenant_id": tenantID,
		"limit":     limit,
		"offset":    offset,
	}

	if domain != "" {
		query = `
			MATCH (s:Skill)
			WHERE s.tenant_id = $tenant_id AND s.domain = $domain
			RETURN s.id, s.tenant_id, s.group_id, s.name, s.domain, s.trigger, s.action,
			       s.confidence, s.usage_count, s.source_memory, s.created_by, s.verified,
			       s.human_reviewed, s.version, s.tags, s.examples, s.metadata,
			       s.created_at, s.updated_at, s.last_used
			ORDER BY s.confidence DESC, s.usage_count DESC
			SKIP $offset LIMIT $limit`
		params["domain"] = domain
	} else {
		query = `
			MATCH (s:Skill)
			WHERE s.tenant_id = $tenant_id
			RETURN s.id, s.tenant_id, s.group_id, s.name, s.domain, s.trigger, s.action,
			       s.confidence, s.usage_count, s.source_memory, s.created_by, s.verified,
			       s.human_reviewed, s.version, s.tags, s.examples, s.metadata,
			       s.created_at, s.updated_at, s.last_used
			ORDER BY s.confidence DESC, s.usage_count DESC
			SKIP $offset LIMIT $limit`
	}

	session, release, err := c.AcquireSession(ctx)
	if err != nil {
		return nil, err
	}
	defer release()

	rec, err := session.Run(ctx, query, params)
	if err != nil {
		return nil, err
	}

	var skills []*types.Skill
	for rec.Next(ctx) {
		skill, err := c.recordToSkill(rec.Record())
		if err != nil {
			continue
		}
		skills = append(skills, skill)
	}

	return skills, nil
}

func (c *Client) UpdateAgentGroup(ctx context.Context, group *types.AgentGroup) error {
	group.UpdatedAt = time.Now()

	query := `
		MATCH (g:AgentGroup {id: $id})
		SET g.name = $name,
		    g.description = $description,
		    g.domain = $domain,
		    g.policy = $policy,
		    g.memory_pool_id = $memory_pool_id,
		    g.metadata = $metadata,
		    g.updated_at = datetime()`

	policy, _ := json.Marshal(group.Policy)
	metadata, _ := json.Marshal(group.Metadata)

	session, release, err := c.AcquireSession(ctx)
	if err != nil {
		return err
	}
	defer release()

	_, err = session.Run(ctx, query, map[string]interface{}{
		"id":             group.ID,
		"name":           group.Name,
		"description":    group.Description,
		"domain":         group.Domain,
		"policy":         string(policy),
		"memory_pool_id": group.MemoryPoolID,
		"metadata":       string(metadata),
	})
	return err
}

func (c *Client) DeleteAgentGroup(ctx context.Context, groupID string) error {
	query := `MATCH (g:AgentGroup {id: $id}) DETACH DELETE g`

	session, release, err := c.AcquireSession(ctx)
	if err != nil {
		return err
	}
	defer release()

	_, err = session.Run(ctx, query, map[string]interface{}{"id": groupID})
	return err
}

func (c *Client) GetGroupSkills(ctx context.Context, groupID string, limit int) ([]*types.Skill, error) {
	query := `
		MATCH (g:AgentGroup {id: $group_id})-[:HAS_SKILL]->(s:Skill)
		RETURN s.id, s.tenant_id, s.group_id, s.name, s.domain, s.trigger, s.action,
		       s.confidence, s.usage_count, s.source_memory, s.created_by, s.verified,
		       s.human_reviewed, s.version, s.tags, s.examples, s.metadata,
		       s.created_at, s.updated_at, s.last_used
		ORDER BY s.confidence DESC
		LIMIT $limit`

	session, release, err := c.AcquireSession(ctx)
	if err != nil {
		return nil, err
	}
	defer release()

	rec, err := session.Run(ctx, query, map[string]interface{}{
		"group_id": groupID,
		"limit":    limit,
	})
	if err != nil {
		return nil, err
	}

	var skills []*types.Skill
	for rec.Next(ctx) {
		skill, err := c.recordToSkill(rec.Record())
		if err != nil {
			continue
		}
		skills = append(skills, skill)
	}

	return skills, nil
}

func (c *Client) GetGroupMemories(ctx context.Context, groupID string) ([]*types.Memory, error) {
	query := `
		MATCH (g:AgentGroup {id: $group_id})-[:SHARED_MEMORY]->(m:Memory)
		RETURN m.id, m.tenant_id, m.user_id, m.org_id, m.agent_id, m.session_id,
		       m.type, m.content, m.category, m.tags, m.importance, m.metadata, m.status, m.immutable,
		       m.expiration_date, m.feedback_score, m.parent_memory_id, m.related_memory_ids,
		       m.version, m.access_count, m.created_at, m.updated_at, m.last_accessed
		ORDER BY m.created_at DESC`

	session, release, err := c.AcquireSession(ctx)
	if err != nil {
		return nil, err
	}
	defer release()

	rec, err := session.Run(ctx, query, map[string]interface{}{"group_id": groupID})
	if err != nil {
		return nil, err
	}

	var memories []*types.Memory
	for rec.Next(ctx) {
		mem, err := c.recordToMemoryPtr(rec.Record())
		if err != nil {
			continue
		}
		memories = append(memories, mem)
	}

	return memories, nil
}

func (c *Client) ShareMemoryToGroup(ctx context.Context, memoryID, groupID, sharedBy string) error {
	sharedID := uuid.New().String()

	query := `
		MATCH (m:Memory {id: $memory_id}), (g:AgentGroup {id: $group_id})
		CREATE (g)-[:SHARED_MEMORY {id: $shared_id, shared_by: $shared_by, shared_at: datetime()}]->(m)
		RETURN id(g)`

	session, release, err := c.AcquireSession(ctx)
	if err != nil {
		return err
	}
	defer release()

	_, err = session.Run(ctx, query, map[string]interface{}{
		"memory_id": memoryID,
		"group_id":  groupID,
		"shared_id": sharedID,
		"shared_by": sharedBy,
	})
	return err
}

func (c *Client) ListPendingReviews(ctx context.Context, tenantID string) ([]*types.SkillReview, error) {
	query := `
		MATCH (r:SkillReview {tenant_id: $tenant_id, status: 'pending'})
		RETURN r.id, r.tenant_id, r.skill_id, r.status, r.reviewed_by, r.notes,
		       r.decision, r.created_at, r.reviewed_at
		ORDER BY r.created_at DESC`

	session, release, err := c.AcquireSession(ctx)
	if err != nil {
		return nil, err
	}
	defer release()

	rec, err := session.Run(ctx, query, map[string]interface{}{"tenant_id": tenantID})
	if err != nil {
		return nil, err
	}

	var reviews []*types.SkillReview
	for rec.Next(ctx) {
		r := rec.Record()
		reviews = append(reviews, &types.SkillReview{
			ID:         getString(r.Values[0]),
			TenantID:   getString(r.Values[1]),
			SkillID:    getString(r.Values[2]),
			Status:     types.ReviewStatus(getString(r.Values[3])),
			ReviewedBy: getString(r.Values[4]),
			Notes:      getString(r.Values[5]),
			Decision:   getString(r.Values[6]),
			CreatedAt:  getTime(r.Values[7]),
			ReviewedAt: parseTime(r.Values[8]),
		})
	}

	return reviews, nil
}

func (c *Client) GetReview(ctx context.Context, reviewID string) (*types.SkillReview, error) {
	query := `
		MATCH (r:SkillReview {id: $id})
		RETURN r.id, r.tenant_id, r.skill_id, r.status, r.reviewed_by, r.notes,
		       r.decision, r.created_at, r.reviewed_at`

	session, release, err := c.AcquireSession(ctx)
	if err != nil {
		return nil, err
	}
	defer release()

	rec, err := session.Run(ctx, query, map[string]interface{}{"id": reviewID})
	if err != nil {
		return nil, err
	}

	if !rec.Next(ctx) {
		return nil, fmt.Errorf("review not found: %s", reviewID)
	}

	r := rec.Record()
	return &types.SkillReview{
		ID:         getString(r.Values[0]),
		TenantID:   getString(r.Values[1]),
		SkillID:    getString(r.Values[2]),
		Status:     types.ReviewStatus(getString(r.Values[3])),
		ReviewedBy: getString(r.Values[4]),
		Notes:      getString(r.Values[5]),
		Decision:   getString(r.Values[6]),
		CreatedAt:  getTime(r.Values[7]),
		ReviewedAt: parseTime(r.Values[8]),
	}, nil
}

func (c *Client) ProcessReview(ctx context.Context, reviewID string, approved bool, notes string) error {
	status := "rejected"
	if approved {
		status = "approved"
	}

	query := `
		MATCH (r:SkillReview {id: $id})
		SET r.status = $status,
		    r.decision = $decision,
		    r.notes = $notes,
		    r.reviewed_at = datetime()`

	if approved {
		query += `
			WITH r
			MATCH (s:Skill {id: r.skill_id})
			SET s.human_reviewed = true, s.verified = true`
	}

	session, release, err := c.AcquireSession(ctx)
	if err != nil {
		return err
	}
	defer release()

	_, err = session.Run(ctx, query, map[string]interface{}{
		"id":       reviewID,
		"status":   status,
		"decision": status,
		"notes":    notes,
	})
	return err
}

func (c *Client) CreateSkillReview(ctx context.Context, review *types.SkillReview) error {
	if review.ID == "" {
		review.ID = uuid.New().String()
	}
	review.CreatedAt = time.Now()
	review.Status = types.ReviewStatusPending

	query := `
		CREATE (r:SkillReview {
			id: $id,
			tenant_id: $tenant_id,
			skill_id: $skill_id,
			status: $status,
			notes: $notes,
			created_at: datetime($created_at)
		})
		RETURN r.id`

	session, release, err := c.AcquireSession(ctx)
	if err != nil {
		return err
	}
	defer release()

	_, err = session.Run(ctx, query, map[string]interface{}{
		"id":         review.ID,
		"tenant_id":  review.TenantID,
		"skill_id":   review.SkillID,
		"status":     string(review.Status),
		"notes":      review.Notes,
		"created_at": review.CreatedAt.Format(time.RFC3339),
	})
	return err
}

// ==================== Helper Functions ====================

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

func getInt(v interface{}) int {
	if v == nil {
		return 0
	}
	if i, ok := v.(int64); ok {
		return int(i)
	}
	if i, ok := v.(int); ok {
		return i
	}
	return 0
}

func getFloat32(v interface{}) float32 {
	if v == nil {
		return 0
	}
	if f, ok := v.(float64); ok {
		return float32(f)
	}
	return 0
}

func getTime(v interface{}) time.Time {
	if v == nil {
		return time.Time{}
	}
	if t, ok := v.(time.Time); ok {
		return t
	}
	return time.Time{}
}

func getInt64(v interface{}) int64 {
	if v == nil {
		return 0
	}
	if i, ok := v.(int64); ok {
		return i
	}
	if i, ok := v.(int); ok {
		return int64(i)
	}
	return 0
}

func getStringSlice(v interface{}) []string {
	if v == nil {
		return nil
	}
	if slice, ok := v.([]interface{}); ok {
		result := make([]string, 0, len(slice))
		for _, item := range slice {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	}
	return nil
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

type APIKey struct {
	ID        string     `json:"id"`
	Key       string     `json:"key"`
	Label     string     `json:"label"`
	TenantID  string     `json:"tenant_id"`
	CreatedAt time.Time  `json:"created_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

func (k *APIKey) IsExpired() bool {
	if k.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*k.ExpiresAt)
}

type APIKeyStore interface {
	Create(ctx context.Context, key *APIKey) error
	Get(ctx context.Context, id string) (*APIKey, error)
	GetByKey(ctx context.Context, key string) (*APIKey, error)
	List(ctx context.Context) ([]*APIKey, error)
	ListByTenant(ctx context.Context, tenantID string) ([]*APIKey, error)
	Delete(ctx context.Context, id string) error
	DeleteExpired(ctx context.Context) (int, error)
}

func (c *Client) CreateAPIKey(ctx context.Context, key *APIKey) error {
	if key.ID == "" {
		key.ID = fmt.Sprintf("key_%s", uuid.New().String())
	}
	key.CreatedAt = time.Now()

	query := `
		CREATE (k:APIKey {
			id: $id,
			key_hash: $key_hash,
			label: $label,
			tenant_id: $tenant_id,
			created_at: datetime($created_at),
			expires_at: $expires_at
		})
		RETURN k.id
	`

	keyHash := hashAPIKey(key.Key)

	session, release, err := c.AcquireSession(ctx)
	if err != nil {
		return err
	}
	defer release()

	_, err = session.Run(ctx, query, map[string]interface{}{
		"id":         key.ID,
		"key_hash":   keyHash,
		"label":      key.Label,
		"tenant_id":  key.TenantID,
		"created_at": key.CreatedAt.Format(time.RFC3339),
		"expires_at": nilIfZeroTime(key.ExpiresAt),
	})
	return err
}

func (c *Client) GetAPIKey(ctx context.Context, id string) (*APIKey, error) {
	query := `
		MATCH (k:APIKey {id: $id})
		RETURN k.id, k.key_hash, k.label, k.tenant_id, k.created_at, k.expires_at
	`

	session, release, err := c.AcquireSession(ctx)
	if err != nil {
		return nil, err
	}
	defer release()

	rec, err := session.Run(ctx, query, map[string]interface{}{"id": id})
	if err != nil {
		return nil, err
	}
	if !rec.Next(ctx) {
		return nil, fmt.Errorf("api key not found: %s", id)
	}

	return c.recordToAPIKey(rec.Record())
}

func (c *Client) GetAPIKeyByKey(ctx context.Context, keyStr string) (*APIKey, error) {
	keyHash := hashAPIKey(keyStr)

	query := `
		MATCH (k:APIKey {key_hash: $key_hash})
		RETURN k.id, k.key_hash, k.label, k.tenant_id, k.created_at, k.expires_at
	`

	session, release, err := c.AcquireSession(ctx)
	if err != nil {
		return nil, err
	}
	defer release()

	rec, err := session.Run(ctx, query, map[string]interface{}{"key_hash": keyHash})
	if err != nil {
		return nil, err
	}
	if !rec.Next(ctx) {
		return nil, fmt.Errorf("api key not found")
	}

	return c.recordToAPIKey(rec.Record())
}

func (c *Client) ListAPIKeys(ctx context.Context) ([]*APIKey, error) {
	query := `
		MATCH (k:APIKey)
		RETURN k.id, k.key_hash, k.label, k.tenant_id, k.created_at, k.expires_at
		ORDER BY k.created_at DESC
	`

	session, release, err := c.AcquireSession(ctx)
	if err != nil {
		return nil, err
	}
	defer release()

	recs, err := session.Run(ctx, query, nil)
	if err != nil {
		return nil, err
	}

	var keys []*APIKey
	for recs.Next(ctx) {
		key, err := c.recordToAPIKey(recs.Record())
		if err != nil {
			continue
		}
		keys = append(keys, key)
	}

	return keys, nil
}

func (c *Client) ListAPIKeysByTenant(ctx context.Context, tenantID string) ([]*APIKey, error) {
	query := `
		MATCH (k:APIKey {tenant_id: $tenant_id})
		RETURN k.id, k.key_hash, k.label, k.tenant_id, k.created_at, k.expires_at
		ORDER BY k.created_at DESC
	`

	session, release, err := c.AcquireSession(ctx)
	if err != nil {
		return nil, err
	}
	defer release()

	recs, err := session.Run(ctx, query, map[string]interface{}{"tenant_id": tenantID})
	if err != nil {
		return nil, err
	}

	var keys []*APIKey
	for recs.Next(ctx) {
		key, err := c.recordToAPIKey(recs.Record())
		if err != nil {
			continue
		}
		keys = append(keys, key)
	}

	return keys, nil
}

func (c *Client) DeleteAPIKey(ctx context.Context, id string) error {
	query := `
		MATCH (k:APIKey {id: $id})
		DELETE k
	`

	session, release, err := c.AcquireSession(ctx)
	if err != nil {
		return err
	}
	defer release()

	_, err = session.Run(ctx, query, map[string]interface{}{"id": id})
	return err
}

func (c *Client) DeleteExpiredAPIKeys(ctx context.Context) (int, error) {
	query := `
		MATCH (k:APIKey)
		WHERE k.expires_at IS NOT NULL AND datetime(k.expires_at) < datetime()
		DELETE k
		RETURN count(k) as deleted
	`

	session, release, err := c.AcquireSession(ctx)
	if err != nil {
		return 0, err
	}
	defer release()

	result, err := session.Run(ctx, query, nil)
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

func (c *Client) recordToAPIKey(rec *neo4jdriver.Record) (*APIKey, error) {
	expiresAt := parseTime(rec.Values[5])

	return &APIKey{
		ID:        getString(rec.Values[0]),
		Key:       "",
		Label:     getString(rec.Values[2]),
		TenantID:  getString(rec.Values[3]),
		CreatedAt: getTime(rec.Values[4]),
		ExpiresAt: expiresAt,
	}, nil
}

func (c *Client) Create(ctx context.Context, key *APIKey) error {
	return c.CreateAPIKey(ctx, key)
}

func (c *Client) Get(ctx context.Context, id string) (*APIKey, error) {
	return c.GetAPIKey(ctx, id)
}

func (c *Client) GetByKey(ctx context.Context, key string) (*APIKey, error) {
	return c.GetAPIKeyByKey(ctx, key)
}

func (c *Client) List(ctx context.Context) ([]*APIKey, error) {
	return c.ListAPIKeys(ctx)
}

func (c *Client) ListByTenant(ctx context.Context, tenantID string) ([]*APIKey, error) {
	return c.ListAPIKeysByTenant(ctx, tenantID)
}

func (c *Client) Delete(ctx context.Context, id string) error {
	return c.DeleteAPIKey(ctx, id)
}

func (c *Client) DeleteExpired(ctx context.Context) (int, error) {
	return c.DeleteExpiredAPIKeys(ctx)
}

func hashAPIKey(key string) string {
	return fmt.Sprintf("%x", time.Now().UnixNano()) + "_" + key[:8]
}

func nilIfZeroTime(t *time.Time) interface{} {
	if t == nil {
		return nil
	}
	return t.Format(time.RFC3339)
}
