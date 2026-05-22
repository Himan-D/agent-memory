package memory

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"agent-memory/internal/audit"
	"agent-memory/internal/config"
	"agent-memory/internal/embedding"
	"agent-memory/internal/llm"
	"agent-memory/internal/memory/decay"
	"agent-memory/internal/memory/neo4j"
	"agent-memory/internal/memory/ontology"
	"agent-memory/internal/memory/pool"
	"agent-memory/internal/memory/privacy"
	"agent-memory/internal/memory/provenance"
	"agent-memory/internal/memory/qdrant"
	"agent-memory/internal/memory/scoring"
	searchpkg "agent-memory/internal/memory/search"
	"agent-memory/internal/memory/temporal"
	"agent-memory/internal/memory/tier"
	"agent-memory/internal/memory/types"
	"agent-memory/internal/reranker"
	"agent-memory/internal/retrieval"
	"agent-memory/internal/webhook"
)

type Service struct {
	graph              GraphStore
	vector             VectorStore
	neo4jClient        *neo4j.Client
	embedder           *embedding.OpenAIEmbedding
	config             *config.Config
	msgBuffer          *MessageBuffer
	processor          *MemoryProcessor
	llmClient          llm.Provider
	apiKeys            neo4j.APIKeyStore
	reranker           reranker.Provider
	compStats          *CompressionStats
	multiSignal        *retrieval.MultiSignalRetrieval
	multiSignalAdapter *retrieval.ServiceAdapter
	ontologies         []*ontology.Ontology
	ontologyLoader     *ontology.Loader
	tierRouter         *tier.MemoryRouter
	auditLogger        audit.Logger
	webhookSvc         *webhook.Service
	privacyFilter      *privacy.Filter
	hooks              []*types.Hook
	context            map[string]interface{}

	// New subsystem references
	temporalScorer       *temporal.TemporalScorer
	phaseRotator         *temporal.PhaseRotator
	volatilityClassifier *temporal.VolatilityClassifier
	compositeScorer      *scoring.CompositeScorer
	ucbScorer            *searchpkg.UCBScorer
	searchRouter         *searchpkg.Router
	distiller            *searchpkg.Distiller
	provenanceDAG        *provenance.DAG
	creditAssigner       *provenance.CreditAssigner
	dualPool             *pool.DualPool
	decayScorer          *decay.DecayScorer

	// Runtime state
	totalRetrievals int64 // for UCB — accessed atomically
	decayEnabled    bool
	temporalEnabled bool
	compressionMode string
	tierPolicy      TierPolicy
}

type TierPolicy string

const (
	TierPolicyAggressive   TierPolicy = "aggressive"
	TierPolicyBalanced     TierPolicy = "balanced"
	TierPolicyConservative TierPolicy = "conservative"
)

func NewService(cfg *config.Config) (*Service, error) {
	neo, _ := neo4j.NewClient(cfg.Neo4j)
	qdr, _ := qdrant.NewClient(cfg.Qdrant)
	svc := &Service{
		graph: neo, vector: qdr, neo4jClient: neo, config: cfg, apiKeys: neo,
	}
	svc.msgBuffer = NewMessageBuffer(cfg.App.MessageBuffer, cfg.App.BufferTimeout, neo)
	if cfg.LLM.APIKey != "" {
		llmCfg := &llm.Config{Provider: llm.ProviderType(cfg.LLM.Provider), APIKey: cfg.LLM.APIKey}
		svc.llmClient, _ = llm.NewProvider(llmCfg)
	}
	if svc.llmClient != nil && cfg.Memory.ProcessingEnabled {
		svc.processor = NewMemoryProcessorWithConfig(svc.llmClient, nil)
	}
	svc.reranker, _ = reranker.NewProvider(cfg.Reranker, svc.llmClient)
	svc.compStats = &CompressionStats{}
	svc.privacyFilter = privacy.NewFilter(privacy.FilterConfig{Enabled: cfg.Privacy.Enabled})

	// Initialize new subsystems
	svc.temporalScorer = temporal.NewTemporalScorer(temporal.DefaultConfig())
	svc.phaseRotator = temporal.NewPhaseRotator()
	svc.volatilityClassifier = temporal.NewVolatilityClassifier()
	svc.compositeScorer = scoring.NewCompositeScorer()
	svc.ucbScorer = searchpkg.NewUCBScorer()
	svc.searchRouter = searchpkg.NewRouter()
	svc.provenanceDAG = provenance.NewDAG()
	svc.creditAssigner = provenance.NewCreditAssigner()
	svc.dualPool = pool.NewDualPool()
	svc.decayScorer = decay.NewDecayScorer(decay.DefaultConfig())
	svc.decayEnabled = true
	svc.temporalEnabled = true
	svc.compressionMode = "extract"
	svc.tierPolicy = TierPolicyBalanced

	if svc.llmClient != nil {
		svc.distiller = searchpkg.NewDistiller(svc.llmClient)
	}

	return svc, nil
}

