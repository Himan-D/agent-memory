package retrieval

import (
	"context"
	"math"
	"sync"

	"agent-memory/internal/memory/types"
)

type RetrievalConfig struct {
	SemanticWeight float64
	KeywordWeight  float64
	EntityWeight   float64
	TopK           int
}

func DefaultRetrievalConfig() *RetrievalConfig {
	return &RetrievalConfig{
		SemanticWeight: 0.60,
		KeywordWeight:  0.25,
		EntityWeight:   0.15,
		TopK:           10,
	}
}

type SignalResult struct {
	MemoryID string
	Content  string
	Score    float64
	Signal   string
}

type MultiSignalSearcher interface {
	SearchSemantic(ctx context.Context, query string, limit int) ([]types.MemoryResult, error)
	SearchKeyword(ctx context.Context, query string, limit int) ([]types.MemoryResult, error)
	SearchEntities(ctx context.Context, entities []string, limit int) ([]types.MemoryResult, error)
	ExtractQueryEntities(ctx context.Context, query string) ([]string, error)
}

type MultiSignalRetrieval struct {
	searcher MultiSignalSearcher
	config   *RetrievalConfig
}

func NewMultiSignalRetrieval(searcher MultiSignalSearcher, config *RetrievalConfig) *MultiSignalRetrieval {
	if config == nil {
		config = DefaultRetrievalConfig()
	}
	return &MultiSignalRetrieval{
		searcher: searcher,
		config:   config,
	}
}

func (m *MultiSignalRetrieval) Retrieve(ctx context.Context, query string) ([]types.MemoryResult, error) {
	var (
		semanticResults []types.MemoryResult
		keywordResults []types.MemoryResult
		entityResults   []types.MemoryResult
		errSem, errKey, errEnt error
		wg sync.WaitGroup
	)

	limit := m.config.TopK * 2

	wg.Add(3)
	go func() {
		defer wg.Done()
		semanticResults, errSem = m.searcher.SearchSemantic(ctx, query, limit)
	}()
	go func() {
		defer wg.Done()
		keywordResults, errKey = m.searcher.SearchKeyword(ctx, query, limit)
	}()
	go func() {
		defer wg.Done()
		entities, errEnt := m.searcher.ExtractQueryEntities(ctx, query)
		if errEnt == nil && len(entities) > 0 {
			entityResults, _ = m.searcher.SearchEntities(ctx, entities, limit)
		}
	}()
	wg.Wait()

	if errSem != nil && errKey != nil && errEnt != nil {
		return nil, errSem
	}

	combined := m.fuseResults(semanticResults, keywordResults, entityResults)
	results := m.rankAndSelect(combined)

	return results, nil
}

func (m *MultiSignalRetrieval) fuseResults(semantic, keyword, entities []types.MemoryResult) map[string]*SignalResult {
	scores := make(map[string]*SignalResult)
	idToContent := make(map[string]string)

	addToScores := func(results []types.MemoryResult, weight float64, signal string) {
		if len(results) == 0 {
			return
		}
		maxScore := float64(0)
		for _, r := range results {
			if float64(r.Score) > maxScore {
				maxScore = float64(r.Score)
			}
		}
		if maxScore == 0 {
			maxScore = 1
		}

		for _, r := range results {
			normalizedScore := float64(r.Score) / maxScore
			weightedScore := normalizedScore * weight

			if existing, ok := scores[r.MemoryID]; ok {
				existing.Score += weightedScore
			} else {
				scores[r.MemoryID] = &SignalResult{
					MemoryID: r.MemoryID,
					Content:  r.Text,
					Score:    weightedScore,
					Signal:   signal,
				}
			}
			idToContent[r.MemoryID] = r.Text
		}
	}

	addToScores(semantic, m.config.SemanticWeight, "semantic")
	addToScores(keyword, m.config.KeywordWeight, "keyword")
	addToScores(entities, m.config.EntityWeight, "entity")

	for id, result := range scores {
		if content, ok := idToContent[id]; ok {
			result.Content = content
		}
	}

	return scores
}

func (m *MultiSignalRetrieval) rankAndSelect(combined map[string]*SignalResult) []types.MemoryResult {
	var results []types.MemoryResult

	for _, result := range combined {
		results = append(results, types.MemoryResult{
			MemoryID: result.MemoryID,
			Text:     result.Content,
			Score:    float32(result.Score),
		})
	}

	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].Score > results[i].Score {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	if len(results) > m.config.TopK {
		results = results[:m.config.TopK]
	}

	return results
}

func (m *MultiSignalRetrieval) SetConfig(config *RetrievalConfig) {
	m.config = config
}

func (m *MultiSignalRetrieval) GetConfig() *RetrievalConfig {
	return m.config
}

func NormalizeScore(score float64, min, max float64) float64 {
	if max == min {
		return 0.5
	}
	normalized := (score - min) / (max - min)
	return math.Max(0, math.Min(1, normalized))
}