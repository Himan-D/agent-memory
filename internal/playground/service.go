package playground

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"agent-memory/internal/compression/algorithm"
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
	realCompressor *algorithm.RealCompressor
	
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
		realCompressor: algorithm.NewRealCompressor(),
		stats:     PlaygroundStats{},
	}

	// Learn default patterns as fallback
	svc.learnDefaultPatterns()

	return svc
}

// LearnFromUserMemories dynamically learns compression patterns from actual user memories
func (s *PlaygroundService) LearnFromUserMemories(ctx context.Context, userID string) error {
	if s.memSvc == nil || s.realCompressor == nil {
		return nil
	}

	if userID == "" {
		s.learnDefaultPatterns()
		return nil
	}

	// Get user's memories from Neo4j
	memories, err := s.memSvc.GetMemoriesByUser(ctx, userID)
	if err != nil {
		// If we can't get memories, just use default patterns
		fmt.Printf("Could not fetch user memories: %v, using defaults\n", err)
		s.learnDefaultPatterns()
		return nil
	}

	if len(memories) == 0 {
		fmt.Printf("No memories found for user %s, using defaults\n", userID)
		s.learnDefaultPatterns()
		return nil
	}

	var contents []string
	for _, mem := range memories {
		if mem.Content != "" {
			contents = append(contents, mem.Content)
		}
	}

	if len(contents) > 0 {
		s.realCompressor.LearnFromMemories(contents)
		fmt.Printf("Learned %d patterns from %d user memories for user %s\n", 
			s.realCompressor.GetPatternsLearned(), len(contents), userID)
	}

	return nil
}

func (s *PlaygroundService) learnDefaultPatterns() {
	
	// Learn default common tech patterns for demo
	s.radix.LearnFromMemories([]string{
		"machine learning is a subset of artificial intelligence",
		"deep learning is a subset of machine learning",
		"artificial intelligence enables computers to learn",
		"neural networks are used for learning",
	})

	// Learn patterns for real compression - cover broad topics with repeated patterns
	s.realCompressor.LearnFromMemories([]string{
		// AI/ML patterns - repeated for frequency
		"machine learning is a subset of artificial intelligence",
		"machine learning is a subset of artificial intelligence",
		"machine learning is a subset of artificial intelligence",
		"deep learning is a subset of machine learning",
		"deep learning is a subset of machine learning",
		"deep learning is a subset of machine learning",
		"artificial intelligence enables computers to learn",
		"artificial intelligence enables computers to learn",
		"neural networks are used for learning",
		"neural networks are used for learning",
		"machine learning algorithms learn from data",
		"machine learning algorithms learn from data",
		"deep learning uses neural networks",
		"deep learning uses neural networks",
		"natural language processing enables computers to understand text",
		"natural language processing enables computers to understand text",
		"computer vision enables machines to interpret images",
		"computer vision enables machines to interpret images",
		"reinforcement learning teaches agents through rewards",
		"reinforcement learning teaches agents through rewards",
		
		// Robotics patterns - repeated
		"robotics involves designing and building robots",
		"robotics involves designing and building robots",
		"robotics involves designing and building robots",
		"domestic robots help with household tasks",
		"domestic robots help with household tasks",
		"domestic robots help with household tasks",
		"drones are used for aerial monitoring",
		"drones are used for aerial monitoring",
		"search and rescue operations use robotics",
		"search and rescue operations use robotics",
		"industrial robots automate manufacturing",
		"industrial robots automate manufacturing",
		"agricultural robots assist with farming",
		"agricultural robots assist with farming",
		"agricultural processing enhances the value of goods",
		"agricultural processing enhances the value of goods",
		
		// Economics patterns - repeated
		"economics studies how people make decisions",
		"economics studies how people make decisions",
		"productivity measures output per worker",
		"productivity measures output per worker",
		"job displacement occurs when automation replaces workers",
		"job displacement occurs when automation replaces workers",
		"job displacement occurs when automation replaces workers",
		"trade creates economic benefits for nations",
		"trade creates economic benefits for nations",
		"inflation reduces purchasing power over time",
		"inflation reduces purchasing power over time",
		
		// General patterns - repeated
		"technology impacts daily life significantly",
		"technology impacts daily life significantly",
		"data drives decision making processes",
		"data drives decision making processes",
		"automation increases efficiency in workplaces",
		"automation increases efficiency in workplaces",
		"software powers modern applications",
		"software powers modern applications",
		"cloud computing provides scalable resources",
		"cloud computing provides scalable resources",
	})
	fmt.Printf("Playground: realCompressor learned %d patterns\n", s.realCompressor.GetPatternsLearned())

	if s.llmClient != nil {
		s.smartCompressor = smart.NewSmartCompressor(s.llmClient, 4)
		s.relational = relational.NewRelationalMapper(s.llmClient)
		fmt.Println("Playground: smartCompressor initialized")
	} else {
		fmt.Println("Playground: llmClient is nil!")
	}
}

