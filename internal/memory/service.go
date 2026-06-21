package memory

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	"agent-memory/internal/audit"
	"agent-memory/internal/compression/pipeline"
	"agent-memory/internal/config"
	"agent-memory/internal/embedding"
	"agent-memory/internal/llm"
	"agent-memory/internal/memory/chroma"
	"agent-memory/internal/memory/decay"
	"agent-memory/internal/memory/neo4j"
	"agent-memory/internal/memory/ontology"
	"agent-memory/internal/memory/pgvector"
	"agent-memory/internal/memory/pinecone"
	"agent-memory/internal/memory/pool"
	"agent-memory/internal/memory/privacy"
	"agent-memory/internal/memory/provenance"
	"agent-memory/internal/memory/qdrant"
	"agent-memory/internal/memory/safety"
	"agent-memory/internal/memory/scoring"
	searchpkg "agent-memory/internal/memory/search"
	"agent-memory/internal/memory/self_improve"
	"agent-memory/internal/memory/temporal"
	"agent-memory/internal/memory/tier"
	"agent-memory/internal/memory/types"
	"agent-memory/internal/metrics"
	"agent-memory/internal/reranker"
	"agent-memory/internal/retrieval"
	"agent-memory/internal/webhook"
)

type Service struct {
	graph               GraphStore
	vector              VectorStore
	neo4jClient         *neo4j.Client
	embedder            *embedding.OpenAIEmbedding
	config              *config.Config
	msgBuffer           *MessageBuffer
	processor           *MemoryProcessor
	llmClient           llm.Provider
	apiKeys             neo4j.APIKeyStore
	reranker            reranker.Provider
	compStats           *CompressionStats
	multiSignal         *retrieval.MultiSignalRetrieval
	multiSignalAdapter  *retrieval.ServiceAdapter
	compressionPipeline *pipeline.CompressionPipeline
	bm25Once            sync.Once
	ontologies          []*ontology.Ontology
	ontologyLoader      *ontology.Loader
	tierRouter          *tier.MemoryRouter
	auditLogger         audit.Logger
	webhookSvc          *webhook.Service
	privacyFilter       *privacy.Filter
	hooks               []*types.Hook

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

	// Research paper subsystems
	safetyClassifier  *safety.Classifier
	prospector        *searchpkg.Prospector
	causalFilter      *searchpkg.CausalFilter
	granularityRouter *searchpkg.GranularityRouter
	shiftDetector     *temporal.ShiftDetector

	// Self-improvement tuning loop
	selfImprover *self_improve.SelfImprover

	// Quota enforcement (Stripe)
	quotaChecker  func(tenantID, operation string) error
	usageRecorder func(tenantID, operation string)

	// Runtime state
	totalRetrievals int64 // for UCB — accessed atomically
	decayEnabled    bool
	temporalEnabled bool
	compressionMode string
	tierPolicy      TierPolicy

	// configMu protects concurrent reads/writes to config fields
	configMu sync.RWMutex

	// wg tracks in-flight background goroutines for graceful shutdown
	wg sync.WaitGroup

	// Tenant isolation: when non-empty, GetMemory and related methods enforce
	// that returned data belongs to this tenant. Empty means admin/internal mode.
	defaultTenantID string
}

type TierPolicy string

const (
	TierPolicyAggressive   TierPolicy = "aggressive"
	TierPolicyBalanced     TierPolicy = "balanced"
	TierPolicyConservative TierPolicy = "conservative"
)

func NewService(cfg *config.Config) (*Service, error) {
	neo, err := neo4j.NewClient(cfg.Neo4j)
	if err != nil {
		log.Printf("warning: neo4j unavailable: %v", err)
		neo = nil // ensure nil so downstream nil-checks work
	}
	qdr, err := qdrant.NewClient(cfg.Qdrant)
	if err != nil {
		log.Printf("warning: qdrant unavailable: %v", err)
		qdr = nil
	}

	var vec VectorStore
	switch cfg.VectorProvider {
	case "pinecone":
		pineClient, err := pinecone.NewClient(cfg.Pinecone)
		if err != nil {
			return nil, fmt.Errorf("pinecone init: %w", err)
		}
		vec = pineClient
	case "pgvector":
		pgClient, err := pgvector.NewClient(cfg.Pgvector)
		if err != nil {
			return nil, fmt.Errorf("pgvector init: %w", err)
		}
		vec = pgClient
	case "chroma":
		chromaClient, err := chroma.NewClient(cfg.Chroma)
		if err != nil {
			return nil, fmt.Errorf("chroma init: %w", err)
		}
		vec = chromaClient
	default:
		if qdr != nil {
			vec = qdr
		}
	}

	svc := &Service{
		neo4jClient: neo,
		config:      cfg,
	}
	if neo != nil {
		svc.graph = neo
		svc.apiKeys = neo
	}
	if qdr != nil {
		if vec == nil {
			svc.vector = qdr
		} else {
			svc.vector = vec
		}
	} else if vec != nil {
		svc.vector = vec
	}
	svc.msgBuffer = NewMessageBuffer(cfg.App.MessageBuffer, cfg.App.BufferTimeout, neo)
	if cfg.LLM.APIKey != "" {
		llmCfg := &llm.Config{
			Provider:     llm.ProviderType(cfg.LLM.Provider),
			APIKey:       cfg.LLM.APIKey,
			BaseURL:      cfg.LLM.BaseURL,
			Organization: cfg.LLM.OrgID,
		}
		switch llmCfg.Provider {
		case llm.ProviderOpenAI:
			llmCfg.OpenAI.Model = cfg.LLM.Model
			llmCfg.OpenAI.MaxTokens = cfg.LLM.MaxTokens
			llmCfg.OpenAI.Temperature = cfg.LLM.Temperature
		case llm.ProviderAnthropic:
			llmCfg.Anthropic.Model = cfg.LLM.Model
			llmCfg.Anthropic.MaxTokens = cfg.LLM.MaxTokens
			llmCfg.Anthropic.Temperature = cfg.LLM.Temperature
		case llm.ProviderGoogle:
			llmCfg.Google.Model = cfg.LLM.Model
			llmCfg.Google.MaxTokens = cfg.LLM.MaxTokens
			llmCfg.Google.Temperature = cfg.LLM.Temperature
		case llm.ProviderGroq:
			llmCfg.Groq.Model = cfg.LLM.Model
			llmCfg.Groq.MaxTokens = cfg.LLM.MaxTokens
			llmCfg.Groq.Temperature = cfg.LLM.Temperature
		case llm.ProviderDeepSeek:
			llmCfg.DeepSeek.Model = cfg.LLM.Model
			llmCfg.DeepSeek.MaxTokens = cfg.LLM.MaxTokens
			llmCfg.DeepSeek.Temperature = cfg.LLM.Temperature
		case llm.ProviderCohere:
			llmCfg.Cohere.Model = cfg.LLM.Model
			llmCfg.Cohere.MaxTokens = cfg.LLM.MaxTokens
			llmCfg.Cohere.Temperature = cfg.LLM.Temperature
		case llm.ProviderLocal:
			llmCfg.Local.Model = cfg.LLM.Model
			llmCfg.Local.MaxTokens = cfg.LLM.MaxTokens
			llmCfg.Local.Temperature = cfg.LLM.Temperature
		case llm.ProviderAWS:
			llmCfg.AWS.Model = cfg.LLM.Model
			llmCfg.AWS.MaxTokens = cfg.LLM.MaxTokens
			llmCfg.AWS.Temperature = cfg.LLM.Temperature
		}
		var llmErr error
		svc.llmClient, llmErr = llm.NewProvider(llmCfg)
		if llmErr != nil {
			log.Printf("warning: llm provider unavailable: %v", llmErr)
		}
	}
	if cfg.OpenAI.APIKey != "" {
		svc.embedder = embedding.NewOpenAI(cfg.OpenAI)
	} else if cfg.LLM.APIKey != "" && cfg.LLM.Provider == "openai" {
		svc.embedder = embedding.NewOpenAI(config.OpenAIConfig{
			APIKey:   cfg.LLM.APIKey,
			Model:    cfg.OpenAI.Model,
			EmbedDim: cfg.OpenAI.EmbedDim,
			BaseURL:  cfg.OpenAI.BaseURL,
		})
	}
	if svc.llmClient != nil && cfg.Memory.ProcessingEnabled {
		svc.processor = NewMemoryProcessorWithConfig(svc.llmClient, nil)
	}
	var rerankerErr error
	svc.reranker, rerankerErr = reranker.NewProvider(cfg.Reranker, svc.llmClient)
	if rerankerErr != nil {
		log.Printf("warning: reranker unavailable: %v", rerankerErr)
	}
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

	// Initialize self-improvement tuning loop (runs async on each feedback)
	if neo != nil {
		fc := self_improve.NewServiceFeedbackCollector(&graphFeedbackAdapter{client: neo})
		ts := self_improve.NewInMemoryTuningStore(&graphTuningAdapter{client: neo})
		svc.selfImprover = self_improve.NewSelfImprover(fc, ts, self_improve.DefaultConfig())
	}

	if svc.llmClient != nil {
		svc.distiller = searchpkg.NewDistiller(svc.llmClient)
	}

	// Initialize research paper subsystems
	svc.safetyClassifier = safety.NewClassifier()
	svc.prospector = searchpkg.NewProspector(svc.llmClient)
	svc.causalFilter = searchpkg.NewCausalFilter(svc.llmClient)
	svc.granularityRouter = searchpkg.NewGranularityRouter()
	svc.shiftDetector = temporal.NewShiftDetector()
	svc.initMultiSignal()

	return svc, nil
}

