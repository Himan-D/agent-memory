package recommendation

import (
	"context"
	"sort"
)

// TopKSelector sorts candidates by score and selects the top K.
type TopKSelector struct {
	MaxResults      int
	AuthorDiversity bool
	MaxPerAuthor    int
}

// NewTopKSelector creates a selector with diversity constraints.
func NewTopKSelector(maxResults int, authorDiversity bool, maxPerAuthor int) *TopKSelector {
	if maxResults == 0 {
		maxResults = 20
	}
	if maxPerAuthor == 0 {
		maxPerAuthor = 3
	}
	return &TopKSelector{
		MaxResults:      maxResults,
		AuthorDiversity: authorDiversity,
		MaxPerAuthor:    maxPerAuthor,
	}
}

func (s *TopKSelector) Name() string {
	return "top_k_selector"
}

func (s *TopKSelector) Select(ctx context.Context, query *QueryContext, candidates []*MemoryCandidate) ([]*MemoryCandidate, error) {
	// Sort by score descending
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Score > candidates[j].Score
	})

	if !s.AuthorDiversity {
		// Simple top-K selection
		if len(candidates) > s.MaxResults {
			return candidates[:s.MaxResults], nil
		}
		return candidates, nil
	}

	// Author diversity: limit results per author to avoid feed monopolization
	authorCounts := make(map[string]int)
	var selected []*MemoryCandidate

	for _, c := range candidates {
		if len(selected) >= s.MaxResults {
			break
		}

		authorID, _ := c.Metadata["author_id"].(string)
		if authorCounts[authorID] >= s.MaxPerAuthor {
			continue // Skip: too many from this author
		}

		authorCounts[authorID]++
		selected = append(selected, c)
	}

	return selected, nil
}
