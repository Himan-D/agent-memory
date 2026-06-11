package embedding

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"agent-memory/internal/config"
)

func TestGenerateBatchEmbeddingsWithContext_OrderAndCache(t *testing.T) {
	// Mock OpenAI server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req embedBatchRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("failed to decode request: %v", err)
		}

		resp := embedResponse{}
		for i, _ := range req.Input {
			// Return dummy embedding where first element is index + 100 for identification
			emb := []float32{float32(i + 100), 0.0}
			resp.Data = append(resp.Data, struct {
				Embedding []float32 `json:"embedding"`
				Index     int       `json:"index"`
			}{Embedding: emb, Index: i})
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cfg := config.OpenAIConfig{
		APIKey:  "dummy",
		Model:   "text-embedding-3-small",
		BaseURL: server.URL,
	}
	e := NewOpenAI(cfg)
	e.client = server.Client()

	// Pre-seed cache with "cached" text at index 1
	cachedEmb := []float32{99.0, 99.0}
	e.cache.Set("cached", cachedEmb)

	texts := []string{"fetch1", "cached", "fetch2"}
	results, err := e.GenerateBatchEmbeddingsWithContext(context.Background(), texts)
	if err != nil {
		t.Fatalf("GenerateBatchEmbeddingsWithContext failed: %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	// Verify order:
	// index 0: fetch1 (first from API batch -> index 0 + 100 = 100.0)
	// index 1: cached (from cache -> 99.0)
	// index 2: fetch2 (second from API batch -> index 1 + 100 = 101.0)

	if results[0][0] != 100.0 {
		t.Errorf("expected results[0] to be 100.0, got %f", results[0][0])
	}
	if !reflect.DeepEqual(results[1], cachedEmb) {
		t.Errorf("expected results[1] to be cached value, got %v", results[1])
	}
	if results[2][0] != 101.0 {
		t.Errorf("expected results[2] to be 101.0, got %f", results[2][0])
	}
}
