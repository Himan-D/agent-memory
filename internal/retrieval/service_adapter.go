package retrieval

import (
	"context"
	"strings"
	"sync"

	"agent-memory/internal/memory/types"
)

type ServiceAdapter struct {
	service interface {
		SearchMemories(ctx context.Context, req *types.SearchRequest) ([]types.MemoryResult, error)
	}
	bm25      *BM25
	bm25Mu    sync.RWMutex
	documents []string
}

func NewServiceAdapter(svc interface {
	SearchMemories(ctx context.Context, req *types.SearchRequest) ([]types.MemoryResult, error)
}) *ServiceAdapter {
	return &ServiceAdapter{
		service: svc,
	}
}

func (a *ServiceAdapter) SearchSemantic(ctx context.Context, query string, limit int) ([]types.MemoryResult, error) {
	req := &types.SearchRequest{
		Query:   query,
		Limit:   limit,
		Offset:  0,
		Filters: nil,
	}
	return a.service.SearchMemories(ctx, req)
}

func (a *ServiceAdapter) SearchKeyword(ctx context.Context, query string, limit int) ([]types.MemoryResult, error) {
	a.bm25Mu.RLock()
	defer a.bm25Mu.RUnlock()

	if a.bm25 == nil {
		return nil, nil
	}

	indices := a.bm25.Search(query, limit*2)
	if len(indices) == 0 {
		return nil, nil
	}

	var results []types.MemoryResult
	a.bm25Mu.RLock()
	for _, idx := range indices {
		if idx < len(a.documents) {
			results = append(results, types.MemoryResult{
				Text:  a.documents[idx],
				Score: float32(a.bm25.Score(query, idx)),
			})
		}
	}
	a.bm25Mu.RUnlock()

	return results, nil
}

func (a *ServiceAdapter) SearchEntities(ctx context.Context, entities []string, limit int) ([]types.MemoryResult, error) {
	if len(entities) == 0 {
		return nil, nil
	}

	req := &types.SearchRequest{
		Query: strings.Join(entities, " "),
		Limit: limit,
	}

	return a.service.SearchMemories(ctx, req)
}

func (a *ServiceAdapter) ExtractQueryEntities(ctx context.Context, query string) ([]string, error) {
	var entities []string

	words := strings.Fields(query)
	for _, word := range words {
		if len(word) > 0 && (word[0] >= 'A' && word[0] <= 'Z') {
			entities = append(entities, word)
		}
	}

	upperStarters := []string{"I", "We", "They", "The", "When", "Where", "How", "Why", "What", "Which"}
	for _, starter := range upperStarters {
		if strings.HasPrefix(query, starter) {
			entities = append(entities, starter)
		}
	}

	return entities, nil
}

func (a *ServiceAdapter) UpdateDocuments(documents []string) {
	a.bm25Mu.Lock()
	defer a.bm25Mu.Unlock()

	a.documents = documents
	if len(documents) > 0 {
		a.bm25 = NewBM25(documents)
	}
}