func (s *Service) SetWebhookService(wh *webhook.Service) { s.webhookSvc = wh }
func (s *Service) SetTierRouter(tr *tier.MemoryRouter) {
	s.tierRouter = tr
	s.syncTierPolicyToRouter()
}
func (s *Service) SetQuotaChecker(checker func(string, string) error, recorder func(string, string)) {
	s.quotaChecker = checker
	s.usageRecorder = recorder
}
func (s *Service) GetNeo4jClient() *neo4j.Client  { return s.neo4jClient }
func (s *Service) APIKeyStore() neo4j.APIKeyStore { return s.apiKeys }
func (s *Service) GetGraph() GraphStore           { return s.graph }
func (s *Service) GetVector() VectorStore         { return s.vector }
func (s *Service) Close() error {
	s.wg.Wait() // wait for in-flight background writes
	if s.neo4jClient != nil {
		return s.neo4jClient.Close()
	}
	return nil
}

func (s *Service) FlushCompression(ctx context.Context) error {
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *Service) GetReranker() reranker.Provider {
	return s.reranker
}

// PingNeo4j checks connectivity to the Neo4j graph store.
func (s *Service) PingNeo4j(ctx context.Context) error {
	if s.graph == nil {
		return fmt.Errorf("neo4j not configured")
	}
	return s.graph.Ping(ctx)
}

// PingQdrant checks connectivity to the Qdrant vector store.
func (s *Service) PingQdrant(ctx context.Context) error {
	if s.vector == nil {
		return fmt.Errorf("qdrant not configured")
	}
	return s.vector.Ping(ctx)
}

// PingRedis checks connectivity to Redis via the cache store if available.
// Returns fmt.Errorf("redis not configured") when no cache store is wired up,
// allowing callers to report the store status as "unknown" rather than "error".
func (s *Service) PingRedis(ctx context.Context) error {
	if s.tierRouter == nil {
		return fmt.Errorf("redis not configured")
	}
	// Probe via a lightweight Exists check; absence of a key is still a
	// successful round-trip. If the cache store is nil the router has no
	// Redis backend, so report as not configured.
	cs := s.tierRouter.CacheStore()
	if cs == nil {
		return fmt.Errorf("redis not configured")
	}
	_, err := cs.Exists(ctx, "__ping__")
	return err
}

func (s *Service) AddToContext(sessionID string, msg interface{}) error {
	if s.msgBuffer == nil {
		return fmt.Errorf("service: no message buffer configured")
	}
	var m types.Message
	switch v := msg.(type) {
	case types.Message:
		m = v
	case *types.Message:
		if v == nil {
			return nil
		}
		m = *v
	case string:
		m = types.Message{ID: generateUUID(), SessionID: sessionID, Content: v, Timestamp: time.Now()}
	default:
		m = types.Message{ID: generateUUID(), SessionID: sessionID, Timestamp: time.Now()}
	}
	if m.SessionID == "" {
		m.SessionID = sessionID
	}
	if m.ID == "" {
		m.ID = generateUUID()
	}
	return s.msgBuffer.Add(m)
}
func (s *Service) GetContext(sessionID string, limit int) ([]types.Message, error) {
	if s.graph == nil {
		return []types.Message{}, nil
	}
	msgs, err := s.graph.GetMessages(sessionID, limit)
	if err != nil {
		return nil, fmt.Errorf("service: get context: %w", err)
	}
	if msgs == nil {
		return []types.Message{}, nil
	}
	return msgs, nil
}
func (s *Service) ClearContext()                         {}
func (s *Service) GetMessages() []map[string]interface{} { return nil }

func (s *Service) CreateSession(agentID string, metadata map[string]interface{}) (*types.Session, error) {
	return &types.Session{ID: generateUUID(), AgentID: agentID}, nil
}

func (s *Service) RunCompaction(ctx context.Context, userID, mode string) (*types.CompactionResult, error) {
	if s.graph == nil {
		return &types.CompactionResult{}, nil
	}
	// Archive memories older than threshold by fetching user/all memories and archiving expired ones
	var memories []*types.Memory
	var err error
	if userID != "" {
		memories, err = s.graph.GetMemoriesByUser(userID)
	} else {
		memories, err = s.graph.GetAllMemories()
	}
	if err != nil {
		return nil, fmt.Errorf("service: run compaction: %w", err)
	}
	result := &types.CompactionResult{}
	for _, mem := range memories {
		if mem.ExpirationDate != nil && time.Now().After(*mem.ExpirationDate) {
			mem.Status = types.MemoryStatusArchived
			mem.UpdatedAt = time.Now()
			if updateErr := s.graph.UpdateMemory(mem); updateErr == nil {
				result.ArchivedCount++
				result.MemoryIDs = append(result.MemoryIDs, mem.ID)
			}
		}
	}
	return result, nil
}
func (s *Service) RunTargetedCompaction(ctx context.Context, ids []string, action string) (*types.CompactionResult, error) {
	if s.graph == nil || len(ids) == 0 {
		return &types.CompactionResult{}, nil
	}

	// Optimization: Resolve N+1 query bottleneck by fetching all requested memories in one batch.
	// Expected impact: Reduces database round-trips from O(N) to O(1).
	memories, err := s.getMemoriesByIDs(ids)
	if err != nil {
		// Log error but attempt to continue if any IDs were fetched (resilience)
		log.Printf("service: targeted compaction batch fetch error: %v", err)
	}

	result := &types.CompactionResult{}
	for _, mem := range memories {
		if mem == nil {
			continue
		}
		id := mem.ID
		switch action {
		case "archive":
			mem.Status = types.MemoryStatusArchived
			mem.UpdatedAt = time.Now()
			if err := s.graph.UpdateMemory(mem); err == nil {
				result.ArchivedCount++
				result.MemoryIDs = append(result.MemoryIDs, id)
			}
		case "delete":
			if err := s.graph.DeleteMemory(id); err == nil {
				result.DeletedCount++
				result.MemoryIDs = append(result.MemoryIDs, id)
			}
		default:
			mem.Status = types.MemoryStatusArchived
			mem.UpdatedAt = time.Now()
			if err := s.graph.UpdateMemory(mem); err == nil {
				result.ArchivedCount++
				result.MemoryIDs = append(result.MemoryIDs, id)
			}
		}
	}
	return result, nil
}
func (s *Service) CompactNegativeFeedback(ctx context.Context, userID string, limit int) (*types.CompactionResult, error) {
	if s.graph == nil {
		return &types.CompactionResult{}, nil
	}
	feedbacks, err := s.graph.GetFeedbackByType(types.FeedbackNegative, limit)
	if err != nil {
		return nil, fmt.Errorf("service: compact negative feedback: %w", err)
	}

	ids := make([]string, 0, len(feedbacks))
	seenIDs := make(map[string]bool)
	for _, fb := range feedbacks {
		if !seenIDs[fb.MemoryID] {
			ids = append(ids, fb.MemoryID)
			seenIDs[fb.MemoryID] = true
		}
	}

	// Optimization: Resolve N+1 query bottleneck by fetching all memories associated with negative feedback in one batch.
	// Expected impact: Reduces latency linearly with the number of unique memories flagged.
	memories, err := s.getMemoriesByIDs(ids)
	if err != nil {
		return nil, fmt.Errorf("service: compact negative feedback fetch: %w", err)
	}

	result := &types.CompactionResult{}
	for _, mem := range memories {
		if mem == nil {
			continue
		}
		if userID != "" && mem.UserID != userID {
			continue
		}
		mem.Status = types.MemoryStatusArchived
		mem.UpdatedAt = time.Now()
		if err := s.graph.UpdateMemory(mem); err == nil {
			result.ArchivedCount++
			result.MemoryIDs = append(result.MemoryIDs, mem.ID)
		}
	}
	return result, nil
}

