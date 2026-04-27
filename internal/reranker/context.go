package reranker

import (
	"context"
	"sort"
	"strings"
	"time"

	"agent-memory/internal/memory/types"
)

type ContextReranker struct {
	base Provider
}

func NewContextReranker(base Provider) *ContextReranker {
	return &ContextReranker{base: base}
}

func (r *ContextReranker) Rerank(ctx context.Context, query string, results []types.MemoryResult, limit int) ([]types.MemoryResult, error) {
	if r.base != nil {
		var err error
		results, err = r.base.Rerank(ctx, query, results, limit*2)
		if err != nil {
			return nil, err
		}
	}

	results = r.contextAwareSort(ctx, query, results)

	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

func (r *ContextReranker) contextAwareSort(ctx context.Context, query string, results []types.MemoryResult) []types.MemoryResult {
	scores := make([]float32, len(results))
	queryLower := strings.ToLower(query)

	for i, result := range results {
		score := result.Score

		if result.Metadata != nil {
			if result.Metadata.Importance == "high" {
				score += 0.1
			} else if result.Metadata.Importance == "critical" {
				score += 0.2
			}

			if !result.Metadata.UpdatedAt.IsZero() {
				daysSince := time.Since(result.Metadata.UpdatedAt).Hours() / 24
				if daysSince < 7 {
					score += 0.05
				} else if daysSince > 30 {
					score -= 0.05
				}
			}

			if !result.Metadata.CreatedAt.IsZero() {
				daysSince := time.Since(result.Metadata.CreatedAt).Hours() / 24
				if daysSince < 1 {
					score += 0.03
				}
			}
		}

		text := strings.ToLower(result.Text)
		if strings.Contains(text, queryLower) {
			score += 0.15
		}

		words := strings.Fields(queryLower)
		matchCount := 0
		for _, word := range words {
			if strings.Contains(text, word) {
				matchCount++
			}
		}
		if len(words) > 0 {
			score += float32(matchCount) / float32(len(words)) * 0.1
		}

		if result.Metadata != nil {
			accessCount, _ := result.Metadata.Metadata["access_count"].(int)
			if accessCount > 10 {
				score += 0.05
			}
		}

		scores[i] = score
	}

	sorted := make([]types.MemoryResult, len(results))
	copy(sorted, results)

	sort.Slice(sorted, func(i, j int) bool {
		return scores[i] > scores[j]
	})

	return sorted
}

func (r *ContextReranker) Name() string {
	return "context_aware"
}

func (r *ContextReranker) Close() error {
	if r.base != nil {
		return r.base.Close()
	}
	return nil
}