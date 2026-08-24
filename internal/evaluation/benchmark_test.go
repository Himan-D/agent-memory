package evaluation

import (
	"context"
	"strings"
	"testing"
)

type fakeMemoryService struct {
	created int
}

func (s *fakeMemoryService) CreateMemory(ctx context.Context, content, userID string) (string, error) {
	s.created++
	return "created-" + userID, nil
}

func (s *fakeMemoryService) GetMemories(ctx context.Context, sessionID string) ([]MemoryResult, error) {
	return nil, nil
}

func TestRunBenchmarkReportsIngestAndMissingEvaluator(t *testing.T) {
	runner := NewBenchmarkRunner(nil, BenchmarkConfig{ParallelLimit: 2})
	memSvc := &fakeMemoryService{}
	dataset := &BenchmarkDataset{
		Name: "test",
		Memories: []BenchmarkMemory{
			{ID: "m1", Content: "User prefers Python", UserID: "demo-user"},
		},
		Questions: []BenchmarkQuestion{
			{
				ID:          "q1",
				Question:    "What does the user prefer?",
				SessionID:   "s1",
				MemoryID:    "m1",
				Category:    "single_hop",
				GroundTruth: "Python",
			},
		},
	}
	searchFn := func(ctx context.Context, sessionID, query string) ([]MemoryResult, error) {
		return []MemoryResult{{ID: "m1", Content: "User prefers Python", Score: 1}}, nil
	}

	results := runner.runBenchmark(context.Background(), dataset, memSvc, searchFn)
	summary := runner.summarizeResults(dataset.Name, results)

	if memSvc.created != 1 {
		t.Fatalf("expected one ingested memory, got %d", memSvc.created)
	}
	if summary.MemoriesIngested != 1 {
		t.Fatalf("expected memories_ingested=1, got %d", summary.MemoriesIngested)
	}
	if summary.ScoredQuestions != 0 {
		t.Fatalf("expected no scored questions without evaluator, got %d", summary.ScoredQuestions)
	}
	if summary.MemoryHitRate != 1 {
		t.Fatalf("expected memory_hit_rate=1, got %f", summary.MemoryHitRate)
	}
	if summary.MRR != 1 {
		t.Fatalf("expected mrr=1, got %f", summary.MRR)
	}
	if summary.EvaluatorConfigured {
		t.Fatal("expected evaluator_configured=false")
	}
	if summary.Publishable {
		t.Fatal("expected unscored benchmark to be non-publishable")
	}
	if summary.ScoreMethod != "unscored" {
		t.Fatalf("expected score_method=unscored, got %q", summary.ScoreMethod)
	}
	if len(summary.Warnings) == 0 || !strings.Contains(summary.Warnings[0], "no questions were scored") {
		t.Fatalf("expected scoring warning, got %+v", summary.Warnings)
	}
}

func TestLoadDatasetMissingReturnsError(t *testing.T) {
	runner := NewBenchmarkRunner(nil, BenchmarkConfig{ParallelLimit: 1})
	if _, err := runner.LoadDataset("does_not_exist"); err == nil {
		t.Fatal("expected missing dataset error")
	}
}

func TestHitRank(t *testing.T) {
	results := []MemoryResult{
		{ID: "m2", Content: "second"},
		{ID: "m1", Content: "first"},
	}
	if got := hitRank(results, "m1"); got != 2 {
		t.Fatalf("expected hit rank 2, got %d", got)
	}
	if got := hitRank(results, "missing"); got != 0 {
		t.Fatalf("expected missing hit rank 0, got %d", got)
	}
}

func TestBuildRetrievalContextTopK(t *testing.T) {
	results := []MemoryResult{
		{ID: "a", Content: "Alice likes tea"},
		{ID: "b", Content: "Bob likes coffee"},
		{ID: "c", Content: "Carol likes water"},
	}
	got := buildRetrievalContext(results, 2)
	if !strings.Contains(got, "[1] Alice likes tea") {
		t.Fatalf("missing first: %q", got)
	}
	if !strings.Contains(got, "[2] Bob likes coffee") {
		t.Fatalf("missing second: %q", got)
	}
	if strings.Contains(got, "Carol") {
		t.Fatalf("topK=2 should exclude third: %q", got)
	}
	if buildRetrievalContext(nil, 10) != "" {
		t.Fatal("empty results should yield empty context")
	}
}

func TestExpandRetrievalQueries(t *testing.T) {
	qs := ExpandRetrievalQueries("When did Caroline go to the LGBTQ support group?")
	if len(qs) < 2 {
		t.Fatalf("expected original + keyword form, got %v", qs)
	}
	joined := strings.ToLower(strings.Join(qs, " | "))
	if !strings.Contains(joined, "caroline") {
		t.Fatalf("expected keyword expansion with caroline: %v", qs)
	}
}

