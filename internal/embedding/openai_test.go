package embedding

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"agent-memory/internal/config"
)

func TestGenerateBatchEmbeddings_Mapping(t *testing.T) {
	// Mock OpenAI API server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req embedBatchRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("failed to decode request: %v", err)
		}

		resp := embedResponse{
			Model: "text-embedding-3-small",
		}
		for i := range req.Input {
			// Return a unique embedding for each input to verify mapping
			// Input order in request must match output order via Index
			val := float32(i + 10)
			resp.Data = append(resp.Data, struct {
				Embedding []float32 `json:"embedding"`
				Index     int       `json:"index"`
			}{
				Embedding: []float32{val, val},
				Index:     i,
			})
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cfg := config.OpenAIConfig{
		APIKey: "test-key",
		Model:  "text-embedding-3-small",
	}
	e := NewOpenAI(cfg)
	e.BaseURL = server.URL

	// Pre-fill cache for one item
	e.cache.Set("cached", []float32{1.0, 1.0})

	texts := []string{"fetch1", "cached", "fetch2"}

	results, err := e.GenerateBatchEmbeddingsWithContext(context.Background(), texts)
	if err != nil {
		t.Fatalf("GenerateBatchEmbeddingsWithContext failed: %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	// Verify mapping:
	// "fetch1" -> index 0 in fetch list -> val 10
	// "cached" -> from cache -> val 1.0
	// "fetch2" -> index 1 in fetch list -> val 11

	if results[0][0] != 10.0 {
		t.Errorf("results[0] mismatch: expected 10.0, got %f", results[0][0])
	}
	if results[1][0] != 1.0 {
		t.Errorf("results[1] mismatch: expected 1.0 (from cache), got %f", results[1][0])
	}
	if results[2][0] != 11.0 {
		t.Errorf("results[2] mismatch: expected 11.0, got %f", results[2][0])
	}
}
