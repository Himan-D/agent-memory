package memory

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"agent-memory/internal/config"
	"agent-memory/internal/memory/types"
)

func BenchmarkSearchMemories(b *testing.B) {
	// Mock OpenAI API
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Return multiple embeddings for prospection expansion
		w.Write([]byte(`{"data": [{"embedding": [0.1, 0.2], "index": 0}, {"embedding": [0.3, 0.4], "index": 1}, {"embedding": [0.5, 0.6], "index": 2}, {"embedding": [0.7, 0.8], "index": 3}, {"embedding": [0.9, 1.0], "index": 4}, {"embedding": [1.1, 1.2], "index": 5}]}`))
	}))
	defer server.Close()

	cfg := &config.Config{
		OpenAI: config.OpenAIConfig{
			APIKey:  "test",
			Model:   "text-embedding-3-small",
			BaseURL: server.URL,
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
		b.Fatalf("failed to create service: %v", err)
	}

	// Mock vector store
	mockVector := &mockVectorStore{delay: 10 * time.Millisecond}
	svc.vector = mockVector

	// Mock graph store
	mockGraph := &mockGraphStore{}
	svc.graph = mockGraph

	req := &types.SearchRequest{
		Query: "test query with many expansion possibilities",
		Limit: 10,
	}

	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = svc.SearchMemories(ctx, req)
	}
}

type mockVectorStore struct {
	VectorStore
	delay time.Duration
}

func (m *mockVectorStore) Search(ctx context.Context, query []float32, limit int, threshold float32, filters map[string]interface{}) ([]types.MemoryResult, error) {
	time.Sleep(m.delay)
	return []types.MemoryResult{{MemoryID: "1", Score: 0.9}}, nil
}

func (m *mockVectorStore) Ping(ctx context.Context) error { return nil }

type mockGraphStore struct {
	GraphStore
}

func (m *mockGraphStore) GetMemory(id string) (*types.Memory, error) {
	return &types.Memory{ID: id, CreatedAt: time.Now()}, nil
}

func (m *mockGraphStore) GetMemoriesByIDs(ids []string) ([]*types.Memory, error) {
	results := make([]*types.Memory, len(ids))
	for i, id := range ids {
		results[i] = &types.Memory{ID: id, CreatedAt: time.Now()}
	}
	return results, nil
}

func (m *mockGraphStore) UpdateMemory(mem *types.Memory) error { return nil }
func (m *mockGraphStore) Ping(ctx context.Context) error      { return nil }
