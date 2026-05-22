package temporal

import (
	"math"
	"sort"
	"strings"
	"time"

	"agent-memory/internal/memory/types"
)

type ReasoningConfig struct {
	Enabled       bool
	HalfLifeDays  float64
	StateKeyBoost float64
	CurrentBoost  float64
}

func DefaultConfig() ReasoningConfig {
	return ReasoningConfig{
		Enabled:       true,
		HalfLifeDays:  7.0,
		StateKeyBoost: 1.3,
		CurrentBoost:  1.2,
	}
}

type TemporalScorer struct {
	config ReasoningConfig
	now    func() time.Time
}

func NewTemporalScorer(cfg ReasoningConfig) *TemporalScorer {
	return &TemporalScorer{
		config: cfg,
		now:    time.Now,
	}
}

func (ts *TemporalScorer) SetNowFunc(fn func() time.Time) {
	ts.now = fn
}

type TemporalScore struct {
	Context       types.TemporalContext
	RecencyScore  float64
	StateKeyScore float64
	FinalBoost    float64
}

func (ts *TemporalScorer) ScoreResult(r *types.MemoryResult, timeRef *time.Time) TemporalScore {
	if r.Metadata == nil {
		return TemporalScore{
			Context:      types.TemporalCurrent,
			RecencyScore: 1.0,
			FinalBoost:   1.0,
		}
	}

	mem := r.Metadata
	now := ts.now()
	if timeRef != nil {
		now = *timeRef
	}

	ctx := ts.classifyTemporal(mem, now)
	recency := ts.computeRecency(mem, now)
	stateBoost := ts.computeStateKeyBoost(mem)

	finalBoost := recency * stateBoost

	switch ctx {
	case types.TemporalCurrent:
		finalBoost *= ts.config.CurrentBoost
	case types.TemporalUpcoming:
		finalBoost *= 1.1
	case types.TemporalHistorical:
		finalBoost *= 0.95
	}

	if finalBoost < 0.1 {
		finalBoost = 0.1
	}

	return TemporalScore{
		Context:       ctx,
		RecencyScore:  recency,
		StateKeyScore: stateBoost,
		FinalBoost:    finalBoost,
	}
}

func (ts *TemporalScorer) classifyTemporal(mem *types.Memory, now time.Time) types.TemporalContext {
	if mem.ExpirationDate != nil && mem.ExpirationDate.After(now) {
		age := now.Sub(mem.CreatedAt)
		if age < 24*time.Hour {
			return types.TemporalUpcoming
		}
	}

	hoursSinceUpdate := now.Sub(mem.UpdatedAt).Hours()
	if hoursSinceUpdate < 48 {
		return types.TemporalCurrent
	}

	return types.TemporalHistorical
}

func (ts *TemporalScorer) computeRecency(mem *types.Memory, now time.Time) float64 {
	hoursSinceUpdate := now.Sub(mem.UpdatedAt).Hours()
	if hoursSinceUpdate < 0 {
		hoursSinceUpdate = 0
	}
	daysSinceUpdate := hoursSinceUpdate / 24.0

	lambda := math.Ln2 / ts.config.HalfLifeDays
	recency := math.Exp(-lambda * daysSinceUpdate)

	if mem.LastAccessed != nil {
		hoursSinceAccess := now.Sub(*mem.LastAccessed).Hours()
		if hoursSinceAccess < 24 {
			accessBoost := 1.0 + 0.2*(1.0-hoursSinceAccess/24.0)
			recency *= accessBoost
		}
	}

	return recency
}

func (ts *TemporalScorer) computeStateKeyBoost(mem *types.Memory) float64 {
	if mem.StateKey == "" {
		return 1.0
	}
	return ts.config.StateKeyBoost
}

type StateKeyResolver struct{}

func NewStateKeyResolver() *StateKeyResolver {
	return &StateKeyResolver{}
}

type StateKeyEntry struct {
	Key      string
	Memory   *types.Memory
	IsLatest bool
}

func (r *StateKeyResolver) Resolve(memories []*types.Memory) map[string]*types.Memory {
	stateMap := make(map[string]*types.Memory)

	for _, mem := range memories {
		if mem.StateKey == "" {
			continue
		}

		existing, found := stateMap[mem.StateKey]
		if !found || mem.UpdatedAt.After(existing.UpdatedAt) {
			stateMap[mem.StateKey] = mem
		}
	}

	return stateMap
}

func (r *StateKeyResolver) SuppressOutdated(results []types.MemoryResult, latest map[string]*types.Memory) []types.MemoryResult {
	filtered := make([]types.MemoryResult, 0, len(results))

	for _, r := range results {
		if r.Metadata == nil || r.Metadata.StateKey == "" {
			filtered = append(filtered, r)
			continue
		}

		latestMem, exists := latest[r.Metadata.StateKey]
		if !exists || latestMem.ID == r.Metadata.ID {
			filtered = append(filtered, r)
			continue
		}

		if r.Metadata.UpdatedAt.Equal(latestMem.UpdatedAt) {
			filtered = append(filtered, r)
		}
	}

	return filtered
}

var temporalIndicators = []string{"today", "yesterday", "this week", "last week", "this month", "last month",
	"recently", "currently", "now", "latest", "newest", "current", "old", "previous", "before", "after",
	"upcoming", "next", "future", "past", "ago", "since", "until", "tomorrow", "monday", "tuesday",
	"wednesday", "thursday", "friday", "saturday", "sunday", "january", "february", "march", "april",
	"may", "june", "july", "august", "september", "october", "november", "december"}

func HasTemporalSignals(query string) bool {
	lower := strings.ToLower(query)
	for _, indicator := range temporalIndicators {
		if strings.Contains(lower, indicator) {
			return true
		}
	}
	return false
}

func ApplyTemporalScoring(results []types.MemoryResult, scorer *TemporalScorer, timeRef *time.Time) []types.MemoryResult {
	for i := range results {
		score := scorer.ScoreResult(&results[i], timeRef)
		results[i].TemporalContext = score.Context
		results[i].Score = float32(float64(results[i].Score) * score.FinalBoost)
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	return results
}
