package decay

import (
	"math"
	"time"

	"agent-memory/internal/memory/types"
)

type DecayConfig struct {
	Enabled           bool
	MinMultiplier     float64
	MaxMultiplier     float64
	ActiveWindow      time.Duration
	DecayHalfLife     time.Duration
	AccessBoostFactor float64
}

func DefaultConfig() DecayConfig {
	return DecayConfig{
		Enabled:           true,
		MinMultiplier:     0.3,
		MaxMultiplier:     1.5,
		ActiveWindow:      24 * time.Hour,
		DecayHalfLife:     7 * 24 * time.Hour,
		AccessBoostFactor: 0.1,
	}
}

type DecayScorer struct {
	config DecayConfig
	now    func() time.Time
}

func NewDecayScorer(cfg DecayConfig) *DecayScorer {
	return &DecayScorer{
		config: cfg,
		now:    time.Now,
	}
}

func (ds *DecayScorer) SetNowFunc(fn func() time.Time) {
	ds.now = fn
}

type DecayResult struct {
	Multiplier float64
	Reason     string
}

func (ds *DecayScorer) Compute(mem *types.Memory) DecayResult {
	if !ds.config.Enabled {
		return DecayResult{Multiplier: 1.0, Reason: "decay_disabled"}
	}

	now := ds.now()
	baseDecay := ds.computeTimeDecay(mem, now)
	accessBoost := ds.computeAccessBoost(mem, now)
	recencyBoost := ds.computeRecencyBoost(mem, now)

	rawMultiplier := baseDecay + accessBoost + recencyBoost
	clamped := ds.clamp(rawMultiplier)

	reason := "time_decay"
	if accessBoost > 0 {
		reason = "time_decay+access_boost"
	}
	if recencyBoost > 0 {
		reason = "time_decay+recency_boost"
	}

	return DecayResult{
		Multiplier: clamped,
		Reason:     reason,
	}
}

func (ds *DecayScorer) computeTimeDecay(mem *types.Memory, now time.Time) float64 {
	hoursSinceUpdate := now.Sub(mem.UpdatedAt).Hours()
	if hoursSinceUpdate < 0 {
		return 1.0
	}

	halfLifeHours := ds.config.DecayHalfLife.Hours()
	if halfLifeHours <= 0 {
		halfLifeHours = 168
	}
	lambda := math.Ln2 / halfLifeHours
	decay := math.Exp(-lambda * hoursSinceUpdate)

	return ds.config.MinMultiplier + (1.0-ds.config.MinMultiplier)*decay
}

func (ds *DecayScorer) computeAccessBoost(mem *types.Memory, now time.Time) float64 {
	if mem.AccessCount <= 0 {
		return 0
	}

	logAccess := math.Log2(float64(mem.AccessCount) + 1)
	boost := logAccess * ds.config.AccessBoostFactor

	return math.Min(boost, 0.3)
}

func (ds *DecayScorer) computeRecencyBoost(mem *types.Memory, now time.Time) float64 {
	if mem.LastAccessed == nil {
		return 0
	}

	hoursSinceAccess := now.Sub(*mem.LastAccessed).Hours()
	if hoursSinceAccess < 0 {
		hoursSinceAccess = 0
	}

	activeWindowHours := ds.config.ActiveWindow.Hours()
	if hoursSinceAccess < activeWindowHours {
		ratio := 1.0 - (hoursSinceAccess / activeWindowHours)
		return ratio * 0.5
	}

	return 0
}

func (ds *DecayScorer) clamp(multiplier float64) float64 {
	if multiplier < ds.config.MinMultiplier {
		return ds.config.MinMultiplier
	}
	if multiplier > ds.config.MaxMultiplier {
		return ds.config.MaxMultiplier
	}
	return multiplier
}

func ApplyDecay(results []types.MemoryResult, scorer *DecayScorer) []types.MemoryResult {
	if scorer == nil {
		return results
	}

	for i := range results {
		if results[i].Metadata == nil {
			results[i].DecayMultiplier = 1.0
			continue
		}

		decay := scorer.Compute(results[i].Metadata)
		results[i].DecayMultiplier = decay.Multiplier
		results[i].Metadata.DecayMultiplier = decay.Multiplier
		results[i].Score = float32(float64(results[i].Score) * decay.Multiplier)
	}

	return results
}
