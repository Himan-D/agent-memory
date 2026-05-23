package evaluation

import (
	"math"
	"time"
)

// FAMAScorer implements Forgetting-Aware Memory Accuracy.
// Based on paper arXiv:2604.20006.
// Penalizes answers derived from stale/outdated memories.
type FAMAScorer struct {
	StalenessThreshold time.Duration // memories older than this get penalized
	DecayRate          float64       // how fast staleness penalty grows (default 0.5)
}

// FAMAInput represents a single retrieval result for batch scoring.
type FAMAInput struct {
	Correct        bool
	MemoryAge      time.Duration
	ValidityWindow time.Duration
	WasSuperseded  bool
}

// NewFAMAScorer creates a FAMAScorer with sensible defaults.
// StalenessThreshold = 7 days, DecayRate = 0.5.
func NewFAMAScorer() *FAMAScorer {
	return &FAMAScorer{
		StalenessThreshold: 7 * 24 * time.Hour,
		DecayRate:          0.5,
	}
}

// Score computes FAMA-adjusted accuracy for a single retrieval.
//
// correctness: standard accuracy (0.0 or 1.0)
// memoryAge: how old the source memory is
// memoryValidityWindow: how long the memory type typically stays valid
//
// The formula applies an exponential staleness penalty:
//
//	FAMA = correctness * freshness_multiplier
//	freshness_multiplier = exp(-decayRate * max(0, staleness_ratio - 1))
//	staleness_ratio = memoryAge / memoryValidityWindow
//
// A memory within its validity window gets no penalty (multiplier = 1.0).
// Beyond the window, the penalty grows exponentially.
func (f *FAMAScorer) Score(correctness float64, memoryAge, memoryValidityWindow time.Duration) float64 {
	if memoryValidityWindow <= 0 {
		memoryValidityWindow = f.StalenessThreshold
	}

	stalenessRatio := float64(memoryAge) / float64(memoryValidityWindow)

	// No penalty for memories within their validity window.
	excess := stalenessRatio - 1.0
	if excess < 0 {
		excess = 0
	}

	freshness := math.Exp(-f.DecayRate * excess)

	return correctness * freshness
}

// BatchScore scores a batch of retrieval results and returns the average FAMA score.
func (f *FAMAScorer) BatchScore(results []FAMAInput) float64 {
	if len(results) == 0 {
		return 0.0
	}

	var total float64
	for _, r := range results {
		correctness := 0.0
		if r.Correct {
			correctness = 1.0
		}

		score := f.Score(correctness, r.MemoryAge, r.ValidityWindow)

		// Additional penalty for superseded memories: halve the score.
		if r.WasSuperseded {
			score *= 0.5
		}

		total += score
	}

	return total / float64(len(results))
}
