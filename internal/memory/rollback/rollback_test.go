package rollback

import (
	"context"
	"errors"
	"testing"
)

func TestInMemoryLedger_RecordAndGet(t *testing.T) {
	l := NewInMemoryLedger()
	if err := l.Record(context.Background(), LedgerEntry{
		PipelineRunID: "r1",
		NodeIDs:       []string{"n1", "n2"},
	}); err != nil {
		t.Fatal(err)
	}
	e, err := l.Get(context.Background(), "r1")
	if err != nil {
		t.Fatal(err)
	}
	if len(e.NodeIDs) != 2 || e.Status != StatusPending {
		t.Fatalf("unexpected: %+v", e)
	}
}

func TestInMemoryLedger_MarkCommitted(t *testing.T) {
	l := NewInMemoryLedger()
	_ = l.Record(context.Background(), LedgerEntry{PipelineRunID: "r2"})
	if err := l.MarkCommitted(context.Background(), "r2"); err != nil {
		t.Fatal(err)
	}
	e, _ := l.Get(context.Background(), "r2")
	if e.Status != StatusCommitted {
		t.Fatalf("expected committed, got %s", e.Status)
	}
}

func TestInMemoryLedger_Rollback(t *testing.T) {
	l := NewInMemoryLedger()
	_ = l.Record(context.Background(), LedgerEntry{PipelineRunID: "r3", NodeIDs: []string{"n1"}})
	nodes, err := l.Rollback(context.Background(), "r3")
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 1 {
		t.Fatalf("expected 1 node id, got %d", len(nodes))
	}
	e, _ := l.Get(context.Background(), "r3")
	if e.Status != StatusRolled {
		t.Fatalf("expected rolled back, got %s", e.Status)
	}
}

func TestInMemoryLedger_RollbackCommittedFails(t *testing.T) {
	l := NewInMemoryLedger()
	_ = l.Record(context.Background(), LedgerEntry{PipelineRunID: "r4"})
	_ = l.MarkCommitted(context.Background(), "r4")
	if _, err := l.Rollback(context.Background(), "r4"); err == nil {
		t.Fatal("expected error rolling back committed run")
	}
}

func TestRollbackWithDeleter_CallsDeleter(t *testing.T) {
	l := NewInMemoryLedger()
	_ = l.Record(context.Background(), LedgerEntry{
		PipelineRunID: "r5",
		NodeIDs:       []string{"a"},
		EdgeIDs:       []string{"e"},
		VectorIDs:     []string{"v"},
	})
	called := false
	var gotNodes, gotEdges, gotVectors []string
	if err := RollbackWithDeleter(context.Background(), l, "r5", func(ctx context.Context, n, e, v []string) error {
		called = true
		gotNodes = n
		gotEdges = e
		gotVectors = v
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("deleter was not called")
	}
	if len(gotNodes) != 1 || len(gotEdges) != 1 || len(gotVectors) != 1 {
		t.Fatalf("deleter did not receive ids: %v %v %v", gotNodes, gotEdges, gotVectors)
	}
}

func TestRollbackWithDeleter_MarksFailedOnDeleterError(t *testing.T) {
	l := NewInMemoryLedger()
	_ = l.Record(context.Background(), LedgerEntry{PipelineRunID: "r6"})
	delErr := errors.New("deleter boom")
	err := RollbackWithDeleter(context.Background(), l, "r6", func(ctx context.Context, n, e, v []string) error {
		return delErr
	})
	if !errors.Is(err, delErr) {
		t.Fatalf("expected wrapped deleter error, got %v", err)
	}
	e, _ := l.Get(context.Background(), "r6")
	if e.Status != StatusFailed {
		t.Fatalf("expected failed status, got %s", e.Status)
	}
}
