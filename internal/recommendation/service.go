package recommendation

import (
	"context"
	"fmt"
	"time"

	"agent-memory/internal/llm"
)

// Service orchestrates the recommendation pipeline (Home Mixer equivalent).
// It assembles sources, hydrators, filters, scorers, and selectors into a cohesive
// "For You Memories" feed for AI agents.
type Service struct {
	pipeline *Pipeline
	config   *ServiceConfig
}

// ServiceConfig holds configuration for the recommendation service.
type ServiceConfig struct {
	MaxResults       int
	PipelineTimeout  time.Duration
	EnableDiversity  bool
	MaxPerAuthor     int
	EnableLLMScoring bool
}

// DefaultServiceConfig returns sensible defaults.
func DefaultServiceConfig() *ServiceConfig {
	return &ServiceConfig{
		MaxResults:       20,
		PipelineTimeout:  5 * time.Second,
		EnableDiversity:  true,
		MaxPerAuthor:     3,
		EnableLLMScoring: true,
	}
}

// Builder provides a fluent API for constructing the recommendation service.
type Builder struct {
	config          *ServiceConfig
	pipelineConfig  *PipelineConfig
	memoryStore     MemoryDetailStore
	engagementStore EngagementStore
	authorStore     AuthorStore
	vectorStore     VectorSearchStore
	graphStore      GraphSearchStore
	embedder        Embedder
	llmProvider     llm.Provider
	cache           RecommendationCache
	analyticsLogger AnalyticsLogger
	servedTracker   ServedTracker
}

// NewBuilder creates a new recommendation service builder.
func NewBuilder() *Builder {
	return &Builder{
		config:         DefaultServiceConfig(),
		pipelineConfig: DefaultPipelineConfig(),
	}
}

// WithConfig sets the service configuration.
func (b *Builder) WithConfig(cfg *ServiceConfig) *Builder {
	b.config = cfg
	return b
}

// WithMemoryStore sets the memory detail store.
func (b *Builder) WithMemoryStore(store MemoryDetailStore) *Builder {
	b.memoryStore = store
	return b
}

// WithEngagementStore sets the engagement history store.
func (b *Builder) WithEngagementStore(store EngagementStore) *Builder {
	b.engagementStore = store
	return b
}

// WithAuthorStore sets the author information store.
func (b *Builder) WithAuthorStore(store AuthorStore) *Builder {
	b.authorStore = store
	return b
}

// WithVectorStore sets the vector similarity search store.
func (b *Builder) WithVectorStore(store VectorSearchStore) *Builder {
	b.vectorStore = store
	return b
}

// WithGraphStore sets the graph search store (for spreading activation).
func (b *Builder) WithGraphStore(store GraphSearchStore) *Builder {
	b.graphStore = store
	return b
}

// WithEmbedder sets the embedding provider.
func (b *Builder) WithEmbedder(embedder Embedder) *Builder {
	b.embedder = embedder
	return b
}

// WithLLMProvider sets the LLM provider for Phoenix scoring.
func (b *Builder) WithLLMProvider(provider llm.Provider) *Builder {
	b.llmProvider = provider
	return b
}

// WithCache sets the recommendation cache.
func (b *Builder) WithCache(cache RecommendationCache) *Builder {
	b.cache = cache
	return b
}

// WithAnalytics sets the analytics logger.
func (b *Builder) WithAnalytics(logger AnalyticsLogger) *Builder {
	b.analyticsLogger = logger
	return b
}

// WithServedTracker sets the served memory tracker.
func (b *Builder) WithServedTracker(tracker ServedTracker) *Builder {
	b.servedTracker = tracker
	return b
}

