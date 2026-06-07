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
	bm25        *BM25
	bm25Mu      sync.RWMutex
	documents   []string
	documentIDs []string
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
		Mode:    "semantic",
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
	for _, idx := range indices {
		if idx < len(a.documents) {
			memoryID := ""
			if idx < len(a.documentIDs) {
				memoryID = a.documentIDs[idx]
			}
			results = append(results, types.MemoryResult{
				MemoryID: memoryID,
				Text:     a.documents[idx],
				Score:    float32(a.bm25.Score(query, idx)),
			})
		}
	}

	return results, nil
}

func (a *ServiceAdapter) SearchEntities(ctx context.Context, entities []string, limit int) ([]types.MemoryResult, error) {
	if len(entities) == 0 {
		return nil, nil
	}

	req := &types.SearchRequest{
		Query: strings.Join(entities, " "),
		Limit: limit,
		Mode:  "entity",
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
	a.documentIDs = make([]string, len(documents))
	if len(documents) > 0 {
		a.bm25 = NewBM25(documents)
	}
}

func (a *ServiceAdapter) AppendDocument(doc string) {
	a.AppendDocumentWithID("", doc)
}

func (a *ServiceAdapter) AppendDocumentWithID(id, doc string) {
	if doc == "" {
		return
	}
	a.bm25Mu.Lock()
	defer a.bm25Mu.Unlock()

	a.documents = append(a.documents, doc)
	a.documentIDs = append(a.documentIDs, id)
	a.bm25 = NewBM25(a.documents)
}

func (a *ServiceAdapter) UpdateMemoryDocuments(memories []*types.Memory) {
	a.bm25Mu.Lock()
	defer a.bm25Mu.Unlock()

	a.documents = make([]string, 0, len(memories))
	a.documentIDs = make([]string, 0, len(memories))
	for _, mem := range memories {
		if mem == nil || mem.Content == "" {
			continue
		}
		a.documents = append(a.documents, mem.Content)
		a.documentIDs = append(a.documentIDs, mem.ID)
	}
	if len(a.documents) > 0 {
		a.bm25 = NewBM25(a.documents)
	} else {
		a.bm25 = nil
	}
}
