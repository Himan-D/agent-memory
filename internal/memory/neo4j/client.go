package neo4j

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	neo4jdriver "github.com/neo4j/neo4j-go-driver/v6/neo4j"

	"agent-memory/internal/auth"
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
		"MEMBER_OF":  true,
		"OWNS":       true,
		"WORKS_WITH": true,
		"WORKS_AT":   true,
		"MANAGES":    true,
		// Ontology-aware edges (Phase 7)
		"CONTRADICTS": true, // Opposite/contradicts relation
		"IMPLIES":     true, // One memory implies another
		"MERGES":      true, // Similar/merged memory
		"SUPPORTS":    true, // Supports/confirms another
		"REFUTES":     true, // Disproves another
		"SPECIALIZES": true, // More specific than
		"GENERALIZES": true, // More general than
		"ENTAILS":     true, // Logically entails
	}
)

// tenantContextKey is the context key type for tenant ID values.
type tenantContextKey struct{}

// ContextWithTenantID returns a new context carrying the given tenant ID.
// Use this when calling Neo4j methods that support tenant filtering via context.
func ContextWithTenantID(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, tenantContextKey{}, tenantID)
}

// tenantIDFromContext extracts the tenant ID from the context.
// Returns an empty string if no tenant ID is set (admin/internal mode — returns all).
func tenantIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(tenantContextKey{}).(string); ok {
		return v
	}
	return ""
}

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
	driver     neo4jdriver.Driver
	config     config.Neo4jConfig
	pool       chan neo4jdriver.SessionWithContext
	maxConns   int
	closeMu    sync.Mutex
	closed     bool
	poolClosed int32 // atomic flag; 1 means pool channel has been closed
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

	// Use context.Background() for pool sessions — they are long-lived and must
	// not be tied to the initialization timeout context (which is cancelled by
	// defer cancel() when NewClient returns, Issue #12).
	for i := 0; i < maxConns; i++ {
		session := driver.NewSession(context.Background(), neo4jdriver.SessionConfig{
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

func (c *Client) queryTimeout() time.Duration {
	timeout := c.config.QueryTimeout
	if timeout <= 0 {
		timeout = 60
	}
	return time.Duration(timeout) * time.Second
}

func (c *Client) shortTimeout() time.Duration {
	timeout := c.config.QueryTimeout
	if timeout <= 0 {
		timeout = 30
	}
	if timeout > 30 {
		timeout = 30
	}
	return time.Duration(timeout) * time.Second
}

func (c *Client) AcquireSession(ctx context.Context) (neo4jdriver.SessionWithContext, func(), error) {
	select {
	case session := <-c.pool:
		// Return pool session; use select/default to avoid blocking/panicking if
		// the pool was closed while the session was in-flight (Issue #9).
		return session, func() {
			if atomic.LoadInt32(&c.poolClosed) == 1 {
				session.Close(context.Background())
				return
			}
			select {
			case c.pool <- session:
				// returned to pool
			default:
				// pool full, close instead of blocking
				session.Close(context.Background())
			}
		}, nil
	default:
		if c.pool == nil || cap(c.pool) == 0 {
			return c.driver.NewSession(ctx, neo4jdriver.SessionConfig{
				AccessMode: neo4jdriver.AccessModeWrite,
			}), func() {}, nil
		}
		// Pool is full — create an overflow session and close it on release.
		// DO NOT push it back into the bounded pool channel; that blocks forever
		// when the pool is at capacity (Issue #5).
		newSession := c.driver.NewSession(ctx, neo4jdriver.SessionConfig{
			AccessMode: neo4jdriver.AccessModeWrite,
		})
		return newSession, func() {
			newSession.Close(context.Background())
		}, nil
	}
}

// Session returns a session from the pool or creates a new one.
// NOTE (Issue #13): this method accepts no context, so overflow sessions are
// created with context.Background() and have no deadline. Callers should
// prefer AcquireSession or GetSession where a context is available.
func (c *Client) Session() neo4jdriver.SessionWithContext {
	select {
	case s, ok := <-c.pool:
		if ok {
			return s
		}
	default:
	}
	return c.driver.NewSession(context.Background(), neo4jdriver.SessionConfig{
		AccessMode: neo4jdriver.AccessModeWrite,
	})
}

func (c *Client) GetSession(ctx context.Context) (neo4jdriver.SessionWithContext, func()) {
	session, cleanup, err := c.AcquireSession(ctx)
	if err != nil {
		return c.driver.NewSession(ctx, neo4jdriver.SessionConfig{
			AccessMode: neo4jdriver.AccessModeWrite,
		}), func() {}
	}
	return session, func() { cleanup() }
}

func (c *Client) Close() error {
	c.closeMu.Lock()
	defer c.closeMu.Unlock()
	if c.closed {
		return nil
	}
	c.closed = true
	// Set atomic flag BEFORE closing the channel so any in-flight release
	// closures see it and close their sessions rather than sending on the
	// already-closed channel (Issue #9).
	atomic.StoreInt32(&c.poolClosed, 1)
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
	// NOTE: Constraints on Memory.id, Entity.id, Session.id, Agent.id,
	// Skill.id, and Concept.id are managed by the schema migration system.
	// Do NOT create plain indexes on those properties here — Neo4j will
	// refuse to later create a uniqueness constraint on a property that
	// already has a non-constraint index.
	indexes := []string{
		"CREATE INDEX message_session_idx IF NOT EXISTS FOR (m:Message) ON (m.session_id)",
		"CREATE INDEX memory_user_id_idx IF NOT EXISTS FOR (m:Memory) ON (m.user_id)",
		"CREATE INDEX memory_org_id_idx IF NOT EXISTS FOR (m:Memory) ON (m.org_id)",
		"CREATE INDEX memory_tenant_id_idx IF NOT EXISTS FOR (m:Memory) ON (m.tenant_id)",
		"CREATE INDEX memory_status_idx IF NOT EXISTS FOR (m:Memory) ON (m.status)",
		"CREATE INDEX memory_content_hash_idx IF NOT EXISTS FOR (m:Memory) ON (m.content_hash)",
		"CREATE INDEX memory_state_key_idx IF NOT EXISTS FOR (m:Memory) ON (m.state_key)",
		"CREATE INDEX memory_type_idx IF NOT EXISTS FOR (m:Memory) ON (m.type)",
		"CREATE INDEX memory_updated_at_idx IF NOT EXISTS FOR (m:Memory) ON (m.updated_at)",
		"CREATE INDEX concept_name_idx IF NOT EXISTS FOR (c:Concept) ON (c.name)",
	}

	for _, idx := range indexes {
		session, cleanup := c.GetSession(ctx)
		_, err := session.Run(ctx, idx, nil)
		cleanup()
		if err != nil {
			errStr := err.Error()
			// Already exists is expected on restart — not a real error.
			if strings.Contains(errStr, "IndexAlreadyExists") || strings.Contains(errStr, "ConstraintAlreadyExists") || strings.Contains(errStr, "EquivalentSchemaRuleAlreadyExists") {
				continue
			}
			log.Printf("WARN: index/constraint creation: %v", err)
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

func (c *Client) ListSessions() ([]*types.Session, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	session := c.driver.NewSession(ctx, neo4jdriver.SessionConfig{
		AccessMode: neo4jdriver.AccessModeRead,
	})
	defer session.Close(ctx)

	query := `
		MATCH (s:Session)
		RETURN s.id, s.agent_id, s.created_at, s.updated_at
		ORDER BY s.updated_at DESC
		LIMIT 100
	`

	result, err := session.Run(ctx, query, nil)
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}

	var sessions []*types.Session
	for result.Next(ctx) {
		rec := result.Record()
		id, _ := rec.Get("s.id")
		agentID, _ := rec.Get("s.agent_id")
		createdAt, _ := rec.Get("s.created_at")
		updatedAt, _ := rec.Get("s.updated_at")

		sess := &types.Session{
			ID:      id.(string),
			AgentID: agentID.(string),
		}
		if ca, ok := createdAt.(time.Time); ok {
			sess.CreatedAt = ca
		}
		if ua, ok := updatedAt.(time.Time); ok {
			sess.UpdatedAt = ua
		}
		sessions = append(sessions, sess)
	}

	return sessions, nil
}

func (c *Client) AddMessage(sessionID string, msg types.Message) error {
	return c.AddMessages(sessionID, []types.Message{msg})
}

func (c *Client) AddMessages(sessionID string, msgs []types.Message) error {
	if len(msgs) == 0 {
		return nil
	}

	ctx := context.Background()
	session, cleanup := c.GetSession(ctx)
	defer cleanup()

	query := `
		MATCH (s:Session {id: $sessionID})
		UNWIND $msgs AS msg
		CREATE (m:Message {
			id: msg.id,
			role: msg.role,
			content: msg.content,
			timestamp: datetime(COALESCE(msg.timestamp, datetime()))
		})
		CREATE (s)-[:HAS_MESSAGE]->(m)
	`

	msgData := make([]map[string]interface{}, 0, len(msgs))
	for _, msg := range msgs {
		id := msg.ID
		if id == "" {
			id = uuid.New().String()
		}

		var timestamp interface{}
		if !msg.Timestamp.IsZero() {
			timestamp = msg.Timestamp.Format(time.RFC3339)
		} else {
			timestamp = nil // COALESCE in Cypher needs null to trigger fallback
		}

		msgData = append(msgData, map[string]interface{}{
			"id":        id,
			"role":      msg.Role,
			"content":   msg.Content,
			"timestamp": timestamp,
		})
	}

	_, err := session.Run(ctx, query, map[string]interface{}{
		"sessionID": sessionID,
		"msgs":      msgData,
	})
	if err != nil {
		return fmt.Errorf("add messages: %w", err)
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

	messages := []types.Message{}
	for result.Next(ctx) {
		rec := result.Record()
		msg := types.Message{
			SessionID: sessionID,
		}
		for i, key := range keys {
			val := rec.Values[i]
			if val == nil {
				continue
			}
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
	session, cleanup := c.GetSession(ctx)
	defer cleanup()

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
		ON CREATE SET e.created_at = datetime($createdAt)
		SET e.type = $type,
			e.name = $name,
			e.tenant_id = $tenant_id,
			e.properties = $properties,
			e.updated_at = datetime($updatedAt)
		RETURN e.id
	`

	tenantID := "default"
	if entity.TenantID != "" {
		tenantID = entity.TenantID
	}

	propertiesJSON, _ := json.Marshal(entity.Properties)
	result, err := session.Run(ctx, query, map[string]interface{}{
		"id":         entity.ID,
		"type":       entity.Type,
		"name":       entity.Name,
		"tenant_id":  tenantID,
		"properties": string(propertiesJSON),
		"createdAt":  time.Now().Format(time.RFC3339),
		"updatedAt":  time.Now().Format(time.RFC3339),
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
	session, cleanup := c.GetSession(ctx)
	defer cleanup()

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
	session, cleanup := c.GetSession(ctx)
	defer cleanup()

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
		props := decodeEntityProperties(rec.Values[3])
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

func decodeEntityProperties(raw interface{}) map[string]interface{} {
	props := map[string]interface{}{}
	switch v := raw.(type) {
	case map[string]interface{}:
		return v
	case string:
		_ = json.Unmarshal([]byte(v), &props)
	}
	return props
}

func (c *Client) ListEntities(tenantID string, limit int) ([]types.Entity, error) {
	ctx := context.Background()
	session, cleanup := c.GetSession(ctx)
	defer cleanup()

	if limit <= 0 {
		limit = 100
	}

	var query string
	var params map[string]interface{}

	if tenantID == "" {
		query = `
			MATCH (e:Entity)
			RETURN e.id, e.type, e.name, e.properties, e.created_at, e.updated_at, e.last_synced
			ORDER BY e.created_at DESC
			LIMIT $limit
		`
		params = map[string]interface{}{
			"limit": limit,
		}
	} else {
		query = `
			MATCH (e:Entity)
			WHERE e.tenant_id = $tenant_id
			RETURN e.id, e.type, e.name, e.properties, e.created_at, e.updated_at, e.last_synced
			ORDER BY e.created_at DESC
			LIMIT $limit
		`
		params = map[string]interface{}{
			"tenant_id": tenantID,
			"limit":     limit,
		}
	}

	result, err := session.Run(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("list entities: %w", err)
	}

	entities := []types.Entity{}
	for result.Next(ctx) {
		rec := result.Record()
		props := decodeEntityProperties(rec.Values[3])
		entity := types.Entity{
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
		entities = append(entities, entity)
	}

	return entities, nil
}

func (c *Client) BatchUpdateSyncTime(entityIDs []string) error {
	if len(entityIDs) == 0 {
		return nil
	}

	ctx := context.Background()
	session, cleanup := c.GetSession(ctx)
	defer cleanup()

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
	session, cleanup := c.GetSession(ctx)
	defer cleanup()

	relID := uuid.New().String()
	weight := 1.0
	if w, ok := props["weight"].(float64); ok {
		weight = w
	}

	query := fmt.Sprintf(`
		MATCH (a:Entity {id: $fromID}), (b:Entity {id: $toID})
		MERGE (a)-[r:%s]->(b)
		ON CREATE SET r.id = $relID, r.created_at = datetime($createdAt)
		SET r.weight = $weight,
			r.updated_at = datetime($updatedAt),
			r.metadata = $metadata
		RETURN r.id
	`, relType)

	_, err := session.Run(ctx, query, map[string]interface{}{
		"fromID":    fromID,
		"toID":      toID,
		"relID":     relID,
		"weight":    weight,
		"metadata":  props,
		"createdAt": time.Now().Format(time.RFC3339),
		"updatedAt": time.Now().Format(time.RFC3339),
	})
	if err != nil {
		return fmt.Errorf("add relation: %w", err)
	}
	return nil
}

func (c *Client) DeleteRelation(fromID, toID, relType string) error {
	if err := ValidateRelationType(relType); err != nil {
		return fmt.Errorf("invalid relation type: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), c.queryTimeout())
	defer cancel()

	session, cleanup := c.GetSession(ctx)
	defer cleanup()

	query := fmt.Sprintf(`
		MATCH (a:Entity {id: $fromID})-[r:%s]->(b:Entity {id: $toID})
		DELETE r
	`, relType)

	_, err := session.Run(ctx, query, map[string]interface{}{
		"fromID": fromID,
		"toID":   toID,
	})
	if err != nil {
		return fmt.Errorf("delete relation: %w", err)
	}
	return nil
}

func (c *Client) QueryGraph(cypher string, params map[string]interface{}) ([]map[string]interface{}, error) {
	ctx := context.Background()
	session, cleanup := c.GetSession(ctx)
	defer cleanup()

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
	session, cleanup := c.GetSession(ctx)
	defer cleanup()

	if depth <= 0 {
		depth = 3
	}

	query := fmt.Sprintf(`
		MATCH path = (start:Entity {id: $id})-[*1..%d]-(end:Entity)
		RETURN nodes(path) as nodes, relationships(path) as edges
		LIMIT 100
	`, depth)

	result, err := session.Run(ctx, query, map[string]interface{}{
		"id": fromEntityID,
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
	ctx := context.Background()
	session, cleanup := c.GetSession(ctx)
	defer cleanup()

	var query string
	var params map[string]interface{}

	if relType == "" {
		query = `
			MATCH (a:Entity {id: $id})-[r]->(b:Entity)
			RETURN r.id, type(r), b.id
		`
		params = map[string]interface{}{"id": entityID}
	} else {
		if err := ValidateRelationType(relType); err != nil {
			return nil, fmt.Errorf("invalid relation type: %w", err)
		}
		query = fmt.Sprintf(`
			MATCH (a:Entity {id: $id})-[r:%s]->(b:Entity)
			RETURN r.id, type(r), b.id
		`, relType)
		params = map[string]interface{}{"id": entityID}
	}

	result, err := session.Run(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("get relations: %w", err)
	}

	var relations []types.Relation
	for result.Next(ctx) {
		rec := result.Record()
		relTypeVal := ""
		if len(rec.Values) > 1 {
			if rt, ok := rec.Values[1].(string); ok {
				relTypeVal = rt
			}
		}
		relations = append(relations, types.Relation{
			ID:     rec.Values[0].(string),
			FromID: entityID,
			ToID:   rec.Values[2].(string),
			Type:   relTypeVal,
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
			content_hash: $content_hash,
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
			updated_at: datetime($updated_at),
			state_key: $state_key
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
		"content_hash":       mem.ContentHash,
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
		"state_key":          mem.StateKey,
	})
	if err != nil {
		return fmt.Errorf("create memory: %w", err)
	}
	_, err = result.Consume(ctx)
	return err
}

func (c *Client) BatchCreateMemories(memories []*types.Memory) error {
	if len(memories) == 0 {
		return nil
	}

	timeout := c.queryTimeout() * 2
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	session := c.driver.NewSession(ctx, neo4jdriver.SessionConfig{
		AccessMode: neo4jdriver.AccessModeWrite,
	})
	defer session.Close(ctx)

	query := `
		UNWIND $memories AS mem
		CREATE (m:Memory {
			id: mem.id,
			content_hash: mem.content_hash,
			tenant_id: mem.tenant_id,
			user_id: mem.user_id,
			org_id: mem.org_id,
			agent_id: mem.agent_id,
			session_id: mem.session_id,
			type: mem.type,
			content: mem.content,
			category: mem.category,
			tags: mem.tags,
			importance: mem.importance,
			metadata: mem.metadata,
			status: mem.status,
			immutable: mem.immutable,
			expiration_date: mem.expiration_date,
			feedback_score: mem.feedback_score,
			parent_memory_id: mem.parent_memory_id,
			related_memory_ids: mem.related_memory_ids,
			version: mem.version,
			access_count: mem.access_count,
			created_at: datetime(mem.created_at),
			updated_at: datetime(mem.updated_at),
			state_key: mem.state_key
		})
		RETURN count(m) AS count
	`

	memData := make([]map[string]interface{}, 0, len(memories))
	for _, mem := range memories {
		expirationDate := ""
		if mem.ExpirationDate != nil {
			expirationDate = mem.ExpirationDate.Format(time.RFC3339)
		}
		metadataJSON, _ := json.Marshal(mem.Metadata)

		memData = append(memData, map[string]interface{}{
			"id":                 mem.ID,
			"content_hash":       mem.ContentHash,
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
			"state_key":          mem.StateKey,
		})
	}

	result, err := session.Run(ctx, query, map[string]interface{}{"memories": memData})
	if err != nil {
		return fmt.Errorf("batch create memories: %w", err)
	}
	_, err = result.Consume(ctx)
	return err
}

func (c *Client) GetMemory(id string) (*types.Memory, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.shortTimeout())
	defer cancel()

	session, cleanup := c.GetSession(ctx)
	defer cleanup()

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

// GetMemoryForTenant retrieves a memory by ID with tenant isolation.
// When tenantID is non-empty, only memories belonging to that tenant are returned.
// When tenantID is empty, any memory is returned (admin/internal use).
func (c *Client) GetMemoryForTenant(id, tenantID string) (*types.Memory, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	session := c.driver.NewSession(ctx, neo4jdriver.SessionConfig{
		AccessMode: neo4jdriver.AccessModeRead,
	})
	defer session.Close(ctx)

	query := `
		MATCH (m:Memory {id: $id})
		WHERE $tenantID = '' OR m.tenant_id = $tenantID
		RETURN m.id, m.tenant_id, m.user_id, m.org_id, m.agent_id, m.session_id,
		       m.type, m.content, m.category, m.tags, m.importance, m.metadata, m.status, m.immutable,
		       m.expiration_date, m.feedback_score, m.parent_memory_id, m.related_memory_ids,
		       m.version, m.access_count, m.created_at, m.updated_at, m.last_accessed
	`

	result, err := session.Run(ctx, query, map[string]interface{}{
		"id":       id,
		"tenantID": tenantID,
	})
	if err != nil {
		return nil, fmt.Errorf("get memory for tenant: %w", err)
	}

	if result.Next(ctx) {
		rec := result.Record()
		return c.recordToMemory(rec)
	}
	return nil, fmt.Errorf("memory not found: %s", id)
}

func (c *Client) GetMemoriesByIDs(ids []string) ([]*types.Memory, error) {
	if len(ids) == 0 {
		return []*types.Memory{}, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	session := c.driver.NewSession(ctx, neo4jdriver.SessionConfig{
		AccessMode: neo4jdriver.AccessModeRead,
	})
	defer session.Close(ctx)

	query := `
		MATCH (m:Memory)
		WHERE m.id IN $ids
		RETURN m.id, m.tenant_id, m.user_id, m.org_id, m.agent_id, m.session_id,
		       m.type, m.content, m.category, m.tags, m.importance, m.metadata, m.status, m.immutable,
		       m.expiration_date, m.feedback_score, m.parent_memory_id, m.related_memory_ids,
		       m.version, m.access_count, m.created_at, m.updated_at, m.last_accessed
	`

	result, err := session.Run(ctx, query, map[string]interface{}{"ids": ids})
	if err != nil {
		return nil, fmt.Errorf("get memories by ids: %w", err)
	}

	var memories []*types.Memory
	for result.Next(ctx) {
		rec := result.Record()
		mem, err := c.recordToMemory(rec)
		if err != nil {
			continue
		}
		memories = append(memories, mem)
	}

	return memories, nil
}

// GetMemoriesByIDsForTenant retrieves memories by IDs with tenant isolation.
// When tenantID is non-empty, only memories belonging to that tenant are returned.
// When tenantID is empty, any matching memory is returned (admin/internal use).
func (c *Client) GetMemoriesByIDsForTenant(ids []string, tenantID string) ([]*types.Memory, error) {
	if len(ids) == 0 {
		return []*types.Memory{}, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	session := c.driver.NewSession(ctx, neo4jdriver.SessionConfig{
		AccessMode: neo4jdriver.AccessModeRead,
	})
	defer session.Close(ctx)

	query := `
		MATCH (m:Memory)
		WHERE m.id IN $ids AND ($tenantID = '' OR m.tenant_id = $tenantID)
		RETURN m.id, m.tenant_id, m.user_id, m.org_id, m.agent_id, m.session_id,
		       m.type, m.content, m.category, m.tags, m.importance, m.metadata, m.status, m.immutable,
		       m.expiration_date, m.feedback_score, m.parent_memory_id, m.related_memory_ids,
		       m.version, m.access_count, m.created_at, m.updated_at, m.last_accessed
	`

	result, err := session.Run(ctx, query, map[string]interface{}{
		"ids":      ids,
		"tenantID": tenantID,
	})
	if err != nil {
		return nil, fmt.Errorf("get memories by ids for tenant: %w", err)
	}

	var memories []*types.Memory
	for result.Next(ctx) {
		rec := result.Record()
		mem, err := c.recordToMemory(rec)
		if err != nil {
			continue
		}
		memories = append(memories, mem)
	}

	return memories, nil
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
		    m.updated_at = datetime($updated_at),
		    m.state_key = $state_key,
		    m.importance = $importance,
		    m.tags = $tags,
		    m.type = $type,
		    m.content_hash = $content_hash,
		    m.version = $version,
		    m.access_count = $access_count,
		    m.parent_memory_id = $parent_memory_id,
		    m.related_memory_ids = $related_memory_ids,
		    m.agent_id = $agent_id,
		    m.session_id = $session_id
		RETURN m.id
	`

	expirationDate := ""
	if mem.ExpirationDate != nil {
		expirationDate = mem.ExpirationDate.Format(time.RFC3339)
	}

	metadataJSON, _ := json.Marshal(mem.Metadata)
	_, err := session.Run(ctx, query, map[string]interface{}{
		"id":                 mem.ID,
		"content":            mem.Content,
		"category":           mem.Category,
		"metadata":           string(metadataJSON),
		"status":             string(mem.Status),
		"immutable":          mem.Immutable,
		"expiration_date":    expirationDate,
		"feedback_score":     string(mem.FeedbackScore),
		"updated_at":         mem.UpdatedAt.Format(time.RFC3339),
		"state_key":          mem.StateKey,
		"importance":         string(mem.Importance),
		"tags":               mem.Tags,
		"type":               string(mem.Type),
		"content_hash":       mem.ContentHash,
		"version":            mem.Version,
		"access_count":       mem.AccessCount,
		"parent_memory_id":   mem.ParentMemoryID,
		"related_memory_ids": mem.RelatedMemoryIDs,
		"agent_id":           mem.AgentID,
		"session_id":         mem.SessionID,
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

func (c *Client) BatchDeleteMemories(ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	timeout := c.queryTimeout() * 2
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	session := c.driver.NewSession(ctx, neo4jdriver.SessionConfig{
		AccessMode: neo4jdriver.AccessModeWrite,
	})
	defer session.Close(ctx)

	query := `
		MATCH (m:Memory)
		WHERE m.id IN $ids
		DETACH DELETE m
		RETURN count(m) AS count
	`

	result, err := session.Run(ctx, query, map[string]interface{}{"ids": ids})
	if err != nil {
		return fmt.Errorf("batch delete memories: %w", err)
	}
	_, err = result.Consume(ctx)
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

func (c *Client) GetMemoriesByHash(userID, hash string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	session := c.driver.NewSession(ctx, neo4jdriver.SessionConfig{
		AccessMode: neo4jdriver.AccessModeRead,
	})
	defer session.Close(ctx)

	query := `
		MATCH (m:Memory {user_id: $user_id, content_hash: $hash, status: 'active'})
		RETURN m.id
		LIMIT 1
	`

	result, err := session.Run(ctx, query, map[string]interface{}{
		"user_id": userID,
		"hash":    hash,
	})
	if err != nil {
		return "", err
	}

	if result.Next(ctx) {
		record := result.Record()
		if id, ok := record.Get("m.id"); ok {
			if idStr, ok := id.(string); ok {
				return idStr, nil
			}
		}
	}
	return "", nil
}

func (c *Client) SearchByContent(query string, limit int) ([]types.MemoryResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	session := c.driver.NewSession(ctx, neo4jdriver.SessionConfig{
		AccessMode: neo4jdriver.AccessModeRead,
	})
	defer session.Close(ctx)

	queryCypher := fmt.Sprintf(`
		MATCH (m:Memory)
		WHERE m.status = 'active' AND toLower(m.content) CONTAINS toLower($query)
		RETURN m.id, m.content, m.created_at
		ORDER BY m.created_at DESC
		LIMIT %d
	`, limit)

	result, err := session.Run(ctx, queryCypher, map[string]interface{}{
		"query": query,
	})
	if err != nil {
		return nil, err
	}

	var results []types.MemoryResult
	for result.Next(ctx) {
		record := result.Record()
		results = append(results, types.MemoryResult{
			MemoryID: getString(record.Values[0]),
			Text:     getString(record.Values[1]),
			Score:    0.8,
		})
	}
	return results, nil
}

func (c *Client) SearchByEntities(entities []string, limit int) ([]types.MemoryResult, error) {
	if len(entities) == 0 {
		return nil, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	session := c.driver.NewSession(ctx, neo4jdriver.SessionConfig{
		AccessMode: neo4jdriver.AccessModeRead,
	})
	defer session.Close(ctx)

	entityMatch := make([]string, len(entities))
	params := make(map[string]interface{})
	for i, entity := range entities {
		key := fmt.Sprintf("entity%d", i)
		params[key] = strings.ToLower(entity)
		entityMatch[i] = fmt.Sprintf("toLower(m.content) CONTAINS toLower($%s)", key)
	}

	queryCypher := fmt.Sprintf(`
		MATCH (m:Memory)
		WHERE m.status = 'active' AND (%s)
		RETURN m.id, m.content, m.user_id, m.created_at
		ORDER BY m.created_at DESC
		LIMIT %d
	`, strings.Join(entityMatch, " OR "), limit)

	result, err := session.Run(ctx, queryCypher, params)
	if err != nil {
		return nil, err
	}

	var results []types.MemoryResult
	for result.Next(ctx) {
		record := result.Record()
		results = append(results, types.MemoryResult{
			MemoryID: getString(record.Values[0]),
			Text:     getString(record.Values[1]),
			Score:    0.7,
		})
	}
	return results, nil
}

func (c *Client) GetAllMemories() ([]*types.Memory, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	session := c.driver.NewSession(ctx, neo4jdriver.SessionConfig{
		AccessMode: neo4jdriver.AccessModeRead,
	})
	defer session.Close(ctx)

	query := `
		MATCH (m:Memory)
		WHERE m.status = 'active'
		RETURN m.id, m.tenant_id, m.user_id, m.org_id, m.agent_id, m.session_id,
		       m.type, m.content, m.category, m.tags, m.importance, m.metadata,
		       m.status, m.immutable, m.feedback_score,
		       m.parent_memory_id, m.related_memory_ids, m.version, m.access_count,
		       m.created_at, m.updated_at
		ORDER BY m.created_at DESC
		LIMIT 1000
	`

	result, err := session.Run(ctx, query, map[string]interface{}{})
	if err != nil {
		return nil, err
	}

	var memories []*types.Memory
	for result.Next(ctx) {
		rec := result.Record()
		vals := rec.Values

		mem := &types.Memory{}
		if len(vals) > 0 {
			mem.ID = getString(vals[0])
		}
		if len(vals) > 1 {
			mem.TenantID = getString(vals[1])
		}
		if len(vals) > 2 {
			mem.UserID = getString(vals[2])
		}
		if len(vals) > 3 {
			mem.OrgID = getString(vals[3])
		}
		if len(vals) > 4 {
			mem.AgentID = getString(vals[4])
		}
		if len(vals) > 5 {
			mem.SessionID = getString(vals[5])
		}
		if len(vals) > 6 {
			mem.Type = types.MemoryType(getString(vals[6]))
		}
		if len(vals) > 7 {
			mem.Content = getString(vals[7])
		}
		if len(vals) > 8 {
			mem.Category = getString(vals[8])
		}
		if len(vals) > 9 {
			mem.Tags = getStringSlice(vals[9])
		}
		if len(vals) > 10 {
			mem.Importance = types.ImportanceLevel(getString(vals[10]))
		}
		if len(vals) > 11 && vals[11] != nil {
			if metaStr, ok := vals[11].(string); ok {
				_ = json.Unmarshal([]byte(metaStr), &mem.Metadata)
			}
		}
		if len(vals) > 12 {
			mem.Status = types.MemoryStatus(getString(vals[12]))
		}
		if len(vals) > 13 {
			mem.Immutable = getBool(vals[13])
		}
		if len(vals) > 14 {
			mem.FeedbackScore = types.FeedbackType(getString(vals[14]))
		}
		if len(vals) > 15 {
			mem.ParentMemoryID = getString(vals[15])
		}
		if len(vals) > 16 {
			mem.RelatedMemoryIDs = getStringSlice(vals[16])
		}
		if len(vals) > 17 {
			mem.Version = getInt(vals[17])
		}
		if len(vals) > 18 {
			mem.AccessCount = getInt64(vals[18])
		}
		if len(vals) > 19 && vals[19] != nil {
			mem.CreatedAt = vals[19].(time.Time)
		}
		if len(vals) > 20 && vals[20] != nil {
			mem.UpdatedAt = vals[20].(time.Time)
		}
		if len(vals) > 21 {
			mem.StateKey = getString(vals[21])
		}
		memories = append(memories, mem)
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
		SET m.entity_id = coalesce(m.entity_id, $entity_id)
	`

	_, err := session.Run(ctx, query, map[string]interface{}{
		"memory_id": memoryID,
		"entity_id": entityID,
	})
	return err
}

func (c *Client) GetMemoryIDsByEntity(entityID string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.shortTimeout())
	defer cancel()

	session, cleanup := c.GetSession(ctx)
	defer cleanup()

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

func (c *Client) GetEntitiesByMemory(memoryID string) ([]types.Entity, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.shortTimeout())
	defer cancel()

	session, cleanup := c.GetSession(ctx)
	defer cleanup()

	query := `
		MATCH (m:Memory {id: $memory_id})<-[:MEMORY_OF]-(e:Entity)
		RETURN e.id, e.type, e.name, e.properties, e.created_at, e.updated_at, e.last_synced
	`

	result, err := session.Run(ctx, query, map[string]interface{}{"memory_id": memoryID})
	if err != nil {
		return nil, err
	}

	var entities []types.Entity
	for result.Next(ctx) {
		rec := result.Record()
		props := decodeEntityProperties(rec.Values[3])
		entity := types.Entity{
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
		entities = append(entities, entity)
	}
	return entities, nil
}

func (c *Client) GetMemoriesPaginated(req *types.SearchRequest) ([]*types.Memory, int64, error) {
	if req == nil {
		return nil, 0, fmt.Errorf("search request required")
	}
	if req.Limit <= 0 {
		req.Limit = 20
	}
	if req.Offset < 0 {
		req.Offset = 0
	}

	query := `
		MATCH (m:Memory)
		WHERE ($query = '' OR toLower(m.content) CONTAINS toLower($query))
		  AND ($user_id = '' OR m.user_id = $user_id)
		  AND ($org_id = '' OR m.org_id = $org_id)
		  AND ($category = '' OR m.category = $category)
		  AND ($memory_type = '' OR m.memory_type = $memory_type)
		  AND ($agent_id = '' OR m.agent_id = $agent_id)
		RETURN m.id, m.tenant_id, m.user_id, m.org_id, m.agent_id, m.session_id,
		       m.type, m.content, m.category, m.tags, m.importance, m.metadata, m.status, m.immutable,
		       m.expiration_date, m.feedback_score, m.parent_memory_id, m.related_memory_ids,
		       m.version, m.access_count, m.created_at, m.updated_at, m.last_accessed
		ORDER BY m.created_at DESC
		SKIP $offset
		LIMIT $limit
	`

	countQuery := `
		MATCH (m:Memory)
		WHERE ($query = '' OR toLower(m.content) CONTAINS toLower($query))
		  AND ($user_id = '' OR m.user_id = $user_id)
		  AND ($org_id = '' OR m.org_id = $org_id)
		  AND ($category = '' OR m.category = $category)
		  AND ($memory_type = '' OR m.memory_type = $memory_type)
		  AND ($agent_id = '' OR m.agent_id = $agent_id)
		RETURN count(m)
	`

	params := map[string]interface{}{
		"query":       strings.TrimSpace(req.Query),
		"user_id":     req.UserID,
		"org_id":      req.OrgID,
		"category":    req.Category,
		"memory_type": req.MemoryType,
		"agent_id":    req.AgentID,
		"offset":      req.Offset,
		"limit":       req.Limit,
	}

	session, release, err := c.AcquireSession(context.Background())
	if err != nil {
		return nil, 0, err
	}
	defer release()

	countResult, err := session.Run(context.Background(), countQuery, params)
	if err != nil {
		return nil, 0, err
	}
	var total int64
	if countResult.Next(context.Background()) {
		rec := countResult.Record()
		if countVal, ok := rec.Values[0].(int64); ok {
			total = countVal
		}
	}

	result, err := session.Run(context.Background(), query, params)
	if err != nil {
		return nil, 0, err
	}

	var memories []*types.Memory
	for result.Next(context.Background()) {
		rec := result.Record()
		if mem, err := c.recordToMemoryPtr(rec); err == nil {
			memories = append(memories, mem)
		}
	}
	return memories, total, nil
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

// GetFeedbackByMemory returns all feedback nodes associated with a specific memory ID.
func (c *Client) GetFeedbackByMemory(ctx context.Context, memoryID string) ([]*types.Feedback, error) {
	tctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	session := c.driver.NewSession(tctx, neo4jdriver.SessionConfig{
		AccessMode: neo4jdriver.AccessModeRead,
	})
	defer session.Close(tctx)

	query := `
		MATCH (f:Feedback {memory_id: $memory_id})
		RETURN f.id, f.memory_id, f.type, f.comment, f.session_id, f.user_id, f.created_at
		ORDER BY f.created_at DESC
	`

	result, err := session.Run(tctx, query, map[string]interface{}{
		"memory_id": memoryID,
	})
	if err != nil {
		return nil, err
	}

	var feedbacks []*types.Feedback
	for result.Next(tctx) {
		rec := result.Record()
		getString := func(v interface{}) string {
			if v == nil {
				return ""
			}
			if s, ok := v.(string); ok {
				return s
			}
			return ""
		}
		getTime := func(v interface{}) time.Time {
			if v == nil {
				return time.Time{}
			}
			if t, ok := v.(time.Time); ok {
				return t
			}
			return time.Time{}
		}
		feedbacks = append(feedbacks, &types.Feedback{
			ID:        getString(rec.Values[0]),
			MemoryID:  getString(rec.Values[1]),
			Type:      types.FeedbackType(getString(rec.Values[2])),
			Comment:   getString(rec.Values[3]),
			SessionID: getString(rec.Values[4]),
			UserID:    getString(rec.Values[5]),
			CreatedAt: getTime(rec.Values[6]),
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
	allowedFields := map[string]bool{
		"id": true, "user_id": true, "agent_id": true, "org_id": true,
		"content": true, "type": true, "category": true, "status": true,
		"priority": true, "scope": true, "tenant_id": true, "hash": true,
		"created_at": true, "updated_at": true, "last_accessed": true,
		"expiration": true, "feedback_score": true, "access_count": true,
		"importance": true, "embedding_model": true, "source": true,
	}

	sanitizeField := func(field string) (string, bool) {
		field = strings.TrimSpace(field)
		if !allowedFields[field] {
			return "", false
		}
		return field, true
	}

	conditions := []string{}
	params := map[string]interface{}{}
	paramIndex := 0

	for _, rule := range filters.Rules {
		safeField, ok := sanitizeField(rule.Field)
		if !ok {
			continue
		}

		paramName := fmt.Sprintf("p%d", paramIndex)
		paramIndex++

		switch rule.Operator {
		case "eq", "==", "=":
			conditions = append(conditions, fmt.Sprintf("m.%s = $%s", safeField, paramName))
			params[paramName] = rule.Value
		case "ne", "!=":
			conditions = append(conditions, fmt.Sprintf("m.%s <> $%s", safeField, paramName))
			params[paramName] = rule.Value
		case "gt", ">":
			conditions = append(conditions, fmt.Sprintf("m.%s > $%s", safeField, paramName))
			params[paramName] = rule.Value
		case "gte", ">=":
			conditions = append(conditions, fmt.Sprintf("m.%s >= $%s", safeField, paramName))
			params[paramName] = rule.Value
		case "lt", "<":
			conditions = append(conditions, fmt.Sprintf("m.%s < $%s", safeField, paramName))
			params[paramName] = rule.Value
		case "lte", "<=":
			conditions = append(conditions, fmt.Sprintf("m.%s <= $%s", safeField, paramName))
			params[paramName] = rule.Value
		case "contains":
			conditions = append(conditions, fmt.Sprintf("m.%s CONTAINS $%s", safeField, paramName))
			params[paramName] = rule.Value
		case "icontains":
			conditions = append(conditions, fmt.Sprintf("toLower(m.%s) CONTAINS toLower($%s)", safeField, paramName))
			params[paramName] = rule.Value
		case "in":
			if values, ok := rule.Value.([]interface{}); ok {
				conditions = append(conditions, fmt.Sprintf("m.%s IN $%s", safeField, paramName))
				params[paramName] = values
			}
		case "starts_with":
			conditions = append(conditions, fmt.Sprintf("m.%s STARTS WITH $%s", safeField, paramName))
			params[paramName] = rule.Value
		case "ends_with":
			conditions = append(conditions, fmt.Sprintf("m.%s ENDS WITH $%s", safeField, paramName))
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
	vals := rec.Values

	if len(vals) > 11 && vals[11] != nil {
		if metaStr, ok := vals[11].(string); ok {
			_ = json.Unmarshal([]byte(metaStr), &metadata)
		}
	}

	var expirationDate *time.Time
	if len(vals) > 17 {
		expirationDate = parseTime(vals[17])
	}

	var lastAccessed *time.Time
	if len(vals) > 23 {
		lastAccessed = parseTime(vals[23])
	}

	mem := &types.Memory{}
	if len(vals) > 0 {
		mem.ID = getString(vals[0])
	}
	if len(vals) > 1 {
		mem.TenantID = getString(vals[1])
	}
	if len(vals) > 2 {
		mem.UserID = getString(vals[2])
	}
	if len(vals) > 3 {
		mem.OrgID = getString(vals[3])
	}
	if len(vals) > 4 {
		mem.AgentID = getString(vals[4])
	}
	if len(vals) > 5 {
		mem.SessionID = getString(vals[5])
	}
	if len(vals) > 6 {
		mem.Type = types.MemoryType(getString(vals[6]))
	}
	if len(vals) > 7 {
		mem.Content = getString(vals[7])
	}
	if len(vals) > 8 {
		mem.Category = getString(vals[8])
	}
	if len(vals) > 9 {
		mem.Tags = getStringSlice(vals[9])
	}
	if len(vals) > 10 {
		mem.Importance = types.ImportanceLevel(getString(vals[10]))
	}
	mem.Metadata = metadata
	if len(vals) > 12 {
		mem.Status = types.MemoryStatus(getString(vals[12]))
	}
	if len(vals) > 13 {
		mem.Immutable = getBool(vals[13])
	}
	mem.ExpirationDate = expirationDate
	if len(vals) > 14 {
		mem.FeedbackScore = types.FeedbackType(getString(vals[14]))
	}
	if len(vals) > 15 {
		mem.ParentMemoryID = getString(vals[15])
	}
	if len(vals) > 16 {
		mem.RelatedMemoryIDs = getStringSlice(vals[16])
	}
	if len(vals) > 18 {
		mem.Version = getInt(vals[18])
	}
	if len(vals) > 19 {
		mem.AccessCount = getInt64(vals[19])
	}
	if len(vals) > 20 && vals[20] != nil {
		mem.CreatedAt = vals[20].(time.Time)
	}
	if len(vals) > 21 && vals[21] != nil {
		mem.UpdatedAt = vals[21].(time.Time)
	}
	if len(vals) > 22 {
		mem.StateKey = getString(vals[22])
	}
	mem.LastAccessed = lastAccessed

	return mem, nil
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

// GetSkillForTenant retrieves a skill by ID with tenant isolation.
// When tenantID is non-empty, only skills belonging to that tenant are returned.
// When tenantID is empty, any skill is returned (admin/internal use).
func (c *Client) GetSkillForTenant(ctx context.Context, skillID, tenantID string) (*types.Skill, error) {
	query := `
		MATCH (s:Skill {id: $id})
		WHERE $tenantID = '' OR s.tenant_id = $tenantID
		RETURN s.id, s.tenant_id, s.group_id, s.name, s.domain, s.trigger, s.action,
		       s.confidence, s.usage_count, s.source_memory, s.created_by, s.verified,
		       s.human_reviewed, s.version, s.tags, s.examples, s.metadata,
		       s.created_at, s.updated_at, s.last_used`

	session, release, err := c.AcquireSession(ctx)
	if err != nil {
		return nil, err
	}
	defer release()

	rec, err := session.Run(ctx, query, map[string]interface{}{
		"id":       skillID,
		"tenantID": tenantID,
	})
	if err != nil {
		return nil, err
	}

	if !rec.Next(ctx) {
		return nil, fmt.Errorf("skill not found: %s", skillID)
	}

	return c.recordToSkill(rec.Record())
}

// ==================== Concept Methods (GAAMA paper) ====================

func (c *Client) CreateConcept(ctx context.Context, concept *types.Concept) error {
	if concept.ID == "" {
		concept.ID = uuid.New().String()
	}
	concept.CreatedAt = time.Now()
	concept.UpdatedAt = time.Now()

	props, _ := json.Marshal(concept.Properties)

	query := `
		CREATE (c:Concept {
			id: $id,
			name: $name,
			description: $description,
			tenant_id: $tenant_id,
			properties: $properties,
			created_at: datetime($created_at),
			updated_at: datetime($updated_at)
		})`

	session, release, err := c.AcquireSession(ctx)
	if err != nil {
		return err
	}
	defer release()

	_, err = session.Run(ctx, query, map[string]interface{}{
		"id":          concept.ID,
		"name":        concept.Name,
		"description": concept.Description,
		"tenant_id":   concept.TenantID,
		"properties":  string(props),
		"created_at":  concept.CreatedAt.Format(time.RFC3339),
		"updated_at":  concept.UpdatedAt.Format(time.RFC3339),
	})
	return err
}

func (c *Client) ListConcepts(ctx context.Context, tenantID string, limit int) ([]*types.Concept, error) {
	query := `
		MATCH (c:Concept)
		WHERE $tenantID = '' OR c.tenant_id = $tenantID
		RETURN c.id, c.name, c.description, c.tenant_id, c.properties, c.created_at, c.updated_at
		LIMIT $limit`

	session, release, err := c.AcquireSession(ctx)
	if err != nil {
		return nil, err
	}
	defer release()

	result, err := session.Run(ctx, query, map[string]interface{}{
		"tenantID": tenantID,
		"limit":    limit,
	})
	if err != nil {
		return nil, err
	}

	var concepts []*types.Concept
	for result.Next(ctx) {
		rec := result.Record()
		vals := rec.Values
		c := &types.Concept{
			ID:          getString(vals[0]),
			Name:        getString(vals[1]),
			Description: getString(vals[2]),
			TenantID:    getString(vals[3]),
		}
		if len(vals) > 4 && vals[4] != nil {
			json.Unmarshal([]byte(getString(vals[4])), &c.Properties)
		}
		if len(vals) > 5 && vals[5] != nil {
			if t, ok := vals[5].(time.Time); ok {
				c.CreatedAt = t
			}
		}
		if len(vals) > 6 && vals[6] != nil {
			if t, ok := vals[6].(time.Time); ok {
				c.UpdatedAt = t
			}
		}
		concepts = append(concepts, c)
	}
	return concepts, nil
}

func (c *Client) GetConceptMemories(ctx context.Context, conceptID string, limit int) ([]*types.Memory, error) {
	query := `
		MATCH (concept:Concept {id: $conceptID})-[:BELONGS_TO|HAS_CONCEPT|RELATED_TO]-(m:Memory)
		RETURN m.id, m.tenant_id, m.user_id, m.org_id, m.agent_id, m.session_id,
		       m.type, m.content, m.category, m.tags, m.importance, m.metadata,
		       m.status, m.immutable, m.feedback_score, m.parent_memory_id,
		       m.related_memory_ids, m.expiration_date, m.version, m.access_count,
		       m.created_at, m.updated_at, m.state_key, m.last_accessed
		LIMIT $limit`

	session, release, err := c.AcquireSession(ctx)
	if err != nil {
		return nil, err
	}
	defer release()

	result, err := session.Run(ctx, query, map[string]interface{}{
		"conceptID": conceptID,
		"limit":     limit,
	})
	if err != nil {
		return nil, err
	}

	var memories []*types.Memory
	for result.Next(ctx) {
		mem, err := c.recordToMemory(result.Record())
		if err != nil {
			continue
		}
		memories = append(memories, mem)
	}
	return memories, nil
}

func (c *Client) LinkToConcept(ctx context.Context, nodeID, conceptID, relType string) error {
	if relType == "" {
		relType = "BELONGS_TO"
	}

	query := fmt.Sprintf(`
		MATCH (n {id: $nodeID}), (concept:Concept {id: $conceptID})
		MERGE (n)-[r:%s]->(concept)
		RETURN r`, relType)

	session, release, err := c.AcquireSession(ctx)
	if err != nil {
		return err
	}
	defer release()

	_, err = session.Run(ctx, query, map[string]interface{}{
		"nodeID":    nodeID,
		"conceptID": conceptID,
	})
	return err
}

// ==================== Prospective Memory (Reminders) ====================

func (c *Client) GetDueReminders(ctx context.Context, before time.Time) ([]*types.Memory, error) {
	query := `
		MATCH (m:Memory)
		WHERE m.remind_at IS NOT NULL AND m.remind_at <= datetime($before)
		RETURN m.id, m.tenant_id, m.user_id, m.org_id, m.agent_id, m.session_id,
		       m.type, m.content, m.category, m.tags, m.importance, m.metadata,
		       m.status, m.immutable, m.feedback_score, m.parent_memory_id,
		       m.related_memory_ids, m.expiration_date, m.version, m.access_count,
		       m.created_at, m.updated_at, m.state_key, m.last_accessed
		LIMIT 100`

	session, release, err := c.AcquireSession(ctx)
	if err != nil {
		return nil, err
	}
	defer release()

	result, err := session.Run(ctx, query, map[string]interface{}{
		"before": before.Format(time.RFC3339),
	})
	if err != nil {
		return nil, err
	}

	var memories []*types.Memory
	for result.Next(ctx) {
		mem, err := c.recordToMemory(result.Record())
		if err != nil {
			continue
		}
		memories = append(memories, mem)
	}
	return memories, nil
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
	tenantID := tenantIDFromContext(ctx)
	query := `
		MATCH (s:Skill)
		WHERE (s.trigger CONTAINS $trigger OR $trigger CONTAINS s.trigger)
		  AND ($tenantID = '' OR s.tenant_id = $tenantID)
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
		"trigger":  trigger,
		"limit":    limit,
		"tenantID": tenantID,
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
	tenantID := tenantIDFromContext(ctx)
	query := `
		MATCH (s:Skill {domain: $domain})
		WHERE $tenantID = '' OR s.tenant_id = $tenantID
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
		"domain":   domain,
		"limit":    limit,
		"tenantID": tenantID,
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
			config: $config,
			groups: $groups,
			metadata: $metadata,
			created_at: datetime($created_at),
			updated_at: datetime($updated_at)
		})
		RETURN a.id`

	groups, _ := json.Marshal(agent.Groups)
	config, _ := json.Marshal(agent.Config)
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
		"config":      string(config),
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
		RETURN a.id, a.tenant_id, a.name, a.description, a.status, a.config, a.groups, a.metadata,
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
	var config types.AgentConfig
	var groups []string
	var metadata map[string]interface{}
	json.Unmarshal([]byte(getString(rec.Values[5])), &config)
	json.Unmarshal([]byte(getString(rec.Values[6])), &groups)
	json.Unmarshal([]byte(getString(rec.Values[7])), &metadata)

	return &types.Agent{
		ID:          getString(rec.Values[0]),
		TenantID:    getString(rec.Values[1]),
		Name:        getString(rec.Values[2]),
		Description: getString(rec.Values[3]),
		Status:      types.AgentStatus(getString(rec.Values[4])),
		Config:      config,
		Groups:      groups,
		Metadata:    metadata,
		CreatedAt:   getTime(rec.Values[8]),
		UpdatedAt:   getTime(rec.Values[9]),
		LastActive:  parseTime(rec.Values[10]),
	}, nil
}

func (c *Client) UpdateAgent(ctx context.Context, agent *types.Agent) error {
	agent.UpdatedAt = time.Now()

	query := `
		MATCH (a:Agent {id: $id})
		SET a.name = $name,
		    a.description = $description,
		    a.status = $status,
		    a.config = $config,
		    a.groups = $groups,
		    a.metadata = $metadata,
		    a.updated_at = datetime()`

	groups, _ := json.Marshal(agent.Groups)
	config, _ := json.Marshal(agent.Config)
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
		"config":      string(config),
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
		RETURN a.id, a.tenant_id, a.name, a.description, a.status, a.config, a.groups, a.metadata,
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
	var policy types.GroupPolicy
	var metadata map[string]interface{}
	var members []types.AgentMember

	json.Unmarshal([]byte(getString(rec.Values[5])), &policy)
	json.Unmarshal([]byte(getString(rec.Values[7])), &metadata)

	membersData, ok := rec.Values[10].([]interface{})
	if ok {
		for _, m := range membersData {
			if m == nil {
				continue
			}
			if memberMap, ok := m.(map[string]interface{}); ok {
				members = append(members, types.AgentMember{
					AgentID:  getString(memberMap["agent_id"]),
					GroupID:  getString(rec.Values[0]),
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
		Policy:       policy,
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
		MERGE (a)-[r:MEMBER_OF]->(g)
		ON CREATE SET r.joined_at = datetime()
		SET r.role = $role`

	session, release, err := c.AcquireSession(ctx)
	if err != nil {
		return err
	}
	defer release()

	result, err := session.Run(ctx, query, map[string]interface{}{
		"agent_id": agentID,
		"group_id": groupID,
		"role":     string(role),
	})
	if err != nil {
		return err
	}
	_, err = result.Consume(ctx)
	return err
}

func (c *Client) RemoveAgentFromGroup(ctx context.Context, agentID, groupID string) error {
	query := `
		MATCH (a:Agent {id: $agent_id})-[r:MEMBER_OF]->(g:AgentGroup {id: $group_id})
		DELETE r`

	session, release, err := c.AcquireSession(ctx)
	if err != nil {
		return err
	}
	defer release()

	result, err := session.Run(ctx, query, map[string]interface{}{
		"agent_id": agentID,
		"group_id": groupID,
	})
	if err != nil {
		return err
	}
	_, err = result.Consume(ctx)
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
	var policy types.GroupPolicy
	var metadata map[string]interface{}

	json.Unmarshal([]byte(getString(rec.Values[5])), &policy)
	json.Unmarshal([]byte(getString(rec.Values[7])), &metadata)

	return &types.AgentGroup{
		ID:           getString(rec.Values[0]),
		TenantID:     getString(rec.Values[1]),
		Name:         getString(rec.Values[2]),
		Description:  getString(rec.Values[3]),
		Domain:       getString(rec.Values[4]),
		Policy:       policy,
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
		MERGE (g)-[r:SHARED_MEMORY]->(m)
		ON CREATE SET r.id = $shared_id, r.shared_at = datetime()
		SET r.shared_by = $shared_by
		RETURN id(g)`

	session, release, err := c.AcquireSession(ctx)
	if err != nil {
		return err
	}
	defer release()

	result, err := session.Run(ctx, query, map[string]interface{}{
		"memory_id": memoryID,
		"group_id":  groupID,
		"shared_id": sharedID,
		"shared_by": sharedBy,
	})
	if err != nil {
		return err
	}
	_, err = result.Consume(ctx)
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

func (c *Client) CreateChain(ctx context.Context, chain *types.SkillChain) error {
	if chain.ID == "" {
		chain.ID = uuid.New().String()
	}

	stepsJSON, _ := json.Marshal(chain.Steps)
	conditionsJSON, _ := json.Marshal(chain.Conditions)
	tagsJSON, _ := json.Marshal(chain.Tags)
	metadataJSON, _ := json.Marshal(chain.Metadata)

	query := `
		CREATE (ch:SkillChain {
			id: $id,
			tenant_id: $tenant_id,
			name: $name,
			description: $description,
			trigger: $trigger,
			steps: $steps,
			conditions: $conditions,
			confidence: $confidence,
			usage_count: 0,
			success_count: 0,
			avg_duration_ms: 0,
			tags: $tags,
			metadata: $metadata,
			created_at: datetime($created_at),
			updated_at: datetime($updated_at),
			last_used: null
		})
		RETURN ch.id`

	session, release, err := c.AcquireSession(ctx)
	if err != nil {
		return err
	}
	defer release()

	_, err = session.Run(ctx, query, map[string]interface{}{
		"id":          chain.ID,
		"tenant_id":   chain.TenantID,
		"name":        chain.Name,
		"description": chain.Description,
		"trigger":     chain.Trigger,
		"steps":       string(stepsJSON),
		"conditions":  string(conditionsJSON),
		"confidence":  chain.Confidence,
		"tags":        string(tagsJSON),
		"metadata":    string(metadataJSON),
		"created_at":  chain.CreatedAt.Format(time.RFC3339),
		"updated_at":  chain.UpdatedAt.Format(time.RFC3339),
	})
	return err
}

func (c *Client) GetChain(ctx context.Context, chainID string) (*types.SkillChain, error) {
	query := `
		MATCH (ch:SkillChain {id: $id})
		RETURN ch.id, ch.tenant_id, ch.name, ch.description, ch.trigger, ch.steps,
			   ch.conditions, ch.confidence, ch.usage_count, ch.success_count,
			   ch.avg_duration_ms, ch.tags, ch.metadata, ch.created_at, ch.updated_at, ch.last_used`

	session, release, err := c.AcquireSession(ctx)
	if err != nil {
		return nil, err
	}
	defer release()

	rec, err := session.Run(ctx, query, map[string]interface{}{"id": chainID})
	if err != nil {
		return nil, err
	}

	if !rec.Next(ctx) {
		return nil, fmt.Errorf("chain not found: %s", chainID)
	}

	return c.chainFromRecord(rec.Record()), nil
}

func (c *Client) ListChains(ctx context.Context, tenantID string, query *types.ChainQuery) ([]*types.SkillChain, error) {
	if query == nil {
		query = &types.ChainQuery{Limit: 50}
	}
	if query.Limit <= 0 {
		query.Limit = 50
	}

	var cypher string
	var params map[string]interface{}

	if query.Offset > 0 {
		cypher = `
		MATCH (ch:SkillChain)
		WHERE ch.tenant_id = $tenant_id OR $tenant_id = ""
		RETURN ch.id, ch.tenant_id, ch.name, ch.description, ch.trigger, ch.steps,
			   ch.conditions, ch.confidence, ch.usage_count, ch.success_count,
			   ch.avg_duration_ms, ch.tags, ch.metadata, ch.created_at, ch.updated_at, ch.last_used
		ORDER BY ch.usage_count DESC
		LIMIT $limit SKIP $offset`
		params = map[string]interface{}{
			"tenant_id": tenantID,
			"limit":     query.Limit,
			"offset":    query.Offset,
		}
	} else {
		cypher = `
		MATCH (ch:SkillChain)
		WHERE ch.tenant_id = $tenant_id OR $tenant_id = ""
		RETURN ch.id, ch.tenant_id, ch.name, ch.description, ch.trigger, ch.steps,
			   ch.conditions, ch.confidence, ch.usage_count, ch.success_count,
			   ch.avg_duration_ms, ch.tags, ch.metadata, ch.created_at, ch.updated_at, ch.last_used
		ORDER BY ch.usage_count DESC
		LIMIT $limit`
		params = map[string]interface{}{
			"tenant_id": tenantID,
			"limit":     query.Limit,
		}
	}

	session, release, err := c.AcquireSession(ctx)
	if err != nil {
		return nil, err
	}
	defer release()

	rec, err := session.Run(ctx, cypher, params)
	if err != nil {
		return nil, err
	}

	var chains []*types.SkillChain
	for rec.Next(ctx) {
		chains = append(chains, c.chainFromRecord(rec.Record()))
	}

	return chains, nil
}

func (c *Client) UpdateChain(ctx context.Context, chain *types.SkillChain) error {
	stepsJSON, _ := json.Marshal(chain.Steps)
	conditionsJSON, _ := json.Marshal(chain.Conditions)
	tagsJSON, _ := json.Marshal(chain.Tags)
	metadataJSON, _ := json.Marshal(chain.Metadata)

	query := `
		MATCH (ch:SkillChain {id: $id})
		SET ch.name = $name,
			ch.description = $description,
			ch.trigger = $trigger,
			ch.steps = $steps,
			ch.conditions = $conditions,
			ch.confidence = $confidence,
			ch.tags = $tags,
			ch.metadata = $metadata,
			ch.updated_at = datetime($updated_at)
		RETURN ch.id`

	session, release, err := c.AcquireSession(ctx)
	if err != nil {
		return err
	}
	defer release()

	_, err = session.Run(ctx, query, map[string]interface{}{
		"id":          chain.ID,
		"name":        chain.Name,
		"description": chain.Description,
		"trigger":     chain.Trigger,
		"steps":       string(stepsJSON),
		"conditions":  string(conditionsJSON),
		"confidence":  chain.Confidence,
		"tags":        string(tagsJSON),
		"metadata":    string(metadataJSON),
		"updated_at":  chain.UpdatedAt.Format(time.RFC3339),
	})
	return err
}

func (c *Client) DeleteChain(ctx context.Context, chainID string) error {
	query := `MATCH (ch:SkillChain {id: $id}) DELETE ch`

	session, release, err := c.AcquireSession(ctx)
	if err != nil {
		return err
	}
	defer release()

	_, err = session.Run(ctx, query, map[string]interface{}{"id": chainID})
	return err
}

func (c *Client) GetChainExecutions(ctx context.Context, chainID string, limit int) ([]*types.ChainExecution, error) {
	query := `
		MATCH (e:ChainExecution {chain_id: $chain_id})
		RETURN e.id, e.chain_id, e.status, e.results, e.started_at, e.completed_at, e.error, e.metadata
		ORDER BY e.started_at DESC
		LIMIT $limit`

	if limit <= 0 {
		limit = 10
	}

	session, release, err := c.AcquireSession(ctx)
	if err != nil {
		return nil, err
	}
	defer release()

	rec, err := session.Run(ctx, query, map[string]interface{}{
		"chain_id": chainID,
		"limit":    limit,
	})
	if err != nil {
		return nil, err
	}

	var executions []*types.ChainExecution
	for rec.Next(ctx) {
		executions = append(executions, c.chainExecutionFromRecord(rec.Record()))
	}

	return executions, nil
}

func (c *Client) UpdateChainExecution(ctx context.Context, exec *types.ChainExecution) error {
	resultsJSON, _ := json.Marshal(exec.Results)
	metadataJSON, _ := json.Marshal(exec.Metadata)

	query := `
		MERGE (e:ChainExecution {id: $id})
		SET e.chain_id = $chain_id,
			e.status = $status,
			e.results = $results,
			e.started_at = datetime($started_at),
			e.completed_at = CASE WHEN $completed_at = '' THEN null ELSE datetime($completed_at) END,
			e.error = $error,
			e.metadata = $metadata
		RETURN e.id`

	var completedAt string
	if exec.CompletedAt != nil {
		completedAt = exec.CompletedAt.Format(time.RFC3339)
	}

	session, release, err := c.AcquireSession(ctx)
	if err != nil {
		return err
	}
	defer release()

	_, err = session.Run(ctx, query, map[string]interface{}{
		"id":           exec.ID,
		"chain_id":     exec.ChainID,
		"status":       string(exec.Status),
		"results":      string(resultsJSON),
		"started_at":   exec.StartedAt.Format(time.RFC3339),
		"completed_at": completedAt,
		"error":        exec.Error,
		"metadata":     string(metadataJSON),
	})
	return err
}

func (c *Client) IncrementChainUsage(ctx context.Context, chainID string) error {
	query := `
		MATCH (ch:SkillChain {id: $id})
		SET ch.usage_count = ch.usage_count + 1,
			ch.last_used = datetime()
		RETURN ch.usage_count`

	session, release, err := c.AcquireSession(ctx)
	if err != nil {
		return err
	}
	defer release()

	_, err = session.Run(ctx, query, map[string]interface{}{"id": chainID})
	return err
}

func (c *Client) chainFromRecord(record *neo4jdriver.Record) *types.SkillChain {
	id, _ := record.Get("ch.id")
	tenantID, _ := record.Get("ch.tenant_id")
	name, _ := record.Get("ch.name")
	description, _ := record.Get("ch.description")
	trigger, _ := record.Get("ch.trigger")
	stepsStr, _ := record.Get("ch.steps")
	conditionsStr, _ := record.Get("ch.conditions")
	confidence, _ := record.Get("ch.confidence")
	usageCount, _ := record.Get("ch.usage_count")
	successCount, _ := record.Get("ch.success_count")
	avgDuration, _ := record.Get("ch.avg_duration_ms")
	tagsStr, _ := record.Get("ch.tags")
	metadataStr, _ := record.Get("ch.metadata")
	createdAt, _ := record.Get("ch.created_at")
	updatedAt, _ := record.Get("ch.updated_at")
	lastUsed, _ := record.Get("ch.last_used")

	var steps []types.ChainStep
	var conditions []types.ChainCondition
	var tags []string
	var metadata map[string]interface{}
	json.Unmarshal([]byte(getString(stepsStr)), &steps)
	json.Unmarshal([]byte(getString(conditionsStr)), &conditions)
	json.Unmarshal([]byte(getString(tagsStr)), &tags)
	json.Unmarshal([]byte(getString(metadataStr)), &metadata)

	var lastUsedTime *time.Time
	lastUsedTime = parseTime(lastUsed)

	return &types.SkillChain{
		ID:           getString(id),
		TenantID:     getString(tenantID),
		Name:         getString(name),
		Description:  getString(description),
		Trigger:      getString(trigger),
		Steps:        steps,
		Conditions:   conditions,
		Confidence:   getFloat32(confidence),
		UsageCount:   getInt64(usageCount),
		SuccessCount: getInt64(successCount),
		AvgDuration:  getInt64(avgDuration),
		Tags:         tags,
		Metadata:     metadata,
		CreatedAt:    getTime(createdAt),
		UpdatedAt:    getTime(updatedAt),
		LastUsed:     lastUsedTime,
	}
}

func (c *Client) chainExecutionFromRecord(record *neo4jdriver.Record) *types.ChainExecution {
	id, _ := record.Get("e.id")
	chainID, _ := record.Get("e.chain_id")
	status, _ := record.Get("e.status")
	resultsStr, _ := record.Get("e.results")
	startedAt, _ := record.Get("e.started_at")
	completedAt, _ := record.Get("e.completed_at")
	errorStr, _ := record.Get("e.error")
	metadataStr, _ := record.Get("e.metadata")

	var results []types.ChainStepResult
	var metadata map[string]interface{}
	json.Unmarshal([]byte(getString(resultsStr)), &results)
	json.Unmarshal([]byte(getString(metadataStr)), &metadata)

	return &types.ChainExecution{
		ID:          getString(id),
		ChainID:     getString(chainID),
		Status:      types.ChainStatus(getString(status)),
		Results:     results,
		StartedAt:   getTime(startedAt),
		CompletedAt: parseTime(completedAt),
		Error:       getString(errorStr),
		Metadata:    metadata,
	}
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
	if m, ok := v.(map[string]interface{}); ok {
		if year, ok := m["year"].(int64); ok {
			month := int64(1)
			day := int64(1)
			hour := int64(0)
			min := int64(0)
			sec := int64(0)
			if m["month"] != nil {
				if mv, ok := m["month"].(int64); ok {
					month = mv
				}
			}
			if m["day"] != nil {
				if dv, ok := m["day"].(int64); ok {
					day = dv
				}
			}
			if m["hour"] != nil {
				if hv, ok := m["hour"].(int64); ok {
					hour = hv
				}
			}
			if m["minute"] != nil {
				if minv, ok := m["minute"].(int64); ok {
					min = minv
				}
			}
			if m["second"] != nil {
				if sv, ok := m["second"].(int64); ok {
					sec = sv
				}
			}
			return time.Date(int(year), time.Month(month), int(day), int(hour), int(min), int(sec), 0, time.UTC)
		}
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
	if m, ok := v.(map[string]interface{}); ok {
		if year, ok := m["year"].(int64); ok {
			month := int64(1)
			day := int64(1)
			hour := int64(0)
			min := int64(0)
			sec := int64(0)
			if m["month"] != nil {
				if mv, ok := m["month"].(int64); ok {
					month = mv
				}
			}
			if m["day"] != nil {
				if dv, ok := m["day"].(int64); ok {
					day = dv
				}
			}
			if m["hour"] != nil {
				if hv, ok := m["hour"].(int64); ok {
					hour = hv
				}
			}
			if m["minute"] != nil {
				if minv, ok := m["minute"].(int64); ok {
					min = minv
				}
			}
			if m["second"] != nil {
				if sv, ok := m["second"].(int64); ok {
					sec = sv
				}
			}
			t := time.Date(int(year), time.Month(month), int(day), int(hour), int(min), int(sec), 0, time.UTC)
			return &t
		}
	}
	return nil
}

type APIKey struct {
	ID         string     `json:"id"`
	Key        string     `json:"key"`
	Label      string     `json:"label"`
	Scope      string     `json:"scope"` // read, write, admin
	TenantID   string     `json:"tenant_id"`
	CreatedAt  time.Time  `json:"created_at"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	UsageCount int64      `json:"usage_count"`
}

const (
	ScopeRead  = "read"
	ScopeWrite = "write"
	ScopeAdmin = "admin"
)

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
			scope: COALESCE($scope, 'read'),
			tenant_id: $tenant_id,
			created_at: datetime($created_at),
			expires_at: $expires_at,
			usage_count: 0
		})
		RETURN k.id, k.key
	`

	keyHash := auth.HashAPIKey(key.Key)
	if key.Scope == "" {
		key.Scope = ScopeRead
	}

	session, release, err := c.AcquireSession(ctx)
	if err != nil {
		return err
	}
	defer release()

	_, err = session.Run(ctx, query, map[string]interface{}{
		"id":         key.ID,
		"key_hash":   keyHash,
		"label":      key.Label,
		"scope":      key.Scope,
		"tenant_id":  key.TenantID,
		"created_at": key.CreatedAt.Format(time.RFC3339),
		"expires_at": nilIfZeroTime(key.ExpiresAt),
	})
	return err
}

func (c *Client) GetAPIKey(ctx context.Context, id string) (*APIKey, error) {
	query := `
		MATCH (k:APIKey {id: $id})
		RETURN k.id, k.key_hash, k.label, k.scope, k.tenant_id, k.created_at, k.expires_at, k.last_used_at, k.usage_count
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
	keyHash := auth.HashAPIKey(keyStr)

	query := `
		MATCH (k:APIKey {key_hash: $key_hash})
		RETURN k.id, k.key_hash, k.label, k.scope, k.tenant_id, k.created_at, k.expires_at, k.last_used_at, k.usage_count
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
		RETURN k.id, k.key_hash, k.label, k.scope, k.tenant_id, k.created_at, k.expires_at, k.last_used_at, k.usage_count
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
		RETURN k.id, k.key_hash, k.label, k.scope, k.tenant_id, k.created_at, k.expires_at, k.last_used_at, k.usage_count
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

func (c *Client) recordToAPIKey(record *neo4jdriver.Record) (*APIKey, error) {
	id, _ := record.Get("k.id")
	label, _ := record.Get("k.label")
	scope, _ := record.Get("k.scope")
	tenantID, _ := record.Get("k.tenant_id")
	createdAt, _ := record.Get("k.created_at")
	expiresAt, _ := record.Get("k.expires_at")
	lastUsedAt, _ := record.Get("k.last_used_at")
	usageCount, _ := record.Get("k.usage_count")

	return &APIKey{
		ID:         getString(id),
		Key:        "",
		Label:      getString(label),
		Scope:      getString(scope),
		TenantID:   getString(tenantID),
		CreatedAt:  getTime(createdAt),
		ExpiresAt:  parseTime(expiresAt),
		LastUsedAt: parseTime(lastUsedAt),
		UsageCount: getInt64(usageCount),
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

func nilIfZeroTime(t *time.Time) interface{} {
	if t == nil {
		return nil
	}
	return t.Format(time.RFC3339)
}