func (s *Service) SetWebhookService(wh *webhook.Service) { s.webhookSvc = wh }
func (s *Service) GetNeo4jClient() *neo4j.Client         { return s.neo4jClient }
func (s *Service) APIKeyStore() neo4j.APIKeyStore        { return s.apiKeys }
func (s *Service) GetGraph() GraphStore                  { return s.graph }
func (s *Service) GetVector() VectorStore                { return s.vector }
func (s *Service) Close() error                          { return nil }

func (s *Service) AddToContext(sessionID string, msg interface{}) error            { return nil }
func (s *Service) GetContext(sessionID string, limit int) ([]types.Message, error) { return nil, nil }
func (s *Service) ClearContext()                                                   {}
func (s *Service) GetMessages() []map[string]interface{}                           { return nil }

func (s *Service) CreateSession(agentID string, metadata map[string]interface{}) (*types.Session, error) {
	return &types.Session{ID: generateUUID(), AgentID: agentID}, nil
}

func (s *Service) RunCompaction(ctx context.Context, userID, mode string) (*types.CompactionResult, error) {
	return nil, nil
}
func (s *Service) RunTargetedCompaction(ctx context.Context, ids []string, action string) (*types.CompactionResult, error) {
	return nil, nil
}
func (s *Service) CompactNegativeFeedback(ctx context.Context, userID string, limit int) (*types.CompactionResult, error) {
	return nil, nil
}

func (s *Service) InferMemoryContent(ctx context.Context, content, userID string, memType types.MemoryType) (*types.MemoryProcessingResult, error) {
	return &types.MemoryProcessingResult{ProcessedContent: content, Importance: "medium", ShouldStore: true}, nil
}

func (s *Service) CreateMemoryInternal(ctx context.Context, mem *types.Memory) (string, error) {
	if mem.ID == "" {
		mem.ID = generateUUID()
	}
	s.graph.CreateMemory(mem)
	return mem.ID, nil
}

func (s *Service) CreateMemory(ctx context.Context, mem *types.Memory) (*types.Memory, error) {
	if mem.ID == "" {
		mem.ID = generateUUID()
	}
	mem.CreatedAt = time.Now()
	mem.UpdatedAt = time.Now()
	if mem.Version == 0 {
		mem.Version = 1
	}

	// Set initial validity
	if mem.ValidityStatus == "" {
		mem.ValidityStatus = types.ValidityCurrent
	}

	// Compute volatility
	if s.volatilityClassifier != nil {
		mem.VolatilityScore = s.volatilityClassifier.ClassifyContent(mem.Content)
	}

	// Route to pool
	if s.dualPool != nil {
		mem.PoolType = s.dualPool.Route(mem.Content, mem.WorthScore, mem.Version == 1)
	}

	// Process with LLM if available (extract facts, entities, importance)
	if s.processor != nil {
		result, err := s.processor.ProcessContent(ctx, mem.Content, mem.UserID, MemoryType(mem.Type))
		if err == nil && result != nil {
			mem.Importance = types.ImportanceLevel(result.Importance)
			if mem.Metadata == nil {
				mem.Metadata = make(map[string]interface{})
			}
			mem.Metadata["facts"] = result.Facts
			mem.Metadata["entities"] = result.Entities
		}
	}

	// Write to graph store
	if err := s.graph.CreateMemory(mem); err != nil {
		return nil, fmt.Errorf("service: create memory graph: %w", err)
	}

	// Write embedding to vector store if embedder available
	if s.embedder != nil {
		emb, err := s.embedder.GenerateEmbeddingWithContext(ctx, mem.Content)
		if err == nil && len(emb) > 0 {
			// Apply phase rotation if temporal features enabled
			if s.phaseRotator != nil && s.temporalEnabled {
				angle := s.phaseRotator.ComputePhaseAngle(mem.VolatilityScore, 0) // age=0 for new
				mem.PhaseAngle = angle
			}
			s.vector.StoreEmbedding(ctx, mem.Content, mem.ID, emb, s.buildMemoryMetadata(mem))
		}
	}

	// Webhook notification
	if s.webhookSvc != nil {
		go s.webhookSvc.EmitEvent(context.Background(), types.WebhookEventMemoryCreated, "", mem)
	}

	return mem, nil
}

func (s *Service) GetMemory(ctx context.Context, id string) (*types.Memory, error) {
	mem, err := s.graph.GetMemory(id)
	if err != nil {
		return nil, fmt.Errorf("service: get memory: %w", err)
	}
	if mem != nil {
		mem.AccessCount++
		now := time.Now()
		mem.LastRetrievedAt = &now
		go s.graph.UpdateMemory(mem)
	}
	return mem, nil
}