func TestNormalizeDatasetScope(t *testing.T) {
	ds := &BenchmarkDataset{
		Memories: []BenchmarkMemory{
			{ID: "m1", UserID: "demo-user", Content: "prefers python"},
			{ID: "m2", UserID: "demo-user", Content: "uses react"},
		},
		Questions: []BenchmarkQuestion{
			{ID: "q1", SessionID: "s001", GroundTruth: "Python"},
			{ID: "q2", SessionID: "s001", MemoryID: "m2", GroundTruth: "React"},
		},
	}
	NormalizeDatasetScope(ds)
	if ds.Questions[0].SessionID != "demo-user" {
		t.Fatalf("expected sole-user remap, got %q", ds.Questions[0].SessionID)
	}
	if ds.Questions[1].SessionID != "demo-user" {
		t.Fatalf("expected memory_id user remap, got %q", ds.Questions[1].SessionID)
	}
}

func TestRerankLexical(t *testing.T) {
	in := []MemoryResult{
		{ID: "a", Content: "unrelated hobby details", Score: 0.9},
		{ID: "b", Content: "Caroline went to the LGBTQ support group", Score: 0.5},
	}
	out := RerankLexical("When did Caroline go to the LGBTQ support group?", in, 2)
	if len(out) == 0 || out[0].ID != "b" {
		t.Fatalf("expected lexical lift of b, got %+v", out)
	}
}

func TestFuseRRF(t *testing.T) {
	a := []MemoryResult{{ID: "m1", Content: "Caroline LGBTQ support", Score: 0.9}}
	b := []MemoryResult{{ID: "m2", Content: "other", Score: 0.8}, {ID: "m1", Content: "Caroline LGBTQ support", Score: 0.5}}
	out := FuseRRF([][]MemoryResult{a, b}, "Caroline LGBTQ", 0)
	if len(out) == 0 || out[0].ID != "m1" {
		t.Fatalf("expected m1 first after RRF, got %+v", out)
	}
}

func TestBlendQAScore(t *testing.T) {
	if BlendQAScore(0.8, 0.2) != 0.8 {
		t.Fatal("expected max to prefer higher F1")
	}
	if BlendQAScore(0.1, 0.4) != 0.4 {
		t.Fatal("expected max to prefer higher LLM score")
	}
}

// TestBEAM10MContextTokenBudget verifies that when max_context_tokens is set (e.g. <7K
// for the 10M scale), retrieved answers are truncated to stay within the token budget.
func TestBEAM10MContextTokenBudget(t *testing.T) {
	const maxTokens = 7000
	const maxChars = maxTokens * 4 // ~4 chars per token

	runner := NewBenchmarkRunner(nil, BenchmarkConfig{ParallelLimit: 1})
	memSvc := &fakeMemoryService{}

	longContent := strings.Repeat("x", maxChars+100)

	dataset := &BenchmarkDataset{
		Name:             "beam_10m",
		MaxContextTokens: maxTokens,
		Memories: []BenchmarkMemory{
			{ID: "m001", Content: "Prefers Python.", UserID: "u001"},
		},
		Questions: []BenchmarkQuestion{
			{ID: "beam10m_t01", Question: "What language?", SessionID: "s001",
				MemoryID: "m001", Category: "single_hop", GroundTruth: "Python"},
		},
	}

	searchFn := func(ctx context.Context, sessionID, query string) ([]MemoryResult, error) {
		return []MemoryResult{{ID: "m001", Content: longContent, Score: 1}}, nil
	}

	results := runner.runBenchmark(context.Background(), dataset, memSvc, searchFn)
	summary := runner.summarizeResults(dataset.Name, results)
	summary.MaxContextTokens = dataset.MaxContextTokens

	if len(results) == 0 {
		t.Fatal("expected at least one result")
	}
	// find the question result (skip ingest results)
	var gotTokens int
	for _, r := range results {
		if !r.Ingested {
			gotTokens = r.Tokens
		}
	}
	if gotTokens > maxTokens {
		t.Fatalf("answer exceeded 7K token budget: got %d tokens (>%d)", gotTokens, maxTokens)
	}
	if summary.MaxContextTokens != maxTokens {
		t.Fatalf("expected max_context_tokens=%d in result, got %d", maxTokens, summary.MaxContextTokens)
	}
}

// TestBEAM10MDatasetLoads verifies the beam_10m dataset file is present and parseable.
func TestBEAM10MDatasetLoads(t *testing.T) {
	// LoadDataset uses paths relative to repo root; chdir there from internal/evaluation/
	t.Chdir("../..")

	runner := NewBenchmarkRunner(nil, BenchmarkConfig{ParallelLimit: 1})
	dataset, err := runner.LoadDataset("beam_10m")
	if err != nil {
		t.Fatalf("failed to load beam_10m dataset: %v", err)
	}
	if dataset.Name != "beam_10m" {
		t.Fatalf("expected dataset name beam_10m, got %q", dataset.Name)
	}
	if len(dataset.Questions) == 0 {
		t.Fatal("beam_10m dataset has no questions")
	}
	if len(dataset.Memories) == 0 {
		t.Fatal("beam_10m dataset has no memories")
	}
	if dataset.MaxContextTokens != 7000 {
		t.Fatalf("expected max_context_tokens=7000, got %d", dataset.MaxContextTokens)
	}
}
