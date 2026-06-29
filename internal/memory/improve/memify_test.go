package improve

import (
	"context"
	"strings"
	"testing"
)

type fakeIndexer struct {
	indexed []Triplet
	err     error
}

func (f *fakeIndexer) IndexTriplets(ctx context.Context, triplets []Triplet) error {
	if f.err != nil {
		return f.err
	}
	f.indexed = append(f.indexed, triplets...)
	return nil
}

func TestMemifyEnrichment_NilGraphErrors(t *testing.T) {
	s := MemifyEnrichment{Vector: &fakeIndexer{}}
	if _, err := s.Run(context.Background(), Input{UserID: "u1"}); err == nil {
		t.Fatal("expected error for nil graph")
	}
}

func TestMemifyEnrichment_NilVectorErrors(t *testing.T) {
	s := MemifyEnrichment{Graph: &fakeGraph{}}
	if _, err := s.Run(context.Background(), Input{UserID: "u1"}); err == nil {
		t.Fatal("expected error for nil vector")
	}
}

func TestMemifyEnrichment_EmptyRowsReportsZero(t *testing.T) {
	fg := &fakeGraph{}
	fv := &fakeIndexer{}
	s := MemifyEnrichment{Graph: fg, Vector: fv}
	res, err := s.Run(context.Background(), Input{UserID: "u1"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Items != 0 {
		t.Fatalf("expected 0 items, got %d", res.Items)
	}
	if len(fv.indexed) != 0 {
		t.Fatalf("expected no triplets indexed, got %d", len(fv.indexed))
	}
}

func TestMemifyEnrichment_GeneratesAndIndexesTriplets(t *testing.T) {
	fg := &fakeGraph{rows: []map[string]interface{}{
		{"id": "l1", "statement": "Alice lives in Paris with Bob"},
		{"id": "l2", "statement": "no entities here"},
	}}
	fv := &fakeIndexer{}
	s := MemifyEnrichment{Graph: fg, Vector: fv}
	res, err := s.Run(context.Background(), Input{UserID: "u1"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Items == 0 {
		t.Fatalf("expected > 0 triplets indexed, got %d", res.Items)
	}
	// l1 should produce triplets (Alice paired with Paris, Bob); l2 should produce none.
	foundL1 := false
	for _, tr := range fv.indexed {
		if tr.Source == "l1" && tr.Subject == "Alice" {
			foundL1 = true
		}
	}
	if !foundL1 {
		t.Fatalf("expected triplet from l1 with Subject=Alice, got %+v", fv.indexed)
	}
}

func TestMemifyEnrichment_QueryErrorPropagates(t *testing.T) {
	fg := &fakeGraph{err: errFake("query failed")}
	fv := &fakeIndexer{}
	s := MemifyEnrichment{Graph: fg, Vector: fv}
	if _, err := s.Run(context.Background(), Input{UserID: "u1"}); err == nil {
		t.Fatal("expected error from failed query")
	}
}

func TestMemifyEnrichment_IndexErrorPropagates(t *testing.T) {
	fg := &fakeGraph{rows: []map[string]interface{}{
		{"id": "l1", "statement": "Alice and Bob"},
	}}
	fv := &fakeIndexer{err: errFake("qdrant down")}
	s := MemifyEnrichment{Graph: fg, Vector: fv}
	if _, err := s.Run(context.Background(), Input{UserID: "u1"}); err == nil {
		t.Fatal("expected error from index failure")
	}
}

func TestMemifyEnrichment_Name(t *testing.T) {
	if (MemifyEnrichment{}).Name() != "memify_enrichment" {
		t.Fatal("unexpected name")
	}
}

func TestHeuristicTripletGenerator_SingleEntityReturnsNil(t *testing.T) {
	g := HeuristicTripletGenerator{}
	got, err := g.Generate(context.Background(), "only Alice here", "l1")
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatalf("expected nil for single entity, got %+v", got)
	}
}

func TestHeuristicTripletGenerator_NoEntitiesReturnsNil(t *testing.T) {
	g := HeuristicTripletGenerator{}
	got, err := g.Generate(context.Background(), "no entities here", "l1")
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatalf("expected nil for no entities, got %+v", got)
	}
}

func TestHeuristicTripletGenerator_MultipleEntities(t *testing.T) {
	g := HeuristicTripletGenerator{}
	got, err := g.Generate(context.Background(), "Alice met Bob Smith in Paris", "l1")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) < 2 {
		t.Fatalf("expected >= 2 triplets, got %d", len(got))
	}
	if got[0].Subject != "Alice" {
		t.Fatalf("expected first triplet Subject=Alice, got %q", got[0].Subject)
	}
	if got[0].Predicate != "is_related_to" {
		t.Fatalf("expected default predicate, got %q", got[0].Predicate)
	}
}

func TestHeuristicTripletGenerator_CustomPredicate(t *testing.T) {
	g := HeuristicTripletGenerator{Predicate: "knows"}
	got, err := g.Generate(context.Background(), "Alice met Bob", "l1")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) == 0 || got[0].Predicate != "knows" {
		t.Fatalf("expected custom predicate, got %+v", got)
	}
}

func TestExtractCapitalizedPhrases(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"Alice lives in Paris", []string{"Alice", "Paris"}},
		{"Bob Smith went to New York", []string{"Bob Smith", "New York"}},
		{"no entities here", nil},
		{"", nil},
		{"Alice", []string{"Alice"}},
	}
	for _, tc := range cases {
		got := extractCapitalizedPhrases(tc.in)
		if !equalStringSlices(got, tc.want) {
			t.Errorf("extractCapitalizedPhrases(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// silence unused import warning if test helpers aren't called.
var _ = strings.ToLower