func (s *Service) UpdateMemory(ctx context.Context, id, content string, meta map[string]interface{}) error {
	mem, err := s.graph.GetMemory(id)
	if err != nil {
		return fmt.Errorf("service: update memory: %w", err)
	}
	if mem == nil {
		return fmt.Errorf("service: memory not found: %s", id)
	}
	mem.PreviousVersionID = mem.ID
	mem.Version++
	mem.Content = content
	mem.UpdatedAt = time.Now()
	if meta != nil {
		if mem.Metadata == nil {
			mem.Metadata = make(map[string]interface{})
		}
		for k, v := range meta {
			mem.Metadata[k] = v
		}
	}
	return s.graph.UpdateMemory(mem)
}

func (s *Service) DeleteMemory(ctx context.Context, id string) error { return s.graph.DeleteMemory(id) }
func (s *Service) DeleteMemories(ctx context.Context, ids []string) error {
	return s.graph.BatchDeleteMemories(ids)
}
func (s *Service) ArchiveMemory(ctx context.Context, id string) error { return nil }

func (s *Service) SearchMemories(ctx context.Context, req *types.SearchRequest) ([]types.MemoryResult, error) {
	if req == nil || req.Query == "" {
		return nil, fmt.Errorf("service: search requires a query")
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 10
	}

	var results []types.MemoryResult

	// Step 1: Get embedding for query and run vector search
	if s.embedder != nil {
		emb, err := s.embedder.GenerateEmbeddingWithContext(ctx, req.Query)
		if err == nil && len(emb) > 0 {
			vectorResults, err := s.vector.Search(ctx, emb, limit*3, req.Threshold, s.filtersToMap(req.Filters))
			if err == nil {
				results = vectorResults
			}
		}
	}

	// Step 2: Apply temporal scoring
	if s.temporalEnabled && s.temporalScorer != nil && len(results) > 0 {
		results = temporal.ApplyTemporalScoring(results, s.temporalScorer, req.TimeReference)
	}

	// Step 3: Apply decay scoring
	if s.decayEnabled && s.decayScorer != nil && len(results) > 0 {
		results = decay.ApplyDecay(results, s.decayScorer)
	}

	// Step 4: Apply phase rotation adjustment and composite/UCB scoring
	for i := range results {
		mem, err := s.graph.GetMemory(results[i].MemoryID)
		if err != nil || mem == nil {
			continue
		}

		// Phase rotation
		if s.phaseRotator != nil && s.temporalEnabled {
			ageDays := time.Since(mem.CreatedAt).Hours() / 24
			angle := s.phaseRotator.ComputePhaseAngle(mem.VolatilityScore, ageDays)
			results[i].Score = float32(s.phaseRotator.ApplyPhaseRotation(float64(results[i].Score), angle))
		}

		// Step 5: Composite scoring
		if s.compositeScorer != nil {
			input := scoring.ScoreInput{
				SemanticScore:   float64(results[i].Score),
				TemporalScore:   scoring.ComputeTemporalScore(time.Since(mem.CreatedAt).Hours()/24, mem.AccessCount, 7),
				ConfidenceScore: scoring.ComputeConfidenceFromMW(mem.SuccessCount, mem.FailureCount),
				CentralityScore: scoring.ComputeCentrality(len(mem.ProvenanceEdges), 20),
			}
			output := s.compositeScorer.Score(input)
			results[i].Score = float32(output.CompositeScore)
		}

		// Step 6: UCB exploration bonus
		if s.ucbScorer != nil {
			total := atomic.LoadInt64(&s.totalRetrievals)
			results[i].Score = float32(s.ucbScorer.Score(
				total,
				mem.RetrievalCount,
				float64(results[i].Score),
				mem.WorthScore,
			))
		}
	}

	// Step 7: Rerank if requested and available
	if req.Rerank && s.reranker != nil && len(results) > 0 {
		rerankLimit := req.RerankTopK
		if rerankLimit <= 0 {
			rerankLimit = limit
		}
		reranked, err := s.reranker.Rerank(ctx, req.Query, results, rerankLimit)
		if err == nil {
			results = reranked
		}
	}

	// Sort by score descending and limit
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})
	if len(results) > limit {
		results = results[:limit]
	}

	// Increment retrieval counter
	atomic.AddInt64(&s.totalRetrievals, 1)

	return results, nil
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

func (s *Service) BatchCreateMemories(ctx context.Context, memories []*types.Memory) ([]string, error) {
	if len(memories) == 0 {
		return nil, nil
	}
	var ids []string
	for _, mem := range memories {
		created, err := s.CreateMemory(ctx, mem)
		if err != nil {
			return ids, err
		}
		ids = append(ids, created.ID)
	}
	return ids, nil
}

func (s *Service) HybridSearch(ctx context.Context, req *types.HybridSearchRequest) ([]types.MemoryResult, error) {
	if req == nil || req.Query == "" {
		return nil, nil
	}
	// Delegate to SearchMemories with translated request
	searchReq := &types.SearchRequest{
		Query:      req.Query,
		Limit:      req.SemanticLimit,
		Filters:    req.Filters,
		MemoryType: req.MemoryType,
		UserID:     req.UserID,
		OrgID:      req.OrgID,
		AgentID:    req.AgentID,
		Rerank:     req.Rerank,
		RerankTopK: req.RerankLimit,
	}
	if searchReq.Limit <= 0 {
		searchReq.Limit = 10
	}
	return s.SearchMemories(ctx, searchReq)
}

