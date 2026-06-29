package session

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestDistiller_NoTurnsProducesEmpty(t *testing.T) {
	m := NewManager(NewInMemoryStore())
	d := NewDistiller(m, DistillOptions{
		Curator: func(ctx context.Context, _ string) ([]ProposedLesson, error) {
			t.Fatal("curator should not be called with no turns")
			return nil, nil
		},
		Writer: acceptAllWriter,
	})
	res := d.Distill(context.Background(), "u1", "missing")
	if res.Proposed != 0 || res.Accepted != 0 {
		t.Fatalf("unexpected counts: %+v", res)
	}
}

func TestDistiller_HappyPath(t *testing.T) {
	m := NewManager(NewInMemoryStore())
	ctx := context.Background()

	turns := []time.Time{time.Unix(1, 0), time.Unix(2, 0)}
	for i, ts := range turns {
		_, _ = m.AddQATurn(ctx, "u1", "s1", "Q"+itoaS(i), "A"+itoaS(i), "", nil, nil)
		_ = ts
	}

	d := NewDistiller(m, DistillOptions{
		Curator: func(ctx context.Context, batch string) ([]ProposedLesson, error) {
			return []ProposedLesson{{WorkingStatement: "user asked about things"}}, nil
		},
		Writer: acceptAllWriter,
	})
	res := d.Distill(ctx, "u1", "s1")
	if res.Proposed == 0 {
		t.Fatalf("expected at least 1 proposal, got %+v", res)
	}
	if res.Accepted == 0 {
		t.Fatalf("expected at least 1 accepted lesson, got %+v", res)
	}
	if len(res.Errors) != 0 {
		t.Fatalf("unexpected errors: %v", res.Errors)
	}
}

func TestDistiller_FailOpenPerBatch(t *testing.T) {
	m := NewManager(NewInMemoryStore())
	ctx := context.Background()
	// Add enough turns to produce multiple batches.
	for i := 0; i < 4; i++ {
		_, _ = m.AddQATurn(ctx, "u1", "s1", "Q"+itoaS(i), "A"+itoaS(i), "", nil, nil)
	}

	// Force 2 batches by capping blocks per batch at 2.
	b := NewBatcher()
	b.BlocksPerBatch = 2

	calls := 0
	d := NewDistiller(m, DistillOptions{
		Batcher: b,
		Curator: func(ctx context.Context, batch string) ([]ProposedLesson, error) {
			calls++
			if calls == 1 {
				return nil, context.DeadlineExceeded
			}
			return []ProposedLesson{{WorkingStatement: "ok"}}, nil
		},
		Writer: acceptAllWriter,
	})
	res := d.Distill(ctx, "u1", "s1")
	if res.Accepted == 0 {
		t.Fatalf("expected at least one acceptance despite batch 1 failure, got %+v", res)
	}
	if len(res.Errors) == 0 {
		t.Fatalf("expected at least one recorded error")
	}
}

func TestDistiller_Rejection(t *testing.T) {
	m := NewManager(NewInMemoryStore())
	ctx := context.Background()
	_, _ = m.AddQATurn(ctx, "u1", "s1", "Q", "A", "", nil, nil)

	writer := func(ctx context.Context, p ProposedLesson, _ []string, _ []string, _ []string) (WrittenLesson, error) {
		return WrittenLesson{Accept: false, Reason: "already_known"}, nil
	}
	d := NewDistiller(m, DistillOptions{
		Curator: func(ctx context.Context, _ string) ([]ProposedLesson, error) {
			return []ProposedLesson{{WorkingStatement: "x"}}, nil
		},
		Writer: writer,
	})
	res := d.Distill(ctx, "u1", "s1")
	if res.Rejected != 1 || res.Accepted != 0 {
		t.Fatalf("unexpected counts: %+v", res)
	}
}

func TestRenderLesson_WithWhy(t *testing.T) {
	l := DistilledLesson{
		SessionID:   "s1",
		Statement:   "User prefers dark mode.",
		WhyLearned:   "User said it explicitly",
		DistilledOn: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
	}
	got := RenderLesson(l)
	if !strings.Contains(got, "User prefers dark mode.") {
		t.Fatalf("missing statement: %q", got)
	}
	if !strings.Contains(got, "User said it explicitly.") {
		t.Fatalf("missing why-learned: %q", got)
	}
	if !strings.Contains(got, "# Session learning — 2026-01-02") {
		t.Fatalf("missing title: %q", got)
	}
}

func TestRenderLesson_NoWhy(t *testing.T) {
	l := DistilledLesson{SessionID: "s1", Statement: "X."}
	got := RenderLesson(l)
	if strings.Contains(got, "()") {
		t.Fatalf("expected no empty parens: %q", got)
	}
}
