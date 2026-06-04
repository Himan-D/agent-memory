package retrieval

import (
	"context"
	"fmt"
	"sort"

	"agent-memory/internal/memory/types"
)

// GraphStore is a minimal interface for graph operations needed by spreading activation.
// Defined locally to avoid an import cycle with internal/memory.
type GraphStore interface {
	GetEntityRelations(entityID string, relType string) ([]types.Relation, error)
	GetEntity(id string) (*types.Entity, error)
}

// VectorStore is a minimal interface for vector operations needed by spreading activation.
// Defined locally to avoid an import cycle with internal/memory.
type VectorStore interface {
	Search(ctx context.Context, query []float32, limit int, threshold float32, filters map[string]interface{}) ([]types.MemoryResult, error)
}

// MemoryService is a minimal interface for memory search used by spreading activation.
// It intentionally does not reference internal/memory to avoid import cycles.
type MemoryService interface {
	SearchMemories(ctx context.Context, req *types.SearchRequest) ([]types.MemoryResult, error)
}

type SpreadingActivation struct {
	memSvc        MemoryService
	graphStore    GraphStore
	vectorStore   VectorStore
	initialBudget float64
	decayFactor   float64
	threshold     float64
	maxHops       int
}

type ActivationResult struct {
	Nodes        []ActivatedNode
	TotalScore  float64
	HopBreakdown []int
}

type ActivatedNode struct {
	ID        string
	Label     string
	Score    float64
	Hop      int
	MemoryID string
}

type SearchMode string

const (
	SearchModeVector    SearchMode = "vector"
	SearchModeSpreading SearchMode = "spreading"
	SearchModeHybrid   SearchMode = "hybrid"
)

// NewSpreadingActivation creates a SpreadingActivation.
// graph and vector may be nil; in that case spreading operations degrade gracefully.
func NewSpreadingActivation(memSvc MemoryService, graph GraphStore, vector VectorStore) *SpreadingActivation {
	return &SpreadingActivation{
		memSvc:        memSvc,
		graphStore:    graph,
		vectorStore:   vector,
		initialBudget: 1.0,
		decayFactor:   0.85,
		threshold:     0.1,
		maxHops:       3,
	}
}

func (s *SpreadingActivation) SetHyperparameters(initialBudget, decayFactor, threshold float64, maxHops int) {
	s.initialBudget = initialBudget
	s.decayFactor = decayFactor
	s.threshold = threshold
	s.maxHops = maxHops
}

func (s *SpreadingActivation) Retrieve(ctx context.Context, query string, mode SearchMode) ([]*types.Memory, error) {
	switch mode {
	case SearchModeSpreading:
		return s.retrieveSpreading(ctx, query)
	case SearchModeHybrid:
		return s.retrieveHybrid(ctx, query)
	default:
		return s.retrieveVector(ctx, query)
	}
}

func (s *SpreadingActivation) retrieveVector(ctx context.Context, query string) ([]*types.Memory, error) {
	if s.memSvc == nil {
		return nil, fmt.Errorf("memory service not configured")
	}

	req := &types.SearchRequest{
		Query:     query,
		Limit:     50,
		Threshold: 0.7,
	}
	results, err := s.memSvc.SearchMemories(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("vector search: %w", err)
	}

	var memories []*types.Memory
	for _, r := range results {
		if r.Metadata != nil {
			mem := r.Metadata
			if mem.Metadata == nil {
				mem.Metadata = make(map[string]interface{})
			}
			mem.Metadata["activation_score"] = float64(r.Score)
			mem.Metadata["activation_hops"] = 0
			memories = append(memories, mem)
		}
	}

	return memories, nil
}

func (s *SpreadingActivation) retrieveSpreading(ctx context.Context, query string) ([]*types.Memory, error) {
	if s.memSvc == nil {
		return s.retrieveVector(ctx, query)
	}

	req := &types.SearchRequest{
		Query:     query,
		Limit:     50,
		Threshold: 0.5,
	}
	initialResults, err := s.memSvc.SearchMemories(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("vector search: %w", err)
	}

	activationMap := s.initializeActivation(initialResults)

	for hop := 0; hop < s.maxHops; hop++ {
		activationMap = s.propagate(ctx, activationMap)
	}

	results := s.rankByActivation(ctx, activationMap)

	var memories []*types.Memory
	for _, r := range results {
		if r.MemoryID != "" || r.ID != "" {
			memID := r.MemoryID
			if memID == "" {
				memID = r.ID
			}
			memories = append(memories, &types.Memory{
				ID:      memID,
				Metadata: map[string]interface{}{
					"activation_score": r.Score,
					"activation_hops": r.Hop,
				},
			})
		}
	}

	return memories, nil
}

