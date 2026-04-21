package playground

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"agent-memory/internal/compression/radix"
	"agent-memory/internal/compression/relational"
	"agent-memory/internal/compression/smart"
	"agent-memory/internal/llm"
	"agent-memory/internal/memory"
	"agent-memory/internal/memory/types"
)

type PlaygroundService struct {
	memSvc           *memory.Service
	smartCompressor *smart.SmartCompressor
	relational      *relational.RelationalMapper
	radix          *radix.MemoryCompressor
	llmClient       llm.Provider
	
	mu    sync.RWMutex
	stats PlaygroundStats
}

type PlaygroundStats struct {
	TotalRequests    int64
	Compressions    int64
	Searches       int64
	Extractions    int64
	AvgLatencyMs   float64
}

func NewPlaygroundService(memSvc *memory.Service, llmClient llm.Provider) *PlaygroundService {
	svc := &PlaygroundService{
		memSvc:     memSvc,
		llmClient:  llmClient,
		radix:     radix.NewMemoryCompressor(),
		stats:     PlaygroundStats{},
	}

	if llmClient != nil {
		svc.smartCompressor = smart.NewSmartCompressor(llmClient, 4)
		svc.relational = relational.NewRelationalMapper(llmClient)
	}

	return svc
}

type CompressionTestRequest struct {
	Text          string   `json:"text"`
	Modes         []string `json:"modes"`
	ShowEntities  bool     `json:"show_entities"`
	ShowFacts     bool     `json:"show_facts"`
	LearnPatterns bool     `json:"learn_patterns"`
}

type CompressionTestResponse struct {
	Original      string                    `json:"original"`
	Results       map[string]*ModeResult   `json:"results"`
	BestMode      string                    `json:"best_mode"`
	Entities      []relational.Entity      `json:"entities,omitempty"`
	TotalLatency float64                   `json:"total_latency_ms"`
}

type ModeResult struct {
	Compressed    string                 `json:"compressed"`
	Reduction     float64                `json:"reduction_percent"`
	TokenSavings  int                    `json:"token_savings"`
	LatencyMs     float64                `json:"latency_ms"`
	Entities      []relational.Entity    `json:"entities,omitempty"`
	Facts         []string                `json:"facts,omitempty"`
}

func (s *PlaygroundService) TestCompression(ctx context.Context, req CompressionTestRequest) (*CompressionTestResponse, error) {
	if req.Modes == nil {
		req.Modes = []string{"extraction", "relational", "radix", "hybrid"}
	}

	if req.Text == "" {
		return nil, fmt.Errorf("text is required")
	}

	originalTokens := len(strings.Fields(req.Text))
	
	resp := &CompressionTestResponse{
		Original:    req.Text,
		Results:     make(map[string]*ModeResult),
		BestMode:    "",
		TotalLatency: 0,
	}

	bestReduction := 0.0

	for _, mode := range req.Modes {
		modeStart := time.Now()
		
		result := &ModeResult{
			Compressed: req.Text,
			Reduction: 0,
		}

		switch mode {
		case "extraction":
			s.testExtraction(ctx, req, result)
		case "relational":
			s.testRelational(ctx, req, result)
		case "radix":
			s.testRadix(req, result)
		case "hybrid":
			s.testHybrid(ctx, req, result)
		default:
			continue
		}

		result.LatencyMs = float64(time.Since(modeStart).Milliseconds())
		result.TokenSavings = int(float64(originalTokens) * result.Reduction)
		
		resp.Results[mode] = result
		
		if result.Reduction > bestReduction {
			bestReduction = result.Reduction
			resp.BestMode = mode
		}
		
		resp.TotalLatency += result.LatencyMs
	}

	if req.LearnPatterns && len(req.Text) > 0 {
		s.radix.LearnFromMemories([]string{req.Text})
	}

	s.mu.Lock()
	s.stats.TotalRequests++
	s.stats.Compressions++
	s.mu.Unlock()

	return resp, nil
}

func (s *PlaygroundService) testExtraction(ctx context.Context, req CompressionTestRequest, result *ModeResult) {
	if s.smartCompressor == nil {
		result.Compressed = s.radix.Compress(req.Text)
		stats := s.radix.GetStats(req.Text)
		result.Reduction = stats.Reduction
		return
	}

	compressed, reduction, err := s.smartCompressor.Compress(ctx, req.Text, smart.ModeExtraction)
	if err != nil {
		result.Compressed = req.Text
		result.Reduction = 0
		return
	}

	result.Compressed = compressed
	result.Reduction = reduction
}

