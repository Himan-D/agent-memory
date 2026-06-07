package decay

import (
	"testing"
	"time"

	"agent-memory/internal/memory/types"
)

func TestDecayScorer_RecentlyAccessed(t *testing.T) {
	now := time.Now()
	cfg := DefaultConfig()
	scorer := NewDecayScorer(cfg)
	scorer.SetNowFunc(func() time.Time { return now })

	lastAccessed := now.Add(-1 * time.Hour)
	mem := &types.Memory{
		ID:           "mem1",
		UpdatedAt:    now,
		AccessCount:  10,
		LastAccessed: &lastAccessed,
	}

	result := scorer.Compute(mem)
	if result.Multiplier < 1.0 {
		t.Errorf("expected multiplier >= 1.0 for recently accessed memory, got %f", result.Multiplier)
	}
	if result.Multiplier > cfg.MaxMultiplier {
		t.Errorf("expected multiplier <= %f, got %f", cfg.MaxMultiplier, result.Multiplier)
	}
}

func TestDecayScorer_IdleMemory(t *testing.T) {
	now := time.Now()
	cfg := DefaultConfig()
	scorer := NewDecayScorer(cfg)
	scorer.SetNowFunc(func() time.Time { return now })

	mem := &types.Memory{
		ID:          "mem2",
		UpdatedAt:   now.Add(-30 * 24 * time.Hour),
		AccessCount: 0,
	}

	result := scorer.Compute(mem)
	if result.Multiplier > 0.5 {
		t.Errorf("expected low multiplier for 30-day idle memory, got %f", result.Multiplier)
	}
	if result.Multiplier < cfg.MinMultiplier {
		t.Errorf("expected multiplier >= %f, got %f", cfg.MinMultiplier, result.Multiplier)
	}
}

func TestDecayScorer_ClampsToMax(t *testing.T) {
	now := time.Now()
	cfg := DefaultConfig()
	cfg.MaxMultiplier = 1.5
	scorer := NewDecayScorer(cfg)
	scorer.SetNowFunc(func() time.Time { return now })

	lastAccessed := now.Add(-10 * time.Minute)
	mem := &types.Memory{
		ID:           "mem3",
		UpdatedAt:    now,
		AccessCount:  1000,
		LastAccessed: &lastAccessed,
	}

	result := scorer.Compute(mem)
	if result.Multiplier > 1.5 {
		t.Errorf("expected multiplier <= 1.5, got %f", result.Multiplier)
	}
}

func TestDecayScorer_ClampsToMin(t *testing.T) {
	now := time.Now()
	cfg := DefaultConfig()
	cfg.MinMultiplier = 0.3
	scorer := NewDecayScorer(cfg)
	scorer.SetNowFunc(func() time.Time { return now })

	mem := &types.Memory{
		ID:          "mem4",
		UpdatedAt:   now.Add(-365 * 24 * time.Hour),
		AccessCount: 0,
	}

	result := scorer.Compute(mem)
	if result.Multiplier < 0.3 {
		t.Errorf("expected multiplier >= 0.3, got %f", result.Multiplier)
	}
}

func TestDecayScorer_Disabled(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = false
	scorer := NewDecayScorer(cfg)

	mem := &types.Memory{
		ID:        "mem5",
		UpdatedAt: time.Now().Add(-365 * 24 * time.Hour),
	}

	result := scorer.Compute(mem)
	if result.Multiplier != 1.0 {
		t.Errorf("expected 1.0 multiplier when disabled, got %f", result.Multiplier)
	}
}

func TestApplyDecay(t *testing.T) {
	now := time.Now()
	cfg := DefaultConfig()
	scorer := NewDecayScorer(cfg)
	scorer.SetNowFunc(func() time.Time { return now })

	lastAccessed := now.Add(-1 * time.Hour)
	results := []types.MemoryResult{
		{Score: 0.9, Metadata: &types.Memory{ID: "recent", UpdatedAt: now, AccessCount: 10, LastAccessed: &lastAccessed}},
		{Score: 0.9, Metadata: &types.Memory{ID: "idle", UpdatedAt: now.Add(-30 * 24 * time.Hour), AccessCount: 0}},
	}

	decayed := ApplyDecay(results, scorer)

	if decayed[0].Metadata.ID != "recent" {
		t.Errorf("expected 'recent' memory to rank first after decay, got %s", decayed[0].Metadata.ID)
	}
	if decayed[0].DecayMultiplier == 0 {
		t.Error("expected decay multiplier to be set")
	}
}

func TestApplyDecay_NilScorer(t *testing.T) {
	results := []types.MemoryResult{
		{Score: 0.9, Metadata: &types.Memory{ID: "mem1"}},
	}
	decayed := ApplyDecay(results, nil)
	if len(decayed) != 1 {
		t.Error("expected results to pass through with nil scorer")
	}
}

func TestDecayScorer_NilMetadata(t *testing.T) {
	cfg := DefaultConfig()
	scorer := NewDecayScorer(cfg)

	results := []types.MemoryResult{
		{Score: 0.9, Metadata: nil},
	}

	decayed := ApplyDecay(results, scorer)
	if decayed[0].Score != 0.9 {
		t.Errorf("expected score unchanged for nil metadata, got %f", decayed[0].Score)
	}
	if decayed[0].DecayMultiplier != 1.0 {
		t.Errorf("expected 1.0 multiplier for nil metadata, got %f", decayed[0].DecayMultiplier)
	}
}
