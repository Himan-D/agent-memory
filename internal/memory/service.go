package memory

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"agent-memory/internal/compression/extractor"
	"agent-memory/internal/compression/pipeline"
	"agent-memory/internal/config"
	"agent-memory/internal/embedding"
	"agent-memory/internal/llm"
	"agent-memory/internal/memory/neo4j"
	"agent-memory/internal/memory/qdrant"
	"agent-memory/internal/memory/types"
	"agent-memory/internal/reranker"
)

type Service struct {
	graph       GraphStore
	vector      VectorStore
	embedder    *embedding.OpenAIEmbedding
	config     *config.Config
	msgBuffer   *MessageBuffer
	processor  *MemoryProcessor
	llmClient  llm.Provider
	apiKeys    neo4j.APIKeyStore
	reranker   reranker.Provider
	compressor *pipeline.CompressionPipeline
	compStats  *CompressionStats
}

type CompressionStats struct {
	mu                sync.RWMutex
	TotalProcessed     int64
	TotalTokensSaved   int64
	ExtractionsDone  int64
	RadixCompressDone int64
	AvgLatencyMs     float64
	AccuracyRetention float64
	TokenReduction   float64
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
		graph:    neo,
		vector:   qdr,
		embedder: emb,
		config:   cfg,
		apiKeys:  neo,
	}

	svc.msgBuffer = NewMessageBuffer(cfg.App.MessageBuffer, cfg.App.BufferTimeout, neo)

	if cfg.LLM.APIKey != "" || cfg.LLM.BaseURL != "" {
		llmCfg := &llm.Config{
			Provider: llm.ProviderType(cfg.LLM.Provider),
			APIKey:   cfg.LLM.APIKey,
			BaseURL:  cfg.LLM.BaseURL,
		}
		if llmCfg.Provider == "" {
			llmCfg.Provider = llm.ProviderOpenAI
		}
		svc.llmClient, _ = llm.NewProvider(llmCfg)
	}

	if svc.llmClient != nil && cfg.Memory.ProcessingEnabled {
		memCfg := &Config{
			Enabled:             cfg.Memory.ProcessingEnabled,
			AutoExtractFacts:    cfg.Memory.AutoExtractFacts,
			AutoExtractEntities: cfg.Memory.AutoExtractEntities,
			DefaultImportance:   cfg.Memory.DefaultImportance,
		}
		svc.processor = NewMemoryProcessorWithConfig(svc.llmClient, memCfg)
	}

	rerankProvider, err := reranker.NewProvider(cfg.Reranker, svc.llmClient)
	if err != nil {
		return nil, fmt.Errorf("reranker init: %w", err)
	}
	svc.reranker = rerankProvider

	svc.compStats = &CompressionStats{}

	if cfg.Compression.Enabled {
		workerCount := cfg.Compression.WorkerCount
		if workerCount <= 0 {
			workerCount = 4
		}

		var memExtractor *extractor.MemoryExtractor
		if svc.llmClient != nil {
			memExtractor = extractor.NewMemoryExtractor(svc.llmClient)
		}

		svc.compressor = pipeline.NewCompressionPipeline(workerCount, memExtractor)
		if cfg.Compression.AsyncEnabled {
			svc.compressor.Start()
		}
		log.Printf("Compression pipeline started with %d workers", workerCount)
	}

	return svc, nil
}

func (s *Service) APIKeyStore() neo4j.APIKeyStore {
	return s.apiKeys
}

