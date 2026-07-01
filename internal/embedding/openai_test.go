package embedding

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"agent-memory/internal/config"
)

func TestEmbeddingCacheLRU(t *testing.T) {
	cache := NewEmbeddingCache(2)

	cache.Set("one", []float32{1.0})
	cache.Set("two", []float32{2.0})

	if _, found := cache.Get("one"); !found {
		t.Error("expected to find 'one'")
	}

	// 'one' is now most recent, 'two' is oldest
	cache.Set("three", []float32{3.0})

	if _, found := cache.Get("two"); found {
		t.Error("expected 'two' to be evicted")
	}
	if _, found := cache.Get("one"); !found {
		t.Error("expected to find 'one'")
	}
	if _, found := cache.Get("three"); !found {
		t.Error("expected to find 'three'")
	}
}

func TestGenerateBatchEmbeddingsOrdering(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{
			"data": [
				{"embedding": [0.1], "index": 0},
				{"embedding": [0.2], "index": 1}
			]
		}`)
	}))
	defer ts.Close()

	cfg := config.OpenAIConfig{
		APIKey:        "test",
		EmbedBaseURL: ts.URL,
		EmbedModel:    "test-model",
	}
	e := NewOpenAI(cfg)

	// Pre-seed cache for one item
	e.cache.Set("cached", []float32{0.9})

	texts := []string{"miss1", "cached", "miss2"}
	results, err := e.GenerateBatchEmbeddingsWithContext(context.Background(), texts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	if results[1][0] != 0.9 {
		t.Errorf("expected results[1] to be cached value 0.9, got %v", results[1][0])
	}
	if results[0][0] != 0.1 {
		t.Errorf("expected results[0] to be 0.1, got %v", results[0][0])
	}
	if results[2][0] != 0.2 {
		t.Errorf("expected results[2] to be 0.2, got %v", results[2][0])
	}
}