func (s *Service) InferMemoryContent(ctx context.Context, content, userID string, memType types.MemoryType) (*types.MemoryProcessingResult, error) {
	return &types.MemoryProcessingResult{ProcessedContent: content, Importance: "medium", ShouldStore: true}, nil
}

func (s *Service) CreateMemoryInternal(ctx context.Context, mem *types.Memory) (string, error) {
	if mem.ID == "" {
		mem.ID = generateUUID()
	}
	if s.graph == nil {
		return "", fmt.Errorf("service: no graph store configured")
	}
	if err := s.graph.CreateMemory(mem); err != nil {
		return "", fmt.Errorf("service: create memory: %w", err)
	}
	return mem.ID, nil
}

func (s *Service) CreateMemory(ctx context.Context, mem *types.Memory) (*types.Memory, error) {
	if mem.ID == "" {
		mem.ID = generateUUID()
	}
	if mem.ContentHash == "" && mem.Content != "" {
		hash := sha256.Sum256([]byte(mem.Content))
		mem.ContentHash = hex.EncodeToString(hash[:])
	}
	// Propagate default tenant ID when the caller has not specified one.
	if mem.TenantID == "" && s.defaultTenantID != "" {
		mem.TenantID = s.defaultTenantID
	}
	mem.CreatedAt = time.Now()
	mem.UpdatedAt = time.Now()
	if mem.Version == 0 {
		mem.Version = 1
	}

	// Set initial status and validity
	if mem.Status == "" {
		mem.Status = types.MemoryStatusActive
	}
	if mem.ValidityStatus == "" {
		mem.ValidityStatus = types.ValidityCurrent
	}

	// TriMem — preserve raw segment before extraction
	mem.RawSegment = mem.Content

	// GAM dual graph — new memories start as events
	if mem.GraphLayer == "" {
		mem.GraphLayer = types.GraphLayerEvent
	}

	// Source authority (MEMTIER paper)
	if mem.SourceType == "" {
		mem.SourceType = types.SourceTold // default: user told the agent
	}
	if score, ok := types.SourceAuthorityScores[mem.SourceType]; ok {
		mem.SourceAuthority = score
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

	// Dimension extraction (DimMem paper — 24% token reduction)
	if mem.Dimensions == nil {
		words := strings.Fields(strings.ToLower(mem.Content))
		keywords := make([]string, 0)
		for _, w := range words {
			if len(w) > 5 { // simple heuristic: longer words are more likely keywords
				keywords = append(keywords, w)
			}
		}
		if len(keywords) > 10 {
			keywords = keywords[:10]
		}
		mem.Dimensions = &types.MemoryDimensions{
			Keywords: keywords,
		}
	}

	// Smart dedup: check for semantically similar existing memories (ADD/UPDATE/NOOP)
	if s.embedder != nil && s.vector != nil && mem.Content != "" {
		dedupEmb, dedupErr := s.embedder.GenerateEmbeddingWithContext(ctx, mem.Content)
		if dedupErr == nil && len(dedupEmb) > 0 {
			dedupFilter := map[string]interface{}{}
			if mem.UserID != "" {
				dedupFilter["user_id"] = mem.UserID
			}
			if mem.OrgID != "" {
				dedupFilter["org_id"] = mem.OrgID
			}
			if mem.TenantID != "" {
				dedupFilter["tenant_id"] = mem.TenantID
			}

			similar, searchErr := s.vector.Search(ctx, dedupEmb, 3, 0.85, dedupFilter)
			if searchErr == nil && len(similar) > 0 {
				topMatch := similar[0]
				existingMem, getErr := s.graph.GetMemory(topMatch.MemoryID)
				if getErr == nil && existingMem != nil && s.processor != nil {
					resolution, resolveErr := s.processor.ResolveConflict(
						ctx,
						existingMem.Content,
						string(existingMem.Importance),
						mem.Content,
					)
					if resolveErr == nil && resolution != nil {
						switch resolution.Action {
						case ConflictActionDiscardNew:
							return existingMem, nil
						case ConflictActionUpdate:
							existingMem.Content = resolution.UpdatedContent
							existingMem.Version++
							existingMem.UpdatedAt = time.Now()
							if updateErr := s.graph.UpdateMemory(existingMem); updateErr == nil {
								if newEmb, embErr := s.embedder.GenerateEmbeddingWithContext(ctx, existingMem.Content); embErr == nil {
									if _, storeErr := s.vector.StoreEmbedding(ctx, existingMem.Content, existingMem.ID, newEmb, dedupFilter); storeErr != nil {
										log.Printf("service: dedup re-embed %s: %v", existingMem.ID, storeErr)
									}
								}
								if s.webhookSvc != nil {
									s.wg.Add(1)
									go func() {
										defer s.wg.Done()
										s.webhookSvc.EmitEvent(context.Background(), types.WebhookEventMemoryUpdated, "", existingMem)
									}()
								}
								return existingMem, nil
							}
						case ConflictActionKeepBoth:
							mem.RelatedMemoryIDs = append(mem.RelatedMemoryIDs, existingMem.ID)
						}
					}
				}
			}
		}
	}

	// Safety check — reject malicious/harmful content (FSFM paper)
	if s.safetyClassifier != nil {
		result := s.safetyClassifier.Classify(mem.Content)
		if !result.Safe && (result.Category == "malicious" || result.Category == "sensitive") {
			return nil, fmt.Errorf("service: memory rejected by safety classifier: %s (%s)", result.Category, result.Reason)
		}
		if !result.Safe {
			if mem.Metadata == nil {
				mem.Metadata = make(map[string]interface{})
			}
			mem.Metadata["safety_category"] = result.Category
			mem.Metadata["safety_reason"] = result.Reason
		}
	}

	// Quota check before graph write
	if s.quotaChecker != nil && mem.TenantID != "" {
		if err := s.quotaChecker(mem.TenantID, "memory_create"); err != nil {
			return nil, fmt.Errorf("service: quota exceeded: %w", err)
		}
	}

	// Route to storage tier before persisting
	s.routeMemoryTier(ctx, mem)

	// Write to graph store
	if err := s.graph.CreateMemory(mem); err != nil {
		return nil, fmt.Errorf("service: create memory graph: %w", err)
	}
	if s.materializeMemoryEntities(ctx, mem) {
		if err := s.graph.UpdateMemory(mem); err != nil {
			log.Printf("service: update memory entity metadata %s: %v", mem.ID, err)
		}
	}

	// Write embedding to vector store if embedder available
	if s.embedder != nil {
		emb, err := s.embedder.GenerateEmbeddingWithContext(ctx, mem.Content)
		if err == nil && len(emb) > 0 {
			// Track embeddings for semantic shift detection (GAM paper)
			if s.shiftDetector != nil {
				if s.shiftDetector.DetectShift(emb) {
					mem.GraphLayer = types.GraphLayerTopic // promote to topic on shift
				}
				s.shiftDetector.AddEmbedding(emb)
			}

			// Apply phase rotation if temporal features enabled
			if s.phaseRotator != nil && s.IsTemporalReasoningEnabled() {
				angle := s.phaseRotator.ComputePhaseAngle(mem.VolatilityScore, 0) // age=0 for new
				mem.PhaseAngle = angle
			}
			if _, err := s.vector.StoreEmbedding(ctx, mem.Content, mem.ID, emb, s.buildMemoryMetadata(mem)); err != nil {
				log.Printf("service: failed to store embedding: %v", err)
				// Don't fail the whole create — graph write already succeeded
			}
		}
	}

	// Webhook notification (tracked for graceful shutdown)
	if s.webhookSvc != nil {
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			s.webhookSvc.EmitEvent(context.Background(), types.WebhookEventMemoryCreated, "", mem)
		}()
	}

	// Record usage after successful create
	if s.usageRecorder != nil && mem.TenantID != "" {
		s.usageRecorder(mem.TenantID, "memory_create")
	}

	// Index for BM25 keyword search
	s.indexMemoryForSearch(mem)

	// Async compression (non-blocking write path)
	s.scheduleCompression(mem.ID, mem.Content)

	// Auto-chunk large memories when chunking is enabled.
	// Each chunk is stored as a child memory node linked to the parent.
	if s.config != nil && s.config.Memory.ChunkingEnabled && s.graph != nil {
		if len(mem.Content) > s.config.Memory.ChunkingMaxBytes {
			s.wg.Add(1)
			parentID := mem.ID
			content := mem.Content
			go func() {
				defer s.wg.Done()
				if err := s.chunkAndStoreChildren(context.Background(), parentID, content, mem); err != nil {
					log.Printf("service: chunking memory %s: %v", parentID, err)
				}
			}()
		}
	}

	// Business metric: count successfully created memories by type and tenant.
	metrics.MemoryCreatedTotal.WithLabelValues(string(mem.Type), mem.TenantID).Inc()

	return mem, nil
}