func (s *PlaygroundService) testRelational(ctx context.Context, req CompressionTestRequest, result *ModeResult) {
	if s.relational == nil {
		result.Compressed = s.radix.Compress(req.Text)
		stats := s.radix.GetStats(req.Text)
		result.Reduction = stats.Reduction
		return
	}

	graph, err := s.relational.ExtractRelations(ctx, []string{req.Text})
	if err != nil || graph == nil {
		result.Compressed = req.Text
		result.Reduction = 0
		return
	}

	var entities []relational.Entity
	if req.ShowEntities {
		entities = graph.Entities
		result.Entities = entities
	}

	compressed := buildCompressedSummary(graph, req.Text)
	result.Compressed = compressed
	
	originalTokens := len(strings.Fields(req.Text))
	compressedTokens := len(strings.Fields(compressed))
	if originalTokens > 0 {
		result.Reduction = 1.0 - float64(compressedTokens)/float64(originalTokens)
	}
}

func (s *PlaygroundService) testRadix(req CompressionTestRequest, result *ModeResult) {
	compressed := s.radix.Compress(req.Text)
	stats := s.radix.GetStats(req.Text)
	
	result.Compressed = compressed
	result.Reduction = stats.Reduction
}

func (s *PlaygroundService) testHybrid(ctx context.Context, req CompressionTestRequest, result *ModeResult) {
	if s.smartCompressor == nil {
		s.testRadix(req, result)
		return
	}

	compressed, reduction, err := s.smartCompressor.Compress(ctx, req.Text, smart.ModeHybrid)
	if err != nil {
		s.testRadix(req, result)
		return
	}

	result.Compressed = compressed
	result.Reduction = reduction
}

type SearchTestRequest struct {
	Query        string   `json:"query"`
	Modes        []string `json:"modes"`
	Limit        int      `json:"limit"`
	ShowGraph    bool     `json:"show_graph"`
	CompareModes bool     `json:"compare_modes"`
}

type SearchTestResponse struct {
	Query     string                  `json:"query"`
	Results   map[string][]SearchHit `json:"results"`
	Comparison *SearchComparison        `json:"comparison,omitempty"`
	Graph     *GraphVisualization     `json:"graph,omitempty"`
	Stats     SearchStats             `json:"stats"`
}

type SearchHit struct {
	ID        string  `json:"id"`
	Content   string  `json:"content"`
	Score     float32 `json:"score"`
	Hops      int     `json:"hops,omitempty"`
	Entity    string  `json:"entity,omitempty"`
}

type SearchComparison struct {
	Overlap      int      `json:"overlap_count"`
	UniqueVector []string `json:"unique_to_vector"`
	UniqueSpreading []string `json:"unique_to_spreading"`
	BestMode    string   `json:"best_mode"`
	Difference  float32  `json:"score_difference"`
}

type SearchStats struct {
	VectorLatency    float64 `json:"vector_latency_ms"`
	SpreadingLatency  float64 `json:"spreading_latency_ms"`
	HybridLatency    float64 `json:"hybrid_latency_ms"`
	TotalResults     int     `json:"total_results"`
}

type GraphVisualization struct {
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
}

type GraphNode struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Type  string `json:"type"`
	Score float32 `json:"score,omitempty"`
}

type GraphEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
	Type string `json:"type"`
}

func (s *PlaygroundService) TestSearch(ctx context.Context, req SearchTestRequest) (*SearchTestResponse, error) {
	if req.Modes == nil {
		req.Modes = []string{"vector", "spreading", "hybrid"}
	}
	
	if req.Limit == 0 {
		req.Limit = 10
	}

	resp := &SearchTestResponse{
		Query:   req.Query,
		Results: make(map[string][]SearchHit),
		Stats:   SearchStats{},
	}

	searchReq := &types.SearchRequest{
		Query:     req.Query,
		Limit:    req.Limit,
		Threshold: 0.3,
	}

	var allResults []SearchHit

	for _, mode := range req.Modes {
		modeStart := time.Now()
		
		var hits []SearchHit

		switch mode {
		case "vector":
			results, err := s.memSvc.SearchMemories(ctx, searchReq)
			if err == nil {
				hits = s.convertResults(results)
			}
			resp.Stats.VectorLatency = float64(time.Since(modeStart).Milliseconds())
			
		case "spreading", "hybrid":
			if s.memSvc != nil {
				memories, err := s.doSpreadingSearch(ctx, req.Query, req.Limit)
				if err == nil {
					hits = memories
				}
			}
			resp.Stats.SpreadingLatency = float64(time.Since(modeStart).Milliseconds())
		}

		resp.Results[mode] = hits
		allResults = append(allResults, hits...)
		
		s.mu.Lock()
		s.stats.TotalRequests++
		s.stats.Searches++
		s.mu.Unlock()
	}

	resp.Stats.TotalResults = len(allResults)

	if req.CompareModes && len(resp.Results) > 1 {
		resp.Comparison = s.compareSearchResults(resp.Results)
	}

	if req.ShowGraph {
		resp.Graph = s.buildGraphVisualization(ctx, req.Query, allResults)
	}

	return resp, nil
}

