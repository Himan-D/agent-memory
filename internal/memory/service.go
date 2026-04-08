package memory

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"agent-memory/internal/config"
	"agent-memory/internal/embedding"
	"agent-memory/internal/memory/neo4j"
	"agent-memory/internal/memory/qdrant"
	"agent-memory/internal/memory/types"
)

// Service implements UnifiedMemory with Neo4j + Qdrant
type Service struct {
	neo4j     *neo4j.Client
	qdrant    *qdrant.Client
	embedder  *embedding.OpenAIEmbedding
	config    *config.Config
	msgBuffer *MessageBuffer
}

func NewService(cfg *config.Config) (*Service, error) {
	neo, err := neo4j.NewClient(cfg.Neo4j)
	if err != nil {
		return nil, fmt.Errorf("neo4j init: %w", err)
	}

	qdr, err := qdrant.NewClient(cfg.Qdrant)
	if err != nil {
		return nil, fmt.Errorf("qdrant init: %w", err)
	}

	emb := embedding.NewOpenAI(cfg.OpenAI)

	svc := &Service{
		neo4j:    neo,
		qdrant:   qdr,
		embedder: emb,
		config:   cfg,
	}

	svc.msgBuffer = NewMessageBuffer(cfg.App.MessageBuffer, cfg.App.BufferTimeout, neo)

	return svc, nil
}

func (s *Service) Close() error {
	if s.msgBuffer != nil {
		if err := s.msgBuffer.Close(); err != nil {
			fmt.Printf("warn: close message buffer: %v\n", err)
		}
	}

	var errs []error
	if err := s.neo4j.Close(); err != nil {
		errs = append(errs, err)
	}
	if err := s.qdrant.Close(); err != nil {
		errs = append(errs, err)
	}
	if len(errs) > 0 {
		return fmt.Errorf("close errors: %v", errs)
	}
	return nil
}

type HealthStatus struct {
	Neo4j  string `json:"neo4j"`
	Qdrant string `json:"qdrant"`
}

func (s *Service) HealthCheck(ctx context.Context) HealthStatus {
	status := HealthStatus{Neo4j: "unhealthy", Qdrant: "unhealthy"}

	if err := s.neo4j.Ping(ctx); err != nil {
		status.Neo4j = fmt.Sprintf("unhealthy: %v", err)
	} else {
		status.Neo4j = "healthy"
	}

	if err := s.qdrant.Ping(ctx); err != nil {
		status.Qdrant = fmt.Sprintf("unhealthy: %v", err)
	} else {
		status.Qdrant = "healthy"
	}

	return status
}

// ==================== Short-term Memory ====================

func (s *Service) CreateSession(agentID string, metadata map[string]interface{}) (*types.Session, error) {
	return s.neo4j.CreateSession(agentID, metadata)
}

func (s *Service) AddToContext(sessionID string, msg types.Message) error {
	msg.ID = uuid.New().String()
	msg.SessionID = sessionID
	msg.Timestamp = time.Now()
	return s.msgBuffer.Add(msg)
}

func (s *Service) GetContext(sessionID string, limit int) ([]types.Message, error) {
	if limit <= 0 {
		limit = s.config.App.ContextWindow
	}
	return s.neo4j.GetMessages(sessionID, limit)
}

func (s *Service) ClearContext(sessionID string) error {
	return s.neo4j.ClearMessages(sessionID)
}

// ==================== Knowledge Graph ====================

func (s *Service) AddEntity(entity types.Entity) (*types.Entity, error) {
	if s.config.OpenAI.APIKey != "" {
		text := entity.Name
		if entity.Type != "" {
			text = entity.Type + ": " + text
		}
		emb, err := s.embedder.GenerateEmbedding(text)
		if err == nil {
			entity.Embedding = emb
		}
	}

	if entity.ID == "" {
		entity.ID = uuid.New().String()
	}

	entity.CreatedAt = time.Now()
	entity.UpdatedAt = time.Now()

	if err := s.neo4j.AddEntity(entity); err != nil {
		return nil, fmt.Errorf("neo4j add entity: %w", err)
	}

	if entity.Embedding != nil {
		text := entity.Name
		metadata := map[string]interface{}{
			"entity_type": entity.Type,
			"created_at":  entity.CreatedAt.Format(time.RFC3339),
		}
		for k, v := range entity.Properties {
			metadata[k] = v
		}

		_, err := s.qdrant.StoreEmbedding(context.Background(), text, entity.ID, entity.Embedding, metadata)
		if err != nil {
			fmt.Printf("warn: qdrant sync failed for entity %s: %v\n", entity.ID, err)
		}
	}

	return &entity, nil
}

func (s *Service) GetEntity(id string) (*types.Entity, error) {
	return s.neo4j.GetEntity(id)
}

