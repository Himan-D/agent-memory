package retrieval

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"agent-memory/internal/memory"
	"agent-memory/internal/memory/types"
	"agent-memory/internal/tenant"
)

type contextKey string

const OrgIDContextKey contextKey = "org_id"
const UserIDContextKey contextKey = "user_id"

func (s *SpreadingActivation) getOrgID(ctx context.Context) string {
	if val := ctx.Value(OrgIDContextKey); val != nil {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func (s *SpreadingActivation) getUserID(ctx context.Context) string {
	if val := ctx.Value(UserIDContextKey); val != nil {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

// searchRequest builds a SearchRequest that preserves org/user scope from context.
// Mode "vector" forces the filtered vector path (skips multi-signal which ignores UserID).
func (s *SpreadingActivation) searchRequest(query string, limit int, threshold float32, ctx context.Context) *types.SearchRequest {
	if limit <= 0 {
		limit = 50
	}
	req := &types.SearchRequest{
		Query:     query,
		Limit:     limit,
		Threshold: threshold,
		OrgID:     s.getOrgID(ctx),
		UserID:    s.getUserID(ctx),
		Mode:      "vector",
		Rerank:    false,
	}
	// Propagate tenant so Qdrant uses agent_memory_{tenant} (same as ingest).
	if tid := tenant.IDFromContext(ctx); tid != "" {
		req.TenantID = tid
	}
	return req
}

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

	req := s.searchRequest(query, 50, 0.0, ctx)
	results, err := s.memSvc.SearchMemories(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("vector search: %w", err)
	}

	var memories []*types.Memory
	if len(results) > 0 {
		// Batch-fetch all memories by ID
		var ids []string
		for _, r := range results {
			if r.MemoryID != "" {
				ids = append(ids, r.MemoryID)
			}
		}
		memByID := make(map[string]*types.Memory)
		if len(ids) > 0 {
			if fetched, err := s.graphStore.GetMemoriesByIDs(ids); err == nil {
				for _, m := range fetched {
					memByID[m.ID] = m
				}
			}
		}
		for _, r := range results {
			if r.MemoryID != "" {
				if m, ok := memByID[r.MemoryID]; ok {
					memories = append(memories, m)
				} else {
					memories = append(memories, &types.Memory{
						ID:      r.MemoryID,
						Content: r.Text,
					})
				}
			}
		}
	}

	return memories, nil
}

func (s *SpreadingActivation) retrieveVectorWithScores(ctx context.Context, query string) ([]RetrieveResult, error) {
	if s.memSvc == nil {
		return nil, fmt.Errorf("memory service not configured")
	}

	req := s.searchRequest(query, 50, 0.0, ctx)
	results, err := s.memSvc.SearchMemories(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("vector search: %w", err)
	}

	var out []RetrieveResult
	if len(results) > 0 {
		// Batch-fetch all memories by ID
		var ids []string
		for _, r := range results {
			if r.MemoryID != "" {
				ids = append(ids, r.MemoryID)
			}
		}
		memByID := make(map[string]*types.Memory)
		if len(ids) > 0 {
			if fetched, err := s.graphStore.GetMemoriesByIDs(ids); err == nil {
				for _, m := range fetched {
					memByID[m.ID] = m
				}
			}
		}
		for _, r := range results {
			var mem *types.Memory
			if r.MemoryID != "" {
				if m, ok := memByID[r.MemoryID]; ok {
					mem = m
				} else {
					mem = &types.Memory{ID: r.MemoryID, Content: r.Text}
				}
			} else if r.Metadata != nil {
				mem = r.Metadata
			}
			if mem != nil {
				out = append(out, RetrieveResult{Memory: mem, Score: float64(r.Score), Hops: 0})
			}
		}
	}
	return out, nil
}

func (s *SpreadingActivation) retrieveSpreadingWithScores(ctx context.Context, query string) ([]RetrieveResult, error) {
	if s.memSvc == nil {
		return s.retrieveVectorWithScores(ctx, query)
	}

	req := s.searchRequest(query, 50, 0.0, ctx)
	initialResults, err := s.memSvc.SearchMemories(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("vector search: %w", err)
	}

	if len(initialResults) == 0 {
		return s.retrieveVectorWithScores(ctx, query)
	}

	activationMap := s.initializeActivationWithHops(initialResults)

	actualHops := 0
	for hop := 0; hop < s.maxHops; hop++ {
		newMap := s.propagate(ctx, activationMap)
		activationMap = newMap
		// Check if any node reached via graph traversal (Hop > 0) is present.
		for _, node := range activationMap {
			if node.Hop > actualHops {
				actualHops = node.Hop
			}
		}
	}

	ranked := s.rankByActivation(ctx, activationMap)

	if s.metrics != nil {
		s.metrics.RecordSpreadingActivation(actualHops)
	}

	hasGraphConnections := actualHops > 0
	if len(ranked) == 0 || !hasGraphConnections {
		return s.retrieveVectorWithScores(ctx, query)
	}

	var results []RetrieveResult
	if len(ranked) > 0 {
		// Batch-fetch all result memories in one query
		var rankedIDs []string
		for _, r := range ranked {
			rankedIDs = append(rankedIDs, r.ID)
		}
		memByID := make(map[string]*types.Memory)
		if fetched, err := s.graphStore.GetMemoriesByIDs(rankedIDs); err == nil {
			for _, m := range fetched {
				memByID[m.ID] = m
			}
		}
		for _, r := range ranked {
			mem, ok := memByID[r.ID]
			if !ok {
				mem = &types.Memory{ID: r.ID, Content: r.Label}
			}
			results = append(results, RetrieveResult{
				Memory: mem,
				Score:  r.Score,
				Hops:   r.Hop,
			})
		}
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

	// Prefer the robust vector path (uses Text/MemoryID even when graph Metadata is nil).
	// Previously hybrid required r.Metadata != nil and dropped every hit after Qdrant/Neo4j
	// drift — that produced empty retrieval on live LoCoMo runs.
	vectorResults, err := s.retrieveVectorWithScores(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("vector search: %w", err)
	}

	seen := make(map[string]bool)
	var results []RetrieveResult
	for _, r := range vectorResults {
		if r.Memory == nil {
			continue
		}
		if seen[r.Memory.ID] {
			continue
		}
		seen[r.Memory.ID] = true
		results = append(results, r)
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

	// If graph-dependent spreading yielded nothing but vector had hits, keep vector hits.
	if len(results) == 0 {
		return s.retrieveVectorWithScores(ctx, query)
	}
	return results, nil
}

func (s *SpreadingActivation) retrieveSpreading(ctx context.Context, query string) ([]*types.Memory, error) {
	if s.memSvc == nil {
		return s.retrieveVector(ctx, query)
	}

	req := s.searchRequest(query, 50, 0.0, ctx)
	initialResults, err := s.memSvc.SearchMemories(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("vector search: %w", err)
	}

	if len(initialResults) == 0 {
		return s.retrieveVector(ctx, query)
	}

	activationMap := s.initializeActivationWithHops(initialResults)

	actualHops := 0
	for hop := 0; hop < s.maxHops; hop++ {
		newMap := s.propagate(ctx, activationMap)
		activationMap = newMap
		for _, node := range activationMap {
			if node.Hop > actualHops {
				actualHops = node.Hop
			}
		}
	}

	results := s.rankByActivation(ctx, activationMap)

	if s.metrics != nil {
		s.metrics.RecordSpreadingActivation(actualHops)
	}

	if len(results) == 0 || actualHops == 0 {
		return s.retrieveVector(ctx, query)
	}

	var memories []*types.Memory
	if len(results) > 0 {
		// Batch-fetch all result memories in one query
		var resultIDs []string
		for _, r := range results {
			if r.MemoryID != "" {
				resultIDs = append(resultIDs, r.MemoryID)
			}
		}
		if len(resultIDs) > 0 {
			fetched, err := s.graphStore.GetMemoriesByIDs(resultIDs)
			if err == nil {
				memByID := make(map[string]*types.Memory, len(fetched))
				for _, m := range fetched {
					memByID[m.ID] = m
				}
				for _, r := range results {
					if r.MemoryID != "" {
						if m, ok := memByID[r.MemoryID]; ok {
							memories = append(memories, m)
						} else {
							memories = append(memories, &types.Memory{
								ID:      r.MemoryID,
								Content: "Memory content not found",
							})
						}
					}
				}
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

	vectorReq := s.searchRequest(query, 25, 0.0, ctx)
	vectorResults, err := s.memSvc.SearchMemories(ctx, vectorReq)
	if err != nil {
		return nil, fmt.Errorf("vector search: %w", err)
	}

	var vectorMemories []*types.Memory
	if len(vectorResults) > 0 {
		var ids []string
		for _, r := range vectorResults {
			if r.MemoryID != "" {
				ids = append(ids, r.MemoryID)
			}
		}
		memByID := make(map[string]*types.Memory)
		if len(ids) > 0 {
			if fetched, err := s.graphStore.GetMemoriesByIDs(ids); err == nil {
				for _, m := range fetched {
					memByID[m.ID] = m
				}
			}
		}
		for _, r := range vectorResults {
			if r.MemoryID != "" {
				if m, ok := memByID[r.MemoryID]; ok {
					vectorMemories = append(vectorMemories, m)
				}
			} else if r.Metadata != nil {
				vectorMemories = append(vectorMemories, r.Metadata)
			}
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

	// Collect IDs that need timestamp lookup
	var idsToFetch []string
	idToScore := make(map[string]float64)

	for _, r := range results {
		memID := r.MemoryID
		if memID == "" && r.Metadata != nil {
			memID = r.Metadata.ID
		}
		if memID == "" {
			continue
		}
		score := float64(r.Score) * s.initialBudget
		idToScore[memID] = score

		// Use timestamp from vector result metadata if available
		if r.Metadata != nil && !r.Metadata.UpdatedAt.IsZero() {
			activationMap[memID] = ActivationNode{
				Score:     score,
				Hop:       0,
				MemoryID:  memID,
				UpdatedAt: r.Metadata.UpdatedAt,
			}
		} else {
			idsToFetch = append(idsToFetch, memID)
		}
	}

	// Batch-fetch timestamps for IDs not in vector metadata
	if len(idsToFetch) > 0 {
		memories, err := s.graphStore.GetMemoriesByIDs(idsToFetch)
		if err == nil {
			tsMap := make(map[string]time.Time, len(memories))
			for _, m := range memories {
				tsMap[m.ID] = m.UpdatedAt
			}
			for _, id := range idsToFetch {
				activationMap[id] = ActivationNode{
					Score:     idToScore[id],
					Hop:       0,
					MemoryID:  id,
					UpdatedAt: tsMap[id],
				}
			}
		} else {
			// Fallback: use zero timestamps (no decay applied)
			for _, id := range idsToFetch {
				activationMap[id] = ActivationNode{
					Score:    idToScore[id],
					Hop:      0,
					MemoryID: id,
				}
			}
		}
	}

	return activationMap
}

type ActivationNode struct {
	Score     float64
	Hop       int
	MemoryID  string
	UpdatedAt time.Time
}

// propagate implements the SYNAPSE spreading activation algorithm (arXiv:2601.02744):
// - Temporal decay: older memories lose activation strength
// - Edge-type weights: SIMILAR_TO > RELATES_TO > CONTRADICTS
// - Lateral inhibition: highly-connected nodes are suppressed to prevent echo chambers
//
// OPT-5: Uses batch Cypher query to fetch all neighbors in a single round-trip
// instead of per-node GetMemory + GetEntitiesByMemory + GetEntityRelations + GetMemoryIDsByEntity.
func (s *SpreadingActivation) propagate(ctx context.Context, activationMap map[string]ActivationNode) map[string]ActivationNode {
	newActivation := make(map[string]ActivationNode)
	neighborCounts := make(map[string]int)

	// Collect active node IDs for batch neighbor prefetch
	var activeIDs []string
	activeNodes := make(map[string]ActivationNode)
	for nodeID, node := range activationMap {
		if node.Score < s.threshold {
			continue
		}
		temporalScore := node.Score * s.computeTemporalDecayFromTime(node.UpdatedAt)
		if temporalScore < s.threshold {
			continue
		}
		// Store decayed score for this hop
		decayed := node
		decayed.Score = temporalScore
		activeNodes[nodeID] = decayed

		if existing, ok := newActivation[nodeID]; !ok || temporalScore > existing.Score {
			newActivation[nodeID] = ActivationNode{Score: temporalScore, Hop: node.Hop, MemoryID: node.MemoryID, UpdatedAt: node.UpdatedAt}
		}
		activeIDs = append(activeIDs, nodeID)
	}

	if len(activeIDs) == 0 {
		return newActivation
	}

	// Batch-fetch all neighbors in a single Cypher query
	allNeighbors := s.batchGetNeighbors(ctx, activeIDs)

	// Process neighbors in-memory
	for _, edge := range allNeighbors {
		sourceNode, ok := activeNodes[edge.SourceID]
		if !ok {
			continue
		}
		neighborCounts[edge.MemoryID]++
		relScore := sourceNode.Score * s.decayFactor * s.edgeWeight(edge.RelType)
		if relScore < s.threshold {
			continue
		}
		nextHop := sourceNode.Hop + 1
		if existing, ok := newActivation[edge.MemoryID]; !ok || relScore > existing.Score {
			newActivation[edge.MemoryID] = ActivationNode{Score: relScore, Hop: nextHop, MemoryID: edge.MemoryID}
		}
	}

	// Batch-fetch timestamps for newly discovered neighbors
	var newIDs []string
	for id, node := range newActivation {
		if node.UpdatedAt.IsZero() {
			newIDs = append(newIDs, id)
		}
	}
	if len(newIDs) > 0 {
		memories, err := s.graphStore.GetMemoriesByIDs(newIDs)
		if err == nil {
			for _, m := range memories {
				if node, ok := newActivation[m.ID]; ok {
					node.UpdatedAt = m.UpdatedAt
					newActivation[m.ID] = node
				}
			}
		}
	}

	// Lateral inhibition: suppress highly-connected nodes
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

// neighborEdge with source tracking for batch processing.
type batchNeighborEdge struct {
	SourceID string
	MemoryID string
	RelType  string
}

// batchGetNeighbors fetches all memory-to-memory neighbors for a set of memory IDs
// in a single Cypher query via QueryGraph, replacing N * (3-4 graph calls) with 1 query.
func (s *SpreadingActivation) batchGetNeighbors(ctx context.Context, memoryIDs []string) []batchNeighborEdge {
	if len(memoryIDs) == 0 {
		return nil
	}

	query := `
		MATCH (m:Memory)<-[:MEMORY_OF]-(e:Entity)-[:MEMORY_OF]->(peerMem:Memory)
		WHERE m.id IN $memoryIDs AND peerMem.id <> m.id
		RETURN m.id AS source_id, peerMem.id AS neighbor_id, "SAME_ENTITY" AS rel_type
		UNION
		MATCH (m:Memory)<-[:MEMORY_OF]-(e:Entity)-[r]-(peer:Entity)-[:MEMORY_OF]->(peerMem:Memory)
		WHERE m.id IN $memoryIDs AND peerMem.id <> m.id
		RETURN m.id AS source_id, peerMem.id AS neighbor_id, type(r) AS rel_type
	`

	records, err := s.graphStore.QueryGraph(query, map[string]interface{}{"memoryIDs": memoryIDs})
	if err != nil {
		return nil
	}

	edges := make([]batchNeighborEdge, 0, len(records))
	for _, rec := range records {
		sourceID, _ := rec["source_id"].(string)
		neighborID, _ := rec["neighbor_id"].(string)
		relType, _ := rec["rel_type"].(string)
		if sourceID != "" && neighborID != "" {
			edges = append(edges, batchNeighborEdge{
				SourceID: sourceID,
				MemoryID: neighborID,
				RelType:  relType,
			})
		}
	}
	return edges
}

// computeTemporalDecayFromTime computes e^(-λ * hours_since) from a stored timestamp.
// No graph call needed — timestamp is carried in the ActivationNode.
func (s *SpreadingActivation) computeTemporalDecayFromTime(updatedAt time.Time) float64 {
	if updatedAt.IsZero() {
		return 1.0 // unknown age → no decay
	}
	hoursSince := time.Since(updatedAt).Hours()
	if hoursSince < 0 {
		hoursSince = 0
	}
	return math.Exp(-0.01 * hoursSince)
}

// edgeWeight maps relationship types to spreading strength multipliers.
func (s *SpreadingActivation) edgeWeight(relType string) float64 {
	switch relType {
	case "SAME_ENTITY":
		return 1.0
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
	// Collect all memory IDs that need label lookup
	var memIDs []string
	for nodeID, node := range activationMap {
		if node.Score >= s.threshold {
			if node.MemoryID != "" {
				memIDs = append(memIDs, nodeID)
			}
		}
	}

	// Batch-fetch memory content for labels
	labelMap := make(map[string]string)
	if len(memIDs) > 0 {
		memories, err := s.graphStore.GetMemoriesByIDs(memIDs)
		if err == nil {
			for _, m := range memories {
				if len(m.Content) > 50 {
					labelMap[m.ID] = m.Content[:50]
				} else {
					labelMap[m.ID] = m.Content
				}
			}
		}
	}

	var nodes []ActivatedNode
	for nodeID, node := range activationMap {
		if node.Score >= s.threshold {
			var label string
			var memoryID string

			if node.MemoryID != "" {
				memoryID = node.MemoryID
				if l, ok := labelMap[nodeID]; ok {
					label = l
				} else {
					label = nodeID
				}
			} else {
				label = nodeID
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

	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].Score > nodes[j].Score
	})

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