func (s *Service) GetMemoryStats(ctx context.Context, userID, orgID string) (*types.MemoryStats, error) {
	return nil, nil
}

func (s *Service) ListEntities(orgID string, limit int) ([]types.Entity, error) {
	if s.graph == nil {
		return nil, nil
	}
	return s.graph.ListEntities(orgID, limit)
}

func (s *Service) GetEntity(entityID string) (*types.Entity, error) {
	if s.graph == nil {
		return nil, nil
	}
	return s.graph.GetEntity(entityID)
}

func (s *Service) GetEntityRelations(entityID, relType string) ([]types.MemoryLink, error) {
	if s.graph == nil {
		return nil, nil
	}
	relations, err := s.graph.GetEntityRelations(entityID, relType)
	if err != nil {
		return nil, err
	}
	var links []types.MemoryLink
	for _, r := range relations {
		links = append(links, types.MemoryLink{
			ID:     r.ID,
			FromID: r.FromID,
			ToID:   r.ToID,
			Type:   types.MemoryLinkType(r.Type),
			Weight: r.Weight,
		})
	}
	return links, nil
}

func (s *Service) DeleteRelation(from, to, relType string) error {
	if s.graph == nil {
		return nil
	}
	return s.graph.DeleteRelation(from, to, relType)
}

func (s *Service) AddRelation(from, to, relType string, meta map[string]interface{}) error {
	if s.graph == nil {
		return nil
	}
	return s.graph.AddRelation(from, to, relType, meta)
}

func (s *Service) Traverse(start string, depth int) ([]types.Memory, error) {
	if s.graph == nil {
		return nil, nil
	}
	paths, err := s.graph.Traverse(start, depth)
	if err != nil {
		return nil, err
	}
	// Collect unique memory IDs from path nodes
	seen := make(map[string]bool)
	var memories []types.Memory
	for _, path := range paths {
		for _, node := range path.Nodes {
			if !seen[node.ID] {
				seen[node.ID] = true
				mem, err := s.graph.GetMemory(node.ID)
				if err == nil && mem != nil {
					memories = append(memories, *mem)
				}
			}
		}
	}
	return memories, nil
}

func (s *Service) AdvancedSearch(ctx context.Context, req *types.SearchRequest) ([]types.MemoryResult, error) {
	return s.SearchMemories(ctx, req)
}

func (s *Service) CreateMemoryWithOptions(ctx context.Context, mem *types.Memory, skip bool) (*types.Memory, error) {
	if skip {
		return mem, nil
	}
	return s.CreateMemory(ctx, mem)
}

func (s *Service) GetMemoryHistory(ctx context.Context, id string) ([]types.MemoryHistory, error) {
	if s.graph == nil {
		return nil, nil
	}
	return s.graph.GetMemoryHistory(id)
}

func (s *Service) SetMemoryExpiration(ctx context.Context, id string, exp time.Time) error {
	mem, err := s.graph.GetMemory(id)
	if err != nil {
		return fmt.Errorf("service: set expiration: %w", err)
	}
	if mem == nil {
		return fmt.Errorf("service: memory not found: %s", id)
	}
	mem.ExpirationDate = &exp
	mem.UpdatedAt = time.Now()
	return s.graph.UpdateMemory(mem)
}

func (s *Service) DeleteMemoryByID(ctx context.Context, id string) error {
	return s.graph.DeleteMemory(id)
}

func (s *Service) GetEntityMemories(ctx context.Context, id string, limit int) ([]types.MemoryResult, error) {
	if s.graph == nil {
		return nil, nil
	}
	return s.graph.SearchByEntities([]string{id}, limit)
}

func (s *Service) LinkMemoryToEntity(ctx context.Context, mid, eid string) error {
	if s.graph == nil {
		return nil
	}
	return s.graph.LinkMemoryEntity(mid, eid)
}

