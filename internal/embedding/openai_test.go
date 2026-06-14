package embedding

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"encoding/json"

	"agent-memory/internal/config"
)

func TestGenerateBatchEmbeddings_MappingFix(t *testing.T) {
	// Create a test server to mock OpenAI embeddings API
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req embedBatchRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("failed to decode request: %v", err)
			return
		}

		resp := embedResponse{
			Model: "text-embedding-3-small",
		}

		for i, text := range req.Input {
			var val float32
			if text == "miss-0" { val = 500 }
			if text == "miss-1" { val = 501 }

			resp.Data = append(resp.Data, struct {
				Embedding []float32 `json:"embedding"`
				Index     int       `json:"index"`
			}{
				Embedding: []float32{val},
				Index:     i,
			})
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cfg := config.OpenAIConfig{
		APIKey:  "test-key",
		Model:   "text-embedding-3-small",
		BaseURL: server.URL,
	}

	e := NewOpenAI(cfg)

	// Prime the cache
	e.cache.Set("hit-0", []float32{100})
	e.cache.Set("hit-1", []float32{101})

	// Mixed hits and misses
	texts := []string{"hit-0", "miss-0", "hit-1", "miss-1"}

	results, err := e.GenerateBatchEmbeddingsWithContext(context.Background(), texts)
	if err != nil {
		t.Fatalf("batch embedding failed: %v", err)
	}

	if len(results) != 4 {
		t.Fatalf("expected 4 results, got %d", len(results))
	}

	// Verify correct mapping
	if len(results[0]) == 0 || results[0][0] != 100 {
		t.Errorf("results[0] mismatch: expected 100, got %v", results[0])
	}
	if len(results[1]) == 0 || results[1][0] != 500 {
		t.Errorf("results[1] mismatch: expected 500, got %v", results[1])
	}
	if len(results[2]) == 0 || results[2][0] != 101 {
		t.Errorf("results[2] mismatch: expected 101, got %v", results[2])
	}
	if len(results[3]) == 0 || results[3][0] != 501 {
		t.Errorf("results[3] mismatch: expected 501, got %v", results[3])
	}
}