func (s *SpreadingActivation) retrieveHybrid(ctx context.Context, query string) ([]*types.Memory, error) {
	if s.memSvc == nil {
		return nil, fmt.Errorf("memory service not configured")
	}

	vectorReq := &types.SearchRequest{
		Query:     query,
		Limit:    25,
		Threshold: 0.7,
	}
	vectorResults, err := s.memSvc.SearchMemories(ctx, vectorReq)
	if err != nil {
		return nil, fmt.Errorf("vector search: %w", err)
	}

	var vectorMemories []*types.Memory
	for _, r := range vectorResults {
		if r.Metadata != nil {
			mem := r.Metadata
			if mem.Metadata == nil {
				mem.Metadata = make(map[string]interface{})
			}
			mem.Metadata["activation_score"] = float64(r.Score)
			mem.Metadata["activation_hops"] = 0
			vectorMemories = append(vectorMemories, mem)
		}
	}

	spreadingResults, err := s.retrieveSpreading(ctx, query)
	if err != nil {
		return vectorMemories, nil
	}

	seen := make(map[string]bool)
	var merged []*types.Memory

	for _, r := range vectorResults {
		if r.Metadata != nil && !seen[r.Metadata.ID] {
			seen[r.Metadata.ID] = true
			merged = append(merged, r.Metadata)
		}
	}

	for _, r := range spreadingResults {
		if !seen[r.ID] {
			seen[r.ID] = true
			merged = append(merged, r)
		}
	}

	return merged, nil
}

func (s *SpreadingActivation) initializeActivation(results []types.MemoryResult) map[string]float64 {
	activationMap := make(map[string]float64)

	for _, r := range results {
		memID := r.MemoryID
		if memID == "" && r.Metadata != nil {
			memID = r.Metadata.ID
		}
		if memID != "" {
			activationMap[memID] = float64(r.Score) * s.initialBudget
		}
	}

	return activationMap
}

func (s *SpreadingActivation) propagate(ctx context.Context, activationMap map[string]float64) map[string]float64 {
	newActivation := make(map[string]float64, len(activationMap))

	for k, v := range activationMap {
		newActivation[k] = v * s.decayFactor
	}

	for nodeID, score := range activationMap {
		if score < s.threshold {
			continue
		}

		relations, err := s.graphStore.GetEntityRelations(nodeID, "")
		if err != nil {
			continue
		}

		for _, rel := range relations {
			relScore := score * s.decayFactor
			newActivation[rel.ToID] += relScore
		}
	}

	for k, v := range newActivation {
		if v < s.threshold {
			delete(newActivation, k)
		}
	}

	return newActivation
}

func (s *SpreadingActivation) rankByActivation(ctx context.Context, activationMap map[string]float64) []ActivatedNode {
	var nodes []ActivatedNode

	for nodeID, score := range activationMap {
		if score >= s.threshold {
			entity, err := s.graphStore.GetEntity(nodeID)
			if err != nil {
				continue
			}

			nodes = append(nodes, ActivatedNode{
				ID:    nodeID,
				Label: entity.Name,
				Score: score,
				Hop:   0,
			})
		}
	}

	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].Score > nodes[j].Score
	})

	return nodes
}

type CompressionStats struct {
	AccuracyRetention   float64 `json:"accuracy_retention"`
	TokenReduction     float64 `json:"token_reduction"`
	TotalTokensSaved   int64   `json:"total_tokens_saved"`
	ExtractionsPerformed int64   `json:"extractions_performed"`
	SpreadingActivations int64   `json:"spreading_activations"`
	AvgLatencyMs       float64 `json:"avg_latency_ms"`
	P95LatencyMs      float64 `json:"p95_latency_ms"`
}

func NewCompressionStats() *CompressionStats {
	return &CompressionStats{
		AccuracyRetention:    0.0,
		TokenReduction:      0.0,
		TotalTokensSaved:    0,
		ExtractionsPerformed: 0,
		SpreadingActivations: 0,
		AvgLatencyMs:        0.0,
		P95LatencyMs:       0.0,
	}
}

type EmbeddingProvider interface {
	Embed(ctx context.Context, text string) ([]float32, error)
}

func GenerateQueryEmbedding(provider EmbeddingProvider, ctx context.Context, query string) ([]float32, error) {
	if provider == nil {
		return nil, nil
	}
	return provider.Embed(ctx, query)
}