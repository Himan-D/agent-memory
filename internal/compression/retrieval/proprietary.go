package retrieval

import (
	"context"
	"fmt"
	"math"
	"time"

	"agent-memory/internal/memory"
	"agent-memory/internal/memory/types"
)

type MemoryService interface {
	GetGraph() memory.GraphStore
	GetVector() memory.VectorStore
	SearchMemories(ctx context.Context, req *types.SearchRequest) ([]types.MemoryResult, error)
}

// SpreadingMetrics is the subset of metrics.MetricsCollector needed here.
type SpreadingMetrics interface {
	RecordSpreadingActivation(hops int)
}

type SpreadingActivation struct {
	memSvc        MemoryService
	graphStore    memory.GraphStore
	vectorStore   memory.VectorStore
	initialBudget float64
	decayFactor   float64
	threshold     float64
	maxHops       int
	metrics       SpreadingMetrics
}

func (s *SpreadingActivation) SetMetrics(m SpreadingMetrics) {
	s.metrics = m
}

type ActivationResult struct {
	Nodes        []ActivatedNode
	TotalScore   float64
	HopBreakdown []int
}

type ActivatedNode struct {
	ID       string
	Label    string
	Score    float64
	Hop      int
	MemoryID string
}

type SearchMode string

const (
	SearchModeVector    SearchMode = "vector"
	SearchModeSpreading SearchMode = "spreading"
	SearchModeHybrid    SearchMode = "hybrid"
)

func NewSpreadingActivation(memSvc MemoryService) *SpreadingActivation {
	return &SpreadingActivation{
		memSvc:        memSvc,
		graphStore:    memSvc.GetGraph(),
		vectorStore:   memSvc.GetVector(),
		initialBudget: 1.0,
		decayFactor:   0.85,
		threshold:     0.1,
		maxHops:       3,
	}
}

// SpreadingConfig holds config-driven hyperparameters for spreading activation.
type SpreadingConfig struct {
	InitialBudget float64
	DecayFactor   float64
	Threshold     float64
	MaxHops       int
}

