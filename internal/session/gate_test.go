package session

import (
	"agent-memory/internal/memory/safety"
	"testing"
)

type fakeClassifier struct {
	safe bool
}

func (f *fakeClassifier) Classify(content string) *safety.ClassificationResult {
	return &safety.ClassificationResult{Safe: f.safe, Category: "test", Reason: "test"}
}

type fakeScorer struct {
	score float64
}

func (f *fakeScorer) Score(content string) float64 { return f.score }

func TestGate_AllowPassesDefault(t *testing.T) {
	g := NewGate()
	e := ContextEntry{ID: "a"}
	if !g.Allow(e) {
		t.Fatal("default entry should pass gate")
	}
}

func TestGate_RejectsHarmful(t *testing.T) {
	g := NewGate()
	e := ContextEntry{ID: "a", HarmfulCount: 1}
	if g.Allow(e) {
		t.Fatal("harmful entry should be rejected")
	}
}

func TestGate_RejectsLowConfidence(t *testing.T) {
	g := NewGate()
	g.MinConfidence = 0.9
	e := ContextEntry{ID: "a", Confidence: 0.5}
	if g.Allow(e) {
		t.Fatal("low-confidence entry should be rejected")
	}
}

func TestGate_AllowAllPreservesOrder(t *testing.T) {
	g := NewGate()
	entries := []ContextEntry{
		{ID: "a", HarmfulCount: 1}, // rejected
		{ID: "b"},                  // passes
		{ID: "c"},                  // passes
	}
	got := g.AllowAll(entries)
	if len(got) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(got))
	}
	if got[0].ID != "b" || got[1].ID != "c" {
		t.Fatalf("order broken: %+v", got)
	}
}

func TestLoadScores_SafeClassifierSetsHarmfulZero(t *testing.T) {
	e := ContextEntry{Content: "hello"}
	LoadScores(&e, &fakeClassifier{safe: true}, nil)
	if e.HarmfulCount != 0 {
		t.Fatalf("safe content should have HarmfulCount=0, got %d", e.HarmfulCount)
	}
}

func TestLoadScores_UnsafeClassifierSetsHarmfulOne(t *testing.T) {
	e := ContextEntry{Content: "dangerous"}
	LoadScores(&e, &fakeClassifier{safe: false}, nil)
	if e.HarmfulCount != 1 {
		t.Fatalf("unsafe content should have HarmfulCount=1, got %d", e.HarmfulCount)
	}
}

func TestLoadScores_ScorerSetsConfidence(t *testing.T) {
	e := ContextEntry{Content: "x"}
	LoadScores(&e, nil, &fakeScorer{score: 0.95})
	if e.Confidence != 0.95 {
		t.Fatalf("expected confidence=0.95, got %v", e.Confidence)
	}
}

func TestLoadScores_ScorerZeroPreservesExisting(t *testing.T) {
	e := ContextEntry{Content: "x", Confidence: 0.8}
	LoadScores(&e, nil, &fakeScorer{score: 0})
	if e.Confidence != 0.8 {
		t.Fatalf("zero score should not overwrite existing confidence, got %v", e.Confidence)
	}
}

func TestLoadScores_NilDependenciesNoOp(t *testing.T) {
	e := ContextEntry{Content: "x", Confidence: 0.7}
	LoadScores(&e, nil, nil)
	if e.Confidence != 0.7 {
		t.Fatalf("nil deps should not modify entry, got %v", e.Confidence)
	}
}

func TestLoadScores_NilEntryReturnsNil(t *testing.T) {
	if got := LoadScores(nil, nil, nil); got != nil {
		t.Fatal("nil entry should return nil")
	}
}

func TestLoadScoresAll_AppliesToAll(t *testing.T) {
	entries := []ContextEntry{
		{Content: "a"},
		{Content: "b"},
		{Content: "c"},
	}
	LoadScoresAll(entries, &fakeClassifier{safe: true}, &fakeScorer{score: 0.9})
	for i, e := range entries {
		if e.Confidence != 0.9 {
			t.Fatalf("entry[%d] confidence=%v, expected 0.9", i, e.Confidence)
		}
	}
}

func TestLoadScoresAll_MixedUnsafe(t *testing.T) {
	entries := []ContextEntry{
		{Content: "a"},
		{Content: "dangerous"},
		{Content: "c"},
	}
	// fakeClassifier that always says unsafe
	alwaysUnsafe := &alwaysUnsafeClassifier{}
	LoadScoresAll(entries, alwaysUnsafe, nil)
	if entries[0].HarmfulCount != 1 || entries[1].HarmfulCount != 1 || entries[2].HarmfulCount != 1 {
		t.Fatalf("expected all marked harmful, got %+v", entries)
	}
}

type alwaysUnsafeClassifier struct{}

func (a *alwaysUnsafeClassifier) Classify(content string) *safety.ClassificationResult {
	return &safety.ClassificationResult{Safe: false, Category: "dangerous", Reason: "test"}
}

func TestGate_MinConfidenceDefault(t *testing.T) {
	g := NewGate()
	if g.minConfidence() != MinGateConfidence {
		t.Fatalf("expected default MinGateConfidence=%v, got %v", MinGateConfidence, g.minConfidence())
	}
}

func TestGate_MinConfidenceZeroFallsBack(t *testing.T) {
	g := &Gate{MinConfidence: 0}
	if g.minConfidence() != MinGateConfidence {
		t.Fatal("zero MinConfidence should fall back to package default")
	}
}
