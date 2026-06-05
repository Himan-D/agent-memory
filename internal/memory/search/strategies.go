package search

import (
	"context"
	"sort"
)

// AdaptiveResult holds a single result from an adaptive retrieval strategy.
type AdaptiveResult struct {
	MemoryID string
	Content  string
	Score    float64
	Source   string // which strategy produced this result
}

// Strategy defines the interface for adaptive retrieval strategies.
type Strategy interface {
	Execute(ctx context.Context, query string, limit int) ([]AdaptiveResult, error)
}

// DirectStrategy performs simple vector similarity search.
// It delegates to the underlying vector store for a single query.
type DirectStrategy struct {
	VectorStore VectorStore
	Collection  string
	EmbedFn     func(ctx context.Context, query string) ([]float32, error)
}

// Execute performs a direct vector similarity search.
func (ds *DirectStrategy) Execute(ctx context.Context, query string, limit int) ([]AdaptiveResult, error) {
	if ds.VectorStore == nil || ds.EmbedFn == nil {
		return nil, nil
	}

	embedding, err := ds.EmbedFn(ctx, query)
	if err != nil {
		return nil, err
	}

	collection := ds.Collection
	if collection == "" {
		collection = "memories"
	}

	results, err := ds.VectorStore.Search(collection, embedding, limit)
	if err != nil {
		return nil, err
	}

	adaptive := make([]AdaptiveResult, 0, len(results))
	for _, r := range results {
		content := ""
		if c, ok := r.Payload["content"].(string); ok {
			content = c
		}
		adaptive = append(adaptive, AdaptiveResult{
			MemoryID: r.ID,
			Content:  content,
			Score:    r.Score,
			Source:   "direct",
		})
	}

	return adaptive, nil
}

// ParallelStrategy decomposes a query into sub-queries, searches each in parallel, and merges results.
type ParallelStrategy struct {
	Router      *Router
	VectorStore VectorStore
	Collection  string
	EmbedFn     func(ctx context.Context, query string) ([]float32, error)
}

// Execute decomposes the query into sub-queries and merges results.
func (ps *ParallelStrategy) Execute(ctx context.Context, query string, limit int) ([]AdaptiveResult, error) {
	if ps.VectorStore == nil || ps.EmbedFn == nil || ps.Router == nil {
		return nil, nil
	}

	subQueries := ps.Router.DecomposeQuery(query)
	if len(subQueries) == 0 {
		subQueries = []string{query}
	}

	// Collect all results from sub-queries
	seen := make(map[string]AdaptiveResult)
	for _, sq := range subQueries {
		embedding, err := ps.EmbedFn(ctx, sq)
		if err != nil {
			continue
		}

		collection := ps.Collection
		if collection == "" {
			collection = "memories"
		}

		results, err := ps.VectorStore.Search(collection, embedding, limit)
		if err != nil {
			continue
		}

		for _, r := range results {
			existing, exists := seen[r.ID]
			if !exists || r.Score > existing.Score {
				content := ""
				if c, ok := r.Payload["content"].(string); ok {
					content = c
				}
				seen[r.ID] = AdaptiveResult{
					MemoryID: r.ID,
					Content:  content,
					Score:    r.Score,
					Source:   "parallel",
				}
			}
		}
	}

	// Flatten and sort by score descending
	merged := make([]AdaptiveResult, 0, len(seen))
	for _, r := range seen {
		merged = append(merged, r)
	}

	sort.Slice(merged, func(i, j int) bool {
		return merged[i].Score > merged[j].Score
	})

	if len(merged) > limit {
		merged = merged[:limit]
	}

	return merged, nil
}

// IterativeStrategy performs sequential narrowing with chain-of-query refinement.
type IterativeStrategy struct {
	VectorStore  VectorStore
	Collection   string
	EmbedFn      func(ctx context.Context, query string) ([]float32, error)
	MaxRounds    int // maximum refinement rounds (default: 3)
	NarrowFactor int // how many top results to use for narrowing (default: 5)
}

// Execute performs iterative narrowing: search, take top results, refine query context, repeat.
func (is *IterativeStrategy) Execute(ctx context.Context, query string, limit int) ([]AdaptiveResult, error) {
	if is.VectorStore == nil || is.EmbedFn == nil {
		return nil, nil
	}

	maxRounds := is.MaxRounds
	if maxRounds <= 0 {
		maxRounds = 3
	}
	narrowFactor := is.NarrowFactor
	if narrowFactor <= 0 {
		narrowFactor = 5
	}

	collection := is.Collection
	if collection == "" {
		collection = "memories"
	}

	var bestResults []AdaptiveResult
	currentQuery := query

	for round := 0; round < maxRounds; round++ {
		embedding, err := is.EmbedFn(ctx, currentQuery)
		if err != nil {
			break
		}

		results, err := is.VectorStore.Search(collection, embedding, narrowFactor)
		if err != nil {
			break
		}

		roundResults := make([]AdaptiveResult, 0, len(results))
		for _, r := range results {
			content := ""
			if c, ok := r.Payload["content"].(string); ok {
				content = c
			}
			roundResults = append(roundResults, AdaptiveResult{
				MemoryID: r.ID,
				Content:  content,
				Score:    r.Score,
				Source:   "iterative",
			})
		}

		// Merge with existing best results (keep highest scores)
		bestResults = mergeAdaptiveResults(bestResults, roundResults)

		// For next round, the top result's content narrows the search
		if len(roundResults) > 0 && roundResults[0].Content != "" {
			currentQuery = query + " " + roundResults[0].Content
		} else {
			break // No new results to refine with
		}
	}

	if len(bestResults) > limit {
		bestResults = bestResults[:limit]
	}

	return bestResults, nil
}

// mergeAdaptiveResults combines two result lists, deduplicating by MemoryID and keeping highest scores.
func mergeAdaptiveResults(a, b []AdaptiveResult) []AdaptiveResult {
	seen := make(map[string]AdaptiveResult)
	for _, r := range a {
		seen[r.MemoryID] = r
	}
	for _, r := range b {
		existing, exists := seen[r.MemoryID]
		if !exists || r.Score > existing.Score {
			seen[r.MemoryID] = r
		}
	}

	merged := make([]AdaptiveResult, 0, len(seen))
	for _, r := range seen {
		merged = append(merged, r)
	}

	sort.Slice(merged, func(i, j int) bool {
		return merged[i].Score > merged[j].Score
	})

	return merged
}
