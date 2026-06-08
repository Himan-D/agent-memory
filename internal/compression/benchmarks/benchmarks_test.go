package benchmarks

import (
	"context"
	"testing"
)

func TestRunner_RunDefaultCorpus(t *testing.T) {
	runner := NewRunner(nil)
	result, err := runner.Run(context.Background(), Config{
		Corpus:       "agent_memory",
		Algorithms:   []string{"radix", "real_best", "gzip"},
		Iterations:   1,
		MinRetention: 0.8,
	})
	if err != nil {
		t.Fatalf("run benchmark: %v", err)
	}
	if result.SampleCount == 0 {
		t.Fatal("expected benchmark samples")
	}
	if len(result.Algorithms) != 3 {
		t.Fatalf("expected 3 algorithm results, got %d", len(result.Algorithms))
	}
	if result.Winner == "" {
		t.Fatal("expected winner")
	}
	for _, alg := range result.Algorithms {
		if alg.SampleCount != result.SampleCount {
			t.Fatalf("algorithm %s sample count mismatch", alg.Name)
		}
		if alg.Iterations != 1 {
			t.Fatalf("algorithm %s iterations mismatch: %d", alg.Name, alg.Iterations)
		}
	}
}

func TestRunner_RunCustomSamplesWithExamples(t *testing.T) {
	runner := NewRunner(nil)
	result, err := runner.Run(context.Background(), Config{
		Samples: []string{
			"memory memory memory graph graph graph compression compression compression",
			"agent agent agent retrieval retrieval retrieval source source source",
		},
		Algorithms:      []string{"radix"},
		Iterations:      1,
		IncludeExamples: true,
	})
	if err != nil {
		t.Fatalf("run benchmark: %v", err)
	}
	if result.Corpus != "custom" {
		t.Fatalf("expected custom corpus, got %s", result.Corpus)
	}
	if len(result.Algorithms) != 1 {
		t.Fatalf("expected one algorithm, got %d", len(result.Algorithms))
	}
	if len(result.Algorithms[0].Examples) == 0 {
		t.Fatal("expected examples")
	}
}

func TestRunner_UnknownCorpus(t *testing.T) {
	runner := NewRunner(nil)
	if _, err := runner.Run(context.Background(), Config{Corpus: "missing"}); err == nil {
		t.Fatal("expected unknown corpus error")
	}
}

func TestRunner_UnknownAlgorithm(t *testing.T) {
	runner := NewRunner(nil)
	if _, err := runner.Run(context.Background(), Config{
		Corpus:     "agent_memory",
		Algorithms: []string{"missing"},
	}); err == nil {
		t.Fatal("expected unknown algorithm error")
	}
}

func TestLexicalRetention(t *testing.T) {
	retention := lexicalRetention("User prefers Python and graph memory", "Python graph memory")
	if retention <= 0 || retention >= 1 {
		t.Fatalf("expected partial retention, got %f", retention)
	}
	if got := lexicalRetention("same same", "same same"); got != 1 {
		t.Fatalf("expected full retention, got %f", got)
	}
}
