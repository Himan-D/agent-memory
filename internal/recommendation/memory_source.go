package recommendation

import (
	"context"
	"fmt"
	"time"
)

// InNetworkSource fetches memories from entities the agent follows (Thunder equivalent).
// Returns recent memories from followed agents, entities, and skills.
type InNetworkSource struct {
	store        MemoryStore
	retentionMax time.Duration
}

// MemoryStore provides access to stored memories.
type MemoryStore interface {
	GetMemoriesByIDs(ctx context.Context, ids []string) ([]interface{}, error)
	GetRecentMemoriesByEntity(ctx context.Context, entityID string, limit int, maxAge time.Duration) ([]string, error)
}

// NewInNetworkSource creates a source for followed-entity memories.
func NewInNetworkSource(store MemoryStore, retentionMax time.Duration) *InNetworkSource {
	if retentionMax == 0 {
		retentionMax = 48 * time.Hour // Default: last 48 hours
	}
	return &InNetworkSource{
		store:        store,
		retentionMax: retentionMax,
	}
}

func (s *InNetworkSource) Name() string {
	return "in_network_source"
}

func (s *InNetworkSource) Fetch(ctx context.Context, query *QueryContext) ([]string, error) {
	var allIDs []string

	// Fetch recent memories from each followed entity
	for _, entityID := range query.FollowingIDs {
		select {
		case <-ctx.Done():
			return allIDs, ctx.Err()
		default:
		}

		ids, err := s.store.GetRecentMemoriesByEntity(ctx, entityID, 30, s.retentionMax)
		if err != nil {
			continue // Skip failed entity, don't fail the whole source
		}
		allIDs = append(allIDs, ids...)
	}

	// Fallback: if no following IDs specified, fetch all recent memories
	if len(query.FollowingIDs) == 0 {
		ids, err := s.store.GetRecentMemoriesByEntity(ctx, "", 50, s.retentionMax)
		if err == nil {
			allIDs = append(allIDs, ids...)
		}
	}

	return allIDs, nil
}

// OutOfNetworkSource fetches memories from the global corpus using vector similarity
// and spreading activation (Phoenix Retrieval equivalent).
type OutOfNetworkSource struct {
	vectorStore    VectorSearchStore
	graphStore     GraphSearchStore
	topK           int
	activationHops int
}

// VectorSearchStore provides vector similarity search.
type VectorSearchStore interface {
	SearchByVector(ctx context.Context, vector []float32, limit int, filters map[string]interface{}) ([]string, error)
}

// GraphSearchStore provides graph-based retrieval (spreading activation).
type GraphSearchStore interface {
	SearchBySpreadingActivation(ctx context.Context, seedIDs []string, hops int, threshold float64) ([]string, error)
	SearchByTopic(ctx context.Context, topicIDs []string, limit int) ([]string, error)
}

// NewOutOfNetworkSource creates a source for global corpus discovery.
func NewOutOfNetworkSource(vector VectorSearchStore, graph GraphSearchStore, topK int, hops int) *OutOfNetworkSource {
	if topK == 0 {
		topK = 50
	}
	if hops == 0 {
		hops = 2
	}
	return &OutOfNetworkSource{
		vectorStore:    vector,
		graphStore:     graph,
		topK:           topK,
		activationHops: hops,
	}
}

func (s *OutOfNetworkSource) Name() string {
	return "out_of_network_source"
}

func (s *OutOfNetworkSource) Fetch(ctx context.Context, query *QueryContext) ([]string, error) {
	var allIDs []string

	// 1. Topic-based retrieval (if agent has topic preferences)
	if len(query.TopicIDs) > 0 && s.graphStore != nil {
		ids, err := s.graphStore.SearchByTopic(ctx, query.TopicIDs, s.topK)
		if err == nil {
			allIDs = append(allIDs, ids...)
		}
	}

	// 2. Engagement-history-based spreading activation
	// Use the agent's recent engaged memories as seeds for graph propagation
	seedIDs := s.getEngagementSeeds(query)
	if len(seedIDs) > 0 && s.graphStore != nil {
		activatedIDs, err := s.graphStore.SearchBySpreadingActivation(ctx, seedIDs, s.activationHops, 0.1)
		if err == nil {
			allIDs = append(allIDs, activatedIDs...)
		}
	}

	// 3. Fallback: if no graph results, use vector search with user embedding
	if len(allIDs) == 0 && s.vectorStore != nil {
		userVector := s.buildUserVector(query)
		if userVector != nil {
			ids, err := s.vectorStore.SearchByVector(ctx, userVector, s.topK, nil)
			if err == nil {
				allIDs = append(allIDs, ids...)
			}
		}
	}

	return allIDs, nil
}

func (s *OutOfNetworkSource) getEngagementSeeds(query *QueryContext) []string {
	var seeds []string
	seen := make(map[string]bool)

	// Use recently engaged memories as graph propagation seeds
	for i := len(query.EngagementHistory) - 1; i >= 0 && len(seeds) < 10; i-- {
		e := query.EngagementHistory[i]
		if e.Action == "use" || e.Action == "reference" || e.Action == "derive" {
			if !seen[e.MemoryID] {
				seen[e.MemoryID] = true
				seeds = append(seeds, e.MemoryID)
			}
		}
	}

	return seeds
}

func (s *OutOfNetworkSource) buildUserVector(query *QueryContext) []float32 {
	// Build a simple user preference vector from engagement history
	// In a real implementation, this would use a trained two-tower user tower model
	// For now, return nil to signal that vector search should use a default query
	return nil
}

// PromptSource fetches memories suggested by the agent's current prompt/context.
// This is analogous to X's "prompts" candidate source.
type PromptSource struct {
	store       MemoryStore
	embedder    Embedder
	vectorStore VectorSearchStore
	topK        int
}

// Embedder converts text to embedding vectors.
type Embedder interface {
	Embed(ctx context.Context, text string) ([]float32, error)
}

// NewPromptSource creates a source for prompt-relevant memories.
func NewPromptSource(store MemoryStore, embedder Embedder, vector VectorSearchStore, topK int) *PromptSource {
	if topK == 0 {
		topK = 30
	}
	return &PromptSource{
		store:       store,
		embedder:    embedder,
		vectorStore: vector,
		topK:        topK,
	}
}

func (s *PromptSource) Name() string {
	return "prompt_source"
}

func (s *PromptSource) Fetch(ctx context.Context, query *QueryContext) ([]string, error) {
	promptText, _ := query.Metadata["prompt_text"].(string)
	if promptText == "" || s.vectorStore == nil || s.embedder == nil {
		return nil, nil
	}

	// Embed the prompt text and search for semantically similar memories
	vector, err := s.embedder.Embed(ctx, promptText)
	if err != nil {
		return nil, fmt.Errorf("embed prompt: %w", err)
	}

	ids, err := s.vectorStore.SearchByVector(ctx, vector, s.topK, nil)
	if err != nil {
		return nil, fmt.Errorf("vector search: %w", err)
	}

	return ids, nil
}
