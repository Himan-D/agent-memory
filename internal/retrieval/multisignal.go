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
	MemoryID     string
	Content      string
	Score        float64
	Signal       string
	Signals      []string
	SignalScores map[string]float64
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
		semanticResults        []types.MemoryResult
		keywordResults         []types.MemoryResult
		entityResults          []types.MemoryResult
		errSem, errKey, errEnt error
		wg                     sync.WaitGroup
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

// fuseResults implements weighted Reciprocal Rank Fusion (RRF):
// score(d) = Σ weight_i/(k + rank_i(d)), k=60.
// This keeps BM25/entity/semantic weights operational while avoiding raw-score
// normalization issues across heterogeneous search backends.
func (m *MultiSignalRetrieval) fuseResults(semantic, keyword, entities []types.MemoryResult) map[string]*SignalResult {
	const k = 60.0
	scores := make(map[string]*SignalResult)

	addRRF := func(results []types.MemoryResult, signal string, weight float64) {
		if weight <= 0 {
			return
		}
		for rank, r := range results {
			id := r.MemoryID
			if id == "" {
				continue
			}
			rrfScore := weight / (k + float64(rank+1))
			if existing, ok := scores[id]; ok {
				existing.Score += rrfScore
				existing.Signals = appendSignal(existing.Signals, signal)
				existing.SignalScores[signal] += rrfScore
				if existing.Content == "" {
					existing.Content = r.Text
				}
			} else {
				scores[id] = &SignalResult{
					MemoryID:     id,
					Content:      r.Text,
					Score:        rrfScore,
					Signal:       signal,
					Signals:      []string{signal},
					SignalScores: map[string]float64{signal: rrfScore},
				}
			}
		}
	}

	addRRF(semantic, "semantic", m.config.SemanticWeight)
	addRRF(keyword, "keyword", m.config.KeywordWeight)
	addRRF(entities, "entity", m.config.EntityWeight)

	return scores
}

func (m *MultiSignalRetrieval) rankAndSelect(combined map[string]*SignalResult) []types.MemoryResult {
	var results []types.MemoryResult

	for _, result := range combined {
		results = append(results, types.MemoryResult{
			MemoryID: result.MemoryID,
			Text:     result.Content,
			Score:    float32(result.Score),
			Entity: types.Entity{
				Properties: map[string]interface{}{
					"signals":       result.Signals,
					"signal_scores": result.SignalScores,
				},
			},
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

func appendSignal(signals []string, signal string) []string {
	for _, existing := range signals {
		if existing == signal {
			return signals
		}
	}
	return append(signals, signal)
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
