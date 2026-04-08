package types

import (
	"time"
)

// Message represents a single conversation message
type Message struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id,omitempty"`
	SessionID string    `json:"session_id"`
	Role      string    `json:"role"` // "user" or "assistant"
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

// Entity represents a knowledge graph entity
type Entity struct {
	ID         string                 `json:"id"`
	TenantID   string                 `json:"tenant_id,omitempty"`
	Type       string                 `json:"type"`
	Name       string                 `json:"name"`
	Properties map[string]interface{} `json:"properties"`
	Embedding  []float32              `json:"embedding,omitempty"`
	CreatedAt  time.Time              `json:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at"`
	LastSynced *time.Time             `json:"last_synced,omitempty"`
}

// Relation represents a relationship between entities
type Relation struct {
	ID       string                 `json:"id"`
	TenantID string                 `json:"tenant_id,omitempty"`
	FromID   string                 `json:"from_id"`
	ToID     string                 `json:"to_id"`
	Type     string                 `json:"type"`
	Weight   float64                `json:"weight"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// Session represents a conversation session
type Session struct {
	ID        string                 `json:"id"`
	TenantID  string                 `json:"tenant_id,omitempty"`
	AgentID   string                 `json:"agent_id"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// Path represents a graph traversal path
type Path struct {
	Nodes []Entity   `json:"nodes"`
	Edges []Relation `json:"edges"`
}

// MemoryResult represents a search result from memory
type MemoryResult struct {
	Entity Entity  `json:"entity"`
	Score  float32 `json:"score"`
	Text   string  `json:"text"`
	Source string  `json:"source"` // "neo4j" or "qdrant"
}

// UnifiedMemory is the primary interface for agent memory operations
type UnifiedMemory interface {
	// Short-term memory (Neo4j - conversation context)
	AddToContext(sessionID string, msg Message) error
	GetContext(sessionID string, limit int) ([]Message, error)
	ClearContext(sessionID string) error
	CreateSession(agentID string, metadata map[string]interface{}) (*Session, error)

	// Knowledge graph operations (Neo4j)
	AddEntity(entity Entity) (*Entity, error)
	GetEntity(id string) (*Entity, error)
	AddRelation(fromID, toID, relType string, props map[string]interface{}) error
	QueryGraph(cypher string, params map[string]interface{}) ([]map[string]interface{}, error)
	Traverse(fromEntityID string, depth int) ([]Path, error)
	GetEntityRelations(entityID string, relType string) ([]Relation, error)

	// Long-term semantic memory (Qdrant)
	StoreEmbedding(text string, entityID string, metadata map[string]interface{}) (string, error)
	SearchSemantic(query string, limit int, scoreThreshold float32, filters map[string]interface{}) ([]MemoryResult, error)
	UpdateMemory(id string, text string, metadata map[string]interface{}) error
	DeleteMemory(id string) error

	// Cross-database operations
	SyncEntityToVector(entityID string) error
	BatchSyncEntities(entityIDs []string) error

	// Lifecycle
	Close() error
}
