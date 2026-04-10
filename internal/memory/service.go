package memory

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"agent-memory/internal/config"
	"agent-memory/internal/embedding"
	"agent-memory/internal/memory/neo4j"
	"agent-memory/internal/memory/qdrant"
	"agent-memory/internal/memory/types"
)

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

// ==================== Memory CRUD Operations ====================

func (s *Service) CreateMemory(ctx context.Context, mem *types.Memory) (*types.Memory, error) {
	if mem.ID == "" {
		mem.ID = uuid.New().String()
	}
	if mem.Status == "" {
		mem.Status = types.MemoryStatusActive
	}
	mem.CreatedAt = time.Now()
	mem.UpdatedAt = time.Now()

	emb, err := s.embedder.GenerateEmbedding(mem.Content)
	if err != nil {
		return nil, fmt.Errorf("generate embedding: %w", err)
	}

	metadata := s.buildMemoryMetadata(mem)
	metadata["memory_type"] = string(mem.Type)
	if mem.Category != "" {
		metadata["category"] = mem.Category
	}

	pointID, err := s.qdrant.StoreEmbedding(ctx, mem.Content, mem.ID, emb, metadata)
	if err != nil {
		return nil, fmt.Errorf("qdrant store: %w", err)
	}
	mem.EntityID = pointID

	if err := s.neo4j.CreateMemory(mem); err != nil {
		return nil, fmt.Errorf("neo4j create memory: %w", err)
	}

	return mem, nil
}

func (s *Service) GetMemory(ctx context.Context, id string) (*types.Memory, error) {
	mem, err := s.neo4j.GetMemory(id)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	mem.LastAccessed = &now
	_ = s.neo4j.UpdateMemoryAccess(id, now)

	return mem, nil
}

func (s *Service) UpdateMemory(ctx context.Context, id string, content string, metadata map[string]interface{}) error {
	mem, err := s.neo4j.GetMemory(id)
	if err != nil {
		return err
	}
	if mem.Immutable {
		return fmt.Errorf("memory is immutable and cannot be updated")
	}

	oldContent := mem.Content
	mem.Content = content
	mem.UpdatedAt = time.Now()

	if metadata != nil {
		if mem.Metadata == nil {
			mem.Metadata = make(map[string]interface{})
		}
		for k, v := range metadata {
			mem.Metadata[k] = v
		}
	}

	if err := s.neo4j.UpdateMemory(mem); err != nil {
		return fmt.Errorf("neo4j update: %w", err)
	}

	if content != oldContent {
		emb, err := s.embedder.GenerateEmbedding(content)
		if err != nil {
			return fmt.Errorf("generate embedding: %w", err)
		}
		meta := s.buildMemoryMetadata(mem)
		if err := s.qdrant.UpdateMemory(ctx, id, content, meta); err != nil {
			return fmt.Errorf("qdrant update: %w", err)
		}
		_ = s.qdrant.UpdateVector(ctx, id, emb)
	}

	_ = s.neo4j.RecordHistory(id, string(types.HistoryActionUpdate), oldContent, content, "", "")

	return nil
}

func (s *Service) DeleteMemory(ctx context.Context, id string) error {
	mem, err := s.neo4j.GetMemory(id)
	if err != nil {
		return err
	}
	if mem.Immutable {
		return fmt.Errorf("memory is immutable and cannot be deleted")
	}

	if err := s.neo4j.DeleteMemory(id); err != nil {
		return fmt.Errorf("neo4j delete: %w", err)
	}

	_ = s.qdrant.DeleteMemory(ctx, id)
	_ = s.neo4j.RecordHistory(id, string(types.HistoryActionDelete), mem.Content, "", "", "")

	return nil
}

