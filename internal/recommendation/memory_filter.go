package recommendation

import (
	"context"
	"strings"
	"time"
)

// DropDuplicatesFilter removes candidate memories with duplicate IDs.
type DropDuplicatesFilter struct{}

func (f *DropDuplicatesFilter) Name() string { return "drop_duplicates" }

func (f *DropDuplicatesFilter) Filter(ctx context.Context, query *QueryContext, candidates []*MemoryCandidate) ([]*MemoryCandidate, error) {
	seen := make(map[string]bool)
	var result []*MemoryCandidate

	for _, c := range candidates {
		if !seen[c.ID] {
			seen[c.ID] = true
			result = append(result, c)
		}
	}

	return result, nil
}

// SelfMemoryFilter removes the user's own memories (don't recommend what they already wrote).
type SelfMemoryFilter struct{}

func (f *SelfMemoryFilter) Name() string { return "self_memory_filter" }

func (f *SelfMemoryFilter) Filter(ctx context.Context, query *QueryContext, candidates []*MemoryCandidate) ([]*MemoryCandidate, error) {
	var result []*MemoryCandidate

	for _, c := range candidates {
		authorID, _ := c.Metadata["author_id"].(string)
		if authorID != query.UserID {
			result = append(result, c)
		}
	}

	return result, nil
}

// AgeFilter removes memories older than the maximum age threshold.
type AgeFilter struct {
	MaxAge time.Duration
}

func (f *AgeFilter) Name() string { return "age_filter" }

func (f *AgeFilter) Filter(ctx context.Context, query *QueryContext, candidates []*MemoryCandidate) ([]*MemoryCandidate, error) {
	maxAge := f.MaxAge
	if maxAge == 0 {
		maxAge = 30 * 24 * time.Hour // Default: 30 days
	}

	cutoff := time.Now().Add(-maxAge)
	var result []*MemoryCandidate

	for _, c := range candidates {
		createdAt, ok := c.Metadata["created_at"].(time.Time)
		if !ok {
			// If we can't determine age, keep the candidate
			result = append(result, c)
			continue
		}
		if !createdAt.Before(cutoff) {
			result = append(result, c)
		}
	}

	return result, nil
}

// PreviouslySeenFilter removes memories the user has already viewed.
type PreviouslySeenFilter struct{}

func (f *PreviouslySeenFilter) Name() string { return "previously_seen" }

func (f *PreviouslySeenFilter) Filter(ctx context.Context, query *QueryContext, candidates []*MemoryCandidate) ([]*MemoryCandidate, error) {
	seenSet := make(map[string]bool)
	for _, id := range query.SeenIDs {
		seenSet[id] = true
	}

	var result []*MemoryCandidate
	for _, c := range candidates {
		if !seenSet[c.ID] {
			result = append(result, c)
		}
	}

	return result, nil
}

// PreviouslyServedFilter removes memories already served in the current session.
type PreviouslyServedFilter struct{}

func (f *PreviouslyServedFilter) Name() string { return "previously_served" }

func (f *PreviouslyServedFilter) Filter(ctx context.Context, query *QueryContext, candidates []*MemoryCandidate) ([]*MemoryCandidate, error) {
	servedSet := make(map[string]bool)
	for _, id := range query.ServedIDs {
		servedSet[id] = true
	}

	var result []*MemoryCandidate
	for _, c := range candidates {
		if !servedSet[c.ID] {
			result = append(result, c)
		}
	}

	return result, nil
}

// AuthorSocialGraphFilter removes memories from blocked or muted authors.
type AuthorSocialGraphFilter struct{}

func (f *AuthorSocialGraphFilter) Name() string { return "author_social_graph" }

func (f *AuthorSocialGraphFilter) Filter(ctx context.Context, query *QueryContext, candidates []*MemoryCandidate) ([]*MemoryCandidate, error) {
	blockedSet := make(map[string]bool)
	for _, id := range query.BlockedIDs {
		blockedSet[id] = true
	}

	var result []*MemoryCandidate
	for _, c := range candidates {
		authorID, _ := c.Metadata["author_id"].(string)
		if !blockedSet[authorID] {
			result = append(result, c)
		}
	}

	return result, nil
}

// MutedKeywordFilter removes memories containing muted keywords.
type MutedKeywordFilter struct{}

func (f *MutedKeywordFilter) Name() string { return "muted_keyword" }

func (f *MutedKeywordFilter) Filter(ctx context.Context, query *QueryContext, candidates []*MemoryCandidate) ([]*MemoryCandidate, error) {
	if len(query.MutedKeywords) == 0 {
		return candidates, nil
	}

	// Build lowercase keyword set for matching
	keywords := make([]string, 0, len(query.MutedKeywords))
	for _, kw := range query.MutedKeywords {
		keywords = append(keywords, strings.ToLower(kw))
	}

	var result []*MemoryCandidate
	for _, c := range candidates {
		content, _ := c.Metadata["content"].(string)
		lowerContent := strings.ToLower(content)

		muted := false
		for _, kw := range keywords {
			if strings.Contains(lowerContent, kw) {
				muted = true
				break
			}
		}

		if !muted {
			result = append(result, c)
		}
	}

	return result, nil
}

// CoreDataHydrationFilter removes candidates that failed to load core metadata.
type CoreDataHydrationFilter struct{}

func (f *CoreDataHydrationFilter) Name() string { return "core_data_hydration" }

func (f *CoreDataHydrationFilter) Filter(ctx context.Context, query *QueryContext, candidates []*MemoryCandidate) ([]*MemoryCandidate, error) {
	var result []*MemoryCandidate

	for _, c := range candidates {
		// Must have content and type to be useful
		content, hasContent := c.Metadata["content"]
		_, hasType := c.Metadata["type"]

		if hasContent && content != "" && hasType {
			result = append(result, c)
		}
	}

	return result, nil
}

// ApplyAllFilters returns a convenience function that chains multiple filters.
func ApplyAllFilters() []Filter {
	return []Filter{
		&DropDuplicatesFilter{},
		&SelfMemoryFilter{},
		&PreviouslySeenFilter{},
		&PreviouslyServedFilter{},
		&AuthorSocialGraphFilter{},
		&MutedKeywordFilter{},
		&CoreDataHydrationFilter{},
		&AgeFilter{MaxAge: 30 * 24 * time.Hour},
	}
}
