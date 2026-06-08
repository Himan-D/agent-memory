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
