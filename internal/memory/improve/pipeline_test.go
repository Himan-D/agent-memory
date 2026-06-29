package improve

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestPipeline_DefaultStages(t *testing.T) {
	p := NewPipeline()
	got := p.Stages()
	want := []string{
		"feedback_weights",
		"persist_sessions",
		"distill_sessions",
		"memify_enrichment",
		"global_context_index",
		"sync_to_cache",
	}
	if len(got) != len(want) {
		t.Fatalf("expected %d stages, got %d (%v)", len(want), len(got), got)
	}
	for i, w := range want {
		if got[i] != w {
			t.Fatalf("stage[%d] = %s want %s", i, got[i], w)
		}
	}
}

func TestPipeline_RunRecordsAllStages(t *testing.T) {
	p := NewPipeline()
	out := p.Run(context.Background(), Input{UserID: "u1", SessionIDs: []string{"s1"}})
	if len(out.Stages) < 4 {
		t.Fatalf("expected at least 4 stages to run by default, got %d", len(out.Stages))
	}
	if _, ok := out.Stages["feedback_weights"]; !ok {
		t.Fatal("missing feedback_weights stage")
	}
	if _, ok := out.Stages["distill_sessions"]; !ok {
		t.Fatal("missing distill_sessions stage")
	}
}

func TestPipeline_FailOpenPerStage(t *testing.T) {
	boom := errors.New("boom")
	p := NewPipeline().WithStage(failingStage{err: boom})
	ran := false
	p.WithStage(passthroughStage{name: "after", fn: func() { ran = true }})
	out := p.Run(context.Background(), Input{UserID: "u1"})
	if _, ok := out.Stages["after"]; !ok {
		t.Fatal("after stage should still run")
	}
	if !ran {
		t.Fatal("after stage was not invoked despite fail-open")
	}
}

func TestPipeline_SkipsDisabledStages(t *testing.T) {
	p := NewPipeline()
	out := p.Run(context.Background(), Input{UserID: "u1"})
	if _, ok := out.Stages["global_context_index"]; ok {
		t.Fatal("global_context_index should not run when BuildGlobalContext=false")
	}
	if _, ok := out.Stages["sync_to_cache"]; ok {
		t.Fatal("sync_to_cache should not run when RunSyncToCache=false")
	}
}

func TestPipeline_RunWithGlobalContextEnabled(t *testing.T) {
	p := NewPipeline()
	out := p.Run(context.Background(), Input{
		UserID:             "u1",
		BuildGlobalContext: true,
		RunSyncToCache:     true,
	})
	if _, ok := out.Stages["global_context_index"]; !ok {
		t.Fatal("global_context_index should run when BuildGlobalContext=true")
	}
	if _, ok := out.Stages["sync_to_cache"]; !ok {
		t.Fatal("sync_to_cache should run when RunSyncToCache=true")
	}
}

func TestPipeline_WithStage_ReplacesExisting(t *testing.T) {
	p := NewPipeline()
	p.WithStage(FeedbackWeights{}) // should replace, not append
	if got := p.Stages(); len(got) != 6 {
		t.Fatalf("expected 6 stages after replace, got %d (%v)", len(got), got)
	}
}

func TestPipeline_DurationIsRecorded(t *testing.T) {
	p := NewPipeline()
	start := time.Now()
	out := p.Run(context.Background(), Input{UserID: "u1"})
	if out.DurationMs < 0 {
		t.Fatalf("expected non-negative duration, got %d", out.DurationMs)
	}
	if time.Since(start) < 0 {
		t.Fatal("clock went backwards")
	}
}

// failingStage always returns the configured error.
type failingStage struct{ err error }

func (f failingStage) Name() string { return "fail" }
func (f failingStage) Run(ctx context.Context, in Input) (StageResult, error) {
	return StageResult{Name: "fail", Started: time.Now(), Ended: time.Now()}, f.err
}

// passthroughStage invokes fn when Run.
type passthroughStage struct {
	name string
	fn   func()
}

func (p passthroughStage) Name() string { return p.name }
func (p passthroughStage) Run(ctx context.Context, in Input) (StageResult, error) {
	if p.fn != nil {
		p.fn()
	}
	return StageResult{Name: p.name, Started: time.Now(), Ended: time.Now()}, nil
}