// NewSpreadingActivationWithConfig creates a SpreadingActivation using config-driven hyperparameters.
func NewSpreadingActivationWithConfig(memSvc MemoryService, cfg SpreadingConfig) *SpreadingActivation {
	sa := NewSpreadingActivation(memSvc)
	if cfg.InitialBudget > 0 {
		sa.initialBudget = cfg.InitialBudget
	}
	if cfg.DecayFactor > 0 {
		sa.decayFactor = cfg.DecayFactor
	}
	if cfg.Threshold > 0 {
		sa.threshold = cfg.Threshold
	}
	if cfg.MaxHops > 0 {
		sa.maxHops = cfg.MaxHops
	}
	return sa
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

type RetrieveResult struct {
	Memory *types.Memory
	Score  float64
	Hops   int
}

func (s *SpreadingActivation) RetrieveWithScores(ctx context.Context, query string, mode SearchMode) ([]RetrieveResult, error) {
	switch mode {
	case SearchModeSpreading:
		return s.retrieveSpreadingWithScores(ctx, query)
	case SearchModeHybrid:
		return s.retrieveHybridWithScores(ctx, query)
	default:
		return s.retrieveVectorWithScores(ctx, query)
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
		if r.MemoryID != "" {
			mem, err := s.graphStore.GetMemory(r.MemoryID)
			if err == nil {
				memories = append(memories, mem)
			} else {
				memories = append(memories, &types.Memory{
					ID:      r.MemoryID,
					Content: r.Text,
				})
			}
		}
	}

	return memories, nil
}

func (s *SpreadingActivation) retrieveVectorWithScores(ctx context.Context, query string) ([]RetrieveResult, error) {
	memories, err := s.retrieveVector(ctx, query)
	if err != nil {
		return nil, err
	}
	results := make([]RetrieveResult, len(memories))
	for i, m := range memories {
		results[i] = RetrieveResult{Memory: m, Score: 0.7, Hops: 0}
	}
	return results, nil
}

func (s *SpreadingActivation) retrieveSpreadingWithScores(ctx context.Context, query string) ([]RetrieveResult, error) {
	if s.memSvc == nil {
		return s.retrieveVectorWithScores(ctx, query)
	}

	req := &types.SearchRequest{
		Query:     query,
		Limit:     50,
		Threshold: 0.3,
	}
	initialResults, err := s.memSvc.SearchMemories(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("vector search: %w", err)
	}

	if len(initialResults) == 0 {
		return s.retrieveVectorWithScores(ctx, query)
	}

	activationMap := s.initializeActivationWithHops(initialResults)

	hasGraphConnections := false
	for hop := 0; hop < s.maxHops; hop++ {
		newMap := s.propagate(ctx, activationMap)
		if len(newMap) > len(activationMap) {
			hasGraphConnections = true
		}
		activationMap = newMap
	}

	ranked := s.rankByActivation(ctx, activationMap)

	if s.metrics != nil {
		s.metrics.RecordSpreadingActivation(s.maxHops)
	}

	if len(ranked) == 0 || !hasGraphConnections {
		return s.retrieveVectorWithScores(ctx, query)
	}

	var results []RetrieveResult
	for _, r := range ranked {
		mem, err := s.graphStore.GetMemory(r.ID)
		if err != nil {
			mem = &types.Memory{ID: r.ID, Content: r.Label}
		}
		results = append(results, RetrieveResult{
			Memory: mem,
			Score:  r.Score,
			Hops:   r.Hop,
		})
	}

	if len(results) == 0 {
		return s.retrieveVectorWithScores(ctx, query)
	}

	return results, nil
}

func (s *SpreadingActivation) retrieveHybridWithScores(ctx context.Context, query string) ([]RetrieveResult, error) {
	if s.memSvc == nil {
		return nil, fmt.Errorf("memory service not configured")
	}

	vectorReq := &types.SearchRequest{
		Query:     query,
		Limit:     25,
		Threshold: 0.7,
	}
	vectorResults, err := s.memSvc.SearchMemories(ctx, vectorReq)
	if err != nil {
		return nil, fmt.Errorf("vector search: %w", err)
	}

	seen := make(map[string]bool)
	var results []RetrieveResult

	for _, r := range vectorResults {
		if r.Metadata != nil && !seen[r.Metadata.ID] {
			seen[r.Metadata.ID] = true
			results = append(results, RetrieveResult{
				Memory: r.Metadata,
				Score:  float64(r.Score),
				Hops:   0,
			})
		}
	}

	spreadingResults, err := s.retrieveSpreadingWithScores(ctx, query)
	if err != nil {
		return results, nil
	}

	for _, r := range spreadingResults {
		if r.Memory != nil && !seen[r.Memory.ID] {
			seen[r.Memory.ID] = true
			results = append(results, r)
		}
	}

	return results, nil
}

func (s *SpreadingActivation) retrieveSpreading(ctx context.Context, query string) ([]*types.Memory, error) {
	if s.memSvc == nil {
		return s.retrieveVector(ctx, query)
	}

	req := &types.SearchRequest{
		Query:     query,
		Limit:     50,
		Threshold: 0.3,
	}
	initialResults, err := s.memSvc.SearchMemories(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("vector search: %w", err)
	}

	if len(initialResults) == 0 {
		return s.retrieveVector(ctx, query)
	}

	activationMap := s.initializeActivationWithHops(initialResults)

	hasGraphConnections := false
	for hop := 0; hop < s.maxHops; hop++ {
		newMap := s.propagate(ctx, activationMap)
		if len(newMap) > len(activationMap) {
			hasGraphConnections = true
		}
		activationMap = newMap
	}

	results := s.rankByActivation(ctx, activationMap)

	if s.metrics != nil {
		s.metrics.RecordSpreadingActivation(s.maxHops)
	}

	if len(results) == 0 || !hasGraphConnections {
		return s.retrieveVector(ctx, query)
	}

	var memories []*types.Memory
	for _, r := range results {
		if r.MemoryID != "" {
			mem, err := s.graphStore.GetMemory(r.MemoryID)
			if err == nil {
				memories = append(memories, mem)
			} else {
				memories = append(memories, &types.Memory{
					ID:      r.MemoryID,
					Content: "Memory content not found",
				})
			}
		}
	}

	if len(memories) == 0 {
		return s.retrieveVector(ctx, query)
	}

	return memories, nil
}

func (s *SpreadingActivation) retrieveHybrid(ctx context.Context, query string) ([]*types.Memory, error) {
	if s.memSvc == nil {
		return nil, fmt.Errorf("memory service not configured")
	}

	vectorReq := &types.SearchRequest{
		Query:     query,
		Limit:     25,
		Threshold: 0.7,
	}
	vectorResults, err := s.memSvc.SearchMemories(ctx, vectorReq)
	if err != nil {
		return nil, fmt.Errorf("vector search: %w", err)
	}

	var vectorMemories []*types.Memory
	for _, r := range vectorResults {
		if r.MemoryID != "" {
			mem, err := s.graphStore.GetMemory(r.MemoryID)
			if err == nil {
				vectorMemories = append(vectorMemories, mem)
			}
		} else if r.Metadata != nil {
			vectorMemories = append(vectorMemories, r.Metadata)
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

func (s *SpreadingActivation) initializeActivationWithHops(results []types.MemoryResult) map[string]ActivationNode {
	activationMap := make(map[string]ActivationNode)

	for _, r := range results {
		memID := r.MemoryID
		if memID == "" && r.Metadata != nil {
			memID = r.Metadata.ID
		}
		if memID != "" {
			activationMap[memID] = ActivationNode{
				Score: float64(r.Score) * s.initialBudget,
				Hop:   0,
			}
		}
	}

	return activationMap
}

type ActivationNode struct {
	Score    float64
	Hop      int
	MemoryID string
}

// propagate implements the SYNAPSE spreading activation algorithm (arXiv:2601.02744):
// - Temporal decay: older memories lose activation strength
// - Edge-type weights: SIMILAR_TO > RELATES_TO > CONTRADICTS
// - Lateral inhibition: highly-connected nodes are suppressed to prevent echo chambers
func (s *SpreadingActivation) propagate(ctx context.Context, activationMap map[string]ActivationNode) map[string]ActivationNode {
	newActivation := make(map[string]ActivationNode)
	neighborCounts := make(map[string]int)

	for nodeID, node := range activationMap {
		if node.Score < s.threshold {
			continue
		}

		temporalScore := node.Score * s.computeTemporalDecay(nodeID)

		if temporalScore >= s.threshold {
			if existing, ok := newActivation[nodeID]; !ok || temporalScore > existing.Score {
				newActivation[nodeID] = ActivationNode{Score: temporalScore, Hop: node.Hop, MemoryID: node.MemoryID}
			}
		}

		neighbors := s.getNeighborMemories(ctx, nodeID)
		nextHop := node.Hop + 1
		for _, neighbor := range neighbors {
			neighborCounts[neighbor.MemoryID]++
			relScore := temporalScore * s.decayFactor * s.edgeWeight(neighbor.RelType)
			if relScore < s.threshold {
				continue
			}
			if existing, ok := newActivation[neighbor.MemoryID]; !ok || relScore > existing.Score {
				newActivation[neighbor.MemoryID] = ActivationNode{Score: relScore, Hop: nextHop, MemoryID: neighbor.MemoryID}
			}
		}
	}

	for nodeID, count := range neighborCounts {
		if count > 3 {
			if node, ok := newActivation[nodeID]; ok {
				node.Score *= 1.0 / math.Log(float64(count)+1)
				newActivation[nodeID] = node
			}
		}
	}

	return newActivation
}

type neighborEdge struct {
	MemoryID string
	RelType  string
}

func (s *SpreadingActivation) getNeighborMemories(ctx context.Context, memoryID string) []neighborEdge {
	mem, err := s.graphStore.GetMemory(memoryID)
	if err != nil || mem == nil {
		return nil
	}

	entityIDs := []string{}
	if mem.EntityID != "" {
		entityIDs = append(entityIDs, mem.EntityID)
	}
	if len(entityIDs) == 0 {
		entities, err := s.graphStore.GetEntitiesByMemory(memoryID)
		if err == nil {
			for _, entity := range entities {
				if entity.ID != "" {
					entityIDs = append(entityIDs, entity.ID)
				}
			}
		}
	}
	if len(entityIDs) == 0 {
		return nil
	}

	var neighbors []neighborEdge
	for _, entityID := range entityIDs {
		relations, err := s.graphStore.GetEntityRelations(entityID, "")
		if err != nil || len(relations) == 0 {
			continue
		}
		for _, rel := range relations {
			peerEntityID := rel.ToID
			peerMemIDs, err := s.graphStore.GetMemoryIDsByEntity(peerEntityID)
			if err == nil {
				for _, id := range peerMemIDs {
					if id == memoryID {
						continue
					}
					neighbors = append(neighbors, neighborEdge{MemoryID: id, RelType: rel.Type})
				}
			}
		}
	}
	return neighbors
}

// computeTemporalDecay returns e^(-λ * hours_since_access) for a node.
// λ=0.01 gives half-life of ~70 hours (memories accessed 3 days ago retain ~50% activation).
func (s *SpreadingActivation) computeTemporalDecay(nodeID string) float64 {
	mem, err := s.graphStore.GetMemory(nodeID)
	if err != nil || mem == nil {
		return 1.0 // unknown age → no decay
	}
	hoursSince := time.Since(mem.UpdatedAt).Hours()
	if hoursSince < 0 {
		hoursSince = 0
	}
	return math.Exp(-0.01 * hoursSince)
}

// edgeWeight maps relationship types to spreading strength multipliers.
func (s *SpreadingActivation) edgeWeight(relType string) float64 {
	switch relType {
	case "SIMILAR_TO":
		return 0.9
	case "RELATES_TO", "RELATED_TO", "MENTIONS":
		return 0.8
	case "CONTRADICTS":
		return 0.3
	default:
		return 0.5
	}
}

func (s *SpreadingActivation) rankByActivation(ctx context.Context, activationMap map[string]ActivationNode) []ActivatedNode {
	var nodes []ActivatedNode

	for nodeID, node := range activationMap {
		if node.Score >= s.threshold {
			var label string
			var memoryID string

			if node.MemoryID != "" {
				memoryID = node.MemoryID
				mem, err := s.graphStore.GetMemory(memoryID)
				if err == nil {
					label = mem.Content[:min(50, len(mem.Content))]
				} else {
					label = nodeID
				}
			} else {
				entity, err := s.graphStore.GetEntity(nodeID)
				if err == nil {
					label = entity.Name
				} else {
					label = nodeID
				}
			}

			nodes = append(nodes, ActivatedNode{
				ID:       nodeID,
				Label:    label,
				Score:    node.Score,
				Hop:      node.Hop,
				MemoryID: memoryID,
			})
		}
	}

	for i := range nodes {
		for j := i + 1; j < len(nodes); j++ {
			if nodes[j].Score > nodes[i].Score {
				nodes[i], nodes[j] = nodes[j], nodes[i]
			}
		}
	}

	return nodes
}

type CompressionStats struct {
	AccuracyRetention    float64 `json:"accuracy_retention"`
	TokenReduction       float64 `json:"token_reduction"`
	TotalTokensSaved     int64   `json:"total_tokens_saved"`
	ExtractionsPerformed int64   `json:"extractions_performed"`
	SpreadingActivations int64   `json:"spreading_activations"`
	AvgLatencyMs         float64 `json:"avg_latency_ms"`
	P95LatencyMs         float64 `json:"p95_latency_ms"`
}

func NewCompressionStats() *CompressionStats {
	return &CompressionStats{
		AccuracyRetention:    0.0,
		TokenReduction:       0.0,
		TotalTokensSaved:     0,
		ExtractionsPerformed: 0,
		SpreadingActivations: 0,
		AvgLatencyMs:         0.0,
		P95LatencyMs:         0.0,
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