func (s *Service) GetMemory(ctx context.Context, id string) (*types.Memory, error) {
	var mem *types.Memory
	var err error

	// Use tenant-isolated retrieval when a defaultTenantID is configured and the
	// direct Neo4j client is available. Falls back to the graph interface (no
	// tenant filter) for internal/admin callers that have not set a tenant ID.
	if s.defaultTenantID != "" && s.neo4jClient != nil {
		mem, err = s.neo4jClient.GetMemoryForTenant(id, s.defaultTenantID)
	} else {
		mem, err = s.graph.GetMemory(id)
	}
	if err != nil {
		return nil, fmt.Errorf("service: get memory: %w", err)
	}
	if mem != nil {
		s.hydrateMemoryFromTierCache(ctx, mem)
		mem.AccessCount++
		now := time.Now()
		mem.LastRetrievedAt = &now
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			if err := s.graph.UpdateMemory(mem); err != nil {
				log.Printf("service: background update failed: %v", err)
			}
		}()
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

func (s *Service) DeleteMemory(ctx context.Context, id string) error {
	if err := s.graph.DeleteMemory(id); err != nil {
		return err
	}
	if s.vector != nil {
		if err := s.vector.DeleteMemory(ctx, id); err != nil {
			log.Printf("service: delete vector memory %s: %v", id, err)
		}
	}
	return nil
}
func (s *Service) DeleteMemories(ctx context.Context, ids []string) error {
	if err := s.graph.BatchDeleteMemories(ids); err != nil {
		return err
	}
	if s.vector != nil {
		for _, id := range ids {
			if err := s.vector.DeleteMemory(ctx, id); err != nil {
				log.Printf("service: delete vector memory %s: %v", id, err)
			}
		}
	}
	return nil
}
func (s *Service) ArchiveMemory(ctx context.Context, id string) error {
	mem, err := s.graph.GetMemory(id)
	if err != nil {
		return fmt.Errorf("service: archive memory: %w", err)
	}
	if mem == nil {
		return fmt.Errorf("service: memory not found: %s", id)
	}
	mem.Status = types.MemoryStatusArchived
	mem.UpdatedAt = time.Now()
	return s.graph.UpdateMemory(mem)
}

func (s *Service) SearchMemories(ctx context.Context, req *types.SearchRequest) ([]types.MemoryResult, error) {
	if req == nil || req.Query == "" {
		return nil, fmt.Errorf("service: search requires a query")
	}

	// Business metrics: count and time every search.
	searchStart := time.Now()
	metrics.SearchTotal.Inc()

	// Quota check at search entry (use OrgID as tenant scope)
	tenantID := req.OrgID
	if s.quotaChecker != nil && tenantID != "" {
		if err := s.quotaChecker(tenantID, "search"); err != nil {
			return nil, fmt.Errorf("service: quota exceeded: %w", err)
		}
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 10
	}

	var results []types.MemoryResult

	// Multi-signal retrieval (BM25 + semantic + entity fusion)
	if req.Mode == "" && s.multiSignal != nil && s.config != nil && s.config.Memory.MultiSignalEnabled {
		msResults, err := s.searchWithMultiSignal(ctx, req.Query, limit)
		if err == nil && len(msResults) > 0 {
			results = msResults
		}
	}

	// Step 1: Prospection-guided retrieval (PGR paper — 3x recall)
	queries := []string{req.Query}
	if s.prospector != nil {
		expanded := s.prospector.HeuristicExpand(req.Query)
		queries = append(queries, expanded...)
	}

	// Search with all queries (original + prospective), union results
	if len(results) == 0 {
		var allResults []types.MemoryResult
		seen := make(map[string]bool)
		var mu sync.Mutex

		if s.embedder != nil {
			// Optimization: Batch generate embeddings for all prospective queries.
			// Expected impact: Reduces O(N) embedding round-trips to O(1).
			embs, err := s.embedder.GenerateBatchEmbeddingsWithContext(ctx, queries)
			if err != nil {
				log.Printf("service: batch embedding failed: %v, falling back to sequential", err)
				// Fallback to sequential if batch fails
				embs = make([][]float32, 0, len(queries))
				for _, q := range queries {
					emb, err := s.embedder.GenerateEmbeddingWithContext(ctx, q)
					if err != nil || len(emb) == 0 {
						embs = append(embs, nil)
						continue
					}
					embs = append(embs, emb)
				}
			}

			// Optimization: Execute vector searches in parallel.
			// Expected impact: Latency is dominated by the slowest search instead of their sum.
			var wg sync.WaitGroup
			filterMap := s.filtersToMap(req.Filters)
			if filterMap == nil {
				filterMap = make(map[string]interface{})
			}
			if req.OrgID != "" {
				filterMap["org_id"] = req.OrgID
			}
			if req.UserID != "" {
				filterMap["user_id"] = req.UserID
			}

			for i, emb := range embs {
				if len(emb) == 0 {
					continue
				}
				wg.Add(1)
				go func(idx int, e []float32) {
					defer wg.Done()
					res, err := s.vector.Search(ctx, e, limit*2, 0.0, filterMap)
					if err != nil {
						log.Printf("service: parallel vector search failed for query %d: %v", idx, err)
						return
					}

					mu.Lock()
					defer mu.Unlock()
					for _, r := range res {
						if !seen[r.MemoryID] {
							seen[r.MemoryID] = true
							allResults = append(allResults, r)
						}
					}
				}(i, emb)
			}
			wg.Wait()
		}
		results = allResults
	}

	// Pre-pass: populate Metadata on each result in batch so temporal/decay scorers
	// have access to CreatedAt, UpdatedAt, AccessCount, etc. without N+1 queries.
	// Both ApplyTemporalScoring and ApplyDecay require Metadata to be populated.
	if s.graph != nil && len(results) > 0 {
		var ids []string
		idToIndices := make(map[string][]int)
		for i := range results {
			if results[i].MemoryID != "" && results[i].Metadata == nil {
				if _, seen := idToIndices[results[i].MemoryID]; !seen {
					ids = append(ids, results[i].MemoryID)
				}
				idToIndices[results[i].MemoryID] = append(idToIndices[results[i].MemoryID], i)
			}
		}

		if len(ids) > 0 {
			var memories []*types.Memory
			var err error
			// Use tenant-isolated batch retrieval if defaultTenantID is configured
			if s.defaultTenantID != "" && s.neo4jClient != nil {
				memories, err = s.neo4jClient.GetMemoriesByIDsForTenant(ids, s.defaultTenantID)
			} else {
				memories, err = s.graph.GetMemoriesByIDs(ids)
			}

			if err == nil {
				for _, mem := range memories {
					if mem != nil {
						if indices, ok := idToIndices[mem.ID]; ok {
							for _, idx := range indices {
								results[idx].Metadata = mem
							}
						}
					}
				}
			}
		}
	}

	// Step 2: Apply temporal scoring
	if s.IsTemporalReasoningEnabled() && s.temporalScorer != nil && len(results) > 0 {
		results = temporal.ApplyTemporalScoring(results, s.temporalScorer, req.TimeReference)
	}

	// Step 3: Apply decay scoring
	if s.IsDecayEnabled() && s.decayScorer != nil && len(results) > 0 {
		results = decay.ApplyDecay(results, s.decayScorer)
	}

	// Step 4: Apply phase rotation adjustment and composite/UCB scoring
	// Reuse Metadata already fetched in the pre-pass above; fall back to a
	// fresh graph fetch only when the pre-pass didn't populate it.
	for i := range results {
		mem := results[i].Metadata
		if mem == nil {
			var err error
			mem, err = s.graph.GetMemory(results[i].MemoryID)
			if err != nil || mem == nil {
				continue
			}
		}

		// Phase rotation
		if s.phaseRotator != nil && s.IsTemporalReasoningEnabled() {
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
				AuthorityScore:  mem.SourceAuthority, // source authority signal (MEMTIER paper)
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

	// Sort by score descending and apply result limit.
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})
	if len(results) > limit {
		results = results[:limit]
	}

	// Token-budget enforcement: cap total tokens returned per search.
	// Estimate ~4 characters per token (rough but consistent heuristic).
	maxTokens := 7000
	if req.MaxTokens > 0 {
		maxTokens = req.MaxTokens
	}
	tokenCount := 0
	budgetedResults := make([]types.MemoryResult, 0, len(results))
	for _, r := range results {
		text := r.Text
		if text == "" && r.Metadata != nil {
			text = r.Metadata.Content
		}
		tokens := len(text) / 4
		if tokenCount+tokens > maxTokens {
			break
		}
		budgetedResults = append(budgetedResults, r)
		tokenCount += tokens
	}
	results = budgetedResults

	// Business metrics: record search latency and tokens returned.
	metrics.SearchLatency.Observe(time.Since(searchStart).Seconds())
	metrics.SearchTokensReturned.Observe(float64(tokenCount))

	// Increment retrieval counter
	atomic.AddInt64(&s.totalRetrievals, 1)

	// Record usage after successful search
	if s.usageRecorder != nil && tenantID != "" {
		s.usageRecorder(tenantID, "search")
	}

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
	if s.graph == nil {
		return nil, fmt.Errorf("service: no graph store configured")
	}
	if len(memories) == 0 {
		return nil, nil
	}
	var ids []string
	var errs []string
	for _, mem := range memories {
		created, err := s.CreateMemory(ctx, mem)
		if err != nil {
			errs = append(errs, fmt.Sprintf("memory %s: %v", mem.ID, err))
			continue // don't stop on first failure
		}
		ids = append(ids, created.ID)
	}
	if len(errs) > 0 {
		return ids, fmt.Errorf("service: %d/%d memories failed: %s", len(errs), len(memories), strings.Join(errs, "; "))
	}
	return ids, nil
}

func (s *Service) materializeMemoryEntities(ctx context.Context, mem *types.Memory) bool {
	if s.graph == nil || mem == nil || len(mem.Metadata) == 0 {
		return false
	}
	extracted, ok := mem.Metadata["entities"].([]ExtractedEntity)
	if !ok || len(extracted) == 0 {
		return false
	}

	entityIDs := make([]string, 0, len(extracted))
	seen := make(map[string]bool, len(extracted))
	changed := false
	for _, extractedEntity := range extracted {
		name := strings.TrimSpace(extractedEntity.Name)
		if name == "" {
			continue
		}
		entityType := strings.TrimSpace(extractedEntity.Type)
		if entityType == "" {
			entityType = "thing"
		}
		entityID := stableEntityID(mem.TenantID, entityType, name)
		if seen[entityID] {
			continue
		}
		seen[entityID] = true

		entity := types.Entity{
			ID:       entityID,
			TenantID: mem.TenantID,
			Type:     entityType,
			Name:     name,
			Properties: map[string]interface{}{
				"source":   "memory_extraction",
				"mentions": extractedEntity.Mentions,
			},
		}

		// Entity dedup: check if a near-identical entity already exists (cosine >= 0.95)
		if s.vector != nil && s.embedder != nil {
			if emb, err := s.embedder.GenerateEmbeddingWithContext(context.Background(), name); err == nil {
				const entityDedupThreshold float32 = 0.95
				existing, err := s.vector.Search(context.Background(), emb, 1, entityDedupThreshold, map[string]interface{}{
					"entity_type": entityType,
					"tenant_id":   mem.TenantID,
				})
				if err == nil && len(existing) > 0 && existing[0].Score >= entityDedupThreshold {
					entityID = existing[0].MemoryID
					entity.ID = entityID
				}
			}
		}

		if err := s.graph.AddEntity(entity); err != nil {
			log.Printf("service: add extracted entity %s: %v", name, err)
			continue
		}
		if err := s.graph.LinkMemoryEntity(mem.ID, entityID); err != nil {
			log.Printf("service: link memory %s to entity %s: %v", mem.ID, entityID, err)
			continue
		}
		if mem.EntityID == "" {
			mem.EntityID = entityID
			changed = true
		}
		entityIDs = append(entityIDs, entityID)
	}
	if len(entityIDs) > 0 {
		mem.Metadata["entity_ids"] = entityIDs
		mem.Metadata["primary_entity_id"] = mem.EntityID
		changed = true
	}

	for i := 0; i < len(entityIDs); i++ {
		for j := i + 1; j < len(entityIDs); j++ {
			props := map[string]interface{}{
				"source":    "memory_co_mention",
				"memory_id": mem.ID,
				"weight":    1.0,
			}
			if err := s.graph.AddRelation(entityIDs[i], entityIDs[j], "RELATED_TO", props); err != nil {
				log.Printf("service: relate entities %s -> %s: %v", entityIDs[i], entityIDs[j], err)
			}
			if err := s.graph.AddRelation(entityIDs[j], entityIDs[i], "RELATED_TO", props); err != nil {
				log.Printf("service: relate entities %s -> %s: %v", entityIDs[j], entityIDs[i], err)
			}
		}
	}
	return changed
}

func (s *Service) hydrateMemoryFromTierCache(ctx context.Context, mem *types.Memory) {
	if s.tierRouter == nil || mem == nil || mem.ID == "" {
		return
	}
	tierName := strings.ToLower(mem.Tier)
	if tierName != "hot" && tierName != "working" {
		return
	}
	cache := s.tierRouter.CacheStore()
	if cache == nil {
		return
	}
	content, err := cache.Get(ctx, "hot:"+mem.ID)
	if err != nil || content == "" {
		return
	}
	mem.Content = content
}

func stableEntityID(tenantID, entityType, name string) string {
	key := strings.ToLower(strings.TrimSpace(tenantID)) + ":" +
		strings.ToLower(strings.TrimSpace(entityType)) + ":" +
		strings.ToLower(strings.TrimSpace(name))
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(key)).String()
}

