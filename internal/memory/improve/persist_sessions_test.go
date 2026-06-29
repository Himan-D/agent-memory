package improve

import (
	"context"
	"strings"
	"testing"
	"time"

	"agent-memory/internal/session"
)

func TestPersistSessions_NilStoreErrors(t *testing.T) {
	s := PersistSessions{Graph: &fakeGraph{}}
	if _, err := s.Run(context.Background(), Input{UserID: "u1", SessionIDs: []string{"s1"}}); err == nil {
		t.Fatal("expected error for nil store")
	}
}

func TestPersistSessions_NilGraphErrors(t *testing.T) {
	s := PersistSessions{Store: session.NewInMemoryStore()}
	if _, err := s.Run(context.Background(), Input{UserID: "u1", SessionIDs: []string{"s1"}}); err == nil {
		t.Fatal("expected error for nil graph")
	}
}

func TestPersistSessions_NoSessionsRunsEmpty(t *testing.T) {
	f := &fakeGraph{}
	s := PersistSessions{Store: session.NewInMemoryStore(), Graph: f}
	res, err := s.Run(context.Background(), Input{UserID: "u1"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Items != 0 {
		t.Fatalf("expected 0 turns persisted, got %d", res.Items)
	}
	if len(f.cyphers) != 0 {
		t.Fatalf("expected no cypher calls with no sessions, got %d", len(f.cyphers))
	}
}

func TestPersistSessions_WritesTurns(t *testing.T) {
	mgr := session.NewManager(session.NewInMemoryStore())
	ctx := context.Background()
	_, _ = mgr.AddQATurn(ctx, "u1", "s1", "Q1", "A1", "", nil, []string{"mem-1"})
	_, _ = mgr.AddQATurn(ctx, "u1", "s1", "Q2", "A2", "", nil, []string{"mem-2"})

	f := &fakeGraph{}
	s := PersistSessions{Store: mgr.Store(), Graph: f}
	res, err := s.Run(ctx, Input{UserID: "u1", SessionIDs: []string{"s1"}})
	if err != nil {
		t.Fatal(err)
	}
	if res.Items != 2 {
		t.Fatalf("expected 2 turns persisted, got %d", res.Items)
	}
	if len(f.cyphers) != 2 {
		t.Fatalf("expected 2 cypher calls, got %d", len(f.cyphers))
	}
	for i, c := range f.cyphers {
		if !strings.Contains(c, "MERGE (t:QATurn") {
			t.Fatalf("cypher[%d] missing QATurn MERGE: %s", i, c)
		}
		if !strings.Contains(c, "MERGE (t)-[:PART_OF]->(sess)") {
			t.Fatalf("cypher[%d] missing PART_OF: %s", i, c)
		}
	}
}

func TestPersistSessions_PropagatesGraphError(t *testing.T) {
	mgr := session.NewManager(session.NewInMemoryStore())
	ctx := context.Background()
	_, _ = mgr.AddQATurn(ctx, "u1", "s1", "Q", "A", "", nil, nil)
	f := &fakeGraph{err: errFake("boom")}
	s := PersistSessions{Store: mgr.Store(), Graph: f}
	if _, err := s.Run(ctx, Input{UserID: "u1", SessionIDs: []string{"s1"}}); err == nil {
		t.Fatal("expected error from graph failure")
	}
}

func TestPersistSessions_FeedbackScoreNilStaysNil(t *testing.T) {
	mgr := session.NewManager(session.NewInMemoryStore())
	ctx := context.Background()
	_, _ = mgr.AddQATurn(ctx, "u1", "s1", "Q", "A", "", nil, nil)
	f := &fakeGraph{}
	s := PersistSessions{Store: mgr.Store(), Graph: f}
	_, _ = s.Run(ctx, Input{UserID: "u1", SessionIDs: []string{"s1"}})
	// Verify the params map carries a nil for feedback_score (not 0).
	if v, ok := f.params[0]["feedback_score"]; !ok || v != nil {
		t.Fatalf("expected feedback_score=nil, got %v (present=%v)", v, ok)
	}
}

func TestPersistSessions_FeedbackScorePositive(t *testing.T) {
	mgr := session.NewManager(session.NewInMemoryStore())
	ctx := context.Background()
	score := 0.9
	_, _ = mgr.AddQATurn(ctx, "u1", "s1", "Q", "A", "", &score, nil)
	f := &fakeGraph{}
	s := PersistSessions{Store: mgr.Store(), Graph: f}
	_, _ = s.Run(ctx, Input{UserID: "u1", SessionIDs: []string{"s1"}})
	if v := f.params[0]["feedback_score"]; v != 0.9 {
		t.Fatalf("expected feedback_score=0.9, got %v", v)
	}
}

func TestPersistSessions_MaxTurnsCap(t *testing.T) {
	mgr := session.NewManager(session.NewInMemoryStore())
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		_, _ = mgr.AddQATurn(ctx, "u1", "s1", "Q", "A", "", nil, nil)
	}
	f := &fakeGraph{}
	s := PersistSessions{Store: mgr.Store(), Graph: f, MaxTurnsPerSession: 2}
	res, err := s.Run(ctx, Input{UserID: "u1", SessionIDs: []string{"s1"}})
	if err != nil {
		t.Fatal(err)
	}
	if res.Items != 2 {
		t.Fatalf("expected 2 turns persisted (capped), got %d", res.Items)
	}
}

func TestPersistSessions_Name(t *testing.T) {
	if (PersistSessions{}).Name() != "persist_sessions" {
		t.Fatal("unexpected name")
	}
}

func TestFeedbackOrNil(t *testing.T) {
	if v := feedbackOrNil(nil); v != nil {
		t.Fatalf("nil -> nil, got %v", v)
	}
	score := 0.5
	if v := feedbackOrNil(&score); v != 0.5 {
		t.Fatalf("pointer -> value, got %v", v)
	}
}

// silence unused import warning when test helpers aren't called.
var _ = time.Time{}
