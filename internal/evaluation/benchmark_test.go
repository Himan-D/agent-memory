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