// Build constructs the recommendation service.
func (b *Builder) Build() (*Service, error) {
	pipeline := NewPipeline(b.pipelineConfig)

	// === QUERY HYDRATORS ===
	// (User-provided query context is sufficient for now; add custom hydrators if needed)

	// === SOURCES ===
	// 1. In-Network: Memories from followed entities (Thunder equivalent)
	if b.memoryStore != nil {
		// Use Neo4jStoreAdapter directly if available, otherwise fall back to wrapper
		if neoAdapter, ok := b.memoryStore.(*Neo4jStoreAdapter); ok {
			pipeline.WithSources(NewInNetworkSource(neoAdapter, 48*time.Hour))
		} else {
			pipeline.WithSources(NewInNetworkSource(
				adaptMemoryStoreToCandidateStore(b.memoryStore),
				48*time.Hour,
			))
		}
	}

	// 2. Out-of-Network: Global corpus discovery via spreading activation (Phoenix Retrieval equivalent)
	if b.vectorStore != nil && b.graphStore != nil {
		outNetworkSource := NewOutOfNetworkSource(
			b.vectorStore,
			b.graphStore,
			50, // topK
			2,  // hops
		)
		pipeline.WithSources(outNetworkSource)
	}

	// 3. Prompt-Based: Memories relevant to current prompt/context
	if b.memoryStore != nil && b.embedder != nil && b.vectorStore != nil {
		promptSource := NewPromptSource(
			adaptMemoryStoreToCandidateStore(b.memoryStore),
			b.embedder,
			b.vectorStore,
			30,
		)
		pipeline.WithSources(promptSource)
	}

	// === HYDRATORS ===
	if b.memoryStore != nil {
		pipeline.WithHydrators(NewMemoryHydrator(b.memoryStore))
	}
	if b.authorStore != nil {
		pipeline.WithHydrators(NewAuthorHydrator(b.authorStore))
	}
	if b.engagementStore != nil {
		pipeline.WithHydrators(NewEngagementHydrator(b.engagementStore))
	}

	// === FILTERS ===
	pipeline.WithFilters(ApplyAllFilters()...)

	// === SCORERS ===
	if b.llmProvider != nil && b.config.EnableLLMScoring {
		phoenixScorer := NewPhoenixMemoryScorer(b.llmProvider, DefaultActionWeights)
		pipeline.WithScorers(phoenixScorer)
	} else {
		// Fallback: heuristic scorer when LLM is disabled
		phoenixScorer := NewPhoenixMemoryScorer(nil, DefaultActionWeights)
		pipeline.WithScorers(phoenixScorer)
	}

	// Add author diversity scorer
	authorDiversityScorer := NewAuthorDiversityScorer()
	pipeline.WithScorers(authorDiversityScorer)

	// === SELECTOR ===
	pipeline.WithSelector(NewTopKSelector(
		b.config.MaxResults,
		b.config.EnableDiversity,
		b.config.MaxPerAuthor,
	))

	// === SIDE EFFECTS ===
	if b.cache != nil {
		pipeline.WithSideEffects(NewCacheSideEffect(b.cache, 5*time.Minute))
	}
	if b.analyticsLogger != nil {
		pipeline.WithSideEffects(NewAnalyticsSideEffect(b.analyticsLogger))
	}
	if b.servedTracker != nil {
		pipeline.WithSideEffects(NewServedTrackingSideEffect(b.servedTracker))
	}

	return &Service{
		pipeline: pipeline,
		config:   b.config,
	}, nil
}

// Recommend runs the recommendation pipeline and returns ranked memories.
func (s *Service) Recommend(ctx context.Context, query *QueryContext) ([]*MemoryCandidate, error) {
	if query.Metadata == nil {
		query.Metadata = make(map[string]interface{})
	}

	candidates, err := s.pipeline.Execute(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("recommendation pipeline: %w", err)
	}

	return candidates, nil
}

// RecommendWithPrompt runs recommendations with the agent's current prompt context.
func (s *Service) RecommendWithPrompt(ctx context.Context, query *QueryContext, promptText string) ([]*MemoryCandidate, error) {
	if query.Metadata == nil {
		query.Metadata = make(map[string]interface{})
	}
	query.Metadata["prompt_text"] = promptText
	return s.Recommend(ctx, query)
}

// GetCandidateByID retrieves a specific candidate from results (for debugging).
func GetCandidateByID(candidates []*MemoryCandidate, id string) *MemoryCandidate {
	for _, c := range candidates {
		if c.ID == id {
			return c
		}
	}
	return nil
}

// CandidateToIDs extracts memory IDs from candidates.
func CandidateToIDs(candidates []*MemoryCandidate) []string {
	ids := make([]string, 0, len(candidates))
	for _, c := range candidates {
		ids = append(ids, c.ID)
	}
	return ids
}

// CandidateSummary returns a human-readable summary of recommendation results.
func CandidateSummary(candidates []*MemoryCandidate) string {
	if len(candidates) == 0 {
		return "No recommendations"
	}

	totalScore := 0.0
	uniqueAuthors := make(map[string]bool)
	for _, c := range candidates {
		totalScore += c.Score
		authorID, _ := c.Metadata["author_id"].(string)
		if authorID != "" {
			uniqueAuthors[authorID] = true
		}
	}

	return fmt.Sprintf("%d memories, avg score: %.2f, %d unique authors",
		len(candidates), totalScore/float64(len(candidates)), len(uniqueAuthors))
}

// adaptMemoryStoreToCandidateStore wraps a MemoryDetailStore to implement MemoryStore.
func adaptMemoryStoreToCandidateStore(store MemoryDetailStore) MemoryStore {
	return &memoryStoreAdapter{detailStore: store}
}

type memoryStoreAdapter struct {
	detailStore MemoryDetailStore
}

func (a *memoryStoreAdapter) GetMemoriesByIDs(ctx context.Context, ids []string) ([]interface{}, error) {
	var result []interface{}
	for _, id := range ids {
		detail, err := a.detailStore.GetMemoryWithMetadata(ctx, id)
		if err != nil {
			continue
		}
		result = append(result, detail)
	}
	return result, nil
}

func (a *memoryStoreAdapter) GetRecentMemoriesByEntity(ctx context.Context, entityID string, limit int, maxAge time.Duration) ([]string, error) {
	return []string{}, nil
}

func (a *memoryStoreAdapter) GetMemoryWithMetadata(ctx context.Context, id string) (*MemoryDetail, error) {
	return a.detailStore.GetMemoryWithMetadata(ctx, id)
}
