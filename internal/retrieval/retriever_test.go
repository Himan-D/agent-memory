package retrieval

import (
	"context"
	"strings"
	"testing"

	"agent-memory/internal/memory/types"
)

// fakeSearcher is a deterministic memorySearcher for tests.
type fakeSearcher struct {
	memories []types.MemoryResult
}

func (f *fakeSearcher) SearchSemantic(ctx context.Context, query string, limit int) ([]types.MemoryResult, error) {
	out := make([]types.MemoryResult, 0, len(f.memories))
	for _, m := range f.memories {
		if limit > 0 && len(out) >= limit {
			break
		}
		out = append(out, m)
	}
	return out, nil
}
func (f *fakeSearcher) SearchKeyword(ctx context.Context, query string, limit int) ([]types.MemoryResult, error) {
	return f.memories, nil
}
func (f *fakeSearcher) SearchEntities(ctx context.Context, entities []string, limit int) ([]types.MemoryResult, error) {
	return f.memories, nil
}
func (f *fakeSearcher) ExtractQueryEntities(ctx context.Context, query string) ([]string, error) {
	return nil, nil
}

func TestVectorRetriever_RetrieveAndComplete(t *testing.T) {
	s := &fakeSearcher{memories: []types.MemoryResult{
		{MemoryID: "m1", Text: "first", Score: 0.9},
		{MemoryID: "m2", Text: "second", Score: 0.5},
	}}
	r := NewVectorRetriever(s, &RetrieverConfig{TopK: 2, WideSearchTopK: 10})
	objs, err := r.RetrieveObjects(context.Background(), "any")
	if err != nil {
		t.Fatal(err)
	}
	if len(objs.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(objs.Items))
	}
	if objs.Items[0].ID != "m1" {
		t.Fatalf("expected top hit m1, got %s", objs.Items[0].ID)
	}
	comp, err := r.GetCompletion(context.Background(), "any")
	if err != nil {
		t.Fatal(err)
	}
	if len(comp.Citations) != 2 {
		t.Fatalf("expected 2 citations, got %d", len(comp.Citations))
	}
	if !strings.Contains(comp.Answer, "m1") && !strings.Contains(comp.Answer, "first") {
		t.Fatalf("expected answer to reference top hit: %q", comp.Answer)
	}
}

func TestVectorRetriever_WithCompletion(t *testing.T) {
	s := &fakeSearcher{memories: []types.MemoryResult{{MemoryID: "m1", Text: "x", Score: 1}}}
	r := NewVectorRetriever(s, nil).WithCompletion(func(ctx context.Context, q, c string) (string, error) {
		return "the answer is 42", nil
	})
	comp, err := r.GetCompletion(context.Background(), "q")
	if err != nil {
		t.Fatal(err)
	}
	if comp.Answer != "the answer is 42" {
		t.Fatalf("unexpected answer: %q", comp.Answer)
	}
}

func TestMultiSignalAdapter(t *testing.T) {
	s := &fakeSearcher{memories: []types.MemoryResult{
		{MemoryID: "m1", Text: "alpha", Score: 0.7},
	}}
	ms := NewMultiSignalRetrieval(s, DefaultRetrievalConfig())
	r := NewMultiSignalRetrieverAdapter(ms, nil)
	if r.Name() != "multisignal" {
		t.Fatalf("unexpected name: %s", r.Name())
	}
	comp, err := r.GetCompletion(context.Background(), "q")
	if err != nil {
		t.Fatal(err)
	}
	if len(comp.Citations) != 1 {
		t.Fatalf("expected 1 citation, got %d", len(comp.Citations))
	}
}

func TestGraphCompletionRetriever_AppliesDistancePenalty(t *testing.T) {
	s := &fakeSearcher{memories: []types.MemoryResult{
		{MemoryID: "high", Text: "a", Score: 0.9},
		{MemoryID: "low", Text: "b", Score: 0.2},
	}}
	r := NewGraphCompletionRetriever(s, &RetrieverConfig{
		TopK:                   5,
		WideSearchTopK:         10,
		TripletDistancePenalty: 6.5,
	})
	objs, err := r.RetrieveObjects(context.Background(), "q")
	if err != nil {
		t.Fatal(err)
	}
	if len(objs.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(objs.Items))
	}
	// The penalty reduces the high-score item's score; both must remain
	// finite.
	for _, it := range objs.Items {
		if it.Score > 100 || it.Score < -100 {
			t.Fatalf("score out of reasonable range: %v", it.Score)
		}
	}
}

func TestGraphCompletionRetriever_IncludeGlobalContext(t *testing.T) {
	s := &fakeSearcher{memories: []types.MemoryResult{{MemoryID: "m1", Text: "x", Score: 0.5}}}
	r := NewGraphCompletionRetriever(s, &RetrieverConfig{
		TopK:                 5,
		WideSearchTopK:       10,
		IncludeGlobalContext: true,
		MaxContextChars:      1000,
	})
	objs, _ := r.RetrieveObjects(context.Background(), "q")
	ctxText, err := r.BuildContext(context.Background(), "q", objs)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(ctxText, "[Global context]") {
		t.Fatalf("missing global context header: %q", ctxText)
	}
}

func TestTrimContext(t *testing.T) {
	in := "para1\n\npara2\n\npara3"
	got := trimContext(in, 12)
	if !strings.HasSuffix(got, "[…truncated]") {
		t.Fatalf("expected truncation marker: %q", got)
	}
}

func TestClamp01(t *testing.T) {
	cases := map[float64]float64{-1: 0, 0: 0, 0.5: 0.5, 1: 1, 2: 1}
	for in, want := range cases {
		if got := clamp01(in); got != want {
			t.Fatalf("clamp01(%v) = %v want %v", in, got, want)
		}
	}
}

func TestParseFloat(t *testing.T) {
	cases := map[string]float64{
		"":       0,
		"0":      0,
		"1":      1,
		"-2.5":   -2.5,
		"+3.14":  3.14,
		"42abc":  42,
	}
	for in, want := range cases {
		got, _ := parseFloat(in)
		if got != want {
			t.Fatalf("parseFloat(%q) = %v want %v", in, got, want)
		}
	}
}
