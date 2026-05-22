package recommendation

import (
	"context"
	"testing"
)

func TestPipelineBasic(t *testing.T) {
	cfg := DefaultPipelineConfig()
	cfg.EnableParallel = false
	p := NewPipeline(cfg)

	// Add a simple source that returns known IDs
	p.WithSources(&testSource{ids: []string{"m1", "m2", "m3"}})

	// Add a filter that removes "m2"
	p.WithFilters(&testFilter{removeID: "m2"})

	// Add a scorer that sets score based on ID
	p.WithScorers(&testScorer{})

	// Add a selector
	p.WithSelector(NewTopKSelector(10, false, 0))

	query := &QueryContext{
		UserID:  "user1",
		AgentID: "agent1",
	}

	results, err := p.Execute(context.Background(), query)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("Expected 2 results, got %d", len(results))
	}

	// Verify m2 was filtered out
	for _, r := range results {
		if r.ID == "m2" {
			t.Error("m2 should have been filtered out")
		}
	}

	// Verify scores were set
	for _, r := range results {
		if r.Score == 0 {
			t.Errorf("Expected score > 0 for %s", r.ID)
		}
	}
}

func TestTopKSelector(t *testing.T) {
	s := NewTopKSelector(3, false, 0)

	candidates := []*MemoryCandidate{
		{ID: "a", Score: 0.3},
		{ID: "b", Score: 0.9},
		{ID: "c", Score: 0.1},
		{ID: "d", Score: 0.7},
		{ID: "e", Score: 0.5},
	}

	results, err := s.Select(context.Background(), &QueryContext{}, candidates)
	if err != nil {
		t.Fatalf("Select error: %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("Expected 3 results, got %d", len(results))
	}

	// Verify top 3 by score: b (0.9), d (0.7), e (0.5)
	if results[0].ID != "b" || results[1].ID != "d" || results[2].ID != "e" {
		t.Errorf("Expected [b, d, e], got [%s, %s, %s]", results[0].ID, results[1].ID, results[2].ID)
	}
}

func TestAuthorDiversitySelector(t *testing.T) {
	s := NewTopKSelector(6, true, 2)

	candidates := []*MemoryCandidate{
		{ID: "a1", Score: 0.9, Metadata: map[string]interface{}{"author_id": "author1"}},
		{ID: "a2", Score: 0.85, Metadata: map[string]interface{}{"author_id": "author1"}},
		{ID: "a3", Score: 0.8, Metadata: map[string]interface{}{"author_id": "author1"}},
		{ID: "b1", Score: 0.7, Metadata: map[string]interface{}{"author_id": "author2"}},
		{ID: "b2", Score: 0.6, Metadata: map[string]interface{}{"author_id": "author2"}},
		{ID: "c1", Score: 0.5, Metadata: map[string]interface{}{"author_id": "author3"}},
	}

	results, err := s.Select(context.Background(), &QueryContext{}, candidates)
	if err != nil {
		t.Fatalf("Select error: %v", err)
	}

	// author1 should have max 2 results
	author1Count := 0
	for _, r := range results {
		if r.Metadata["author_id"] == "author1" {
			author1Count++
		}
	}
	if author1Count > 2 {
		t.Errorf("Expected max 2 from author1, got %d", author1Count)
	}
}

func TestPhoenixHeuristicScorer(t *testing.T) {
	scorer := NewPhoenixMemoryScorer(nil, DefaultActionWeights)

	candidate := NewMemoryCandidate("test1")
	candidate.Metadata["content"] = "Important meeting about AI strategy tomorrow at 3pm"
	candidate.Metadata["type"] = "conversation"
	candidate.Metadata["source"] = "entity1"
	candidate.Metadata["recency"] = 0.9

	query := &QueryContext{
		UserID:       "user1",
		FollowingIDs: []string{"entity1"},
	}

	err := scorer.Score(context.Background(), query, candidate)
	if err != nil {
		t.Fatalf("Score error: %v", err)
	}

	if candidate.Score == 0 {
		t.Error("Expected non-zero score from heuristic scorer")
	}

	t.Logf("Heuristic score: %.3f", candidate.Score)
}

func TestDropDuplicatesFilter(t *testing.T) {
	f := &DropDuplicatesFilter{}
	candidates := []*MemoryCandidate{
		{ID: "a"}, {ID: "b"}, {ID: "a"}, {ID: "c"}, {ID: "b"},
	}

	results, err := f.Filter(context.Background(), &QueryContext{}, candidates)
	if err != nil {
		t.Fatalf("Filter error: %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("Expected 3 unique results, got %d", len(results))
	}
}

func TestMutedKeywordFilter(t *testing.T) {
	f := &MutedKeywordFilter{}
	candidates := []*MemoryCandidate{
		{ID: "a", Metadata: map[string]interface{}{"content": "meeting about project"}},
		{ID: "b", Metadata: map[string]interface{}{"content": "ignore this spam content"}},
		{ID: "c", Metadata: map[string]interface{}{"content": "good discussion about AI"}},
	}

	query := &QueryContext{MutedKeywords: []string{"spam", "ignore"}}
	results, err := f.Filter(context.Background(), query, candidates)
	if err != nil {
		t.Fatalf("Filter error: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("Expected 2 results after filtering spam, got %d", len(results))
	}
}

// Test helpers
type testSource struct {
	ids []string
}

func (s *testSource) Name() string { return "test_source" }
func (s *testSource) Fetch(ctx context.Context, query *QueryContext) ([]string, error) {
	return s.ids, nil
}

type testFilter struct {
	removeID string
}

func (f *testFilter) Name() string { return "test_filter" }
func (f *testFilter) Filter(ctx context.Context, query *QueryContext, candidates []*MemoryCandidate) ([]*MemoryCandidate, error) {
	var result []*MemoryCandidate
	for _, c := range candidates {
		if c.ID != f.removeID {
			result = append(result, c)
		}
	}
	return result, nil
}

type testScorer struct{}

func (s *testScorer) Name() string { return "test_scorer" }
func (s *testScorer) Score(ctx context.Context, query *QueryContext, candidate *MemoryCandidate) error {
	candidate.Score = 0.5
	return nil
}