func (s *Service) HybridSearch(ctx context.Context, req *types.HybridSearchRequest) ([]types.MemoryResult, error) {
	if req == nil || req.Query == "" {
		return nil, nil
	}

	limit := req.SemanticLimit
	if limit <= 0 {
		limit = 10
	}
	if s.multiSignal != nil && s.config != nil && s.config.Memory.MultiSignalEnabled {
		if results, err := s.searchWithMultiSignal(ctx, req.Query, limit); err == nil && len(results) > 0 {
			return results, nil
		}
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
	if s.graph == nil {
		return &types.MemoryStats{ByCategory: map[string]int64{}, ByType: map[string]int64{}, ByImportance: map[string]int64{}, ByStatus: map[string]int64{}}, nil
	}
	var memories []*types.Memory
	var err error
	if userID != "" {
		memories, err = s.graph.GetMemoriesByUser(userID)
	} else if orgID != "" {
		memories, err = s.graph.GetMemoriesByOrg(orgID)
	} else {
		memories, err = s.graph.GetAllMemories()
	}
	if err != nil {
		return nil, fmt.Errorf("service: get memory stats: %w", err)
	}
	stats := &types.MemoryStats{
		ByCategory:   make(map[string]int64),
		ByType:       make(map[string]int64),
		ByImportance: make(map[string]int64),
		ByStatus:     make(map[string]int64),
	}
	tagCounts := make(map[string]int64)
	var totalAccess int64
	now := time.Now()
	sevenDaysAgo := now.AddDate(0, 0, -7)
	for _, m := range memories {
		stats.TotalMemories++
		stats.ByType[string(m.Type)]++
		stats.ByImportance[string(m.Importance)]++
		stats.ByStatus[string(m.Status)]++
		totalAccess += int64(m.AccessCount)
		if m.CreatedAt.After(sevenDaysAgo) {
			stats.RecentMemories++
		}
		if m.ExpirationDate != nil && now.After(*m.ExpirationDate) {
			stats.ExpiredMemories++
		}
		for _, tag := range m.Tags {
			tagCounts[tag]++
		}
	}
	if stats.TotalMemories > 0 {
		stats.AvgAccessCount = float64(totalAccess) / float64(stats.TotalMemories)
	}
	for tag, count := range tagCounts {
		stats.TopTags = append(stats.TopTags, types.TagCount{Tag: tag, Count: count})
	}
	return stats, nil
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

	// Collect unique memory IDs from path nodes, preserving discovery order.
	var ids []string
	seen := make(map[string]bool)
	for _, path := range paths {
		for _, node := range path.Nodes {
			if !seen[node.ID] {
				seen[node.ID] = true
				ids = append(ids, node.ID)
			}
		}
	}

	// Optimization: Resolve N+1 query bottleneck by fetching all traversed memories in one batch.
	// Expected impact: Significant speedup for deep traversals (O(N) -> O(1) database trips).
	mems, err := s.getMemoriesByIDs(ids)
	if err != nil {
		return nil, err
	}

	// Map fetched memories by ID to restore original discovery order.
	memMap := make(map[string]*types.Memory)
	for _, m := range mems {
		if m != nil {
			memMap[m.ID] = m
		}
	}

	var memories []types.Memory
	for _, id := range ids {
		if m, ok := memMap[id]; ok {
			memories = append(memories, *m)
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
	return s.DeleteMemory(ctx, id)
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
	if s.graph == nil {
		return nil, nil
	}
	return s.graph.GetEntitiesByMemory(mid)
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
	if s.graph == nil {
		return nil, 0, fmt.Errorf("service: no graph store configured")
	}
	if req == nil {
		return nil, 0, fmt.Errorf("service: request required")
	}
	memories, total, err := s.graph.GetMemoriesPaginated(req)
	if err != nil {
		return nil, 0, fmt.Errorf("service: get memories paginated: %w", err)
	}
	var flat []types.Memory
	for _, m := range memories {
		if m != nil {
			flat = append(flat, *m)
		}
	}
	return flat, total, nil
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
		if err := s.graph.CreateFeedback(fb); err != nil {
			log.Printf("service: failed to persist feedback: %v", err)
		}
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
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			if err := s.graph.UpdateMemory(mem); err != nil {
				log.Printf("service: background update failed: %v", err)
			}
		}()

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
					s.wg.Add(1)
					anc := ancestor // capture for goroutine
					go func() {
						defer s.wg.Done()
						if err := s.graph.UpdateMemory(anc); err != nil {
							log.Printf("service: background ancestor update failed: %v", err)
						}
					}()
				}
			}
		}
	}

	// Self-improvement tuning cycle — run asynchronously to avoid blocking the caller.
	// Only triggers if we have accumulated enough feedback (controlled by the engine config).
	if s.selfImprover != nil {
		memoryID := fb.MemoryID
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			if _, err := s.selfImprover.ProcessTuningCycle(context.Background(), memoryID); err != nil {
				log.Printf("service: self-improve tuning cycle for %s: %v", memoryID, err)
			}
		}()
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

	ids := make([]string, 0, len(feedbacks))
	seenIDs := make(map[string]bool)
	for _, f := range feedbacks {
		if !seenIDs[f.MemoryID] {
			ids = append(ids, f.MemoryID)
			seenIDs[f.MemoryID] = true
		}
	}

	// Optimization: Resolve N+1 query bottleneck by fetching all feedback-linked memories in one batch.
	// Expected impact: Reduces database load and latency proportional to the feedback count.
	return s.getMemoriesByIDs(ids)
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
		return 0, fmt.Errorf("service: nil import")
	}
	var count int
	var errs []string
	for i := range imp.Memories {
		mem := &imp.Memories[i]
		if imp.Overwrite && mem.ID != "" {
			existing, _ := s.graph.GetMemory(mem.ID)
			if existing != nil {
				_ = s.graph.DeleteMemory(mem.ID)
			}
		}
		if _, err := s.CreateMemory(ctx, mem); err != nil {
			errs = append(errs, err.Error())
			continue
		}
		count++
	}
	if len(errs) > 0 {
		return count, fmt.Errorf("service: imported %d, %d failed", count, len(errs))
	}
	return count, nil
}

