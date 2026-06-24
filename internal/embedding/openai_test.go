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

func TestGenerateBatchEmbeddingsWithContext_MixedCache(t *testing.T) {
	// Setup mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody embedBatchRequest
		json.NewDecoder(r.Body).Decode(&reqBody)

		w.Header().Set("Content-Type", "application/json")

		// Return embeddings for whatever was requested in the batch
		data := make([]struct {
			Embedding []float32 `json:"embedding"`
			Index     int       `json:"index"`
		}, len(reqBody.Input))

		for i, text := range reqBody.Input {
			data[i].Index = i
			if text == "miss1" {
				data[i].Embedding = []float32{0.1, 0.1}
			} else if text == "miss2" {
				data[i].Embedding = []float32{0.2, 0.2}
			}
		}

		resp := embedResponse{
			Data: data,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cfg := config.OpenAIConfig{
		APIKey:  "test-key",
		Model:   "text-embedding-3-small",
		BaseURL: server.URL,
	}

	e := NewOpenAI(cfg)
	// Pre-fill cache with one item
	e.cache.Set("hit1", []float32{0.9, 0.9})

	texts := []string{"hit1", "miss1", "hit1", "miss2"}
	results, err := e.GenerateBatchEmbeddingsWithContext(context.Background(), texts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := [][]float32{
		{0.9, 0.9},
		{0.1, 0.1},
		{0.9, 0.9},
		{0.2, 0.2},
	}

	if !reflect.DeepEqual(results, expected) {
		t.Errorf("results mismatch.\ngot:  %v\nwant: %v", results, expected)
	}
}
