package rollback

import (
	"context"
	"strings"
	"testing"
	"time"
)

// fakeGraph records cypher/params and returns canned rows.
type fakeGraph struct {
	cyphers []string
	params  []map[string]interface{}
	// rows returned for each QueryGraph call, consumed in order
	rows [][]map[string]interface{}
	err  error
	idx  int
}

func (f *fakeGraph) QueryGraph(cypher string, params map[string]interface{}) ([]map[string]interface{}, error) {
	f.cyphers = append(f.cyphers, cypher)
	f.params = append(f.params, params)
	if f.err != nil {
		return nil, f.err
	}
	if f.idx < len(f.rows) {
		rs := f.rows[f.idx]
		f.idx++
		return rs, nil
	}
	return nil, nil
}

func TestNeo4jLedger_RecordIssuesMerge(t *testing.T) {
	f := &fakeGraph{}
	l := NewNeo4jLedger(f)
	err := l.Record(context.Background(), LedgerEntry{
		PipelineRunID: "r1",
		NodeIDs:       []string{"n1"},
		EdgeIDs:       []string{"e1"},
		VectorIDs:     []string{"v1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	c := f.cyphers[0]
	if !strings.Contains(c, "MERGE (r:RollbackEntry") {
		t.Fatalf("expected MERGE, got %s", c)
	}
	if f.params[0]["pipeline_run_id"] != "r1" {
		t.Fatalf("expected pipeline_run_id=r1")
	}
	if got := f.params[0]["status"]; got != string(StatusPending) {
		t.Fatalf("expected default status=pending, got %v", got)
	}
}

func TestNeo4jLedger_RecordRequiresID(t *testing.T) {
	l := NewNeo4jLedger(&fakeGraph{})
	if err := l.Record(context.Background(), LedgerEntry{}); err == nil {
		t.Fatal("expected error for empty PipelineRunID")
	}
}

func TestNeo4jLedger_RecordDefaultsCreatedAt(t *testing.T) {
	f := &fakeGraph{}
	l := NewNeo4jLedger(f)
	before := time.Now().UTC().Add(-time.Second)
	_ = l.Record(context.Background(), LedgerEntry{PipelineRunID: "r1"})
	after := time.Now().UTC().Add(time.Second)
	ts, _ := f.params[0]["created_at"].(string)
	parsed, err := time.Parse(time.RFC3339Nano, ts)
	if err != nil {
		t.Fatalf("parse created_at: %v", err)
	}
	if parsed.Before(before) || parsed.After(after) {
		t.Fatalf("created_at out of range: %v", parsed)
	}
}

func TestNeo4jLedger_MarkCommittedUpdatesStatus(t *testing.T) {
	f := &fakeGraph{rows: [][]map[string]interface{}{{{"id": "r1"}}}}
	l := NewNeo4jLedger(f)
	if err := l.MarkCommitted(context.Background(), "r1"); err != nil {
		t.Fatal(err)
	}
	c := f.cyphers[0]
	if !strings.Contains(c, "SET r.status = $status") {
		t.Fatalf("cypher missing status SET: %s", c)
	}
	if got := f.params[0]["status"]; got != string(StatusCommitted) {
		t.Fatalf("expected status=committed, got %v", got)
	}
}

func TestNeo4jLedger_MarkCommittedUnknownRun(t *testing.T) {
	f := &fakeGraph{rows: [][]map[string]interface{}{{}}}
	l := NewNeo4jLedger(f)
	if err := l.MarkCommitted(context.Background(), "missing"); err == nil {
		t.Fatal("expected error for unknown run")
	}
}

func TestNeo4jLedger_MarkFailedIncludesNote(t *testing.T) {
	f := &fakeGraph{rows: [][]map[string]interface{}{{{"id": "r1"}}}}
	l := NewNeo4jLedger(f)
	if err := l.MarkFailed(context.Background(), "r1", "deleter exploded"); err != nil {
		t.Fatal(err)
	}
	if got := f.params[0]["note"]; got != "deleter exploded" {
		t.Fatalf("expected note to be passed through, got %v", got)
	}
}

func TestNeo4jLedger_GetReturnsEntry(t *testing.T) {
	f := &fakeGraph{
		rows: [][]map[string]interface{}{{
			{
				"node_ids":   []interface{}{"n1", "n2"},
				"edge_ids":   []interface{}{"e1"},
				"vector_ids": []interface{}{"v1"},
				"status":     "pending",
				"note":       "test",
				"created_at": time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
				"updated_at": time.Date(2026, 1, 1, 0, 0, 1, 0, time.UTC),
			},
		}},
	}
	l := NewNeo4jLedger(f)
	e, err := l.Get(context.Background(), "r1")
	if err != nil {
		t.Fatal(err)
	}
	if e.Status != StatusPending {
		t.Fatalf("status: %v", e.Status)
	}
	if len(e.NodeIDs) != 2 || e.NodeIDs[0] != "n1" {
		t.Fatalf("node_ids: %v", e.NodeIDs)
	}
	if e.Note != "test" {
		t.Fatalf("note: %v", e.Note)
	}
}

func TestNeo4jLedger_GetUnknownRun(t *testing.T) {
	f := &fakeGraph{rows: [][]map[string]interface{}{{}}}
	l := NewNeo4jLedger(f)
	if _, err := l.Get(context.Background(), "missing"); err == nil {
		t.Fatal("expected error for unknown run")
	}
}

func TestNeo4jLedger_RollbackPendingRun(t *testing.T) {
	// Get returns entry; updateStatus succeeds.
	f := &fakeGraph{
		rows: [][]map[string]interface{}{
			{
				{
					"node_ids":   []interface{}{"n1"},
					"status":     "pending",
					"created_at": time.Now(),
					"updated_at": time.Now(),
				},
			},
			{{"id": "r1"}}, // updateStatus RETURN
		},
	}
	l := NewNeo4jLedger(f)
	nodes, err := l.Rollback(context.Background(), "r1")
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 1 || nodes[0] != "n1" {
		t.Fatalf("expected [n1], got %v", nodes)
	}
}

func TestNeo4jLedger_RollbackCommittedRunRejected(t *testing.T) {
	f := &fakeGraph{
		rows: [][]map[string]interface{}{
			{
				{
					"node_ids":   []interface{}{"n1"},
					"status":     "committed",
					"created_at": time.Now(),
					"updated_at": time.Now(),
				},
			},
		},
	}
	l := NewNeo4jLedger(f)
	_, err := l.Rollback(context.Background(), "r1")
	if err == nil {
		t.Fatal("expected error rolling back committed run")
	}
}

func TestNeo4jLedger_NilGraphErrors(t *testing.T) {
	l := NewNeo4jLedger(nil)
	if err := l.Record(context.Background(), LedgerEntry{PipelineRunID: "r1"}); err == nil {
		t.Fatal("expected error from nil graph on Record")
	}
	if err := l.MarkCommitted(context.Background(), "r1"); err == nil {
		t.Fatal("expected error from nil graph on MarkCommitted")
	}
	if _, err := l.Get(context.Background(), "r1"); err == nil {
		t.Fatal("expected error from nil graph on Get")
	}
	if _, err := l.Rollback(context.Background(), "r1"); err == nil {
		t.Fatal("expected error from nil graph on Rollback")
	}
}

func TestNeo4jLedger_QueryErrorPropagates(t *testing.T) {
	f := &fakeGraph{err: errFake("boom")}
	l := NewNeo4jLedger(f)
	if err := l.Record(context.Background(), LedgerEntry{PipelineRunID: "r1"}); err == nil {
		t.Fatal("expected error from failed query")
	}
}

type errFake string

func (e errFake) Error() string { return string(e) }
