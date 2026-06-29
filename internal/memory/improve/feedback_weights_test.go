package improve

import (
	"context"
	"strings"
	"testing"
)

type fakeGraph struct {
	cyphers []string
	params  []map[string]interface{}
	rows    []map[string]interface{}
	err     error
}

func (f *fakeGraph) QueryGraph(cypher string, params map[string]interface{}) ([]map[string]interface{}, error) {
	f.cyphers = append(f.cyphers, cypher)
	f.params = append(f.params, params)
	if f.err != nil {
		return nil, f.err
	}
	return f.rows, nil
}

func TestFeedbackWeights_NilGraphErrors(t *testing.T) {
	s := FeedbackWeights{Graph: nil}
	_, err := s.Run(context.Background(), Input{UserID: "u1"})
	if err == nil {
		t.Fatal("expected error for nil graph")
	}
}

func TestFeedbackWeights_IssuesCypher(t *testing.T) {
	f := &fakeGraph{}
	s := FeedbackWeights{Graph: f, BatchSize: 100}
	if _, err := s.Run(context.Background(), Input{UserID: "u1"}); err != nil {
		t.Fatal(err)
	}
	c := f.cyphers[0]
	if !strings.Contains(c, "MATCH (t:QATurn)") {
		t.Fatalf("cypher missing QATurn match: %s", c)
	}
	if !strings.Contains(c, "SET r.feedback_weight") {
		t.Fatalf("cypher missing feedback_weight SET: %s", c)
	}
	if f.params[0]["user_id"] != "u1" {
		t.Fatalf("expected user_id param, got %v", f.params[0]["user_id"])
	}
}

func TestFeedbackWeights_ReportsEdgeCount(t *testing.T) {
	f := &fakeGraph{rows: []map[string]interface{}{
		{"updated_edges": int64(42), "processed_turns": int64(7)},
	}}
	s := FeedbackWeights{Graph: f}
	res, err := s.Run(context.Background(), Input{UserID: "u1"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Items != 42 {
		t.Fatalf("expected Items=42, got %d", res.Items)
	}
}

func TestFeedbackWeights_HandlesIntEdgeCount(t *testing.T) {
	// Some Neo4j drivers return int instead of int64.
	f := &fakeGraph{rows: []map[string]interface{}{
		{"updated_edges": 17, "processed_turns": 3},
	}}
	s := FeedbackWeights{Graph: f}
	res, err := s.Run(context.Background(), Input{UserID: "u1"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Items != 17 {
		t.Fatalf("expected Items=17, got %d", res.Items)
	}
}

func TestFeedbackWeights_QueryErrorPropagates(t *testing.T) {
	f := &fakeGraph{err: errFake("query failed")}
	s := FeedbackWeights{Graph: f}
	_, err := s.Run(context.Background(), Input{UserID: "u1"})
	if err == nil {
		t.Fatal("expected error from failed query")
	}
}

func TestFeedbackWeights_DefaultBatchSize(t *testing.T) {
	f := &fakeGraph{}
	s := FeedbackWeights{Graph: f}
	_, _ = s.Run(context.Background(), Input{UserID: "u1"})
	if got := f.params[0]["limit"]; got != int64(500) {
		t.Fatalf("expected default limit=500, got %v", got)
	}
}

func TestFeedbackWeights_CustomBatchSize(t *testing.T) {
	f := &fakeGraph{}
	s := FeedbackWeights{Graph: f, BatchSize: 25}
	_, _ = s.Run(context.Background(), Input{UserID: "u1"})
	if got := f.params[0]["limit"]; got != int64(25) {
		t.Fatalf("expected limit=25, got %v", got)
	}
}

func TestFeedbackWeights_Name(t *testing.T) {
	s := FeedbackWeights{}
	if s.Name() != "feedback_weights" {
		t.Fatalf("unexpected name: %s", s.Name())
	}
}

type errFake string

func (e errFake) Error() string { return string(e) }