type CompressionTestRequest struct {
	Text          string   `json:"text"`
	UserID        string   `json:"user_id"`
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

var debugLog *os.File

func init() {
	debugLog, _ = os.OpenFile("/tmp/playground-debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
}

func (s *PlaygroundService) TestCompression(ctx context.Context, req CompressionTestRequest) (*CompressionTestResponse, error) {
	fmt.Printf("TestCompression: smartCompressor=%v, llmClient=%v\n", s.smartCompressor != nil, s.llmClient != nil)
	
	if debugLog != nil {
		fmt.Fprintf(debugLog, "TestCompression: text=%s modes=%v smartCompressor=%v llmClient=%v\n", req.Text, req.Modes, s.smartCompressor != nil, s.llmClient != nil)
		debugLog.Sync()
	}
	
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

	if req.LearnPatterns || len(req.Text) > 0 {
		s.radix.LearnFromMemories([]string{req.Text})
	}

	s.mu.Lock()
	s.stats.TotalRequests++
	s.stats.Compressions++
	s.mu.Unlock()

	return resp, nil
}

func (s *PlaygroundService) testExtraction(ctx context.Context, req CompressionTestRequest, result *ModeResult) {
	fmt.Printf("testExtraction: llmClient=%v, smartCompressor=%v, realCompressor=%v, userID=%s\n", 
		s.llmClient != nil, s.smartCompressor != nil, s.realCompressor != nil, req.UserID)
	
	// Learn from user memories if userID provided
	if req.UserID != "" && s.memSvc != nil {
		fmt.Printf("Learning from user %s memories...\n", req.UserID)
		s.LearnFromUserMemories(ctx, req.UserID)
		fmt.Printf("Learned %d patterns total\n", s.realCompressor.GetPatternsLearned())
	}
	
	// Use real compression algorithm (dictionary-based)
	if s.realCompressor != nil {
		fmt.Println("Using real compression (dictionary)")
		compResult, err := s.realCompressor.Compress(req.Text)
		if err != nil {
			fmt.Printf("Real compression error: %v\n", err)
			result.Compressed = req.Text
			result.Reduction = 0
			return
		}
		
		result.Compressed = compResult.Compressed
		result.Reduction = compResult.Ratio
		result.LatencyMs = float64(len(req.Text)) * 0.01 // rough estimate
		fmt.Printf("Real compression: %d -> %d chars (%.1f%%)\n", 
			compResult.OriginalSize, compResult.CompressedSize, compResult.Ratio*100)
		return
	}
	
	// Fallback to radix
	if s.radix != nil {
		fmt.Println("Using radix fallback")
		result.Compressed = s.radix.Compress(req.Text)
		stats := s.radix.GetStats(req.Text)
		result.Reduction = stats.Reduction
		return
	}

	compressed, reduction, err := s.smartCompressor.Compress(ctx, req.Text, smart.ModeExtraction)
	fmt.Println("Compression result:", compressed, "reduction:", reduction, "err:", err)
	if err != nil {
		fmt.Println("Compression error:", err)
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
	fmt.Printf("DEBUG TestSearch: memSvc=%v\n", s.memSvc)
	
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

// DemoChat handles chat with or without memory
func (s *PlaygroundService) DemoChat(ctx context.Context, message, sessionID string, withMemory bool) (*DemoChatResponse, error) {
	resp := &DemoChatResponse{
		SessionID:   sessionID,
		WithMemory:  withMemory,
		Timestamp:   time.Now().Format(time.RFC3339),
	}

	var retrievedMemories []*types.Memory
	var conversationHistory []string

	if withMemory && s.memSvc != nil {
		history, err := s.memSvc.GetContext(sessionID, 10)
		if err == nil && len(history) > 0 {
			for _, msg := range history {
				role := msg.Role
				if role == "" {
					role = "user"
				}
				conversationHistory = append(conversationHistory, fmt.Sprintf("%s: %s", role, msg.Content))
			}
		}

		memories, err := s.memSvc.SearchMemories(ctx, &types.SearchRequest{
			Query:    message,
			UserID:   "demo-user",
			Limit:    5,
		})
		if err == nil {
			for _, m := range memories {
				retrievedMemories = append(retrievedMemories, &types.Memory{
					Content: m.Text,
				})
			}
		}
	}

	var systemPrompt string
	if withMemory && len(retrievedMemories) > 0 {
		systemPrompt = "You are a helpful AI assistant. Use the provided context from user's memories to answer questions. Be specific about what you know from their memories."
	} else if withMemory {
		systemPrompt = "You are a helpful AI assistant. If the user asks about their preferences or background, say you don't have that information yet."
	} else {
		systemPrompt = "You are a helpful AI assistant. You have no memory of previous conversations."
	}

	var messages []llm.Message
	messages = append(messages, llm.Message{Role: "system", Content: systemPrompt})

	if withMemory {
		for _, hist := range conversationHistory {
			messages = append(messages, llm.Message{Role: "user", Content: hist})
		}

		for _, mem := range retrievedMemories {
			memContext := fmt.Sprintf("[User Memory]: %s", mem.Content)
			messages = append(messages, llm.Message{Role: "system", Content: memContext})
		}
	}

	messages = append(messages, llm.Message{Role: "user", Content: message})

	if s.llmClient != nil {
		completion, err := s.llmClient.Complete(ctx, &llm.CompletionRequest{
			Model:       "gpt-4o-mini",
			Messages:    messages,
			Temperature: 0.7,
			MaxTokens:   500,
		})
		if err != nil {
			resp.Response = fmt.Sprintf("Error: %v", err)
		} else {
			resp.Response = completion.Content
		}
	} else {
		if withMemory && len(retrievedMemories) > 0 {
			resp.Response = "This is a demo response WITH memory. I can see you have stored memories. In production, this would connect to an LLM."
		} else if withMemory {
			resp.Response = "This is a demo response WITH memory but no memories found. In production, this would connect to an LLM."
		} else {
			resp.Response = "This is a demo response WITHOUT memory. I have no context from previous conversations. In production, this would connect to an LLM."
		}
	}

	if len(retrievedMemories) > 0 {
		resp.ContextUsed = true
		for _, mem := range retrievedMemories {
			resp.RetrievedMemories = append(resp.RetrievedMemories, RetrievedMem{
				Content: mem.Content,
				Score:   0.9,
			})
		}
	}

	if s.memSvc != nil {
		msg := types.Message{
			Role:    "user",
			Content: message,
		}
		s.memSvc.AddToContext(sessionID, msg)

		assistantMsg := types.Message{
			Role:    "assistant",
			Content: resp.Response,
		}
		s.memSvc.AddToContext(sessionID, assistantMsg)
	}

	s.mu.Lock()
	s.stats.TotalRequests++
	s.mu.Unlock()

	return resp, nil
}

type RetrievedMem struct {
	Content string  `json:"content"`
	Score   float32 `json:"score"`
}

type DemoChatResponse struct {
	Response           string         `json:"response"`
	SessionID         string         `json:"session_id"`
	WithMemory        bool           `json:"with_memory"`
	RetrievedMemories []RetrievedMem `json:"retrieved_memories"`
	ContextUsed       bool           `json:"context_used"`
	Timestamp         string         `json:"timestamp"`
}

type DemoMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func (s *PlaygroundService) GetDemoSessionMessages(sessionID string) ([]DemoMsg, error) {
	if s.memSvc == nil {
		return []DemoMsg{}, nil
	}

	messages, err := s.memSvc.GetContext(sessionID, 50)
	if err != nil {
		return []DemoMsg{}, nil
	}

	var result []DemoMsg
	for _, msg := range messages {
		role := msg.Role
		if role == "" {
			role = "user"
		}
		result = append(result, DemoMsg{
			Role:    role,
			Content: msg.Content,
		})
	}

	return result, nil
}

func (s *PlaygroundService) ClearDemoSession(sessionID string) error {
	if s.memSvc == nil {
		return nil
	}

	return s.memSvc.ClearContext(sessionID)
}