package recommendation

import (
	"context"
)

// AuthorDiversityScorer attenuates scores for candidates from authors with many high-ranking memories.
// This implements X's "Author Diversity Scorer" pattern to ensure feed diversity.
type AuthorDiversityScorer struct {
	attenuationFactor float64 // How much to penalize repeated authors (0.0 = no penalty, 1.0 = full penalty)
	maxPerAuthor      int     // Soft limit before attenuation kicks in
}

// NewAuthorDiversityScorer creates an author diversity scorer.
func NewAuthorDiversityScorer() *AuthorDiversityScorer {
	return &AuthorDiversityScorer{
		attenuationFactor: 0.15, // 15% score reduction per excess candidate
		maxPerAuthor:      3,    // Start attenuating after 3 memories from same author
	}
}

func (s *AuthorDiversityScorer) Name() string {
	return "author_diversity_scorer"
}

func (s *AuthorDiversityScorer) Score(ctx context.Context, query *QueryContext, candidate *MemoryCandidate) error {
	// This scorer runs after primary scoring and adjusts for diversity.
	// It needs all candidates to count per-author, so it's a no-op at individual level.
	// The actual diversity logic is handled by the TopKSelector's author limit.
	return nil
}

// OONScorer adjusts scores for out-of-network content to balance with in-network.
// X uses this to boost out-of-network content to ensure discovery.
type OONScorer struct {
	oonBoost float64 // Boost factor for out-of-network memories
}

// NewOONScorer creates an out-of-network booster.
func NewOONScorer(oonBoost float64) *OONScorer {
	if oonBoost == 0 {
		oonBoost = 1.05 // 5% boost for OON content
	}
	return &OONScorer{oonBoost: oonBoost}
}

func (s *OONScorer) Name() string {
	return "oon_scorer"
}

func (s *OONScorer) Score(ctx context.Context, query *QueryContext, candidate *MemoryCandidate) error {
	source, _ := candidate.Metadata["source"].(string)
	followingSet := make(map[string]bool)
	for _, fid := range query.FollowingIDs {
		followingSet[fid] = true
	}

	// If source is NOT from a followed entity, it's out-of-network
	if !followingSet[source] && source != "" {
		candidate.Score *= s.oonBoost
		if candidate.Score > 1.0 {
			candidate.Score = 1.0
		}
		candidate.Metadata["is_oon"] = true
	}

	return nil
}

// RecencyDecayScorer applies time-based decay to scores (older memories get lower scores).
type RecencyDecayScorer struct {
	halfLifeHours float64 // Half-life for score decay
}

// NewRecencyDecayScorer creates a recency decay scorer.
func NewRecencyDecayScorer(halfLifeHours float64) *RecencyDecayScorer {
	if halfLifeHours == 0 {
		halfLifeHours = 24 // 24 hour half-life
	}
	return &RecencyDecayScorer{halfLifeHours: halfLifeHours}
}

func (s *RecencyDecayScorer) Name() string {
	return "recency_decay_scorer"
}

func (s *RecencyDecayScorer) Score(ctx context.Context, query *QueryContext, candidate *MemoryCandidate) error {
	recency, ok := candidate.Metadata["recency"].(float64)
	if !ok {
		return nil
	}

	// Apply exponential decay based on recency
	// Newer memories (recency close to 1.0) keep their score
	// Older memories (recency close to 0.0) get attenuated
	decayFactor := recency*0.8 + 0.2 // Scale to [0.2, 1.0] range
	candidate.Score *= decayFactor

	return nil
}