func (s *Service) AddRelation(fromID, toID, relType string, props map[string]interface{}) error {
	return s.neo4j.AddRelation(fromID, toID, relType, props)
}

func (s *Service) QueryGraph(cypher string, params map[string]interface{}) ([]map[string]interface{}, error) {
	return s.neo4j.QueryGraph(cypher, params)
}

func (s *Service) Traverse(fromEntityID string, depth int) ([]types.Path, error) {
	if depth <= 0 {
		depth = 3
	}
	return s.neo4j.Traverse(fromEntityID, depth)
}

func (s *Service) GetEntityRelations(entityID string, relType string) ([]types.Relation, error) {
	return s.neo4j.GetEntityRelations(entityID, relType)
}

// ==================== Long-term Semantic Memory ====================

func (s *Service) StoreEmbedding(text string, entityID string, metadata map[string]interface{}) (string, error) {
	emb, err := s.embedder.GenerateEmbedding(text)
	if err != nil {
		return "", fmt.Errorf("generate embedding: %w", err)
	}

	return s.qdrant.StoreEmbedding(context.Background(), text, entityID, emb, metadata)
}

func (s *Service) SearchSemantic(query string, limit int, scoreThreshold float32, filters map[string]interface{}) ([]types.MemoryResult, error) {
	emb, err := s.embedder.GenerateEmbedding(query)
	if err != nil {
		return nil, fmt.Errorf("generate query embedding: %w", err)
	}

	if scoreThreshold <= 0 {
		scoreThreshold = 0.5
	}
	if limit <= 0 {
		limit = 10
	}

	results, err := s.qdrant.SearchSemantic(context.Background(), emb, limit, scoreThreshold, filters)
	if err != nil {
		return nil, err
	}

	for i := range results {
		if results[i].Entity.ID != "" {
			entity, err := s.neo4j.GetEntity(results[i].Entity.ID)
			if err == nil {
				results[i].Entity = *entity
			}
		}
	}

	return results, nil
}

func (s *Service) UpdateMemory(id string, text string, metadata map[string]interface{}) error {
	return s.qdrant.UpdateMemory(context.Background(), id, text, metadata)
}

func (s *Service) DeleteMemory(id string) error {
	return s.qdrant.DeleteMemory(context.Background(), id)
}

// ==================== Cross-database Sync ====================

func (s *Service) SyncEntityToVector(entityID string) error {
	entity, err := s.neo4j.GetEntity(entityID)
	if err != nil {
		return fmt.Errorf("get entity: %w", err)
	}

	if s.config.OpenAI.APIKey != "" {
		text := entity.Name
		if entity.Type != "" {
			text = entity.Type + ": " + text
		}
		emb, err := s.embedder.GenerateEmbedding(text)
		if err != nil {
			return fmt.Errorf("generate embedding: %w", err)
		}
		entity.Embedding = emb
	}

	if entity.Embedding == nil {
		return fmt.Errorf("no embedding available for entity %s", entityID)
	}

	text := entity.Name
	metadata := map[string]interface{}{
		"entity_type": entity.Type,
	}
	for k, v := range entity.Properties {
		metadata[k] = v
	}

	_, err = s.qdrant.StoreEmbedding(context.Background(), text, entity.ID, entity.Embedding, metadata)
	return err
}

func (s *Service) BatchSyncEntities(entityIDs []string) error {
	entities := make([]types.Entity, 0, len(entityIDs))
	for _, id := range entityIDs {
		entity, err := s.neo4j.GetEntity(id)
		if err != nil {
			fmt.Printf("warn: get entity %s failed: %v\n", id, err)
			continue
		}
		entities = append(entities, *entity)
	}

	if len(entities) == 0 {
		return nil
	}

	texts := make([]string, len(entities))
	for i, e := range entities {
		text := e.Name
		if e.Type != "" {
			text = e.Type + ": " + text
		}
		texts[i] = text
	}

	embeddings, err := s.embedder.GenerateBatchEmbeddings(texts)
	if err != nil {
		return fmt.Errorf("batch embed: %w", err)
	}

	syncedIDs := make([]string, 0, len(entities))
	for i, entity := range entities {
		metadata := map[string]interface{}{
			"entity_type": entity.Type,
		}
		for k, v := range entity.Properties {
			metadata[k] = v
		}

		_, err := s.qdrant.StoreEmbedding(context.Background(), texts[i], entity.ID, embeddings[i], metadata)
		if err != nil {
			fmt.Printf("warn: qdrant store %s failed: %v\n", entity.ID, err)
		} else {
			syncedIDs = append(syncedIDs, entity.ID)
		}
	}

	if len(syncedIDs) > 0 {
		if err := s.neo4j.BatchUpdateSyncTime(syncedIDs); err != nil {
			fmt.Printf("warn: batch update sync time failed: %v\n", err)
		}
	}

	return nil
}