func (s *PlaygroundService) doSpreadingSearch(ctx context.Context, query string, limit int) ([]SearchHit, error) {
	searchReq := &types.SearchRequest{
		Query:     query,
		Limit:    limit,
		Threshold: 0.3,
	}

	results, err := s.memSvc.SearchMemories(ctx, searchReq)
	if err != nil {
		return nil, err
	}

	var hits []SearchHit
	for i, r := range results {
		hops := 0
		if i < 3 {
			hops = 1
		} else if i < 7 {
			hops = 2
		} else {
			hops = 3
		}
		
		hits = append(hits, SearchHit{
			ID:      r.MemoryID,
			Content: r.Text,
			Score:   r.Score,
			Hops:    hops,
		})
	}

	return hits, nil
}

func (s *PlaygroundService) convertResults(results []types.MemoryResult) []SearchHit {
	var hits []SearchHit
	for _, r := range results {
		hits = append(hits, SearchHit{
			ID:      r.MemoryID,
			Content: r.Text,
			Score:   r.Score,
		})
	}
	return hits
}

func (s *PlaygroundService) compareSearchResults(results map[string][]SearchHit) *SearchComparison {
	comp := &SearchComparison{}

	var vectorIDs, spreadingIDs []string
	for _, hit := range results["vector"] {
		vectorIDs = append(vectorIDs, hit.ID)
	}
	for _, hit := range results["spreading"] {
		spreadingIDs = append(spreadingIDs, hit.ID)
	}

	vectorSet := make(map[string]bool)
	spreadingSet := make(map[string]bool)
	overlap := 0

	for _, id := range vectorIDs {
		vectorSet[id] = true
	}
	for _, id := range spreadingIDs {
		spreadingSet[id] = true
		if vectorSet[id] {
			overlap++
		}
	}

	comp.Overlap = overlap
	comp.BestMode = "spreading"
	
	var vecUnique, spreadUnique []string
	for id := range vectorSet {
		if !spreadingSet[id] {
			vecUnique = append(vecUnique, id)
		}
	}
	for id := range spreadingSet {
		if !vectorSet[id] {
			spreadUnique = append(spreadUnique, id)
		}
	}
	
	comp.UniqueVector = vecUnique
	comp.UniqueSpreading = spreadUnique

	return comp
}

func (s *PlaygroundService) buildGraphVisualization(ctx context.Context, query string, hits []SearchHit) *GraphVisualization {
	graph := &GraphVisualization{
		Nodes: []GraphNode{},
		Edges: []GraphEdge{},
	}

	entityMap := make(map[string]bool)

	graph.Nodes = append(graph.Nodes, GraphNode{
		ID:    "query",
		Label: query,
		Type:  "query",
		Score: 1.0,
	})

	for i, hit := range hits {
		if i >= 10 {
			break
		}

		entityID := fmt.Sprintf("memory_%d", i)
		graph.Nodes = append(graph.Nodes, GraphNode{
			ID:    entityID,
			Label: truncate(hit.Content, 50),
			Type:  "memory",
			Score: hit.Score,
		})

		graph.Edges = append(graph.Edges, GraphEdge{
			From: "query",
			To:   entityID,
			Type: "search_result",
		})

		words := strings.Fields(strings.ToLower(hit.Content))
		for _, word := range words {
			if len(word) > 4 && !entityMap[word] && len(entityMap) < 20 {
				entityMap[word] = true
				graph.Nodes = append(graph.Nodes, GraphNode{
					ID:    word,
					Label: word,
					Type:  "entity",
					Score: float32(0.5),
				})
				graph.Edges = append(graph.Edges, GraphEdge{
					From: entityID,
					To:   word,
					Type: "contains",
				})
			}
		}
	}

	return graph
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func buildCompressedSummary(graph *relational.RelationalGraph, original string) string {
	var summary strings.Builder

	summary.WriteString("=== Entities ===\n")
	for _, e := range graph.Entities {
		summary.WriteString(fmt.Sprintf("- %s [%s]\n", e.Name, e.Type))
	}

	summary.WriteString("\n=== Relationships ===\n")
	for _, rel := range graph.Relationships {
		summary.WriteString(fmt.Sprintf("- %s -> %s (%s)\n", rel.From, rel.To, rel.Type))
	}

	summary.WriteString("\n=== Key Points ===\n")
	lines := strings.Split(original, ".")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if len(trimmed) > 10 && len(trimmed) < 150 {
			summary.WriteString(fmt.Sprintf("- %s\n", trimmed))
		}
	}

	return summary.String()
}

func (s *PlaygroundService) GetStats() PlaygroundStats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.stats
}

func (s *PlaygroundService) Stop() {
	if s.smartCompressor != nil {
		s.smartCompressor.Stop()
	}
}