func (s *Service) Close() error {
	if s.compressor != nil {
		s.compressor.Stop()
	}

	if s.msgBuffer != nil {
		if err := s.msgBuffer.Close(); err != nil {
			fmt.Printf("warn: close message buffer: %v\n", err)
		}
	}

	var errs []error
	if err := s.graph.Close(); err != nil {
		errs = append(errs, err)
	}
	if err := s.vector.Close(); err != nil {
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

	if err := s.graph.Ping(ctx); err != nil {
		status.Neo4j = fmt.Sprintf("unhealthy: %v", err)
	} else {
		status.Neo4j = "healthy"
	}

	if err := s.vector.Ping(ctx); err != nil {
		status.Qdrant = fmt.Sprintf("unhealthy: %v", err)
	} else {
		status.Qdrant = "healthy"
	}

	return status
}

func (s *Service) GetGraph() GraphStore {
	return s.graph
}

func (s *Service) GetVector() VectorStore {
	return s.vector
}

func (s *Service) GetCompressor() *pipeline.CompressionPipeline {
	return s.compressor
}

func (s *Service) CompressMemory(ctx context.Context, content string) (string, float64, error) {
	if s.compressor == nil {
		return content, 0.0, nil
	}

	done := make(chan pipeline.Result, 1)
	job := pipeline.CompressionJob{
		Content: content,
		Done:    done,
	}

	if s.compressor != nil {
		s.compressor.CompressAsync(job)
		result := <-done
		return result.Compressed, result.TokenReduction, result.Error
	}

	return content, 0.0, nil
}

func (s *Service) GetCompressionStats() (int64, int64, float64, float64, float64) {
	s.compStats.mu.RLock()
	defer s.compStats.mu.RUnlock()

	totalProcessed := s.compStats.TotalProcessed
	totalTokensSaved := s.compStats.TotalTokensSaved
	avgLatency := s.compStats.AvgLatencyMs
	accuracy := s.compStats.AccuracyRetention
	reduction := s.compStats.TokenReduction

	if s.compressor != nil {
		_, tokens, latency := s.compressor.GetStats()
		totalProcessed++
		totalTokensSaved = tokens
		avgLatency = latency
	}

	return totalProcessed, totalTokensSaved, avgLatency, accuracy, reduction
}

func (s *Service) updateCompressionStats(latencyMs float64, reduction float64) {
	s.compStats.mu.Lock()
	defer s.compStats.mu.Unlock()

	s.compStats.TotalProcessed++
	s.compStats.TotalTokensSaved += int64(float64(reduction * 1000))
	oldAvg := s.compStats.AvgLatencyMs
	count := float64(s.compStats.TotalProcessed)
	s.compStats.AvgLatencyMs = ((oldAvg * (count - 1)) + latencyMs) / count
	s.compStats.TokenReduction = reduction
}

// ==================== Memory CRUD Operations ====================

func (s *Service) CreateMemory(ctx context.Context, mem *types.Memory) (*types.Memory, error) {
	return s.CreateMemoryWithOptions(ctx, mem, false)
}

func (s *Service) CreateMemoryWithOptions(ctx context.Context, mem *types.Memory, skipProcessing bool) (*types.Memory, error) {
	if mem.ID == "" {
		mem.ID = uuid.New().String()
	}
	if mem.Status == "" {
		mem.Status = types.MemoryStatusActive
	}
	mem.CreatedAt = time.Now()
	mem.UpdatedAt = time.Now()

	contentToStore := mem.Content

	if s.processor != nil && !skipProcessing {
		result, err := s.processor.ProcessContent(ctx, mem.Content, mem.UserID, MemoryType(mem.Type))
		if err == nil {
			if result.ProcessedContent != "" {
				contentToStore = result.ProcessedContent
			}
			if len(result.Facts) > 0 {
				if mem.Metadata == nil {
					mem.Metadata = make(map[string]interface{})
				}
				mem.Metadata["facts"] = result.Facts
			}
			if len(result.Entities) > 0 {
				if mem.Metadata == nil {
					mem.Metadata = make(map[string]interface{})
				}
				mem.Metadata["entities"] = result.Entities
			}
			if result.Importance != "" {
				if mem.Metadata == nil {
					mem.Metadata = make(map[string]interface{})
				}
				mem.Metadata["importance"] = result.Importance
			}
			if len(result.Categories) > 0 {
				if mem.Category == "" {
					mem.Category = strings.Join(result.Categories, ",")
				}
			}
			if !result.ShouldStore {
				return nil, fmt.Errorf("memory does not meet importance threshold: %s", result.Reason)
			}
		}
	}

	emb, err := s.embedder.GenerateEmbedding(contentToStore)
	if err != nil {
		return nil, fmt.Errorf("generate embedding: %w", err)
	}

	metadata := s.buildMemoryMetadata(mem)
	metadata["memory_type"] = string(mem.Type)
	if mem.Category != "" {
		metadata["category"] = mem.Category
	}

	pointID, err := s.vector.StoreEmbedding(ctx, contentToStore, mem.ID, emb, metadata)
	if err != nil {
		return nil, fmt.Errorf("qdrant store: %w", err)
	}
	mem.EntityID = pointID

	if err := s.graph.CreateMemory(mem); err != nil {
		return nil, fmt.Errorf("neo4j create memory: %w", err)
	}

	return mem, nil
}

func (s *Service) InferMemoryContent(ctx context.Context, content, userID string, memType types.MemoryType) (*MemoryProcessingResult, error) {
	if s.processor == nil {
		return &MemoryProcessingResult{
			ProcessedContent: content,
			Importance:       "medium",
			ShouldStore:      true,
		}, nil
	}
	return s.processor.ProcessContent(ctx, content, userID, MemoryType(memType))
}

func (s *Service) GetMemory(ctx context.Context, id string) (*types.Memory, error) {
	mem, err := s.graph.GetMemory(id)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	mem.LastAccessed = &now
	if err := s.graph.UpdateMemoryAccess(id, now); err != nil {
		log.Printf("WARN: failed to update memory access for %s: %v", id, err)
	}

	return mem, nil
}

func (s *Service) UpdateMemory(ctx context.Context, id string, content string, metadata map[string]interface{}) error {
	mem, err := s.graph.GetMemory(id)
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

	if err := s.graph.UpdateMemory(mem); err != nil {
		return fmt.Errorf("neo4j update: %w", err)
	}

	if content != oldContent {
		emb, err := s.embedder.GenerateEmbedding(content)
		if err != nil {
			return fmt.Errorf("generate embedding: %w", err)
		}
		meta := s.buildMemoryMetadata(mem)
		if err := s.vector.UpdateMemory(ctx, id, content, meta); err != nil {
			return fmt.Errorf("qdrant update: %w", err)
		}
		if err := s.vector.UpdateVector(ctx, id, emb); err != nil {
			log.Printf("WARN: failed to update vector for memory %s: %v", id, err)
		}
	}

	if err := s.graph.RecordHistory(id, string(types.HistoryActionUpdate), oldContent, content, "", ""); err != nil {
		log.Printf("WARN: failed to record history for memory %s: %v", id, err)
	}

	return nil
}

func (s *Service) DeleteMemory(ctx context.Context, id string) error {
	mem, err := s.graph.GetMemory(id)
	if err != nil {
		return err
	}
	if mem.Immutable {
		return fmt.Errorf("memory is immutable and cannot be deleted")
	}

	if err := s.graph.DeleteMemory(id); err != nil {
		return fmt.Errorf("neo4j delete: %w", err)
	}

	if err := s.vector.DeleteMemory(ctx, id); err != nil {
		log.Printf("WARN: failed to delete vector for memory %s: %v", id, err)
	}
	if err := s.graph.RecordHistory(id, string(types.HistoryActionDelete), mem.Content, "", "", ""); err != nil {
		log.Printf("WARN: failed to record delete history for memory %s: %v", id, err)
	}

	return nil
}

func (s *Service) DeleteMemories(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	for _, id := range ids {
		mem, err := s.graph.GetMemory(id)
		if err != nil {
			return err
		}
		if mem.Immutable {
			return fmt.Errorf("memory %s is immutable and cannot be deleted", id)
		}
	}

	for _, id := range ids {
		if err := s.vector.DeleteMemory(ctx, id); err != nil {
			log.Printf("WARN: failed to delete vector for memory %s: %v", id, err)
		}
	}

	if len(ids) > 1 {
		if err := s.graph.BatchDeleteMemories(ids); err != nil {
			for _, id := range ids {
				if err := s.graph.DeleteMemory(id); err != nil {
					log.Printf("WARN: individual graph delete failed for %s: %v", id, err)
				}
			}
		}
	} else if len(ids) == 1 {
		if err := s.graph.DeleteMemory(ids[0]); err != nil {
			return fmt.Errorf("neo4j delete: %w", err)
		}
	}

	return nil
}

func (s *Service) ArchiveMemory(ctx context.Context, id string) error {
	mem, err := s.graph.GetMemory(id)
	if err != nil {
		return err
	}

	mem.Status = types.MemoryStatusArchived
	mem.UpdatedAt = time.Now()

	if err := s.graph.UpdateMemory(mem); err != nil {
		return fmt.Errorf("neo4j archive: %w", err)
	}

	if err := s.graph.RecordHistory(id, string(types.HistoryActionArchive), "", "", "", ""); err != nil {
		log.Printf("WARN: failed to record archive history for memory %s: %v", id, err)
	}

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

	results, err := s.vector.Search(ctx, emb, req.Limit+req.Offset, req.Threshold, qdrantFilters)
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
			mem, err := s.graph.GetMemory(results[i].Entity.ID)
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
		matches, err := s.graph.AdvancedSearch(req.Filters)
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

	if err := s.graph.CreateFeedback(feedback); err != nil {
		return nil, fmt.Errorf("create feedback: %w", err)
	}

	if err := s.graph.UpdateMemoryFeedbackScore(feedback.MemoryID, feedback.Type); err != nil {
		log.Printf("WARN: failed to update feedback score for memory %s: %v", feedback.MemoryID, err)
	}
	if err := s.graph.RecordHistory(feedback.MemoryID, string(types.HistoryActionFeedback), "", string(feedback.Type), feedback.UserID, feedback.Comment); err != nil {
		log.Printf("WARN: failed to record feedback history for memory %s: %v", feedback.MemoryID, err)
	}

	return feedback, nil
}

func (s *Service) GetMemoriesByFeedback(ctx context.Context, feedbackType types.FeedbackType, limit int) ([]*types.Memory, error) {
	fbs, err := s.graph.GetFeedbackByType(feedbackType, limit)
	if err != nil {
		return nil, err
	}

	if len(fbs) == 0 {
		return []*types.Memory{}, nil
	}

	var memIDs []string
	for _, fb := range fbs {
		memIDs = append(memIDs, fb.MemoryID)
	}

	return s.graph.GetMemoriesByIDs(memIDs)
}

// ==================== History Operations ====================

func (s *Service) GetMemoryHistory(ctx context.Context, memoryID string) ([]types.MemoryHistory, error) {
	return s.graph.GetMemoryHistory(memoryID)
}

// ==================== Batch Operations ====================

func (s *Service) BatchCreateMemories(ctx context.Context, memories []*types.Memory) ([]*types.Memory, error) {
	if len(memories) == 0 {
		return []*types.Memory{}, nil
	}

	processed := make([]*types.Memory, 0, len(memories))

	for _, mem := range memories {
		m, err := s.processMemoryNoGraph(ctx, mem)
		if err != nil {
			return processed, fmt.Errorf("batch create failed at %s: %w", mem.ID, err)
		}
		processed = append(processed, m)
	}

	if len(processed) > 1 {
		if err := s.graph.BatchCreateMemories(processed); err != nil {
			for _, m := range processed {
				if err := s.graph.CreateMemory(m); err != nil {
					log.Printf("WARN: individual graph create failed for %s: %v", m.ID, err)
				}
			}
		}
	} else if len(processed) == 1 {
		if err := s.graph.CreateMemory(processed[0]); err != nil {
			return processed, fmt.Errorf("neo4j create memory: %w", err)
		}
	}

	return processed, nil
}

func (s *Service) processMemoryNoGraph(ctx context.Context, mem *types.Memory) (*types.Memory, error) {
	if mem.ID == "" {
		mem.ID = uuid.New().String()
	}
	if mem.Status == "" {
		mem.Status = types.MemoryStatusActive
	}
	mem.CreatedAt = time.Now()
	mem.UpdatedAt = time.Now()

	contentToStore := mem.Content

	if s.processor != nil {
		result, err := s.processor.ProcessContent(ctx, mem.Content, mem.UserID, MemoryType(mem.Type))
		if err == nil {
			if result.ProcessedContent != "" {
				contentToStore = result.ProcessedContent
			}
			if len(result.Facts) > 0 {
				if mem.Metadata == nil {
					mem.Metadata = make(map[string]interface{})
				}
				mem.Metadata["facts"] = result.Facts
			}
			if len(result.Entities) > 0 {
				if mem.Metadata == nil {
					mem.Metadata = make(map[string]interface{})
				}
				mem.Metadata["entities"] = result.Entities
			}
			if result.Importance != "" {
				if mem.Metadata == nil {
					mem.Metadata = make(map[string]interface{})
				}
				mem.Metadata["importance"] = result.Importance
			}
			if len(result.Categories) > 0 {
				if mem.Category == "" {
					mem.Category = strings.Join(result.Categories, ",")
				}
			}
			if !result.ShouldStore {
				return nil, fmt.Errorf("memory does not meet importance threshold: %s", result.Reason)
			}
		}
	}

	emb, err := s.embedder.GenerateEmbedding(contentToStore)
	if err != nil {
		return nil, fmt.Errorf("generate embedding: %w", err)
	}

	metadata := s.buildMemoryMetadata(mem)
	metadata["memory_type"] = string(mem.Type)
	if mem.Category != "" {
		metadata["category"] = mem.Category
	}

	pointID, err := s.vector.StoreEmbedding(ctx, contentToStore, mem.ID, emb, metadata)
	if err != nil {
		return nil, fmt.Errorf("qdrant store: %w", err)
	}
	mem.EntityID = pointID

	return mem, nil
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
	count, err := s.graph.BulkDeleteByFilter(req.UserID, req.OrgID, req.Category)
	if err != nil {
		return 0, err
	}

	var memIDs []string
	if req.UserID != "" {
		mems, _ := s.graph.GetMemoriesByUser(req.UserID)
		for _, m := range mems {
			memIDs = append(memIDs, m.ID)
		}
	}
	if req.OrgID != "" {
		mems, _ := s.graph.GetMemoriesByOrg(req.OrgID)
		for _, m := range mems {
			memIDs = append(memIDs, m.ID)
		}
	}

	for _, id := range memIDs {
		if err := s.vector.DeleteMemory(ctx, id); err != nil {
			log.Printf("WARN: failed to delete vector for memory %s during bulk delete: %v", id, err)
		}
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
	mem, err := s.graph.GetMemory(id)
	if err != nil {
		return err
	}

	mem.ExpirationDate = &expirationDate
	mem.UpdatedAt = time.Now()

	return s.graph.UpdateMemory(mem)
}

func (s *Service) CleanupExpiredMemories(ctx context.Context) (int, error) {
	expired, err := s.graph.GetExpiredMemories()
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
	return s.graph.LinkMemoryEntity(memoryID, entityID)
}

func (s *Service) GetEntityMemories(ctx context.Context, entityID string, limit int) ([]types.MemoryResult, error) {
	memIDs, err := s.graph.GetMemoryIDsByEntity(entityID)
	if err != nil {
		return nil, err
	}

	if len(memIDs) == 0 {
		return []types.MemoryResult{}, nil
	}

	memories, err := s.graph.GetMemoriesByIDs(memIDs)
	if err != nil {
		return nil, err
	}

	var results []types.MemoryResult
	for _, mem := range memories {
		if limit > 0 && len(results) >= limit {
			break
		}
		results = append(results, types.MemoryResult{
			Entity:   types.Entity{ID: mem.ID},
			Text:     mem.Content,
			Source:   "linked",
			MemoryID: mem.ID,
			Metadata: mem,
		})
	}
	return results, nil
}

// ==================== Short-term Memory ====================

func (s *Service) CreateSession(agentID string, metadata map[string]interface{}) (*types.Session, error) {
	return s.graph.CreateSession(agentID, metadata)
}

func (s *Service) ListSessions() ([]*types.Session, error) {
	return s.graph.ListSessions()
}

func (s *Service) GetSession(sessionID string) (*types.Session, error) {
	sessions, err := s.graph.ListSessions()
	if err != nil {
		return nil, err
	}
	for _, sess := range sessions {
		if sess.ID == sessionID {
			return sess, nil
		}
	}
	return nil, fmt.Errorf("session not found: %s", sessionID)
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
	return s.graph.GetMessages(sessionID, limit)
}

func (s *Service) ClearContext(sessionID string) error {
	return s.graph.ClearMessages(sessionID)
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

	if err := s.graph.AddEntity(entity); err != nil {
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

		_, err := s.vector.StoreEmbedding(context.Background(), text, entity.ID, entity.Embedding, metadata)
		if err != nil {
			fmt.Printf("warn: qdrant sync failed for entity %s: %v\n", entity.ID, err)
		}
	}

	return &entity, nil
}

func (s *Service) GetEntity(id string) (*types.Entity, error) {
	return s.graph.GetEntity(id)
}

func (s *Service) ListEntities(tenantID string, limit int) ([]types.Entity, error) {
	return s.graph.ListEntities(tenantID, limit)
}

func (s *Service) AddRelation(fromID, toID, relType string, props map[string]interface{}) error {
	return s.graph.AddRelation(fromID, toID, relType, props)
}

func (s *Service) QueryGraph(cypher string, params map[string]interface{}) ([]map[string]interface{}, error) {
	return s.graph.QueryGraph(cypher, params)
}

func (s *Service) Traverse(fromEntityID string, depth int) ([]types.Path, error) {
	if depth <= 0 {
		depth = 3
	}
	return s.graph.Traverse(fromEntityID, depth)
}

func (s *Service) GetEntityRelations(entityID string, relType string) ([]types.Relation, error) {
	return s.graph.GetEntityRelations(entityID, relType)
}

// ==================== Long-term Semantic Memory ====================

func (s *Service) StoreEmbedding(text string, entityID string, metadata map[string]interface{}) (string, error) {
	emb, err := s.embedder.GenerateEmbedding(text)
	if err != nil {
		return "", fmt.Errorf("generate embedding: %w", err)
	}

	return s.vector.StoreEmbedding(context.Background(), text, entityID, emb, metadata)
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

	results, err := s.vector.Search(context.Background(), emb, limit, scoreThreshold, filters)
	if err != nil {
		return nil, err
	}

	for i := range results {
		if results[i].Entity.ID != "" {
			entity, err := s.graph.GetEntity(results[i].Entity.ID)
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
	entity, err := s.graph.GetEntity(entityID)
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

	_, err = s.vector.StoreEmbedding(context.Background(), text, entity.ID, entity.Embedding, metadata)
	return err
}

func (s *Service) BatchSyncEntities(entityIDs []string) error {
	entities := make([]types.Entity, 0, len(entityIDs))
	for _, id := range entityIDs {
		entity, err := s.graph.GetEntity(id)
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

		_, err := s.vector.StoreEmbedding(context.Background(), texts[i], entity.ID, embeddings[i], metadata)
		if err != nil {
			fmt.Printf("warn: qdrant store %s failed: %v\n", entity.ID, err)
		} else {
			syncedIDs = append(syncedIDs, entity.ID)
		}
	}

	if len(syncedIDs) > 0 {
		if err := s.graph.BatchUpdateSyncTime(syncedIDs); err != nil {
			fmt.Printf("warn: batch update sync time failed: %v\n", err)
		}
	}

	return nil
}

func (s *Service) GetMemoriesByUser(ctx context.Context, userID string) ([]*types.Memory, error) {
	return s.graph.GetMemoriesByUser(userID)
}

func (s *Service) GetMemoriesByOrg(ctx context.Context, orgID string) ([]*types.Memory, error) {
	return s.graph.GetMemoriesByOrg(orgID)
}

func (s *Service) GetAllMemories(ctx context.Context) ([]*types.Memory, error) {
	return s.graph.GetAllMemories()
}

// ==================== Memory Linking (Relationships) ====================

func (s *Service) LinkMemories(ctx context.Context, fromID, toID string, linkType types.MemoryLinkType, weight float64) (*types.MemoryLink, error) {
	link := &types.MemoryLink{
		ID:     uuid.New().String(),
		FromID: fromID,
		ToID:   toID,
		Type:   linkType,
		Weight: weight,
	}

	if err := s.graph.CreateMemoryLink(link); err != nil {
		return nil, fmt.Errorf("create memory link: %w", err)
	}

	return link, nil
}

func (s *Service) GetMemoryLinks(ctx context.Context, memoryID string) ([]types.MemoryLink, error) {
	return s.graph.GetMemoryLinks(memoryID)
}

func (s *Service) DeleteMemoryLink(ctx context.Context, linkID string) error {
	return s.graph.DeleteMemoryLink(linkID)
}

func (s *Service) GetRelatedMemories(ctx context.Context, memoryID string, linkType types.MemoryLinkType, limit int) ([]*types.Memory, error) {
	links, err := s.GetMemoryLinks(ctx, memoryID)
	if err != nil {
		return nil, err
	}

	var memories []*types.Memory
	for _, link := range links {
		if linkType != "" && link.Type != linkType {
			continue
		}

		var relatedID string
		if link.FromID == memoryID {
			relatedID = link.ToID
		} else {
			relatedID = link.FromID
		}

		if mem, err := s.GetMemory(ctx, relatedID); err == nil {
			memories = append(memories, mem)
			if limit > 0 && len(memories) >= limit {
				break
			}
		}
	}

	return memories, nil
}

// ==================== Memory Versioning ====================

func (s *Service) SaveMemoryVersion(ctx context.Context, memoryID, content, createdBy string) (*types.MemoryVersion, error) {
	mem, err := s.GetMemory(ctx, memoryID)
	if err != nil {
		return nil, fmt.Errorf("get memory: %w", err)
	}

	version := &types.MemoryVersion{
		ID:        uuid.New().String(),
		MemoryID:  memoryID,
		Version:   mem.Version + 1,
		Content:   content,
		Metadata:  mem.Metadata,
		CreatedBy: createdBy,
		CreatedAt: time.Now(),
	}

	if err := s.graph.CreateMemoryVersion(version); err != nil {
		return nil, fmt.Errorf("create version: %w", err)
	}

	mem.Version = version.Version
	if err := s.graph.UpdateMemory(mem); err != nil {
		return nil, fmt.Errorf("update memory version: %w", err)
	}

	return version, nil
}

func (s *Service) GetMemoryVersions(ctx context.Context, memoryID string) ([]types.MemoryVersion, error) {
	return s.graph.GetMemoryVersions(memoryID)
}

func (s *Service) RestoreMemoryVersion(ctx context.Context, memoryID, versionID string) error {
	versions, err := s.GetMemoryVersions(ctx, memoryID)
	if err != nil {
		return err
	}

	var targetVersion *types.MemoryVersion
	for i := range versions {
		if versions[i].ID == versionID {
			targetVersion = &versions[i]
			break
		}
	}

	if targetVersion == nil {
		return fmt.Errorf("version not found: %s", versionID)
	}

	currentMem, err := s.GetMemory(ctx, memoryID)
	if err != nil {
		return err
	}

	_, err = s.SaveMemoryVersion(ctx, memoryID, currentMem.Content, "restore")
	if err != nil {
		return fmt.Errorf("save current version: %w", err)
	}

	return s.UpdateMemory(ctx, memoryID, targetVersion.Content, targetVersion.Metadata)
}

// ==================== Hybrid Search (Semantic + Keyword) ====================

func (s *Service) HybridSearch(ctx context.Context, req *types.HybridSearchRequest) ([]types.MemoryResult, error) {
	if req.SemanticLimit <= 0 {
		req.SemanticLimit = 10
	}
	if req.KeywordLimit <= 0 {
		req.KeywordLimit = 10
	}
	if req.RerankLimit <= 0 {
		req.RerankLimit = 10
	}

	var semanticResults []types.MemoryResult
	var keywordResults []types.MemoryResult
	var err error

	semanticResults, err = s.SearchMemories(ctx, &types.SearchRequest{
		Query:      req.Query,
		Limit:      req.SemanticLimit,
		Threshold:  req.Threshold,
		MemoryType: req.MemoryType,
		UserID:     req.UserID,
		OrgID:      req.OrgID,
		AgentID:    req.AgentID,
		Category:   req.Category,
	})
	if err != nil {
		return nil, fmt.Errorf("semantic search: %w", err)
	}

	keywordResults, err = s.keywordSearch(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("keyword search: %w", err)
	}

	combined := s.mergeSearchResults(semanticResults, keywordResults, req.Boost)

	if req.DateFrom != nil || req.DateTo != nil {
		combined = s.filterByDateRange(combined, req.DateFrom, req.DateTo)
	}

	if req.Tags != nil && len(req.Tags) > 0 {
		combined = s.filterByTags(combined, req.Tags)
	}

	if req.Importance != "" {
		combined = s.filterByImportance(combined, req.Importance)
	}

	if req.Rerank && s.reranker != nil && len(combined) > 0 {
		combined, err = s.reranker.Rerank(ctx, req.Query, combined, req.RerankLimit)
		if err != nil {
			return combined, fmt.Errorf("rerank: %w", err)
		}
	}

	return combined, nil
}

func (s *Service) keywordSearch(ctx context.Context, req *types.HybridSearchRequest) ([]types.MemoryResult, error) {
	if req.Filters == nil {
		req.Filters = &types.SearchFilters{}
	}

	req.Filters.Rules = append(req.Filters.Rules, types.SearchFilter{
		Field:    "content",
		Operator: "contains",
		Value:    req.Query,
	})

	return s.AdvancedSearch(ctx, &types.SearchRequest{
		Query:      req.Query,
		Limit:      req.KeywordLimit,
		Filters:    req.Filters,
		MemoryType: req.MemoryType,
		UserID:     req.UserID,
		OrgID:      req.OrgID,
		AgentID:    req.AgentID,
		Category:   req.Category,
	})
}

func (s *Service) mergeSearchResults(semantic, keyword []types.MemoryResult, boost float32) []types.MemoryResult {
	seen := make(map[string]bool)
	var result []types.MemoryResult

	for _, r := range semantic {
		if !seen[r.Entity.ID] {
			seen[r.Entity.ID] = true
			result = append(result, r)
		}
	}

	for _, r := range keyword {
		if !seen[r.Entity.ID] {
			seen[r.Entity.ID] = true
			existing := false
			for i := range result {
				if result[i].Entity.ID == r.Entity.ID {
					result[i].Score = result[i].Score + (r.Score * boost)
					existing = true
					break
				}
			}
			if !existing {
				r.Score = r.Score * boost
				result = append(result, r)
			}
		}
	}

	for i := 0; i < len(result)-1; i++ {
		for j := i + 1; j < len(result); j++ {
			if result[j].Score > result[i].Score {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	return result
}

func (s *Service) filterByDateRange(results []types.MemoryResult, from, to *time.Time) []types.MemoryResult {
	var filtered []types.MemoryResult
	for _, r := range results {
		if r.Metadata == nil {
			continue
		}
		createdAt := r.Metadata.CreatedAt
		if createdAt.IsZero() {
			continue
		}
		if from != nil && createdAt.Before(*from) {
			continue
		}
		if to != nil && createdAt.After(*to) {
			continue
		}
		filtered = append(filtered, r)
	}
	return filtered
}

func (s *Service) filterByTags(results []types.MemoryResult, tags []string) []types.MemoryResult {
	var filtered []types.MemoryResult
	for _, r := range results {
		if r.Metadata == nil || r.Metadata.Tags == nil {
			continue
		}
		for _, tag := range tags {
			for _, memTag := range r.Metadata.Tags {
				if tag == memTag {
					filtered = append(filtered, r)
					break
				}
			}
		}
	}
	return filtered
}

func (s *Service) filterByImportance(results []types.MemoryResult, importance types.ImportanceLevel) []types.MemoryResult {
	var filtered []types.MemoryResult
	for _, r := range results {
		if r.Metadata == nil {
			continue
		}
		if r.Metadata.Importance == importance {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

// ==================== Memory Statistics & Analytics ====================

func (s *Service) GetMemoryStats(ctx context.Context, userID, orgID string) (*types.MemoryStats, error) {
	var memories []*types.Memory
	var err error

	if userID != "" {
		memories, err = s.GetMemoriesByUser(ctx, userID)
	} else if orgID != "" {
		memories, err = s.GetMemoriesByOrg(ctx, orgID)
	} else {
		return nil, fmt.Errorf("userID or orgID required")
	}

	if err != nil {
		return nil, err
	}

	stats := &types.MemoryStats{
		TotalMemories: int64(len(memories)),
		ByCategory:    make(map[string]int64),
		ByType:        make(map[string]int64),
		ByImportance:  make(map[string]int64),
		ByStatus:      make(map[string]int64),
		TopTags:       []types.TagCount{},
	}

	var totalAccess int64
	tagCounts := make(map[string]int64)
	now := time.Now()

	for _, mem := range memories {
		stats.ByStatus[string(mem.Status)]++

		if mem.Category != "" {
			stats.ByCategory[mem.Category]++
		}

		if mem.Type != "" {
			stats.ByType[string(mem.Type)]++
		}

		if mem.Importance != "" {
			stats.ByImportance[string(mem.Importance)]++
		}

		if mem.Tags != nil {
			for _, tag := range mem.Tags {
				tagCounts[tag]++
			}
		}

		totalAccess += mem.AccessCount

		if mem.ExpirationDate != nil && mem.ExpirationDate.Before(now) {
			stats.ExpiredMemories++
		}

		daysSinceCreation := now.Sub(mem.CreatedAt).Hours() / 24
		if daysSinceCreation <= 7 {
			stats.RecentMemories++
		}
	}

	if len(memories) > 0 {
		stats.AvgAccessCount = float64(totalAccess) / float64(len(memories))
	}

	for tag, count := range tagCounts {
		stats.TopTags = append(stats.TopTags, types.TagCount{Tag: tag, Count: count})
	}

	for i := 0; i < len(stats.TopTags)-1; i++ {
		for j := i + 1; j < len(stats.TopTags); j++ {
			if stats.TopTags[j].Count > stats.TopTags[i].Count {
				stats.TopTags[i], stats.TopTags[j] = stats.TopTags[j], stats.TopTags[i]
			}
		}
	}

	if len(stats.TopTags) > 10 {
		stats.TopTags = stats.TopTags[:10]
	}

	return stats, nil
}

func (s *Service) GetMemoryInsights(ctx context.Context, userID, orgID string) ([]types.MemoryInsight, error) {
	stats, err := s.GetMemoryStats(ctx, userID, orgID)
	if err != nil {
		return nil, err
	}

	var insights []types.MemoryInsight

	if stats.TotalMemories > 100 {
		insights = append(insights, types.MemoryInsight{
			Type:        "high_memory_volume",
			Description: fmt.Sprintf("You have %d memories stored. Consider running compaction to optimize.", stats.TotalMemories),
		})
	}

	if stats.RecentMemories > stats.TotalMemories/2 {
		insights = append(insights, types.MemoryInsight{
			Type:        "recent_activity",
			Description: "Most of your memories are from the last 7 days.",
		})
	}

	var lowImportanceCount int64
	for imp, count := range stats.ByImportance {
		if imp == string(types.ImportanceLow) {
			lowImportanceCount = count
		}
	}
	if lowImportanceCount > stats.TotalMemories/3 {
		insights = append(insights, types.MemoryInsight{
			Type:        "low_importance",
			Description: "A significant portion of your memories are marked as low importance. Consider reviewing them.",
		})
	}

	return insights, nil
}

// ==================== Pagination ====================

func (s *Service) ListMemoriesPaginated(ctx context.Context, userID, orgID string, page, pageSize int) (*types.PaginatedResponse, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	var memories []*types.Memory
	var err error
	var total int64

	if userID != "" {
		memories, err = s.GetMemoriesByUser(ctx, userID)
	} else if orgID != "" {
		memories, err = s.GetMemoriesByOrg(ctx, orgID)
	} else {
		return nil, fmt.Errorf("userID or orgID required")
	}

	if err != nil {
		return nil, err
	}

	total = int64(len(memories))

	start := (page - 1) * pageSize
	end := start + pageSize

	if start >= len(memories) {
		memories = []*types.Memory{}
	} else {
		if end > len(memories) {
			end = len(memories)
		}
		memories = memories[start:end]
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	return &types.PaginatedResponse{
		Items:      memories,
		Page:       page,
		PageSize:   pageSize,
		TotalItems: total,
		TotalPages: totalPages,
		HasMore:    page < totalPages,
	}, nil
}

// ==================== Export/Import ====================

func (s *Service) ExportMemories(ctx context.Context, userID, orgID string) (*types.MemoryExport, error) {
	var memories []*types.Memory
	var err error

	if userID != "" {
		memories, err = s.GetMemoriesByUser(ctx, userID)
	} else if orgID != "" {
		memories, err = s.GetMemoriesByOrg(ctx, orgID)
	} else {
		return nil, fmt.Errorf("userID or orgID required")
	}

	if err != nil {
		return nil, err
	}

	var memTypes []types.Memory
	for _, m := range memories {
		memTypes = append(memTypes, *m)
	}

	return &types.MemoryExport{
		Version:    "1.0",
		ExportedAt: time.Now(),
		Memories:   memTypes,
	}, nil
}

func (s *Service) ImportMemories(ctx context.Context, imp *types.MemoryImport) (int, error) {
	imported := 0

	for _, mem := range imp.Memories {
		if imp.Overwrite {
			existing, _ := s.GetMemory(ctx, mem.ID)
			if existing != nil {
				_ = s.DeleteMemory(ctx, mem.ID)
			}
		}

		mem.ID = ""
		created, err := s.CreateMemory(ctx, &mem)
		if err != nil {
			continue
		}
		if created != nil {
			imported++
		}
	}

	return imported, nil
}

// ==================== Access Tracking ====================

func (s *Service) IncrementAccessCount(ctx context.Context, memoryID string) error {
	mem, err := s.GetMemory(ctx, memoryID)
	if err != nil {
		return err
	}

	mem.AccessCount++
	mem.LastAccessed = &time.Time{}

	now := time.Now()
	mem.LastAccessed = &now

	return s.graph.UpdateMemoryAccess(memoryID, now)
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

type CompactionResult struct {
	MergedCount        int           `json:"merged_count"`
	ArchivedCount      int           `json:"archived_count"`
	DeletedCount       int           `json:"deleted_count"`
	SummarizedCount    int           `json:"summarized_count"`
	KeyPointsExtracted int           `json:"key_points_extracted"`
	TotalMemories      int           `json:"total_memories"`
	ProcessedMemories  int           `json:"processed_memories"`
	Duration           time.Duration `json:"duration"`
	Errors             []string      `json:"errors,omitempty"`
}

type CompactionConfig struct {
	SimilarityThreshold float64       `json:"similarity_threshold"`
	MaxMemoryAge        time.Duration `json:"max_memory_age"`
	MinMemoryLength     int           `json:"min_memory_length"`
	MaxMemoriesPerUser  int           `json:"max_memories_per_user"`
	CompressionRatio    float64       `json:"compression_ratio"`
	EnableMerging       bool          `json:"enable_merging"`
	EnableArchiving     bool          `json:"enable_archiving"`
	EnableDedup         bool          `json:"enable_dedup"`
	EnableSummarize     bool          `json:"enable_summarize"`
	SummarizeMaxWords   int           `json:"summarize_max_words"`
}

func (s *Service) RunCompaction(ctx context.Context, userID, orgID string) (*CompactionResult, error) {
	cfg := &CompactionConfig{
		SimilarityThreshold: 0.92,
		MaxMemoryAge:        30 * 24 * time.Hour,
		MinMemoryLength:     100,
		MaxMemoriesPerUser:  1000,
		CompressionRatio:    0.6,
		EnableMerging:       true,
		EnableArchiving:     true,
		EnableDedup:         true,
		EnableSummarize:     true,
		SummarizeMaxWords:   150,
	}

	result := &CompactionResult{}

	var memories []*types.Memory
	var err error

	if userID != "" {
		memories, err = s.GetMemoriesByUser(ctx, userID)
	} else if orgID != "" {
		memories, err = s.GetMemoriesByOrg(ctx, orgID)
	} else {
		return nil, fmt.Errorf("either userID or orgID required")
	}

	if err != nil {
		return nil, fmt.Errorf("fetch memories: %w", err)
	}

	result.TotalMemories = len(memories)
	result.ProcessedMemories = len(memories)

	start := time.Now()

	activeMemories := make([]*types.Memory, 0)
	for _, m := range memories {
		if m.Status == types.MemoryStatusActive && m.Content != "" {
			activeMemories = append(activeMemories, m)
		}
	}

	if cfg.EnableArchiving {
		cutoff := time.Now().Add(-cfg.MaxMemoryAge)
		for _, m := range activeMemories {
			if m.Immutable {
				continue
			}
			if m.CreatedAt.Before(cutoff) {
				if err := s.ArchiveMemory(ctx, m.ID); err == nil {
					result.ArchivedCount++
				}
			}
		}
	}

	if cfg.EnableDedup {
		seen := make(map[string]string)
		for _, m := range activeMemories {
			if m.Immutable {
				continue
			}
			normalized := strings.ToLower(strings.Join(strings.Fields(m.Content), " "))
			if prev, exists := seen[normalized]; exists {
				if err := s.DeleteMemory(ctx, m.ID); err == nil {
					result.DeletedCount++
					if err := s.graph.RecordHistory(m.ID, string(types.HistoryActionDelete), m.Content, "", "compaction", fmt.Sprintf("Duplicate of %s", prev)); err != nil {
						log.Printf("WARN: failed to record delete history for memory %s during compaction: %v", m.ID, err)
					}
				}
				continue
			}
			seen[normalized] = m.ID
		}
	}

	if cfg.EnableSummarize {
		for _, m := range activeMemories {
			if m.Immutable {
				continue
			}
			if len(m.Content) < cfg.MinMemoryLength {
				continue
			}
			if m.Metadata != nil {
				if v, ok := m.Metadata["summarized"]; ok {
					if b, ok := v.(bool); ok && b {
						continue
					}
				}
			}

			summarized, err := s.summarizeMemoryHelper(ctx, m, cfg.SummarizeMaxWords)
			if err == nil && summarized {
				result.SummarizedCount++
			}
		}
	}

	result.Duration = time.Since(start)
	return result, nil
}

func (s *Service) summarizeMemoryHelper(ctx context.Context, mem *types.Memory, maxWords int) (bool, error) {
	if mem.Content == "" {
		return false, nil
	}

	words := strings.Fields(mem.Content)
	if len(words) <= maxWords {
		return false, nil
	}

	var summary strings.Builder
	summary.WriteString(fmt.Sprintf("Summary of memory (%d words -> %d):\n\n", len(words), maxWords))

	sentences := strings.Split(mem.Content, ". ")
	var keySentences []string
	wordCount := 0

	for _, sentence := range sentences {
		sentence = strings.TrimSpace(sentence)
		if sentence == "" {
			continue
		}

		sWords := strings.Fields(sentence)
		if wordCount+len(sWords) > maxWords {
			break
		}

		keySentences = append(keySentences, sentence)
		wordCount += len(sWords)
	}

	result := strings.Join(keySentences, ". ")
	if !strings.HasSuffix(result, ".") && len(keySentences) > 0 {
		result += "."
	}

	summary.WriteString(result)

	oldContent := mem.Content
	mem.Content = summary.String()
	mem.UpdatedAt = time.Now()

	if mem.Metadata == nil {
		mem.Metadata = make(map[string]interface{})
	}
	mem.Metadata["summarized"] = true
	mem.Metadata["original_length"] = len(oldContent)
	mem.Metadata["summarized_at"] = time.Now().Format(time.RFC3339)

	if err := s.graph.UpdateMemory(mem); err != nil {
		return false, err
	}

	if err := s.graph.RecordHistory(mem.ID, string(types.HistoryActionUpdate), oldContent, summary.String(), "compaction", "Auto-summarized"); err != nil {
		log.Printf("WARN: failed to record summarization history for memory %s: %v", mem.ID, err)
	}

	return true, nil
}

func (s *Service) RunTargetedCompaction(ctx context.Context, memoryIDs []string, action string) (*CompactionResult, error) {
	result := &CompactionResult{}

	memories := make([]*types.Memory, 0, len(memoryIDs))
	for _, id := range memoryIDs {
		if mem, err := s.GetMemory(ctx, id); err == nil {
			memories = append(memories, mem)
		}
	}

	result.TotalMemories = len(memories)
	result.ProcessedMemories = len(memories)
	start := time.Now()

	switch action {
	case "summarize":
		for _, mem := range memories {
			if summarized, err := s.summarizeMemoryHelper(ctx, mem, 150); err == nil && summarized {
				result.SummarizedCount++
			}
		}
	case "archive":
		for _, mem := range memories {
			if err := s.ArchiveMemory(ctx, mem.ID); err == nil {
				result.ArchivedCount++
			}
		}
	case "delete":
		for _, mem := range memories {
			if err := s.DeleteMemory(ctx, mem.ID); err == nil {
				result.DeletedCount++
			}
		}
	default:
		return nil, fmt.Errorf("unknown action: %s", action)
	}

	result.Duration = time.Since(start)
	return result, nil
}

func (s *Service) CompactNegativeFeedback(ctx context.Context, limit int) (*CompactionResult, error) {
	result := &CompactionResult{}

	memories, err := s.GetMemoriesByFeedback(ctx, types.FeedbackNegative, limit)
	if err != nil {
		return nil, err
	}

	result.TotalMemories = len(memories)
	start := time.Now()

	for _, mem := range memories {
		if mem.Metadata != nil {
			if v, ok := mem.Metadata["summarized"]; ok {
				if b, ok := v.(bool); ok && b {
					continue
				}
			}
		}

		if summarized, err := s.summarizeMemoryHelper(ctx, mem, 100); err == nil && summarized {
			result.SummarizedCount++
		}
	}

	result.Duration = time.Since(start)
	return result, nil
}

// ==================== Skill Service Methods ====================

func (s *Service) CreateSkill(ctx context.Context, skill *types.Skill) error {
	if skill.TenantID == "" {
		skill.TenantID = "default"
	}
	return s.graph.CreateSkill(ctx, skill)
}

func (s *Service) ListSkills(ctx context.Context, tenantID, domain string, limit, offset int) ([]*types.Skill, error) {
	if tenantID == "" {
		tenantID = "default"
	}
	if limit <= 0 {
		limit = 50
	}
	return s.graph.ListSkills(ctx, tenantID, domain, limit, offset)
}

func (s *Service) GetSkill(ctx context.Context, skillID string) (*types.Skill, error) {
	return s.graph.GetSkill(ctx, skillID)
}

func (s *Service) UpdateSkill(ctx context.Context, skill *types.Skill) error {
	return s.graph.UpdateSkill(ctx, skill)
}

func (s *Service) DeleteSkill(ctx context.Context, skillID string) error {
	return s.graph.DeleteSkill(ctx, skillID)
}

func (s *Service) SearchSkillsByTrigger(ctx context.Context, trigger string, limit int) ([]*types.Skill, error) {
	if limit <= 0 {
		limit = 20
	}
	return s.graph.GetSkillsByTrigger(ctx, trigger, limit)
}

func (s *Service) GetSkillsByDomain(ctx context.Context, domain string, limit int) ([]*types.Skill, error) {
	if limit <= 0 {
		limit = 50
	}
	return s.graph.GetSkillsByDomain(ctx, domain, limit)
}

func (s *Service) IncrementSkillUsage(ctx context.Context, skillID string) error {
	return s.graph.IncrementSkillUsage(ctx, skillID)
}

func (s *Service) SuggestSkills(ctx context.Context, trigger, context string, limit int) ([]*types.Skill, error) {
	if s.processor == nil {
		return s.graph.GetSkillsByTrigger(ctx, trigger, limit)
	}

	existingSkills, err := s.graph.GetSkillsByTrigger(ctx, trigger, limit*2)
	if err != nil || len(existingSkills) == 0 {
		return s.graph.GetSkillsByTrigger(ctx, trigger, limit)
	}

	var extractedSkills []ExtractedSkill
	for _, skill := range existingSkills {
		extractedSkills = append(extractedSkills, ExtractedSkill{
			Name:       skill.Name,
			Domain:     skill.Domain,
			Trigger:    skill.Trigger,
			Action:     skill.Action,
			Confidence: skill.Confidence,
			Examples:   skill.Examples,
			Tags:       skill.Tags,
		})
	}

	suggestions, err := s.processor.SuggestProcedure(ctx, trigger, context, extractedSkills)
	if err != nil || len(suggestions) == 0 {
		return existingSkills[:min(limit, len(existingSkills))], nil
	}

	var skills []*types.Skill
	for _, sug := range suggestions {
		for _, skill := range existingSkills {
			if len(skills) >= limit {
				break
			}
			if sug.SkillID == "" || sug.SkillID == skill.ID {
				skills = append(skills, skill)
			}
		}
	}

	return skills, nil
}

func (s *Service) SynthesizeSkills(ctx context.Context, skillIDs []string) (*types.SkillSynthesis, error) {
	if s.processor == nil {
		return nil, fmt.Errorf("LLM processor not available")
	}

	var skills []*types.Skill
	for _, id := range skillIDs {
		skill, err := s.graph.GetSkill(ctx, id)
		if err != nil {
			continue
		}
		skills = append(skills, skill)
	}

	if len(skills) < 2 {
		return nil, fmt.Errorf("need at least 2 skills to synthesize")
	}

	var extractedSkills []ExtractedSkill
	for _, skill := range skills {
		extractedSkills = append(extractedSkills, ExtractedSkill{
			Name:       skill.Name,
			Domain:     skill.Domain,
			Trigger:    skill.Trigger,
			Action:     skill.Action,
			Confidence: skill.Confidence,
			Examples:   skill.Examples,
			Tags:       skill.Tags,
		})
	}

	result, err := s.processor.SynthesizeSkills(ctx, extractedSkills)
	if err != nil {
		return nil, fmt.Errorf("synthesize skills: %w", err)
	}

	synthesized := &types.Skill{
		ID:            uuid.New().String(),
		Name:          result.SynthesizedSkill.Name,
		Domain:        result.SynthesizedSkill.Domain,
		Trigger:       result.SynthesizedSkill.Trigger,
		Action:        result.SynthesizedSkill.Action,
		Confidence:    result.SynthesizedSkill.Confidence,
		SourceMemory:  skillIDs[0],
		Verified:      false,
		HumanReviewed: false,
		Version:       1,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := s.graph.CreateSkill(ctx, synthesized); err != nil {
		return nil, fmt.Errorf("create synthesized skill: %w", err)
	}

	return &types.SkillSynthesis{
		ID:             uuid.New().String(),
		SourceSkillIDs: skillIDs,
		ResultSkill:    synthesized,
		Status:         "completed",
		Reason:         result.Reason,
		CreatedAt:      time.Now(),
	}, nil
}

func (s *Service) ExtractSkills(ctx context.Context, content, userID, agentID string) (*SkillExtractionResult, error) {
	if s.processor == nil {
		return &SkillExtractionResult{Skills: []ExtractedSkill{}}, nil
	}

	result, err := s.processor.ExtractSkills(ctx, content, userID, agentID)
	if err != nil {
		return nil, fmt.Errorf("extract skills: %w", err)
	}

	var skills []*types.Skill
	for _, extracted := range result.Skills {
		skill := &types.Skill{
			ID:            uuid.New().String(),
			TenantID:      "default",
			Name:          extracted.Name,
			Domain:        extracted.Domain,
			Trigger:       extracted.Trigger,
			Action:        extracted.Action,
			Confidence:    extracted.Confidence,
			SourceMemory:  content[:min(100, len(content))],
			CreatedBy:     userID,
			Verified:      false,
			HumanReviewed: false,
			Version:       1,
			Tags:          extracted.Tags,
			Examples:      extracted.Examples,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		if err := s.graph.CreateSkill(ctx, skill); err == nil {
			skills = append(skills, skill)

			if s.config.Memory.ProcessingEnabled {
				review := &types.SkillReview{
					ID:        uuid.New().String(),
					TenantID:  skill.TenantID,
					SkillID:   skill.ID,
					Status:    types.ReviewStatusPending,
					CreatedAt: time.Now(),
				}
				if err := s.graph.CreateSkillReview(ctx, review); err != nil {
					log.Printf("WARN: failed to create skill review for skill %s: %v", skill.ID, err)
				}
			}
		}
	}

	return &SkillExtractionResult{
		Skills:      extractedSkillsFromType(skills),
		ShouldStore: result.ShouldStore,
		Reason:      result.Reason,
	}, nil
}

func extractedSkillsFromType(skills []*types.Skill) []ExtractedSkill {
	var result []ExtractedSkill
	for _, skill := range skills {
		result = append(result, ExtractedSkill{
			Name:       skill.Name,
			Domain:     skill.Domain,
			Trigger:    skill.Trigger,
			Action:     skill.Action,
			Confidence: skill.Confidence,
			Examples:   skill.Examples,
			Tags:       skill.Tags,
		})
	}
	return result
}

// ==================== Skill Chain Methods ====================

func (s *Service) CreateChain(ctx context.Context, chain *types.SkillChain) error {
	if chain.ID == "" {
		chain.ID = uuid.New().String()
	}
	if chain.TenantID == "" {
		chain.TenantID = "default"
	}
	chain.CreatedAt = time.Now()
	chain.UpdatedAt = time.Now()
	return s.graph.CreateChain(ctx, chain)
}

func (s *Service) GetChain(ctx context.Context, chainID string) (*types.SkillChain, error) {
	return s.graph.GetChain(ctx, chainID)
}

func (s *Service) ListChains(ctx context.Context, tenantID string, query *types.ChainQuery) ([]*types.SkillChain, error) {
	return s.graph.ListChains(ctx, tenantID, query)
}

func (s *Service) UpdateChain(ctx context.Context, chain *types.SkillChain) error {
	chain.UpdatedAt = time.Now()
	return s.graph.UpdateChain(ctx, chain)
}

func (s *Service) DeleteChain(ctx context.Context, chainID string) error {
	return s.graph.DeleteChain(ctx, chainID)
}

func (s *Service) ExecuteChain(ctx context.Context, req *types.ChainExecutionRequest) (*types.ChainExecution, error) {
	chain, err := s.graph.GetChain(ctx, req.ChainID)
	if err != nil {
		return nil, fmt.Errorf("get chain: %w", err)
	}

	execution := &types.ChainExecution{
		ID:        uuid.New().String(),
		ChainID:   chain.ID,
		Status:    types.ChainStatusRunning,
		Results:   []types.ChainStepResult{},
		StartedAt: time.Now(),
	}

	ctx, cancel := context.WithTimeout(ctx, time.Duration(req.TimeoutMs)*time.Millisecond)
	if req.TimeoutMs <= 0 {
		req.TimeoutMs = 30000
	}
	defer cancel()

	for _, step := range chain.Steps {
		select {
		case <-ctx.Done():
			execution.Status = types.ChainStatusFailed
			execution.Error = "execution timeout"
			return execution, ctx.Err()
		default:
		}

		stepResult := s.executeChainStep(ctx, chain.ID, step, req.Context)
		execution.Results = append(execution.Results, stepResult)

		if !stepResult.Success {
			execution.Status = types.ChainStatusFailed
			execution.Error = fmt.Sprintf("step %d failed: %s", step.Order, stepResult.Error)
			now := time.Now()
			execution.CompletedAt = &now
			s.graph.UpdateChainExecution(ctx, execution)
			return execution, nil
		}
	}

	execution.Status = types.ChainStatusCompleted
	now := time.Now()
	execution.CompletedAt = &now

	if err := s.graph.UpdateChainExecution(ctx, execution); err != nil {
		fmt.Printf("warn: update chain execution: %v\n", err)
	}

	s.graph.IncrementChainUsage(ctx, chain.ID)

	return execution, nil
}

func (s *Service) executeChainStep(ctx context.Context, chainID string, step types.ChainStep, context map[string]interface{}) types.ChainStepResult {
	result := types.ChainStepResult{
		StepOrder: step.Order,
		SkillID:   step.SkillID,
	}

	start := time.Now()

	skill, err := s.graph.GetSkill(ctx, step.SkillID)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("skill not found: %s", step.SkillID)
		result.DurationMs = time.Since(start).Milliseconds()
		return result
	}

	if step.ContinueIf != "" && !s.evaluateCondition(ctx, step.ContinueIf, context) {
		result.Success = true
		result.Output = "skipped due to condition"
		result.DurationMs = time.Since(start).Milliseconds()
		return result
	}

	result.Success = true
	result.Output = fmt.Sprintf("executed skill: %s", skill.Name)
	result.DurationMs = time.Since(start).Milliseconds()

	return result
}

func (s *Service) evaluateCondition(ctx context.Context, condition string, context map[string]interface{}) bool {
	if s.processor == nil {
		return true
	}

	triggerCtx := ""
	for k, v := range context {
		triggerCtx += fmt.Sprintf("%s: %v; ", k, v)
	}

	prompt := fmt.Sprintf(`Given context: %s

Evaluate this condition: %s

Return "true" if the condition is met, "false" otherwise. Only return true or false.`, triggerCtx, condition)

	resp, err := s.llmClient.Complete(ctx, &llm.CompletionRequest{
		Model:       "gpt-4o",
		Messages:    []llm.Message{{Role: "user", Content: prompt}},
		Temperature: 0,
		MaxTokens:   10,
	})
	if err != nil {
		return true
	}

	answer := strings.ToLower(strings.TrimSpace(resp.Content))
	return strings.HasPrefix(answer, "true")
}

func (s *Service) ExtractChains(ctx context.Context, skillIDs []string) ([]*types.SkillChain, error) {
	if s.processor == nil {
		return nil, fmt.Errorf("LLM processor not available")
	}

	var skills []*types.Skill
	for _, id := range skillIDs {
		skill, err := s.graph.GetSkill(ctx, id)
		if err != nil {
			continue
		}
		skills = append(skills, skill)
	}

	if len(skills) < 2 {
		return nil, fmt.Errorf("need at least 2 skills to extract chains")
	}

	var extractedSkills []ExtractedSkill
	for _, skill := range skills {
		extractedSkills = append(extractedSkills, ExtractedSkill{
			Name:       skill.Name,
			Domain:     skill.Domain,
			Trigger:    skill.Trigger,
			Action:     skill.Action,
			Confidence: skill.Confidence,
		})
	}

	result, err := s.processor.ExtractChains(ctx, extractedSkills)
	if err != nil {
		return nil, fmt.Errorf("extract chains: %w", err)
	}

	var chains []*types.SkillChain
	for _, chainResult := range result.Chains {
		chain := &types.SkillChain{
			ID:          uuid.New().String(),
			Name:        chainResult.Name,
			Description: chainResult.Description,
			Trigger:     chainResult.Trigger,
			Confidence:  chainResult.Confidence,
			Tags:        chainResult.Tags,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		for i, stepResult := range chainResult.Steps {
			for _, skill := range skills {
				if skill.Name == stepResult.SkillName {
					chain.Steps = append(chain.Steps, types.ChainStep{
						SkillID:    skill.ID,
						SkillName:  skill.Name,
						Order:      i + 1,
						ContinueIf: stepResult.ContinueIf,
					})
					break
				}
			}
		}

		if len(chain.Steps) >= 2 {
			chains = append(chains, chain)
		}
	}

	return chains, nil
}

func (s *Service) GetChainExecutions(ctx context.Context, chainID string, limit int) ([]*types.ChainExecution, error) {
	return s.graph.GetChainExecutions(ctx, chainID, limit)
}

// ==================== Agent Service Methods ====================

func (s *Service) CreateAgent(ctx context.Context, agent *types.Agent) error {
	if agent.TenantID == "" {
		agent.TenantID = "default"
	}
	return s.graph.CreateAgent(ctx, agent)
}

func (s *Service) GetAgent(ctx context.Context, agentID string) (*types.Agent, error) {
	return s.graph.GetAgent(ctx, agentID)
}

func (s *Service) UpdateAgent(ctx context.Context, agent *types.Agent) error {
	return s.graph.UpdateAgent(ctx, agent)
}

func (s *Service) DeleteAgent(ctx context.Context, agentID string) error {
	return s.graph.DeleteAgent(ctx, agentID)
}

func (s *Service) ListAgents(ctx context.Context, tenantID string, limit, offset int) ([]*types.Agent, int64, error) {
	if tenantID == "" {
		tenantID = "default"
	}
	if limit <= 0 {
		limit = 50
	}
	return s.graph.ListAgents(ctx, tenantID, limit, offset)
}

// ==================== Agent Group Service Methods ====================

func (s *Service) CreateAgentGroup(ctx context.Context, group *types.AgentGroup) error {
	if group.TenantID == "" {
		group.TenantID = "default"
	}
	return s.graph.CreateAgentGroup(ctx, group)
}

func (s *Service) GetAgentGroup(ctx context.Context, groupID string) (*types.AgentGroup, error) {
	return s.graph.GetAgentGroup(ctx, groupID)
}

func (s *Service) UpdateAgentGroup(ctx context.Context, group *types.AgentGroup) error {
	return s.graph.UpdateAgentGroup(ctx, group)
}

func (s *Service) DeleteAgentGroup(ctx context.Context, groupID string) error {
	return s.graph.DeleteAgentGroup(ctx, groupID)
}

func (s *Service) ListAgentGroups(ctx context.Context, tenantID string, limit, offset int) ([]*types.AgentGroup, int64, error) {
	if tenantID == "" {
		tenantID = "default"
	}
	if limit <= 0 {
		limit = 50
	}
	return s.graph.ListAgentGroups(ctx, tenantID, limit, offset)
}

func (s *Service) AddAgentToGroup(ctx context.Context, agentID, groupID string, role types.MemberRole) error {
	return s.graph.AddAgentToGroup(ctx, agentID, groupID, role)
}

func (s *Service) RemoveAgentFromGroup(ctx context.Context, agentID, groupID string) error {
	return s.graph.RemoveAgentFromGroup(ctx, agentID, groupID)
}

func (s *Service) GetGroupSkills(ctx context.Context, groupID string, limit int) ([]*types.Skill, error) {
	if limit <= 0 {
		limit = 50
	}
	return s.graph.GetGroupSkills(ctx, groupID, limit)
}

func (s *Service) GetGroupMemories(ctx context.Context, groupID string) ([]*types.Memory, error) {
	return s.graph.GetGroupMemories(ctx, groupID)
}

func (s *Service) ShareMemoryToGroup(ctx context.Context, memoryID, groupID string) error {
	return s.graph.ShareMemoryToGroup(ctx, memoryID, groupID, "")
}

// ==================== Review Service Methods ====================

func (s *Service) ListPendingReviews(ctx context.Context, tenantID string) ([]*types.SkillReview, error) {
	if tenantID == "" {
		tenantID = "default"
	}
	return s.graph.ListPendingReviews(ctx, tenantID)
}

func (s *Service) GetReview(ctx context.Context, reviewID string) (*types.SkillReview, error) {
	return s.graph.GetReview(ctx, reviewID)
}

func (s *Service) ProcessReview(ctx context.Context, reviewID string, approved bool, notes string) error {
	return s.graph.ProcessReview(ctx, reviewID, approved, notes)
}