func (s *Service) DeleteMemories(ctx context.Context, ids []string) error {
	for _, id := range ids {
		if err := s.DeleteMemory(ctx, id); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) ArchiveMemory(ctx context.Context, id string) error {
	mem, err := s.neo4j.GetMemory(id)
	if err != nil {
		return err
	}

	mem.Status = types.MemoryStatusArchived
	mem.UpdatedAt = time.Now()

	if err := s.neo4j.UpdateMemory(mem); err != nil {
		return fmt.Errorf("neo4j archive: %w", err)
	}

	_ = s.neo4j.RecordHistory(id, string(types.HistoryActionArchive), "", "", "", "")

	return nil
}

// ==================== Search Operations ====================

func (s *Service) SearchMemories(ctx context.Context, req *types.SearchRequest) ([]types.MemoryResult, error) {
	if req.Limit <= 0 {
		req.Limit = 10
	}
	if req.Limit > 100 {
		req.Limit = 100
	}
	if req.Threshold <= 0 {
		req.Threshold = 0.5
	}

	emb, err := s.embedder.GenerateEmbedding(req.Query)
	if err != nil {
		return nil, fmt.Errorf("generate embedding: %w", err)
	}

	var qdrantFilters map[string]interface{}
	if req.Filters != nil {
		qdrantFilters = s.filtersToMap(req.Filters)
	}
	if req.UserID != "" {
		if qdrantFilters == nil {
			qdrantFilters = make(map[string]interface{})
		}
		qdrantFilters["user_id"] = req.UserID
	}
	if req.OrgID != "" {
		if qdrantFilters == nil {
			qdrantFilters = make(map[string]interface{})
		}
		qdrantFilters["org_id"] = req.OrgID
	}
	if req.AgentID != "" {
		if qdrantFilters == nil {
			qdrantFilters = make(map[string]interface{})
		}
		qdrantFilters["agent_id"] = req.AgentID
	}
	if req.Category != "" {
		if qdrantFilters == nil {
			qdrantFilters = make(map[string]interface{})
		}
		qdrantFilters["category"] = req.Category
	}
	if req.MemoryType != "" {
		if qdrantFilters == nil {
			qdrantFilters = make(map[string]interface{})
		}
		qdrantFilters["memory_type"] = string(req.MemoryType)
	}

	results, err := s.qdrant.SearchSemantic(ctx, emb, req.Limit+req.Offset, req.Threshold, qdrantFilters)
	if err != nil {
		return nil, err
	}

	if req.Offset > 0 && req.Offset >= len(results) {
		return []types.MemoryResult{}, nil
	}
	if req.Offset > 0 {
		results = results[req.Offset:]
	}
	if req.Limit < len(results) {
		results = results[:req.Limit]
	}

	for i := range results {
		if results[i].Entity.ID != "" {
			mem, err := s.neo4j.GetMemory(results[i].Entity.ID)
			if err == nil {
				results[i].Metadata = mem
			}
		}
	}

	if req.Rerank && len(results) > 0 {
		rerankTopK := req.RerankTopK
		if rerankTopK <= 0 {
			rerankTopK = 20
		}
		results = s.rerankResults(req.Query, results, rerankTopK)
	}

	return results, nil
}

func (s *Service) AdvancedSearch(ctx context.Context, req *types.SearchRequest) ([]types.MemoryResult, error) {
	if req.Filters != nil && len(req.Filters.Rules) > 0 {
		matches, err := s.neo4j.AdvancedSearch(req.Filters)
		if err != nil {
			return nil, err
		}

		var results []types.MemoryResult
		for _, mem := range matches {
			emb, _ := s.embedder.GenerateEmbedding(mem.Content)
			score := float32(0)
			if emb != nil && len(req.Query) > 0 {
				queryEmb, _ := s.embedder.GenerateEmbedding(req.Query)
				score = s.cosineSimilarity(emb, queryEmb)
			}
			results = append(results, types.MemoryResult{
				Entity:   types.Entity{ID: mem.ID, Properties: mem.Metadata},
				Score:    score,
				Text:     mem.Content,
				Source:   "neo4j",
				MemoryID: mem.ID,
				Metadata: mem,
			})
		}
		return results, nil
	}
	return s.SearchMemories(ctx, req)
}

// ==================== Feedback Operations ====================

func (s *Service) AddFeedback(ctx context.Context, feedback *types.Feedback) (*types.Feedback, error) {
	if feedback.ID == "" {
		feedback.ID = uuid.New().String()
	}
	feedback.CreatedAt = time.Now()

	if err := s.neo4j.CreateFeedback(feedback); err != nil {
		return nil, fmt.Errorf("create feedback: %w", err)
	}

	_ = s.neo4j.UpdateMemoryFeedbackScore(feedback.MemoryID, feedback.Type)
	_ = s.neo4j.RecordHistory(feedback.MemoryID, string(types.HistoryActionFeedback), "", string(feedback.Type), feedback.UserID, feedback.Comment)

	return feedback, nil
}

func (s *Service) GetMemoriesByFeedback(ctx context.Context, feedbackType types.FeedbackType, limit int) ([]*types.Memory, error) {
	fbs, err := s.neo4j.GetFeedbackByType(feedbackType, limit)
	if err != nil {
		return nil, err
	}

	var memIDs []string
	for _, fb := range fbs {
		memIDs = append(memIDs, fb.MemoryID)
	}

	var memories []*types.Memory
	for _, id := range memIDs {
		mem, err := s.GetMemory(ctx, id)
		if err == nil {
			memories = append(memories, mem)
		}
	}
	return memories, nil
}

// ==================== History Operations ====================

func (s *Service) GetMemoryHistory(ctx context.Context, memoryID string) ([]types.MemoryHistory, error) {
	return s.neo4j.GetMemoryHistory(memoryID)
}

// ==================== Batch Operations ====================

func (s *Service) BatchCreateMemories(ctx context.Context, memories []*types.Memory) ([]*types.Memory, error) {
	var created []*types.Memory
	for _, mem := range memories {
		m, err := s.CreateMemory(ctx, mem)
		if err != nil {
			return created, fmt.Errorf("batch create failed at %s: %w", mem.ID, err)
		}
		created = append(created, m)
	}
	return created, nil
}

func (s *Service) BatchUpdateMemories(ctx context.Context, req *types.BatchUpdateRequest) error {
	switch req.Action {
	case "update":
		for _, id := range req.IDs {
			if err := s.UpdateMemory(ctx, id, req.Content, req.Metadata); err != nil {
				return err
			}
		}
	case "archive":
		for _, id := range req.IDs {
			if err := s.ArchiveMemory(ctx, id); err != nil {
				return err
			}
		}
	case "delete":
		return s.DeleteMemories(ctx, req.IDs)
	default:
		return fmt.Errorf("unknown batch action: %s", req.Action)
	}
	return nil
}

func (s *Service) BulkDeleteByFilter(ctx context.Context, req *types.BatchDeleteRequest) (int, error) {
	count, err := s.neo4j.BulkDeleteByFilter(req.UserID, req.OrgID, req.Category)
	if err != nil {
		return 0, err
	}

	var memIDs []string
	if req.UserID != "" {
		mems, _ := s.neo4j.GetMemoriesByUser(req.UserID)
		for _, m := range mems {
			memIDs = append(memIDs, m.ID)
		}
	}
	if req.OrgID != "" {
		mems, _ := s.neo4j.GetMemoriesByOrg(req.OrgID)
		for _, m := range mems {
			memIDs = append(memIDs, m.ID)
		}
	}

	for _, id := range memIDs {
		_ = s.qdrant.DeleteMemory(ctx, id)
	}

	return count, nil
}

// ==================== Async Operations ====================

func (s *Service) SearchMemoriesAsync(ctx context.Context, req *types.SearchRequest) (<-chan []types.MemoryResult, <-chan error) {
	resultsChan := make(chan []types.MemoryResult, 1)
	errorChan := make(chan error, 1)

	go func() {
		results, err := s.SearchMemories(ctx, req)
		if err != nil {
			errorChan <- err
			close(resultsChan)
			close(errorChan)
			return
		}
		resultsChan <- results
		close(resultsChan)
		close(errorChan)
	}()

	return resultsChan, errorChan
}

func (s *Service) CreateMemoryAsync(ctx context.Context, mem *types.Memory) (<-chan *types.Memory, <-chan error) {
	resultChan := make(chan *types.Memory, 1)
	errorChan := make(chan error, 1)

	go func() {
		m, err := s.CreateMemory(ctx, mem)
		if err != nil {
			errorChan <- err
			close(resultChan)
			close(errorChan)
			return
		}
		resultChan <- m
		close(resultChan)
		close(errorChan)
	}()

	return resultChan, errorChan
}

// ==================== Memory Expiration/TTL ====================

func (s *Service) SetMemoryExpiration(ctx context.Context, id string, expirationDate time.Time) error {
	mem, err := s.neo4j.GetMemory(id)
	if err != nil {
		return err
	}

	mem.ExpirationDate = &expirationDate
	mem.UpdatedAt = time.Now()

	return s.neo4j.UpdateMemory(mem)
}

func (s *Service) CleanupExpiredMemories(ctx context.Context) (int, error) {
	expired, err := s.neo4j.GetExpiredMemories()
	if err != nil {
		return 0, err
	}

	count := 0
	for _, mem := range expired {
		if err := s.DeleteMemory(ctx, mem.ID); err == nil {
			count++
		}
	}
	return count, nil
}

// ==================== Entity/Memory Linking ====================

func (s *Service) LinkMemoryToEntity(ctx context.Context, memoryID, entityID string) error {
	return s.neo4j.LinkMemoryEntity(memoryID, entityID)
}

func (s *Service) GetEntityMemories(ctx context.Context, entityID string, limit int) ([]types.MemoryResult, error) {
	memIDs, err := s.neo4j.GetMemoryIDsByEntity(entityID)
	if err != nil {
		return nil, err
	}

	var results []types.MemoryResult
	for _, memID := range memIDs {
		if limit > 0 && len(results) >= limit {
			break
		}
		mem, err := s.GetMemory(ctx, memID)
		if err == nil {
			results = append(results, types.MemoryResult{
				Entity:   types.Entity{ID: mem.ID},
				Text:     mem.Content,
				Source:   "linked",
				MemoryID: mem.ID,
				Metadata: mem,
			})
		}
	}
	return results, nil
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

func (s *Service) UpdateMemoryByID(ctx context.Context, id string, text string, metadata map[string]interface{}) error {
	return s.UpdateMemory(ctx, id, text, metadata)
}

func (s *Service) DeleteMemoryByID(ctx context.Context, id string) error {
	return s.DeleteMemory(ctx, id)
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

func (s *Service) GetMemoriesByUser(ctx context.Context, userID string) ([]*types.Memory, error) {
	return s.neo4j.GetMemoriesByUser(userID)
}

func (s *Service) GetMemoriesByOrg(ctx context.Context, orgID string) ([]*types.Memory, error) {
	return s.neo4j.GetMemoriesByOrg(orgID)
}

// ==================== Helper Methods ====================

func (s *Service) buildMemoryMetadata(mem *types.Memory) map[string]interface{} {
	meta := make(map[string]interface{})
	if mem.TenantID != "" {
		meta["tenant_id"] = mem.TenantID
	}
	if mem.UserID != "" {
		meta["user_id"] = mem.UserID
	}
	if mem.OrgID != "" {
		meta["org_id"] = mem.OrgID
	}
	if mem.AgentID != "" {
		meta["agent_id"] = mem.AgentID
	}
	if mem.SessionID != "" {
		meta["session_id"] = mem.SessionID
	}
	if mem.Category != "" {
		meta["category"] = mem.Category
	}
	if mem.Status != "" {
		meta["status"] = string(mem.Status)
	}
	if mem.ExpirationDate != nil {
		meta["expiration_date"] = mem.ExpirationDate.Format(time.RFC3339)
	}
	for k, v := range mem.Metadata {
		meta[k] = v
	}
	return meta
}

func (s *Service) filtersToMap(filters *types.SearchFilters) map[string]interface{} {
	result := make(map[string]interface{})
	if filters == nil {
		return result
	}

	for _, rule := range filters.Rules {
		key := s.filterKey(rule.Field, rule.Operator)
		result[key] = rule.Value
	}

	if len(filters.Nested) > 0 {
		var nested []map[string]interface{}
		for _, nf := range filters.Nested {
			nested = append(nested, s.filtersToMap(&nf))
		}
		result["_nested"] = nested
	}

	return result
}

func (s *Service) filterKey(field, operator string) string {
	switch operator {
	case "eq", "==":
		return field
	case "ne", "!=":
		return field + "_ne"
	case "gt", ">":
		return field + "_gt"
	case "gte", ">=":
		return field + "_gte"
	case "lt", "<":
		return field + "_lt"
	case "lte", "<=":
		return field + "_lte"
	case "contains":
		return field + "_contains"
	case "icontains":
		return field + "_icontains"
	case "in":
		return field + "_in"
	default:
		return field
	}
}

func (s *Service) cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}

	var dotProd, normA, normB float32
	for i := range a {
		dotProd += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}
	return dotProd / (float32(float64(normA)*float64(normB)) * 0.5)
}

func (s *Service) rerankResults(query string, results []types.MemoryResult, topK int) []types.MemoryResult {
	scored := make([]struct {
		result types.MemoryResult
		score  float32
	}, len(results))

	queryLower := strings.ToLower(query)
	for i, r := range results {
		textLower := strings.ToLower(r.Text)
		score := r.Score

		words := strings.Fields(queryLower)
		matchCount := 0
		for _, word := range words {
			if strings.Contains(textLower, word) {
				matchCount++
			}
		}
		if matchCount > 0 {
			score += float32(matchCount) * 0.1
		}

		if strings.Contains(textLower, queryLower) {
			score += 0.2
		}

		scored[i] = struct {
			result types.MemoryResult
			score  float32
		}{r, score}
	}

	for i := 0; i < len(scored)-1; i++ {
		for j := i + 1; j < len(scored); j++ {
			if scored[j].score > scored[i].score {
				scored[i], scored[j] = scored[j], scored[i]
			}
		}
	}

	if topK > len(scored) {
		topK = len(scored)
	}

	reranked := make([]types.MemoryResult, topK)
	for i := 0; i < topK; i++ {
		reranked[i] = scored[i].result
		reranked[i].Score = scored[i].score
	}

	return reranked
}
