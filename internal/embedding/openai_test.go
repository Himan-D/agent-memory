package embedding

import (
	"context"
	"testing"

	"agent-memory/internal/config"
)

func TestGenerateBatchEmbeddingsMapping(t *testing.T) {
	e := NewOpenAI(config.OpenAIConfig{APIKey: "test"})

	cachedText := "cached"
	cachedEmb := []float32{0.9, 0.9}
	e.cache.Set(cachedText, cachedEmb)

	// Test purely cached results
	texts := []string{cachedText, cachedText}
	results, err := e.GenerateBatchEmbeddingsWithContext(context.Background(), texts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}

	for i, res := range results {
		if res[0] != 0.9 || res[1] != 0.9 {
			t.Errorf("result %d is incorrect: %v", i, res)
		}
	}
}