func (s *Service) QueryGraph(query string, params map[string]interface{}) ([]map[string]interface{}, error) {
	if s.graph == nil {
		return nil, nil
	}
	return s.graph.QueryGraph(query, params)
}

func (s *Service) CreateSkill(ctx context.Context, sk *types.Skill) error {
	if s.graph == nil {
		return fmt.Errorf("service: no graph store configured")
	}
	if sk.GroupID != "" {
		group, err := s.graph.GetAgentGroup(ctx, sk.GroupID)
		if err != nil {
			return fmt.Errorf("service: create skill: get group policy: %w", err)
		}
		if group != nil && !group.Policy.SkillSharingEnabled {
			return fmt.Errorf("service: create skill: skill sharing disabled for group %s", sk.GroupID)
		}
	}
	return s.graph.CreateSkill(ctx, sk)
}
func (s *Service) ListSkills(ctx context.Context, dom, group string, lim, off int) ([]*types.Skill, error) {
	if s.graph == nil {
		return []*types.Skill{}, nil
	}
	skills, err := s.graph.ListSkills(ctx, dom, group, lim, off)
	if err != nil {
		return nil, fmt.Errorf("service: list skills: %w", err)
	}
	if skills == nil {
		return []*types.Skill{}, nil
	}
	return skills, nil
}
func (s *Service) SearchSkillsByTrigger(ctx context.Context, tr string, lim int) ([]*types.Skill, error) {
	if s.graph == nil {
		return []*types.Skill{}, nil
	}
	skills, err := s.graph.GetSkillsByTrigger(ctx, tr, lim)
	if err != nil {
		return nil, fmt.Errorf("service: search skills by trigger: %w", err)
	}
	if skills == nil {
		return []*types.Skill{}, nil
	}
	return skills, nil
}
func (s *Service) GetSkillsByDomain(ctx context.Context, dom string, lim int) ([]*types.Skill, error) {
	if s.graph == nil {
		return []*types.Skill{}, nil
	}
	skills, err := s.graph.GetSkillsByDomain(ctx, dom, lim)
	if err != nil {
		return nil, fmt.Errorf("service: get skills by domain: %w", err)
	}
	if skills == nil {
		return []*types.Skill{}, nil
	}
	return skills, nil
}
func (s *Service) GetSkill(ctx context.Context, id string) (*types.Skill, error) {
	if s.graph == nil {
		return nil, fmt.Errorf("service: no graph store configured")
	}
	sk, err := s.graph.GetSkill(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("service: get skill: %w", err)
	}
	return sk, nil
}
func (s *Service) UpdateSkill(ctx context.Context, sk *types.Skill) error {
	if s.graph == nil {
		return fmt.Errorf("service: no graph store configured")
	}
	return s.graph.UpdateSkill(ctx, sk)
}
func (s *Service) DeleteSkill(ctx context.Context, id string) error {
	if s.graph == nil {
		return fmt.Errorf("service: no graph store configured")
	}
	return s.graph.DeleteSkill(ctx, id)
}
func (s *Service) ExecuteSkill(ctx context.Context, id string, p map[string]interface{}) (string, error) {
	if s.graph == nil {
		return "", fmt.Errorf("service: no graph store configured")
	}
	sk, err := s.graph.GetSkill(ctx, id)
	if err != nil {
		return "", fmt.Errorf("service: execute skill: get: %w", err)
	}
	if sk == nil {
		return "", fmt.Errorf("service: skill not found: %s", id)
	}
	_ = s.graph.IncrementSkillUsage(ctx, id)
	return s.executeSkillWithLLM(ctx, sk, p)
}

