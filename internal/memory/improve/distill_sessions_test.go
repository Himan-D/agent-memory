package improve

import (
	"context"
	"testing"

	"agent-memory/internal/session"
)

func TestDistillSessions_NilManagerErrors(t *testing.T) {
	s := DistillSessions{Distiller: session.NewDistiller(session.NewManager(session.NewInMemoryStore()), session.DistillOptions{})}
	if _, err := s.Run(context.Background(), Input{UserID: "u1"}); err == nil {
		t.Fatal("expected error for nil manager")
	}
}

func TestDistillSessions_NilDistillerErrors(t *testing.T) {
	s := DistillSessions{Manager: session.NewManager(session.NewInMemoryStore())}
	if _, err := s.Run(context.Background(), Input{UserID: "u1"}); err == nil {
		t.Fatal("expected error for nil distiller")
	}
}

func TestDistillSessions_MissingUserIDErrors(t *testing.T) {
	mgr := session.NewManager(session.NewInMemoryStore())
	d := session.NewDistiller(mgr, session.DistillOptions{})
	s := DistillSessions{Manager: mgr, Distiller: d}
	if _, err := s.Run(context.Background(), Input{}); err == nil {
		t.Fatal("expected error for missing user_id")
	}
}

func TestDistillSessions_NoSessionsRunsEmpty(t *testing.T) {
	mgr := session.NewManager(session.NewInMemoryStore())
	d := session.NewDistiller(mgr, session.DistillOptions{
		Writer: session.AcceptAllWriter(),
	})
	s := DistillSessions{Manager: mgr, Distiller: d}
	res, err := s.Run(context.Background(), Input{UserID: "u1"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Items != 0 {
		t.Fatalf("expected 0 lessons, got %d", res.Items)
	}
}

func TestDistillSessions_AccumulatesAcceptedLessons(t *testing.T) {
	mgr := session.NewManager(session.NewInMemoryStore())
	ctx := context.Background()
	_, _ = mgr.AddQATurn(ctx, "u1", "s1", "Q1", "Alice lives in Paris.", "", nil, nil)

	d := session.NewDistiller(mgr, session.DistillOptions{
		Curator: func(ctx context.Context, batch string) ([]session.ProposedLesson, error) {
			return []session.ProposedLesson{{WorkingStatement: "Alice lives in Paris"}}, nil
		},
		Writer: session.AcceptAllWriter(),
	})
	s := DistillSessions{Manager: mgr, Distiller: d}
	res, err := s.Run(ctx, Input{UserID: "u1", SessionIDs: []string{"s1"}})
	if err != nil {
		t.Fatal(err)
	}
	if res.Items < 1 {
		t.Fatalf("expected >= 1 accepted lesson, got %d", res.Items)
	}
}

func TestDistillSessions_Name(t *testing.T) {
	if (DistillSessions{}).Name() != "distill_sessions" {
		t.Fatal("unexpected name")
	}
}
