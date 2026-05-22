package recommendation

import (
	"context"
	"log"
	"time"
)

// CacheSideEffect caches the recommendation results for future use.
type CacheSideEffect struct {
	cache RecommendationCache
	ttl   time.Duration
}

// RecommendationCache provides caching for recommendation results.
type RecommendationCache interface {
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
}

// NewCacheSideEffect creates a cache side effect.
func NewCacheSideEffect(cache RecommendationCache, ttl time.Duration) *CacheSideEffect {
	if ttl == 0 {
		ttl = 5 * time.Minute
	}
	return &CacheSideEffect{
		cache: cache,
		ttl:   ttl,
	}
}

func (s *CacheSideEffect) Name() string {
	return "cache_side_effect"
}

func (s *CacheSideEffect) Execute(ctx context.Context, query *QueryContext, candidates []*MemoryCandidate) error {
	if s.cache == nil {
		return nil
	}

	cacheKey := "recs:" + query.UserID + ":" + query.AgentID
	resultIDs := make([]string, 0, len(candidates))
	for _, c := range candidates {
		resultIDs = append(resultIDs, c.ID)
	}

	if err := s.cache.Set(ctx, cacheKey, resultIDs, s.ttl); err != nil {
		log.Printf("[recommendation] Cache error: %v", err)
	}

	return nil
}

// AnalyticsSideEffect logs recommendation analytics.
type AnalyticsSideEffect struct {
	logger AnalyticsLogger
}

// AnalyticsLogger provides analytics logging.
type AnalyticsLogger interface {
	LogEvent(ctx context.Context, event string, properties map[string]interface{}) error
}

// NewAnalyticsSideEffect creates an analytics side effect.
func NewAnalyticsSideEffect(logger AnalyticsLogger) *AnalyticsSideEffect {
	return &AnalyticsSideEffect{logger: logger}
}

func (s *AnalyticsSideEffect) Name() string {
	return "analytics_side_effect"
}

func (s *AnalyticsSideEffect) Execute(ctx context.Context, query *QueryContext, candidates []*MemoryCandidate) error {
	if s.logger == nil {
		return nil
	}

	properties := map[string]interface{}{
		"user_id":        query.UserID,
		"agent_id":       query.AgentID,
		"total_sourced":  len(query.SeenIDs) + len(query.ServedIDs),
		"results_count":  len(candidates),
		"avg_score":      s.avgScore(candidates),
		"top_score":      s.topScore(candidates),
		"unique_authors": s.uniqueAuthors(candidates),
		"timestamp":      time.Now().UTC().Format(time.RFC3339),
	}

	if err := s.logger.LogEvent(ctx, "recommendation_served", properties); err != nil {
		log.Printf("[recommendation] Analytics error: %v", err)
	}

	return nil
}

func (s *AnalyticsSideEffect) avgScore(candidates []*MemoryCandidate) float64 {
	if len(candidates) == 0 {
		return 0
	}
	var total float64
	for _, c := range candidates {
		total += c.Score
	}
	return total / float64(len(candidates))
}

func (s *AnalyticsSideEffect) topScore(candidates []*MemoryCandidate) float64 {
	if len(candidates) == 0 {
		return 0
	}
	max := candidates[0].Score
	for _, c := range candidates[1:] {
		if c.Score > max {
			max = c.Score
		}
	}
	return max
}

func (s *AnalyticsSideEffect) uniqueAuthors(candidates []*MemoryCandidate) int {
	seen := make(map[string]bool)
	for _, c := range candidates {
		authorID, _ := c.Metadata["author_id"].(string)
		if authorID != "" {
			seen[authorID] = true
		}
	}
	return len(seen)
}

// ServedTrackingSideEffect records which memories were served to the user.
type ServedTrackingSideEffect struct {
	tracker ServedTracker
}

// ServedTracker records served memories.
type ServedTracker interface {
	RecordServed(ctx context.Context, userID string, agentID string, memoryIDs []string) error
}

// NewServedTrackingSideEffect creates a served tracking side effect.
func NewServedTrackingSideEffect(tracker ServedTracker) *ServedTrackingSideEffect {
	return &ServedTrackingSideEffect{tracker: tracker}
}

func (s *ServedTrackingSideEffect) Name() string {
	return "served_tracking"
}

func (s *ServedTrackingSideEffect) Execute(ctx context.Context, query *QueryContext, candidates []*MemoryCandidate) error {
	if s.tracker == nil {
		return nil
	}

	ids := make([]string, 0, len(candidates))
	for _, c := range candidates {
		ids = append(ids, c.ID)
	}

	return s.tracker.RecordServed(ctx, query.UserID, query.AgentID, ids)
}