func (s *Service) BatchUpdateMemories(ctx context.Context, req *types.BatchUpdateRequest) error {
	if req == nil {
		return nil
	}
	for _, update := range req.Updates {
		if err := s.UpdateMemory(ctx, update.ID, update.Content, update.Metadata); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) BatchDeleteMemories(ctx context.Context, ids []string) error {
	return s.graph.BatchDeleteMemories(ids)
}

func (s *Service) GetMemoryByEntity(ctx context.Context, eid string) (*types.Memory, error) {
	if s.graph == nil {
		return nil, nil
	}
	ids, err := s.graph.GetMemoryIDsByEntity(eid)
	if err != nil || len(ids) == 0 {
		return nil, err
	}
	return s.graph.GetMemory(ids[0])
}

func (s *Service) GetEntitiesByMemory(ctx context.Context, mid string) ([]types.Entity, error) {
	// TODO: wire to neo4j entity-by-memory query when available
	return nil, nil
}

func (s *Service) GetMemoryLinks(ctx context.Context, mid string) ([]types.MemoryLink, error) {
	if s.graph == nil {
		return nil, nil
	}
	return s.graph.GetMemoryLinks(mid)
}

func (s *Service) CreateMemoryLink(ctx context.Context, link *types.MemoryLink) error {
	if s.graph == nil {
		return nil
	}
	return s.graph.CreateMemoryLink(link)
}

func (s *Service) DeleteMemoryLink(ctx context.Context, id string) error {
	if s.graph == nil {
		return nil
	}
	return s.graph.DeleteMemoryLink(id)
}

func (s *Service) SearchByEmbedding(ctx context.Context, emb []float32, limit int) ([]types.MemoryResult, error) {
	if s.vector == nil {
		return nil, nil
	}
	return s.vector.Search(ctx, emb, limit, 0, nil)
}

func (s *Service) GetMemoriesPaginated(ctx context.Context, req *types.SearchRequest) ([]types.Memory, int64, error) {
	// TODO: wire to paginated graph query when available
	return nil, 0, nil
}

func (s *Service) BulkDeleteByFilter(ctx context.Context, req *types.BatchDeleteRequest) (int, error) {
	if req == nil {
		return 0, nil
	}
	return s.graph.BulkDeleteByFilter(req.UserID, req.OrgID, req.Category)
}

func (s *Service) AddFeedback(ctx context.Context, fb *types.Feedback) (*types.Feedback, error) {
	if fb.MemoryID == "" {
		return nil, fmt.Errorf("service: feedback requires memory_id")
	}
	if fb.ID == "" {
		fb.ID = generateUUID()
	}
	fb.CreatedAt = time.Now()

	// Persist feedback in graph store
	if s.graph != nil {
		_ = s.graph.CreateFeedback(fb)
	}

	// Update MW counters on the memory
	mem, err := s.graph.GetMemory(fb.MemoryID)
	if err == nil && mem != nil {
		switch fb.Type {
		case types.FeedbackPositive:
			mem.SuccessCount++
		case types.FeedbackNegative, types.FeedbackVeryNegative:
			mem.FailureCount++
		}
		// Recompute worth score
		total := float64(mem.SuccessCount + mem.FailureCount)
		if total > 0 {
			mem.WorthScore = float64(mem.SuccessCount) / total
		}
		go s.graph.UpdateMemory(mem)

		// Credit assignment through provenance chain
		if s.creditAssigner != nil && s.provenanceDAG != nil {
			reward := 1.0
			if fb.Type == types.FeedbackNegative || fb.Type == types.FeedbackVeryNegative {
				reward = -1.0
			}
			ancestors := s.provenanceDAG.GetAncestors(fb.MemoryID, 5)
			credits := s.creditAssigner.AssignCredit(fb.MemoryID, reward, ancestors)
			for ancestorID, delta := range credits {
				if ancestorID == fb.MemoryID {
					continue // already updated above
				}
				if ancestor, err := s.graph.GetMemory(ancestorID); err == nil && ancestor != nil {
					ancestor.QValue += delta
					go s.graph.UpdateMemory(ancestor)
				}
			}
		}
	}

	return fb, nil
}

func (s *Service) GetMemoriesByFeedback(ctx context.Context, fb types.FeedbackType, limit int) ([]*types.Memory, error) {
	if s.graph == nil {
		return nil, nil
	}
	feedbacks, err := s.graph.GetFeedbackByType(fb, limit)
	if err != nil {
		return nil, err
	}
	var memories []*types.Memory
	seen := make(map[string]bool)
	for _, f := range feedbacks {
		if seen[f.MemoryID] {
			continue
		}
		seen[f.MemoryID] = true
		mem, err := s.graph.GetMemory(f.MemoryID)
		if err == nil && mem != nil {
			memories = append(memories, mem)
		}
	}
	return memories, nil
}

func (s *Service) ExportMemories(ctx context.Context, uid, oid string) (*types.MemoryExport, error) {
	var memories []*types.Memory
	var err error
	if uid != "" {
		memories, err = s.graph.GetMemoriesByUser(uid)
	} else if oid != "" {
		memories, err = s.graph.GetMemoriesByOrg(oid)
	} else {
		memories, err = s.graph.GetAllMemories()
	}
	if err != nil {
		return nil, err
	}
	var flat []types.Memory
	for _, m := range memories {
		flat = append(flat, *m)
	}
	return &types.MemoryExport{
		Version:    "1.0",
		ExportedAt: time.Now(),
		Memories:   flat,
	}, nil
}

func (s *Service) ImportMemories(ctx context.Context, imp *types.MemoryImport) (int, error) {
	if imp == nil {
		return 0, nil
	}
	count := 0
	for i := range imp.Memories {
		mem := &imp.Memories[i]
		if imp.Overwrite && mem.ID != "" {
			existing, _ := s.graph.GetMemory(mem.ID)
			if existing != nil {
				_ = s.graph.DeleteMemory(mem.ID)
			}
		}
		if _, err := s.CreateMemory(ctx, mem); err == nil {
			count++
		}
	}
	return count, nil
}

func (s *Service) QueryGraph(query string, params map[string]interface{}) ([]map[string]interface{}, error) {
	if s.graph == nil {
		return nil, nil
	}
	return s.graph.QueryGraph(query, params)
}

func (s *Service) CreateSkill(ctx context.Context, sk *types.Skill) error { return nil }
func (s *Service) ListSkills(ctx context.Context, dom, group string, lim, off int) ([]*types.Skill, error) {
	return nil, nil
}
func (s *Service) SearchSkillsByTrigger(ctx context.Context, tr string, lim int) ([]*types.Skill, error) {
	return nil, nil
}
func (s *Service) GetSkillsByDomain(ctx context.Context, dom string, lim int) ([]*types.Skill, error) {
	return nil, nil
}
func (s *Service) GetSkill(ctx context.Context, id string) (*types.Skill, error) { return nil, nil }
func (s *Service) UpdateSkill(ctx context.Context, sk *types.Skill) error        { return nil }
func (s *Service) DeleteSkill(ctx context.Context, id string) error              { return nil }
func (s *Service) ExecuteSkill(ctx context.Context, id string, p map[string]interface{}) (string, error) {
	return "", nil
}
func (s *Service) SuggestSkills(ctx context.Context, tr, c string, lim int) ([]*types.Skill, error) {
	return nil, nil
}
func (s *Service) SynthesizeSkills(ctx context.Context, ids []string) (*types.Skill, error) {
	if s.graph == nil || len(ids) < 2 {
		return nil, fmt.Errorf("need at least 2 skills and a graph store")
	}

	var skills []*types.Skill
	for _, id := range ids {
		sk, err := s.graph.GetSkill(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("get skill %s: %w", id, err)
		}
		if sk != nil {
			skills = append(skills, sk)
		}
	}

	if len(skills) < 2 {
		return nil, fmt.Errorf("need at least 2 valid skills to synthesize")
	}

	if s.processor == nil {
		return nil, fmt.Errorf("no memory processor (LLM) configured")
	}

	var extracted []ExtractedSkill
	for _, sk := range skills {
		extracted = append(extracted, ExtractedSkill{
			Name:       sk.Name,
			Domain:     sk.Domain,
			Trigger:    sk.Trigger,
			Action:     sk.Action,
			Confidence: sk.Confidence,
		})
	}

	result, err := s.processor.SynthesizeSkills(ctx, extracted)
	if err != nil {
		return nil, fmt.Errorf("synthesize: %w", err)
	}

	synthesized := &types.Skill{
		ID:         generateUUID(),
		Name:       result.SynthesizedSkill.Name,
		Domain:     result.SynthesizedSkill.Domain,
		Trigger:    result.SynthesizedSkill.Trigger,
		Action:     result.SynthesizedSkill.Action,
		Confidence: float32(result.SynthesizedSkill.Confidence),
		Verified:   false,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := s.graph.CreateSkill(ctx, synthesized); err != nil {
		return nil, fmt.Errorf("create synthesized skill: %w", err)
	}

	return synthesized, nil
}
func (s *Service) ExtractSkills(ctx context.Context, c, u, sid string) (*types.SkillExtractionResult, error) {
	return nil, nil
}
func (s *Service) UseSkill(ctx context.Context, id string) error            { return nil }
func (s *Service) IncrementSkillUsage(ctx context.Context, id string) error { return nil }
func (s *Service) GetSimilarSkills(ctx context.Context, id string, lim int) ([]*types.Skill, error) {
	return nil, nil
}
func (s *Service) CreateSkillReview(ctx context.Context, r *types.Review) (*types.Review, error) {
	return nil, nil
}
func (s *Service) ListSkillReviews(ctx context.Context, st string) ([]*types.Review, error) {
	return nil, nil
}
func (s *Service) GetSkillReview(ctx context.Context, id string) (*types.Review, error) {
	return nil, nil
}
func (s *Service) ProcessSkillReview(ctx context.Context, id string, approved bool, notes string) (*types.Review, error) {
	if s.graph == nil {
		return nil, fmt.Errorf("no graph store")
	}

	review, err := s.graph.GetReview(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get review: %w", err)
	}
	if review == nil {
		return nil, fmt.Errorf("review not found: %s", id)
	}

	if err := s.graph.ProcessReview(ctx, id, approved, notes); err != nil {
		return nil, fmt.Errorf("process review: %w", err)
	}

	status := "rejected"
	if approved {
		status = "approved"
	}

	result := &types.Review{
		ID:        id,
		SkillID:   review.SkillID,
		Status:    status,
		Notes:     notes,
		UpdatedAt: time.Now(),
	}
	if review.CreatedAt.IsZero() {
		result.CreatedAt = review.CreatedAt
	}

	return result, nil
}
func (s *Service) ProcessReview(ctx context.Context, id string, approved bool, notes string) error {
	if s.graph == nil {
		return fmt.Errorf("no graph store")
	}
	return s.graph.ProcessReview(ctx, id, approved, notes)
}
func (s *Service) CreateChain(ctx context.Context, ch *types.SkillChain) error { return nil }
func (s *Service) ListChains(ctx context.Context, oid string, q *types.ChainQuery) ([]*types.SkillChain, error) {
	return nil, nil
}
func (s *Service) GetChain(ctx context.Context, id string) (*types.SkillChain, error) {
	return nil, nil
}
func (s *Service) UpdateChain(ctx context.Context, ch *types.SkillChain) error { return nil }
func (s *Service) DeleteChain(ctx context.Context, id string) error            { return nil }
func (s *Service) ExecuteChain(ctx context.Context, req *types.ChainExecutionRequest) (*types.ChainExecution, error) {
	return nil, nil
}
func (s *Service) GetChainExecutions(ctx context.Context, id string, lim int) ([]*types.ChainExecution, error) {
	return nil, nil
}
func (s *Service) ExtractChains(ctx context.Context, ids []string) ([]*types.SkillChain, error) {
	return nil, nil
}

func (s *Service) CreateAgent(ctx context.Context, ag *types.Agent) error {
	if s.graph == nil {
		return nil
	}
	return s.graph.CreateAgent(ctx, ag)
}
func (s *Service) GetAgent(ctx context.Context, id string) (*types.Agent, error) {
	if s.graph == nil {
		return nil, nil
	}
	return s.graph.GetAgent(ctx, id)
}
func (s *Service) UpdateAgent(ctx context.Context, ag *types.Agent) error {
	if s.graph == nil {
		return nil
	}
	return s.graph.UpdateAgent(ctx, ag)
}
func (s *Service) DeleteAgent(ctx context.Context, id string) error {
	if s.graph == nil {
		return nil
	}
	return s.graph.DeleteAgent(ctx, id)
}
func (s *Service) ListAgents(ctx context.Context, oid string, lim, off int) ([]types.Agent, int64, error) {
	if s.graph == nil {
		return nil, 0, nil
	}
	agents, total, err := s.graph.ListAgents(ctx, oid, lim, off)
	if err != nil {
		return nil, 0, err
	}
	var result []types.Agent
	for _, a := range agents {
		if a != nil {
			result = append(result, *a)
		}
	}
	return result, total, nil
}
func (s *Service) CreateAgentGroup(ctx context.Context, gr *types.AgentGroup) error {
	if s.graph == nil {
		return nil
	}
	return s.graph.CreateAgentGroup(ctx, gr)
}
func (s *Service) GetAgentGroup(ctx context.Context, id string) (*types.AgentGroup, error) {
	if s.graph == nil {
		return nil, nil
	}
	return s.graph.GetAgentGroup(ctx, id)
}
func (s *Service) UpdateAgentGroup(ctx context.Context, gr *types.AgentGroup) error {
	if s.graph == nil {
		return nil
	}
	return s.graph.UpdateAgentGroup(ctx, gr)
}
func (s *Service) DeleteAgentGroup(ctx context.Context, id string) error {
	if s.graph == nil {
		return nil
	}
	return s.graph.DeleteAgentGroup(ctx, id)
}
func (s *Service) ListAgentGroups(ctx context.Context, oid string, lim, off int) ([]*types.AgentGroup, int64, error) {
	if s.graph == nil {
		return nil, 0, nil
	}
	return s.graph.ListAgentGroups(ctx, oid, lim, off)
}
func (s *Service) AddAgentToGroup(ctx context.Context, aid, gid string, r types.MemberRole) error {
	if s.graph == nil {
		return nil
	}
	return s.graph.AddAgentToGroup(ctx, aid, gid, r)
}
func (s *Service) RemoveAgentFromGroup(ctx context.Context, aid, gid string) error {
	if s.graph == nil {
		return nil
	}
	return s.graph.RemoveAgentFromGroup(ctx, aid, gid)
}
func (s *Service) GetGroupSkills(ctx context.Context, gid string, lim int) ([]*types.Skill, error) {
	if s.graph == nil {
		return nil, nil
	}
	return s.graph.GetGroupSkills(ctx, gid, lim)
}
func (s *Service) GetGroupMemories(ctx context.Context, gid string) ([]*types.Memory, error) {
	if s.graph == nil {
		return nil, nil
	}
	return s.graph.GetGroupMemories(ctx, gid)
}
func (s *Service) ShareMemoryToGroup(ctx context.Context, mid, gid string) error {
	if s.graph == nil {
		return nil
	}
	return s.graph.ShareMemoryToGroup(ctx, mid, gid, "member")
}
func (s *Service) ListPendingReviews(ctx context.Context, status string) ([]*types.Review, error) {
	// TODO: wire to paginated graph query when available
	return nil, nil
}
func (s *Service) ListSessions(ctx context.Context, uid string) ([]*types.Session, error) {
	if s.graph == nil {
		return nil, nil
	}
	return s.graph.ListSessions()
}
func (s *Service) GetReview(ctx context.Context, id string) (*types.Review, error) {
	if s.graph == nil {
		return nil, nil
	}
	review, err := s.graph.GetReview(ctx, id)
	if err != nil || review == nil {
		return nil, err
	}
	return &types.Review{
		ID:        review.ID,
		SkillID:   review.SkillID,
		Status:    string(review.Status),
		Notes:     review.Notes,
		CreatedAt: review.CreatedAt,
	}, nil
}
func (s *Service) BatchSyncEntitiesByID(ctx context.Context, ids []string) error {
	if s.graph == nil {
		return nil
	}
	return s.graph.BatchUpdateSyncTime(ids)
}
func (s *Service) BatchSyncEntities(ids []string) error {
	if s.graph == nil {
		return nil
	}
	return s.graph.BatchUpdateSyncTime(ids)
}
func (s *Service) AddEntity(en types.Entity) (*types.Entity, error) {
	if s.graph == nil {
		return &en, nil
	}
	if err := s.graph.AddEntity(en); err != nil {
		return nil, err
	}
	return &en, nil
}
func (s *Service) CleanupExpiredMemories(ctx context.Context) (int, error) {
	if s.graph == nil {
		return 0, nil
	}
	expired, err := s.graph.GetExpiredMemories()
	if err != nil {
		return 0, err
	}
	var ids []string
	for _, m := range expired {
		ids = append(ids, m.ID)
	}
	if len(ids) == 0 {
		return 0, nil
	}
	if err := s.graph.BatchDeleteMemories(ids); err != nil {
		return 0, err
	}
	return len(ids), nil
}

// Configuration methods using real state
func (s *Service) GetCompressionMode() string        { return s.compressionMode }
func (s *Service) SetCompressionMode(m string) error { s.compressionMode = m; return nil }
func (s *Service) GetTierPolicy() TierPolicy         { return s.tierPolicy }
func (s *Service) SetTierPolicy(p TierPolicy) error  { s.tierPolicy = p; return nil }
func (s *Service) SetTemporalReasoningEnabled(e bool) {
	s.temporalEnabled = e
}
func (s *Service) SetDecayEnabled(e bool) {
	s.decayEnabled = e
}
func (s *Service) IsDecayEnabled() bool            { return s.decayEnabled }
func (s *Service) IsTemporalReasoningEnabled() bool { return s.temporalEnabled }

func (s *Service) SetCompressionStats(acc, red float64) {
	s.compStats.mu.Lock()
	defer s.compStats.mu.Unlock()
	s.compStats.accuracy = acc
	s.compStats.reduction = red
}

func (s *Service) RecordCompression(tokensSaved, originalSize int64, latency float64) {
	s.compStats.mu.Lock()
	defer s.compStats.mu.Unlock()
	s.compStats.processed++
	if originalSize > 0 {
		s.compStats.reduction = float64(tokensSaved) / float64(originalSize)
	}
	if s.compStats.processed > 0 {
		s.compStats.avgLatency = (s.compStats.avgLatency*float64(s.compStats.processed-1) + latency) / float64(s.compStats.processed)
	}
}

func (s *Service) GetCompressionStats() (float64, float64, int64, float64) {
	s.compStats.mu.RLock()
	defer s.compStats.mu.RUnlock()
	return s.compStats.accuracy, s.compStats.reduction, s.compStats.processed, s.compStats.avgLatency
}

type CompressionStats struct {
	mu         sync.RWMutex
	accuracy   float64
	reduction  float64
	processed  int64
	avgLatency float64
}

func generateUUID() string { return fmt.Sprintf("%d", time.Now().UnixNano()) }

type HealthStatus struct {
	Status, Version, Uptime, Neo4j, Qdrant string
	Services                               map[string]string
	Timestamp                              time.Time
}

func (s *Service) HealthCheck(ctx context.Context) *HealthStatus {
	return &HealthStatus{Status: "healthy"}
}

func (s *Service) buildMemoryMetadata(m *types.Memory) map[string]interface{} {
	if m == nil {
		return nil
	}
	return map[string]interface{}{
		"user_id":          m.UserID,
		"org_id":           m.OrgID,
		"agent_id":         m.AgentID,
		"type":             string(m.Type),
		"importance":       string(m.Importance),
		"pool_type":        m.PoolType,
		"volatility_score": m.VolatilityScore,
		"validity_status":  m.ValidityStatus,
		"version":          m.Version,
	}
}

func (s *Service) filtersToMap(f *types.SearchFilters) map[string]interface{} {
	if f == nil {
		return nil
	}
	result := make(map[string]interface{})
	for _, rule := range f.Rules {
		result[rule.Field] = rule.Value
	}
	return result
}
