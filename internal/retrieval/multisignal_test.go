package retrieval

import (
	"context"
	"testing"

	"agent-memory/internal/memory/types"
)

type mockMultiSignalSearcher struct {
	semanticResults []types.MemoryResult
	keywordResults  []types.MemoryResult
	entityResults  []types.MemoryResult
}

func (m *mockMultiSignalSearcher) SearchSemantic(ctx context.Context, query string, limit int) ([]types.MemoryResult, error) {
	return m.semanticResults, nil
}

func (m *mockMultiSignalSearcher) SearchKeyword(ctx context.Context, query string, limit int) ([]types.MemoryResult, error) {
	return m.keywordResults, nil
}

func (m *mockMultiSignalSearcher) SearchEntities(ctx context.Context, entities []string, limit int) ([]types.MemoryResult, error) {
	return m.entityResults, nil
}

func (m *mockMultiSignalSearcher) ExtractQueryEntities(ctx context.Context, query string) ([]string, error) {
	return []string{}, nil
}

func TestMultiSignalRetrieval_Initialize(t *testing.T) {
	cfg := DefaultRetrievalConfig()
	ms := NewMultiSignalRetrieval(&mockMultiSignalSearcher{}, cfg)
	if ms == nil {
		t.Error("expected non-nil MultiSignalRetrieval")
	}
}

func TestMultiSignalRetrieval_WithNilConfig(t *testing.T) {
	ms := NewMultiSignalRetrieval(&mockMultiSignalSearcher{}, nil)
	if ms == nil {
		t.Error("expected non-nil MultiSignalRetrieval")
	}
	if ms.config == nil {
		t.Error("expected config to be set to default")
	}
	if ms.config.TopK != 10 {
		t.Errorf("expected default TopK 10, got %d", ms.config.TopK)
	}
}

func TestRetrievalConfig_Fields(t *testing.T) {
	cfg := &RetrievalConfig{
		SemanticWeight: 0.6,
		KeywordWeight:  0.25,
		EntityWeight:   0.15,
		TopK:           10,
	}

	if cfg.SemanticWeight != 0.6 {
		t.Errorf("expected 0.6, got %f", cfg.SemanticWeight)
	}
	if cfg.KeywordWeight != 0.25 {
		t.Errorf("expected 0.25, got %f", cfg.KeywordWeight)
	}
	if cfg.TopK != 10 {
		t.Errorf("expected 10, got %d", cfg.TopK)
	}
}

func TestDefaultRetrievalConfig(t *testing.T) {
	cfg := DefaultRetrievalConfig()
	
	if cfg.SemanticWeight != 0.60 {
		t.Errorf("expected semantic 0.60, got %f", cfg.SemanticWeight)
	}
	if cfg.KeywordWeight != 0.25 {
		t.Errorf("expected keyword 0.25, got %f", cfg.KeywordWeight)
	}
	if cfg.EntityWeight != 0.15 {
		t.Errorf("expected entity 0.15, got %f", cfg.EntityWeight)
	}
	if cfg.TopK != 10 {
		t.Errorf("expected topk 10, got %d", cfg.TopK)
	}
}

func TestSignalResult_Fields(t *testing.T) {
	result := SignalResult{
		MemoryID: "mem-1",
		Content:  "test content",
		Score:    0.95,
		Signal:   "semantic",
	}

	if result.MemoryID != "mem-1" {
		t.Errorf("expected mem-1, got %s", result.MemoryID)
	}
	if result.Score != 0.95 {
		t.Errorf("expected 0.95, got %f", result.Score)
	}
	if result.Signal != "semantic" {
		t.Errorf("expected semantic, got %s", result.Signal)
	}
}