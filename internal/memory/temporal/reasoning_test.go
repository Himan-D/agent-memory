package temporal

import (
	"testing"
	"time"

	"agent-memory/internal/memory/types"
)

func TestTemporalScorer_ScoreResult(t *testing.T) {
	now := time.Now()
	cfg := DefaultConfig()
	scorer := NewTemporalScorer(cfg)
	scorer.SetNowFunc(func() time.Time { return now })

	mem := &types.Memory{
		ID:        "mem1",
		Content:   "user prefers dark mode",
		UpdatedAt: now,
		CreatedAt: now,
	}

	result := &types.MemoryResult{
		Score:    0.9,
		Metadata: mem,
	}

	score := scorer.ScoreResult(result, nil)
	if score.Context != types.TemporalCurrent {
		t.Errorf("expected current temporal context, got %s", score.Context)
	}
	if score.FinalBoost < 1.0 {
		t.Errorf("expected boost >= 1.0 for current memory, got %f", score.FinalBoost)
	}
}

func TestTemporalScorer_HistoricalMemory(t *testing.T) {
	now := time.Now()
	cfg := DefaultConfig()
	scorer := NewTemporalScorer(cfg)
	scorer.SetNowFunc(func() time.Time { return now })

	mem := &types.Memory{
		ID:        "mem2",
		Content:   "user started new job",
		UpdatedAt: now.Add(-72 * time.Hour),
		CreatedAt: now.Add(-72 * time.Hour),
	}

	result := &types.MemoryResult{
		Score:    0.85,
		Metadata: mem,
	}

	score := scorer.ScoreResult(result, nil)
	if score.Context != types.TemporalHistorical {
		t.Errorf("expected historical temporal context, got %s", score.Context)
	}
	if score.RecencyScore >= 1.0 {
		t.Errorf("expected recency < 1.0 for old memory, got %f", score.RecencyScore)
	}
}

func TestTemporalScorer_UpcomingMemory(t *testing.T) {
	now := time.Now()
	cfg := DefaultConfig()
	scorer := NewTemporalScorer(cfg)
	scorer.SetNowFunc(func() time.Time { return now })

	future := now.Add(24 * time.Hour)
	mem := &types.Memory{
		ID:             "mem3",
		Content:        "meeting scheduled",
		UpdatedAt:      now,
		CreatedAt:      now,
		ExpirationDate: &future,
	}

	result := &types.MemoryResult{
		Score:    0.8,
		Metadata: mem,
	}

	score := scorer.ScoreResult(result, nil)
	if score.Context != types.TemporalUpcoming {
		t.Errorf("expected upcoming temporal context, got %s", score.Context)
	}
}

func TestTemporalScorer_NilMetadata(t *testing.T) {
	cfg := DefaultConfig()
	scorer := NewTemporalScorer(cfg)

	result := &types.MemoryResult{
		Score:    0.9,
		Metadata: nil,
	}

	score := scorer.ScoreResult(result, nil)
	if score.Context != types.TemporalCurrent {
		t.Errorf("expected current for nil metadata, got %s", score.Context)
	}
	if score.FinalBoost != 1.0 {
		t.Errorf("expected 1.0 boost for nil metadata, got %f", score.FinalBoost)
	}
}

func TestStateKeyResolver_Resolve(t *testing.T) {
	now := time.Now()
	resolver := NewStateKeyResolver()

	memories := []*types.Memory{
		{ID: "mem1", StateKey: "job_title", Content: "Software Engineer", UpdatedAt: now.Add(-48 * time.Hour)},
		{ID: "mem2", StateKey: "job_title", Content: "Senior Engineer", UpdatedAt: now},
		{ID: "mem3", StateKey: "location", Content: "New York", UpdatedAt: now.Add(-24 * time.Hour)},
		{ID: "mem4", StateKey: "", Content: "No state key", UpdatedAt: now},
	}

	latest := resolver.Resolve(memories)

	if len(latest) != 2 {
		t.Fatalf("expected 2 state keys, got %d", len(latest))
	}
	if latest["job_title"].ID != "mem2" {
		t.Errorf("expected mem2 as latest job_title, got %s", latest["job_title"].ID)
	}
	if latest["location"].ID != "mem3" {
		t.Errorf("expected mem3 as latest location, got %s", latest["location"].ID)
	}
}

func TestStateKeyResolver_SuppressOutdated(t *testing.T) {
	now := time.Now()
	resolver := NewStateKeyResolver()

	memories := []*types.Memory{
		{ID: "mem1", StateKey: "job_title", Content: "Software Engineer", UpdatedAt: now.Add(-48 * time.Hour)},
		{ID: "mem2", StateKey: "job_title", Content: "Senior Engineer", UpdatedAt: now},
		{ID: "mem3", StateKey: "", Content: "No state key", UpdatedAt: now},
	}

	latest := resolver.Resolve(memories)

	results := []types.MemoryResult{
		{Score: 0.9, Metadata: memories[0]},
		{Score: 0.85, Metadata: memories[1]},
		{Score: 0.8, Metadata: memories[2]},
	}

	filtered := resolver.SuppressOutdated(results, latest)
	if len(filtered) != 2 {
		t.Fatalf("expected 2 results after suppression, got %d", len(filtered))
	}
	for _, r := range filtered {
		if r.Metadata.StateKey == "job_title" && r.Metadata.ID == "mem1" {
			t.Error("outdated job_title memory should have been suppressed")
		}
	}
}

func TestHasTemporalSignals(t *testing.T) {
	tests := []struct {
		query    string
		expected bool
	}{
		{"what did I work on this week", true},
		{"what is my current project", true},
		{"show me the latest results", true},
		{"hello world", false},
		{"find documents about APIs", false},
		{"what changed recently", true},
		{"upcoming events", true},
	}

	for _, tt := range tests {
		result := HasTemporalSignals(tt.query)
		if result != tt.expected {
			t.Errorf("HasTemporalSignals(%q) = %v, expected %v", tt.query, result, tt.expected)
		}
	}
}

func TestApplyTemporalScoring(t *testing.T) {
	now := time.Now()
	cfg := DefaultConfig()
	scorer := NewTemporalScorer(cfg)
	scorer.SetNowFunc(func() time.Time { return now })

	results := []types.MemoryResult{
		{Score: 0.8, Metadata: &types.Memory{ID: "old", UpdatedAt: now.Add(-72 * time.Hour), CreatedAt: now.Add(-72 * time.Hour)}},
		{Score: 0.7, Metadata: &types.Memory{ID: "new", UpdatedAt: now, CreatedAt: now, StateKey: "preference"}},
	}

	scored := ApplyTemporalScoring(results, scorer, nil)

	if scored[0].Metadata.ID != "new" {
		t.Errorf("expected 'new' memory to rank first after temporal scoring, got %s", scored[0].Metadata.ID)
	}
	if scored[0].TemporalContext == "" {
		t.Error("expected temporal context to be set")
	}
}
