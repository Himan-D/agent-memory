package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
	"net/http"
	"net/http/httptest"
	"testing"

	"agent-memory/internal/config"
	"agent-memory/internal/memory/types"
)

type mockVectorStore struct {
	VectorStore
	shouldReturnResult bool
}

func (m *mockVectorStore) Search(ctx context.Context, query []float32, limit int, threshold float32, filters map[string]interface{}) ([]types.MemoryResult, error) {
	if !m.shouldReturnResult {
		return nil, nil
	}
	return []types.MemoryResult{
		{MemoryID: "mem-1", Score: 0.9, Text: "Result 1"},
	}, nil
}

type mockGraphStore struct {
	GraphStore
}

func (m *mockGraphStore) GetMemoriesByIDs(ids []string) ([]*types.Memory, error) {
	return []*types.Memory{{ID: "mem-1", Content: "Result 1"}}, nil
}

func BenchmarkSearchMemories(b *testing.B) {
	// Mock OpenAI API
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Simulate a batch response or single response depending on input
		resp := struct {
			Data []struct {
				Embedding []float32 `json:"embedding"`
				Index     int       `json:"index"`
			} `json:"data"`
		}{}

		// Simulate network latency
		time.Sleep(2 * time.Millisecond)

		// Very simple mock embedding
		emb := make([]float32, 1536)
		for i := range emb {
			emb[i] = 0.1
		}

		// Try to decode to see if it's batch or single
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		input := body["input"]
		if texts, ok := input.([]interface{}); ok {
			for i := range texts {
				resp.Data = append(resp.Data, struct {
					Embedding []float32 `json:"embedding"`
					Index     int       `json:"index"`
				}{Embedding: emb, Index: i})
			}
		} else {
			resp.Data = append(resp.Data, struct {
				Embedding []float32 `json:"embedding"`
				Index     int       `json:"index"`
			}{Embedding: emb, Index: 0})
		}

		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cfg := &config.Config{
		OpenAI: config.OpenAIConfig{
			APIKey:  "test-key",
			BaseURL: server.URL,
			Model:   "text-embedding-3-small",
		},
		Memory: config.MemoryConfig{
			MultiSignalEnabled: false,
		},
		App: config.AppConfig{
			BufferTimeout: 5 * time.Second,
		},
	}

	svc, err := NewService(cfg)
	if err != nil {
		b.Fatal(err)
	}
	// Setup vector store to NOT return results initially, forcing the expanded
	// search path (Step 1 Prospection-guided retrieval).
	vector := &mockVectorStore{shouldReturnResult: false}
	svc.vector = vector
	svc.graph = &mockGraphStore{}

	req := &types.SearchRequest{
		Query: "search for something expanded",
		Limit: 10,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Change query to bypass embedding cache
		req.Query = fmt.Sprintf("search for something expanded %d", i)

		// Ensure first search (multi-signal/direct) returns nothing to trigger expanded loop
		_, err := svc.SearchMemories(context.Background(), req)
		if err != nil {
			b.Fatal(err)
		}
	}
}