func (s *Service) executeSkillWithLLM(ctx context.Context, sk *types.Skill, params map[string]interface{}) (string, error) {
	if s.llmClient == nil {
		return "", fmt.Errorf("service: execute skill: no LLM provider configured")
	}
	if params == nil {
		params = map[string]interface{}{}
	}
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return "", fmt.Errorf("service: execute skill: marshal params: %w", err)
	}
	examplesJSON, _ := json.Marshal(sk.Examples)
	metadataJSON, _ := json.Marshal(sk.Metadata)

	prompt := fmt.Sprintf(`Execute this procedural memory skill.

Skill:
- Name: %s
- Domain: %s
- Trigger: %s
- Action: %s
- Examples: %s
- Metadata: %s

Runtime context:
%s

Return only the execution result. Do not restate the skill definition.`, sk.Name, sk.Domain, sk.Trigger, sk.Action, string(examplesJSON), string(metadataJSON), string(paramsJSON))

	resp, err := s.llmClient.Complete(ctx, &llm.CompletionRequest{
		Model: "defaultModel",
		Messages: []llm.Message{
			{Role: "system", Content: "You execute procedural memory skills for an AI agent. Be precise, actionable, and grounded in the supplied runtime context."},
			{Role: "user", Content: prompt},
		},
		Temperature: 0.2,
		MaxTokens:   1200,
	})
	if err != nil {
		return "", fmt.Errorf("service: execute skill: llm: %w", err)
	}
	return strings.TrimSpace(resp.Content), nil
}
func (s *Service) SuggestSkills(ctx context.Context, tr, c string, lim int) ([]*types.Skill, error) {
	if s.graph == nil {
		return []*types.Skill{}, nil
	}
	skills, err := s.graph.GetSkillsByTrigger(ctx, tr, lim)
	if err != nil {
		return nil, fmt.Errorf("service: suggest skills: %w", err)
	}
	if skills == nil {
		return []*types.Skill{}, nil
	}
	return skills, nil
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
	// No processor or dedicated method available — return empty result
	return &types.SkillExtractionResult{}, nil
}
func (s *Service) UseSkill(ctx context.Context, id string) error {
	if s.graph == nil {
		return fmt.Errorf("service: no graph store configured")
	}
	return s.graph.IncrementSkillUsage(ctx, id)
}
func (s *Service) IncrementSkillUsage(ctx context.Context, id string) error {
	if s.graph == nil {
		return fmt.Errorf("service: no graph store configured")
	}
	return s.graph.IncrementSkillUsage(ctx, id)
}
func (s *Service) GetSimilarSkills(ctx context.Context, id string, lim int) ([]*types.Skill, error) {
	if s.graph == nil {
		return []*types.Skill{}, nil
	}
	skills, err := s.graph.GetSimilarSkills(ctx, id, lim)
	if err != nil {
		return nil, fmt.Errorf("service: get similar skills: %w", err)
	}
	if skills == nil {
		return []*types.Skill{}, nil
	}
	return skills, nil
}
func (s *Service) CreateSkillReview(ctx context.Context, r *types.Review) (*types.Review, error) {
	if s.graph == nil {
		return nil, fmt.Errorf("service: no graph store configured")
	}
	if r.ID == "" {
		r.ID = generateUUID()
	}
	sr := &types.SkillReview{
		ID:        r.ID,
		SkillID:   r.SkillID,
		Status:    types.ReviewStatus(r.Status),
		Notes:     r.Notes,
		CreatedAt: r.CreatedAt,
	}
	if err := s.graph.CreateSkillReview(ctx, sr); err != nil {
		return nil, fmt.Errorf("service: create skill review: %w", err)
	}
	return r, nil
}
func (s *Service) ListSkillReviews(ctx context.Context, st string) ([]*types.Review, error) {
	if s.graph == nil {
		return []*types.Review{}, nil
	}
	srs, err := s.graph.ListPendingReviews(ctx, st)
	if err != nil {
		return nil, fmt.Errorf("service: list skill reviews: %w", err)
	}
	var reviews []*types.Review
	for _, sr := range srs {
		reviews = append(reviews, &types.Review{
			ID:        sr.ID,
			SkillID:   sr.SkillID,
			Status:    string(sr.Status),
			Notes:     sr.Notes,
			CreatedAt: sr.CreatedAt,
		})
	}
	if reviews == nil {
		return []*types.Review{}, nil
	}
	return reviews, nil
}
func (s *Service) GetSkillReview(ctx context.Context, id string) (*types.Review, error) {
	if s.graph == nil {
		return nil, fmt.Errorf("service: no graph store configured")
	}
	sr, err := s.graph.GetReview(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("service: get skill review: %w", err)
	}
	if sr == nil {
		return nil, nil
	}
	return &types.Review{
		ID:        sr.ID,
		SkillID:   sr.SkillID,
		Status:    string(sr.Status),
		Notes:     sr.Notes,
		CreatedAt: sr.CreatedAt,
	}, nil
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
func (s *Service) CreateChain(ctx context.Context, ch *types.SkillChain) error {
	if s.graph == nil {
		return fmt.Errorf("service: no graph store configured")
	}
	return s.graph.CreateChain(ctx, ch)
}
func (s *Service) ListChains(ctx context.Context, oid string, q *types.ChainQuery) ([]*types.SkillChain, error) {
	if s.graph == nil {
		return []*types.SkillChain{}, nil
	}
	chains, err := s.graph.ListChains(ctx, oid, q)
	if err != nil {
		return nil, fmt.Errorf("service: list chains: %w", err)
	}
	if chains == nil {
		return []*types.SkillChain{}, nil
	}
	return chains, nil
}
func (s *Service) GetChain(ctx context.Context, id string) (*types.SkillChain, error) {
	if s.graph == nil {
		return nil, fmt.Errorf("service: no graph store configured")
	}
	ch, err := s.graph.GetChain(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("service: get chain: %w", err)
	}
	return ch, nil
}
func (s *Service) UpdateChain(ctx context.Context, ch *types.SkillChain) error {
	if s.graph == nil {
		return fmt.Errorf("service: no graph store configured")
	}
	return s.graph.UpdateChain(ctx, ch)
}
func (s *Service) DeleteChain(ctx context.Context, id string) error {
	if s.graph == nil {
		return fmt.Errorf("service: no graph store configured")
	}
	return s.graph.DeleteChain(ctx, id)
}
func (s *Service) ExecuteChain(ctx context.Context, req *types.ChainExecutionRequest) (*types.ChainExecution, error) {
	if s.graph == nil {
		return nil, fmt.Errorf("service: no graph store configured")
	}
	if req == nil {
		return nil, fmt.Errorf("service: execute chain: nil request")
	}
	ch, err := s.graph.GetChain(ctx, req.ChainID)
	if err != nil {
		return nil, fmt.Errorf("service: execute chain: %w", err)
	}
	if ch == nil {
		return nil, fmt.Errorf("service: chain not found: %s", req.ChainID)
	}
	sort.SliceStable(ch.Steps, func(i, j int) bool {
		return ch.Steps[i].Order < ch.Steps[j].Order
	})

	exec := &types.ChainExecution{
		ID:        generateUUID(),
		ChainID:   req.ChainID,
		Status:    types.ChainStatusActive,
		StartedAt: time.Now(),
	}
	if req.Params != nil {
		exec.Metadata = map[string]interface{}{"params": req.Params}
	}
	if err := s.graph.UpdateChainExecution(ctx, exec); err != nil {
		return nil, fmt.Errorf("service: persist chain execution: %w", err)
	}

	currentParams := map[string]interface{}{}
	for k, v := range req.Params {
		currentParams[k] = v
	}

	for _, step := range ch.Steps {
		stepStart := time.Now()
		stepParams := map[string]interface{}{}
		for k, v := range currentParams {
			stepParams[k] = v
		}
		stepParams["chain_id"] = ch.ID
		stepParams["chain_name"] = ch.Name
		stepParams["step_order"] = step.Order
		stepParams["previous_results"] = exec.Results

		output, err := s.ExecuteSkill(ctx, step.SkillID, stepParams)
		stepResult := types.ChainStepResult{
			StepID:    fmt.Sprintf("%s:%d", step.SkillID, step.Order),
			SkillID:   step.SkillID,
			Output:    output,
			Status:    "completed",
			Duration:  float64(time.Since(stepStart).Milliseconds()),
			Timestamp: time.Now(),
		}
		if err != nil {
			stepResult.Status = "failed"
			stepResult.Output = err.Error()
			exec.Results = append(exec.Results, stepResult)
			exec.Status = types.ChainStatusFailed
			exec.Error = err.Error()
			completedAt := time.Now()
			exec.CompletedAt = &completedAt
			_ = s.graph.UpdateChainExecution(ctx, exec)
			return exec, fmt.Errorf("service: execute chain step %s: %w", step.SkillID, err)
		}
		exec.Results = append(exec.Results, stepResult)
		currentParams["last_output"] = output
		_ = s.graph.UpdateChainExecution(ctx, exec)
	}

	completedAt := time.Now()
	exec.CompletedAt = &completedAt
	exec.Status = types.ChainStatusCompleted
	if err := s.graph.UpdateChainExecution(ctx, exec); err != nil {
		return nil, fmt.Errorf("service: persist completed chain execution: %w", err)
	}
	_ = s.graph.IncrementChainUsage(ctx, req.ChainID)
	return exec, nil
}
func (s *Service) GetChainExecutions(ctx context.Context, id string, lim int) ([]*types.ChainExecution, error) {
	if s.graph == nil {
		return []*types.ChainExecution{}, nil
	}
	execs, err := s.graph.GetChainExecutions(ctx, id, lim)
	if err != nil {
		return nil, fmt.Errorf("service: get chain executions: %w", err)
	}
	if execs == nil {
		return []*types.ChainExecution{}, nil
	}
	return execs, nil
}
func (s *Service) ExtractChains(ctx context.Context, ids []string) ([]*types.SkillChain, error) {
	// No dedicated extraction method; fetch existing chains by IDs
	if s.graph == nil || len(ids) == 0 {
		return []*types.SkillChain{}, nil
	}
	var chains []*types.SkillChain
	for _, id := range ids {
		ch, err := s.graph.GetChain(ctx, id)
		if err == nil && ch != nil {
			chains = append(chains, ch)
		}
	}
	if chains == nil {
		return []*types.SkillChain{}, nil
	}
	return chains, nil
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
	if s.graph == nil {
		return []*types.Review{}, nil
	}
	srs, err := s.graph.ListPendingReviews(ctx, status)
	if err != nil {
		return nil, fmt.Errorf("service: list pending reviews: %w", err)
	}
	var reviews []*types.Review
	for _, sr := range srs {
		reviews = append(reviews, &types.Review{
			ID:        sr.ID,
			SkillID:   sr.SkillID,
			Status:    string(sr.Status),
			Notes:     sr.Notes,
			CreatedAt: sr.CreatedAt,
		})
	}
	if reviews == nil {
		return []*types.Review{}, nil
	}
	return reviews, nil
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

// SetDefaultTenantID configures the tenant ID used for tenant-scoped read operations.
// When set to a non-empty value, GetMemory and related methods will only return data
// belonging to this tenant. Set to "" to revert to admin/internal mode (no filtering).
func (s *Service) SetDefaultTenantID(tenantID string) { s.defaultTenantID = tenantID }

// GetDefaultTenantID returns the currently configured default tenant ID.
func (s *Service) GetDefaultTenantID() string { return s.defaultTenantID }

// getMemoriesByIDs is a helper that performs batch retrieval with tenant isolation if configured.
func (s *Service) getMemoriesByIDs(ids []string) ([]*types.Memory, error) {
	if s.graph == nil || len(ids) == 0 {
		return []*types.Memory{}, nil
	}
	if s.defaultTenantID != "" && s.neo4jClient != nil {
		return s.neo4jClient.GetMemoriesByIDsForTenant(ids, s.defaultTenantID)
	}
	return s.graph.GetMemoriesByIDs(ids)
}

// Configuration methods — protected by configMu for concurrent safety
func (s *Service) GetCompressionMode() string {
	s.configMu.RLock()
	defer s.configMu.RUnlock()
	return s.compressionMode
}
func (s *Service) SetCompressionMode(m string) error {
	s.configMu.Lock()
	defer s.configMu.Unlock()
	s.compressionMode = m
	return nil
}
func (s *Service) GetTierPolicy() TierPolicy {
	s.configMu.RLock()
	defer s.configMu.RUnlock()
	return s.tierPolicy
}
func (s *Service) SetTierPolicy(p TierPolicy) error {
	s.configMu.Lock()
	s.tierPolicy = p
	s.configMu.Unlock()
	s.syncTierPolicyToRouter()
	return nil
}
func (s *Service) SetTemporalReasoningEnabled(e bool) {
	s.configMu.Lock()
	defer s.configMu.Unlock()
	s.temporalEnabled = e
}
func (s *Service) SetDecayEnabled(e bool) {
	s.configMu.Lock()
	defer s.configMu.Unlock()
	s.decayEnabled = e
}
func (s *Service) IsDecayEnabled() bool {
	s.configMu.RLock()
	defer s.configMu.RUnlock()
	return s.decayEnabled
}
func (s *Service) IsTemporalReasoningEnabled() bool {
	s.configMu.RLock()
	defer s.configMu.RUnlock()
	return s.temporalEnabled
}

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

func generateUUID() string {
	return uuid.New().String()
}

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
	meta := map[string]interface{}{
		"memory_id":        m.ID,
		"entity_id":        m.EntityID,
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
	if m.Metadata != nil {
		if entityIDs, ok := m.Metadata["entity_ids"]; ok {
			meta["entity_ids"] = entityIDs
		}
		if primary, ok := m.Metadata["primary_entity_id"]; ok {
			meta["primary_entity_id"] = primary
		}
		if facts, ok := m.Metadata["facts"]; ok {
			meta["facts"] = facts
		}
	}
	return meta
}

// filtersToMap converts SearchFilters to a flat map for the vector store.
// Rules with an explicit non-eq operator are encoded under the special key
// "__op_filters__" (as []types.SearchFilter) so the Qdrant client can build
// range / in / not_eq conditions. Simple eq rules remain as field→value pairs.
func (s *Service) filtersToMap(f *types.SearchFilters) map[string]interface{} {
	if f == nil {
		return nil
	}
	result := make(map[string]interface{})
	var opRules []types.SearchFilter
	for _, rule := range f.Rules {
		op := rule.Operator
		if op == "" || op == "eq" {
			result[rule.Field] = rule.Value
		} else {
			opRules = append(opRules, rule)
		}
	}
	if len(opRules) > 0 {
		result["__op_filters__"] = opRules
	}
	return result
}

// chunkAndStoreChildren splits parentContent into smaller chunks and persists each
// chunk as a child memory node in the graph store, linked to the parent via metadata.
func (s *Service) chunkAndStoreChildren(ctx context.Context, parentID, parentContent string, parent *types.Memory) error {
	chunkSize := 2048
	if s.config != nil && s.config.Memory.ChunkingMaxBytes > 0 {
		chunkSize = s.config.Memory.ChunkingMaxBytes
	}
	overlap := chunkSize / 10 // 10% overlap
	if overlap < 50 {
		overlap = 50
	}

	chunker := newChunker(chunkSize, overlap)
	chunks := chunker(parentContent)
	if len(chunks) <= 1 {
		return nil // nothing to do — content fits in a single chunk
	}

	for i, chunkText := range chunks {
		child := &types.Memory{
			ID:             generateUUID(),
			Content:        chunkText,
			UserID:         parent.UserID,
			OrgID:          parent.OrgID,
			TenantID:       parent.TenantID,
			AgentID:        parent.AgentID,
			Type:           parent.Type,
			Category:       parent.Category,
			Status:         types.MemoryStatusActive,
			ValidityStatus: types.ValidityCurrent,
			Importance:     parent.Importance,
			GraphLayer:     types.GraphLayerEvent,
			SourceType:     parent.SourceType,
			CreatedAt:      parent.CreatedAt,
			UpdatedAt:      parent.UpdatedAt,
			Version:        1,
			Metadata: map[string]interface{}{
				"parent_memory_id": parentID,
				"chunk_index":      i,
				"chunk_total":      len(chunks),
				"is_chunk":         true,
			},
		}

		if err := s.graph.CreateMemory(child); err != nil {
			log.Printf("service: create chunk %d for parent %s: %v", i, parentID, err)
			continue
		}

		// Link child to parent in the graph
		if err := s.graph.CreateMemoryLink(&types.MemoryLink{
			FromID: parentID,
			ToID:   child.ID,
			Type:   types.MemoryLinkRelated,
		}); err != nil {
			log.Printf("service: link chunk %s to parent %s: %v", child.ID, parentID, err)
		}

		// Embed chunk for vector search
		if s.embedder != nil {
			emb, err := s.embedder.GenerateEmbeddingWithContext(ctx, chunkText)
			if err == nil && len(emb) > 0 {
				if _, err := s.vector.StoreEmbedding(ctx, chunkText, child.ID, emb, s.buildMemoryMetadata(child)); err != nil {
					log.Printf("service: embed chunk %s: %v", child.ID, err)
				}
			}
		}
	}
	return nil
}

// newChunker returns a function that splits text into chunks of the given size with overlap.
func newChunker(chunkSize, overlap int) func(string) []string {
	return func(text string) []string {
		if len(text) == 0 {
			return nil
		}
		var chunks []string
		runes := []rune(text)
		step := chunkSize - overlap
		if step <= 0 {
			step = chunkSize
		}
		for start := 0; start < len(runes); start += step {
			end := start + chunkSize
			if end > len(runes) {
				end = len(runes)
			}
			chunk := strings.TrimSpace(string(runes[start:end]))
			if chunk != "" {
				chunks = append(chunks, chunk)
			}
			if end >= len(runes) {
				break
			}
		}
		return chunks
	}
}

// ============================================================
// Adapters satisfying self_improve interfaces for neo4j.Client
// ============================================================

// graphFeedbackAdapter wraps *neo4j.Client to satisfy self_improve.GraphFeedbackStore.
type graphFeedbackAdapter struct {
	client *neo4j.Client
}

func (a *graphFeedbackAdapter) GetFeedbackByMemory(ctx context.Context, memoryID string) ([]*types.Feedback, error) {
	return a.client.GetFeedbackByMemory(ctx, memoryID)
}

// graphTuningAdapter wraps *neo4j.Client to satisfy self_improve.GraphTuningOps.
type graphTuningAdapter struct {
	client *neo4j.Client
}

func (a *graphTuningAdapter) GetMemory(id string) (*types.Memory, error) {
	return a.client.GetMemory(id)
}

func (a *graphTuningAdapter) UpdateMemory(mem *types.Memory) error {
	return a.client.UpdateMemory(mem)
}
