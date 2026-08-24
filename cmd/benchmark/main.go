package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"agent-memory/internal/compression/extractor"
	"agent-memory/internal/compression/pipeline"
	"agent-memory/internal/compression/retrieval"
	"agent-memory/internal/config"
	"agent-memory/internal/evaluation"
	"agent-memory/internal/llm"
	"agent-memory/internal/memory"
	"agent-memory/internal/memory/qdrant"
	"agent-memory/internal/memory/types"
	"agent-memory/internal/sources"
	"agent-memory/internal/tenant"

	"github.com/joho/godotenv"
)

type serviceAdapter struct {
	svc                 *memory.Service
	mode                string
	spreadingActivation *retrieval.SpreadingActivation
	searchLimit         int
	rrfK                int
	candidateLimit      int
}

// baseMemoryID strips the chunk suffix (_c0, _c1, …) added by
// chunkConversationMemory so that hit-rank tracking can match a retrieved
// chunk back to its source memory (e.g. "mem_0_c3" → "mem_0").
func baseMemoryID(id string) string {
	// Find the last "_c" followed only by digits.
	if idx := strings.LastIndex(id, "_c"); idx >= 0 {
		suffix := id[idx+2:]
		digitsOnly := true
		for _, r := range suffix {
			if r < '0' || r > '9' {
				digitsOnly = false
				break
			}
		}
		if digitsOnly && len(suffix) > 0 {
			return id[:idx]
		}
	}
	return id
}

func (a *serviceAdapter) CreateMemory(ctx context.Context, content, userID string) (string, error) {
	return a.CreateBenchmarkMemory(ctx, evaluation.BenchmarkMemory{
		ID:      uuid.New().String(),
		Content: content,
		UserID:  userID,
	})
}

func (a *serviceAdapter) CreateBenchmarkMemory(ctx context.Context, benchmarkMem evaluation.BenchmarkMemory) (string, error) {
	memID := benchmarkMem.ID
	if memID == "" {
		memID = uuid.New().String()
	}
	userID := benchmarkMem.UserID
	if userID == "" {
		userID = "benchmark-user"
	}
	mem := &types.Memory{
		ID:       memID,
		Content:  benchmarkMem.Content,
		UserID:   userID,
		OrgID:    "benchmark",
		TenantID: "benchmark",
		Type:     types.MemoryTypeUser,
		Metadata: map[string]interface{}{
			"seed_data":       true,
			"skip_processing": true, // store verbatim dialogue turns (no LLM rewrite)
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	// Ensure per-tenant Qdrant collection (agent_memory_benchmark) is used.
	ctx = tenant.WithContext(ctx, tenant.TenantContext{TenantID: "benchmark"})
	created, err := a.svc.CreateMemory(ctx, mem)
	if err != nil {
		return "", err
	}
	return created.ID, nil
}

func (a *serviceAdapter) GetMemories(ctx context.Context, sessionID string) ([]evaluation.MemoryResult, error) {
	return a.Search(ctx, sessionID, "")
}

func (a *serviceAdapter) Search(ctx context.Context, sessionID, query string) ([]evaluation.MemoryResult, error) {
	// LoCoMo conversations map 1:1 onto memory user_id (sample_id). Scoping search
	// to the conversation under test is required — without it, hybrid retrieval
	// returns cross-conversation distractors and Hit@k collapses.
	scopeUser := sessionID
	if scopeUser == "" {
		scopeUser = "benchmark-user"
	}
	// Per-tenant collection + org/user filters must match ingest path.
	ctx = tenant.WithContext(ctx, tenant.TenantContext{TenantID: "benchmark"})
	ctx = context.WithValue(ctx, retrieval.OrgIDContextKey, "benchmark")
	ctx = context.WithValue(ctx, retrieval.UserIDContextKey, scopeUser)

	// Multi-query expansion → retrieve each → RRF fuse → optional cross-encoder rerank.
	queries := evaluation.ExpandRetrievalQueries(query)
	if len(queries) == 0 {
		queries = []string{query}
	}

	perQueryLimit := a.searchLimit
	if perQueryLimit <= 0 {
		perQueryLimit = 40
	}
	lists := make([][]evaluation.MemoryResult, 0, len(queries)+2)
	var union []evaluation.MemoryResult
	seenUnion := map[string]struct{}{}
	for _, q := range queries {
		list, err := a.searchOnce(ctx, scopeUser, q, perQueryLimit)
		if err != nil {
			continue
		}
		if len(list) > 0 {
			lists = append(lists, list)
			for _, r := range list {
				if r.ID == "" {
					continue
				}
				if _, ok := seenUnion[r.ID]; ok {
					continue
				}
				seenUnion[r.ID] = struct{}{}
				union = append(union, r)
			}
		}
	}
	// Also run hybrid/spreading once on the original query when configured.
	if (a.mode == "spreading" || a.mode == "hybrid") && a.spreadingActivation != nil {
		if list, err := a.searchSpreading(ctx, scopeUser, query); err == nil && len(list) > 0 {
			lists = append(lists, list)
			for _, r := range list {
				if r.ID == "" {
					continue
				}
				if _, ok := seenUnion[r.ID]; ok {
					continue
				}
				seenUnion[r.ID] = struct{}{}
				union = append(union, r)
			}
		}
	}
	// Lexical ranking list for exact-term / name / date lift (Mem0-style multi-signal).
	if len(union) > 0 {
		if lex := evaluation.RankByLexical(query, union); len(lex) > 0 {
			lists = append(lists, lex)
		}
	}

	var results []evaluation.MemoryResult
	if len(lists) == 0 {
		return results, nil
	}
	results = evaluation.FuseRRF(lists, query, a.rrfK)
	results = deduplicateByID(results)

	// Cap candidate pool from config (0 = keep all fused hits).
	if a.candidateLimit > 0 && len(results) > a.candidateLimit {
		results = results[:a.candidateLimit]
	}

	// Cross-encoder / Cohere / LLM rerank when enabled; otherwise lexical precision@1.
	reranked := false
	if len(results) > 0 && a.svc.GetReranker() != nil && a.svc.GetReranker().Name() != "disabled" {
		memResults := make([]types.MemoryResult, len(results))
		for i, r := range results {
			memResults[i] = types.MemoryResult{
				MemoryID: r.ID,
				Text:     r.Content,
				Score:    r.Score,
			}
		}
		out, err := a.svc.GetReranker().Rerank(ctx, query, memResults, len(results))
		if err == nil && len(out) > 0 {
			results = make([]evaluation.MemoryResult, len(out))
			for i, r := range out {
				results[i] = evaluation.MemoryResult{
					ID:      baseMemoryID(r.MemoryID),
					Content: r.Text,
					Score:   r.Score,
				}
			}
			reranked = true
		}
	}
	if !reranked && len(results) > 0 {
		results = evaluation.RerankLexical(query, results, len(results))
	}

	return results, nil
}

func (a *serviceAdapter) searchOnce(ctx context.Context, scopeUser, query string, limit int) ([]evaluation.MemoryResult, error) {
	// limit <= 0 means "provider default" — leave 0 on the request when unset.
	searchResults, err := a.svc.SearchMemories(ctx, &types.SearchRequest{
		Query:     query,
		OrgID:     "benchmark",
		UserID:    scopeUser,
		TenantID:  "benchmark",
		Limit:     limit,
		Mode:      "vector",
		Threshold: 0,
		Rerank:    false,
	})
	if err != nil {
		return nil, err
	}
	out := make([]evaluation.MemoryResult, 0, len(searchResults))
	for _, result := range searchResults {
		content := result.Text
		if result.Metadata != nil {
			if result.Metadata.UserID != "" && result.Metadata.UserID != scopeUser {
				continue
			}
			if result.Metadata.Compressed != "" {
				content = result.Metadata.Compressed
			} else if result.Metadata.Content != "" {
				content = result.Metadata.Content
			}
		}
		if strings.TrimSpace(content) == "" {
			continue
		}
		out = append(out, evaluation.MemoryResult{
			ID:      baseMemoryID(result.MemoryID),
			Content: content,
			Score:   result.Score,
		})
	}
	return out, nil
}

func (a *serviceAdapter) searchSpreading(ctx context.Context, scopeUser, query string) ([]evaluation.MemoryResult, error) {
	if a.spreadingActivation == nil {
		return nil, nil
	}
	searchResults, err := a.spreadingActivation.RetrieveWithScores(ctx, query, retrieval.SearchMode(a.mode))
	if err != nil {
		return nil, err
	}
	out := make([]evaluation.MemoryResult, 0, len(searchResults))
	for _, result := range searchResults {
		if result.Memory == nil {
			continue
		}
		if result.Memory.UserID != "" && result.Memory.UserID != scopeUser {
			continue
		}
		content := result.Memory.Content
		if result.Memory.Compressed != "" {
			content = result.Memory.Compressed
		}
		if strings.TrimSpace(content) == "" {
			continue
		}
		out = append(out, evaluation.MemoryResult{
			ID:      baseMemoryID(result.Memory.ID),
			Content: content,
			Score:   float32(result.Score),
		})
	}
	return out, nil
}

func (a *serviceAdapter) CleanupBenchmarkMemories(ctx context.Context) error {
	// Match the tenant/collection used at ingest time.
	ctx = tenant.WithContext(ctx, tenant.TenantContext{TenantID: "benchmark"})

	// 1. Delete all memory and entity nodes in Neo4j
	if a.svc.GetGraph() != nil {
		_, err := a.svc.GetGraph().QueryGraph("MATCH (n:Memory) DETACH DELETE n", nil)
		if err != nil {
			fmt.Printf("warning: failed to delete Neo4j memories: %v\n", err)
		}
		_, err = a.svc.GetGraph().QueryGraph("MATCH (e:Entity) DETACH DELETE e", nil)
		if err != nil {
			fmt.Printf("warning: failed to delete Neo4j entities: %v\n", err)
		}
	}

	// 2. Delete all points in the benchmark tenant collection.
	// Empty filter matches nothing useful on some Qdrant builds — delete by org_id.
	if a.svc.GetVector() != nil {
		if qdrClient, ok := a.svc.GetVector().(*qdrant.Client); ok {
			if _, err := qdrClient.DeleteByFilter(ctx, map[string]interface{}{"org_id": "benchmark"}); err != nil {
				fmt.Printf("warning: failed to delete Qdrant points: %v\n", err)
			}
			if _, err := qdrClient.DeleteByFilter(ctx, map[string]interface{}{"tenant_id": "benchmark"}); err != nil {
				fmt.Printf("warning: failed to delete Qdrant tenant points: %v\n", err)
			}
		} else {
			fmt.Printf("warning: vector store is not a Qdrant client\n")
		}
	}

	return nil
}

func (a *serviceAdapter) Flush(ctx context.Context) error {
	return a.svc.FlushCompression(ctx)
}

// deduplicateByID collapses multiple results that share the same base memory ID
// (e.g. several chunks from the same source memory) into a single entry, keeping
// the one with the highest score. The relative order of first-seen IDs is preserved
// so that rank positions reflect the best chunk score per source memory.
func deduplicateByID(results []evaluation.MemoryResult) []evaluation.MemoryResult {
	seen := make(map[string]int, len(results)) // ID -> index in out
	out := make([]evaluation.MemoryResult, 0, len(results))
	for _, r := range results {
		if idx, exists := seen[r.ID]; exists {
			// Keep whichever chunk scored higher.
			if r.Score > out[idx].Score {
				out[idx] = r
			}
		} else {
			seen[r.ID] = len(out)
			out = append(out, r)
		}
	}
	// Re-sort by score descending so the best chunk per memory is ranked first.
	sort.Slice(out, func(i, j int) bool { return out[i].Score > out[j].Score })
	return out
}

type mockMemoryService struct {
	memories []evaluation.MemoryResult
}

func (m *mockMemoryService) CreateMemory(ctx context.Context, content, userID string) (string, error) {
	return m.CreateBenchmarkMemory(ctx, evaluation.BenchmarkMemory{
		ID:      uuid.New().String(),
		Content: content,
		UserID:  userID,
	})
}

func (m *mockMemoryService) CreateBenchmarkMemory(ctx context.Context, mem evaluation.BenchmarkMemory) (string, error) {
	id := mem.ID
	if id == "" {
		id = uuid.New().String()
	}
	m.memories = append(m.memories, evaluation.MemoryResult{ID: id, Content: mem.Content, Score: 1})
	return id, nil
}

type benchmarkSourceMemoryService struct {
	memories map[string]*types.Memory
}

func newBenchmarkSourceMemoryService() *benchmarkSourceMemoryService {
	return &benchmarkSourceMemoryService{memories: map[string]*types.Memory{}}
}

func (m *benchmarkSourceMemoryService) CreateMemory(ctx context.Context, mem *types.Memory) (*types.Memory, error) {
	cp := *mem
	m.memories[mem.ID] = &cp
	return &cp, nil
}

func (m *benchmarkSourceMemoryService) GetMemory(ctx context.Context, id string) (*types.Memory, error) {
	mem, ok := m.memories[id]
	if !ok {
		return nil, fmt.Errorf("memory %s not found", id)
	}
	cp := *mem
	return &cp, nil
}

func (m *benchmarkSourceMemoryService) DeleteMemory(ctx context.Context, id string) error {
	delete(m.memories, id)
	return nil
}

func (m *benchmarkSourceMemoryService) GetMemoriesByUser(ctx context.Context, userID string) ([]*types.Memory, error) {
	return m.filter(func(mem *types.Memory) bool { return mem.UserID == userID }), nil
}

func (m *benchmarkSourceMemoryService) GetMemoriesByOrg(ctx context.Context, orgID string) ([]*types.Memory, error) {
	return m.filter(func(mem *types.Memory) bool { return mem.OrgID == orgID }), nil
}

func (m *benchmarkSourceMemoryService) SearchMemories(ctx context.Context, req *types.SearchRequest) ([]types.MemoryResult, error) {
	matches := m.filter(func(mem *types.Memory) bool {
		return strings.Contains(strings.ToLower(mem.Content), strings.ToLower(req.Query))
	})
	results := make([]types.MemoryResult, 0, len(matches))
	for _, mem := range matches {
		results = append(results, types.MemoryResult{MemoryID: mem.ID, Text: mem.Content, Metadata: mem, Score: 1})
	}
	return results, nil
}

func (m *benchmarkSourceMemoryService) filter(match func(*types.Memory) bool) []*types.Memory {
	out := make([]*types.Memory, 0)
	for _, mem := range m.memories {
		if match(mem) {
			cp := *mem
			out = append(out, &cp)
		}
	}
	return out
}

type benchmarkBlobStore struct{}

func (benchmarkBlobStore) Put(ctx context.Context, key string, data []byte, contentType string) error {
	return nil
}

func (benchmarkBlobStore) Delete(ctx context.Context, key string) error {
	return nil
}

func (m *mockMemoryService) CleanupBenchmarkMemories(ctx context.Context) error {
	m.memories = nil
	return nil
}

func (m *mockMemoryService) Flush(ctx context.Context) error {
	return nil
}

func (m *mockMemoryService) GetMemories(ctx context.Context, sessionID string) ([]evaluation.MemoryResult, error) {
	return m.memories, nil
}

func (m *mockMemoryService) Search(ctx context.Context, sessionID, query string) ([]evaluation.MemoryResult, error) {
	queryTokens := tokenSet(query)
	results := make([]evaluation.MemoryResult, 0, len(m.memories))
	for _, mem := range m.memories {
		memTokens := tokenSet(mem.Content)
		var score float32
		for token := range queryTokens {
			if memTokens[token] {
				score++
			}
		}
		if score > 0 {
			mem.Score = score
			results = append(results, mem)
		}
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})
	if len(results) > 10 {
		results = results[:10]
	}
	return results, nil
}

func tokenSet(s string) map[string]bool {
	set := map[string]bool{}
	for _, field := range strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return r < 'a' || r > 'z'
	}) {
		if len(field) > 2 {
			set[field] = true
		}
	}
	return set
}

func main() {
	suite := flag.String("suite", "retrieval", "benchmark suite: retrieval, ingestion, all")
	dataset := flag.String("dataset", "all", "benchmark dataset: locomo, longmemeval, beam_1m, beam_10m, all")
	mode := flag.String("mode", "hybrid", "search mode passed to live backend: vector, hybrid, spreading")
	parallel := flag.Int("parallel", envInt("BENCHMARK_PARALLEL", 1), "concurrent question workers (env BENCHMARK_PARALLEL)")
	mock := flag.Bool("mock", false, "run against in-memory lexical mock instead of live stores")
	limit := flag.Int("limit", envInt("BENCHMARK_LIMIT", 0), "limit questions (0 = all; env BENCHMARK_LIMIT)")
	searchLimit := flag.Int("search-limit", envInt("BENCHMARK_SEARCH_LIMIT", 0), "hits per retrieval query (0 = service default; env BENCHMARK_SEARCH_LIMIT)")
	contextTopK := flag.Int("context-topk", envInt("BENCHMARK_CONTEXT_TOPK", 0), "memories fused into reader context (0 = same as search-limit; env BENCHMARK_CONTEXT_TOPK)")
	rrfK := flag.Int("rrf-k", envInt("BENCHMARK_RRF_K", 0), "RRF constant k (0 = 1/(rank+1); env BENCHMARK_RRF_K)")
	candidateLimit := flag.Int("candidate-limit", envInt("BENCHMARK_CANDIDATE_LIMIT", 0), "max fused candidates before rerank (0 = all; env BENCHMARK_CANDIDATE_LIMIT)")
	maxTokens := flag.Int("max-tokens", envInt("BENCHMARK_MAX_TOKENS", 0), "LLM max tokens for judge/reader (env BENCHMARK_MAX_TOKENS)")
	output := flag.String("output", "", "optional path to write JSON result")
	flag.Parse()

	ctx := context.Background()
	_ = godotenv.Load(".env")
	cfg := config.Load()
	cfg.Memory.ProcessingEnabled = true
	cfg.Compression.Enabled = true
	// Prefer sync compression so ingest finishes before search (async races the eval loop).
	cfg.Compression.AsyncEnabled = false

	// LLM-as-judge configuration (required for publishable results).
	// Priority: EVALUATOR_* → COMPRESSION_VERIFY_* → OPENAI/LLM_* from app config.
	evalApiKey := firstNonEmpty(
		os.Getenv("EVALUATOR_API_KEY"),
		cfg.Compression.VerifyAPIKey,
		os.Getenv("COMPRESSION_VERIFY_API_KEY"),
		cfg.OpenAI.APIKey,
		os.Getenv("OPENAI_API_KEY"),
		cfg.LLM.APIKey,
		os.Getenv("LLM_API_KEY"),
	)
	evalProvider := firstNonEmpty(
		os.Getenv("EVALUATOR_PROVIDER"),
		cfg.Compression.VerifyProvider,
		os.Getenv("LLM_PROVIDER"),
		string(cfg.LLM.Provider),
		"openai",
	)
	evalModel := firstNonEmpty(
		os.Getenv("EVALUATOR_MODEL"),
		cfg.Compression.VerifyModel,
		os.Getenv("OPENAI_MODEL"),
		cfg.OpenAI.Model,
		cfg.LLM.Model,
		"gpt-4o-mini",
	)
	evalBaseURL := firstNonEmpty(
		os.Getenv("EVALUATOR_BASE_URL"),
		cfg.Compression.VerifyBaseURL,
		cfg.OpenAI.BaseURL,
		cfg.LLM.BaseURL,
	)

	// Competitive defaults when unset (Mem0-style reader + retrieval depth).
	if *searchLimit <= 0 {
		*searchLimit = 40
	}
	if *contextTopK <= 0 {
		*contextTopK = 15
	}
	if *rrfK <= 0 {
		*rrfK = 60
	}
	if *candidateLimit <= 0 {
		*candidateLimit = 40
	}
	if *maxTokens <= 0 {
		*maxTokens = 256
	}

	benchCfg := evaluation.BenchmarkConfig{
		Model:           evalModel,
		MaxTokens:       *maxTokens,
		ParallelLimit:   *parallel,
		Limit:           *limit,
		ContextTopK:     *contextTopK,
		SearchLimit:     *searchLimit,
		RRFK:            *rrfK,
		GenerateAnswers: true,
	}

	var llmProvider llm.Provider
	if evalApiKey == "" {
		fmt.Fprintf(os.Stderr, "warning: no evaluator API key (set EVALUATOR_API_KEY or OPENAI_API_KEY); using token_f1 fallback\n")
	} else {
		provider, err := llm.NewProvider(&llm.Config{
			Provider: llm.ProviderType(evalProvider),
			APIKey:   evalApiKey,
			BaseURL:  evalBaseURL,
			OpenAI: llm.OpenAIConfig{
				Model:     evalModel,
				MaxTokens: 4096,
			},
			Anthropic: llm.AnthropicConfig{
				Model:     evalModel,
				MaxTokens: 4096,
			},
			Google: llm.GoogleConfig{
				Model:     evalModel,
				MaxTokens: 4096,
			},
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: evaluator LLM unavailable, using token_f1: %v\n", err)
		} else {
			llmProvider = provider
			fmt.Fprintf(os.Stderr, "LLM judge: provider=%s model=%s\n", evalProvider, evalModel)
		}
	}
	runner := evaluation.NewBenchmarkRunner(evaluation.NewScorer(llmProvider, benchCfg), benchCfg)

	var memSvc evaluation.MemoryService
	var searchFn evaluation.SearchFunc
	var sourceMemSvc sources.MemoryService
	var closeFn func()
	if *mock {
		mockSvc := &mockMemoryService{}
		memSvc = mockSvc
		searchFn = mockSvc.Search
		sourceMemSvc = newBenchmarkSourceMemoryService()
	} else {
		// Enable Cohere reranker when an API key is present. Otherwise rely on
		// evaluation.RerankLexical (fast, local) — per-doc LLM rerank is too slow
		// for full LoCoMo (~1.5k questions).
		if (strings.TrimSpace(cfg.Reranker.Provider) == "" || cfg.Reranker.Provider == "disabled") &&
			(os.Getenv("RERANKER_API_KEY") != "" || cfg.Reranker.APIKey != "") {
			cfg.Reranker.Provider = "cohere"
		}
		svc, err := memory.NewService(cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "init memory service: %v\n", err)
			os.Exit(1)
		}
		if r := svc.GetReranker(); r != nil && r.Name() != "disabled" {
			fmt.Fprintf(os.Stderr, "reranker: %s\n", r.Name())
		} else {
			fmt.Fprintf(os.Stderr, "reranker: lexical (local)\n")
		}
		if cfg.Compression.Enabled && cfg.Compression.AsyncEnabled && llmProvider != nil {
			memoryExtractor := extractor.NewMemoryExtractor(llmProvider)
			compressionPipeline := pipeline.NewCompressionPipeline(cfg.Compression.WorkerCount, memoryExtractor, nil)
			svc.SetCompressionPipeline(compressionPipeline)
			compressionPipeline.Start()
			defer compressionPipeline.Stop()
		}
		closeFn = func() { _ = svc.Close() }
		sa := retrieval.NewSpreadingActivationWithConfig(svc, retrieval.SpreadingConfig{
			InitialBudget: cfg.Compression.SpreadingInitialBudget,
			DecayFactor:   cfg.Compression.SpreadingDecayFactor,
			Threshold:     cfg.Compression.SpreadingThreshold,
			MaxHops:       cfg.Compression.SpreadingMaxHops,
		})
		adapter := &serviceAdapter{
			svc:                 svc,
			mode:                *mode,
			spreadingActivation: sa,
			searchLimit:         benchCfg.SearchLimit,
			rrfK:                benchCfg.RRFK,
			candidateLimit:      *candidateLimit,
		}
		if adapter.candidateLimit <= 0 && benchCfg.SearchLimit > 0 {
			// Derive from search limit when unset (no independent constant).
			adapter.candidateLimit = benchCfg.SearchLimit
		}
		memSvc = adapter
		searchFn = adapter.Search
		sourceMemSvc = svc
	}
	if closeFn != nil {
		defer closeFn()
	}

	var result any
	var err error
	switch *suite {
	case "retrieval":
		result, err = runRetrievalSuite(ctx, runner, memSvc, searchFn, *dataset)
	case "ingestion":
		result, err = runIngestionSuite(ctx, sourceMemSvc)
	case "all":
		retrieval, retrievalErr := runRetrievalSuite(ctx, runner, memSvc, searchFn, *dataset)
		ingestion, ingestionErr := runIngestionSuite(ctx, sourceMemSvc)
		result = map[string]any{"retrieval": retrieval, "ingestion": ingestion}
		if retrievalErr != nil {
			err = retrievalErr
		} else {
			err = ingestionErr
		}
	default:
		err = fmt.Errorf("unknown suite %q", *suite)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if *output != "" {
		if err := os.WriteFile(*output, data, 0o644); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	fmt.Println(string(data))
}

func runRetrievalSuite(ctx context.Context, runner *evaluation.BenchmarkRunner, memSvc evaluation.MemoryService, searchFn evaluation.SearchFunc, dataset string) (any, error) {
	switch dataset {
	case "locomo":
		return runner.RunLoCoMo(ctx, memSvc, searchFn)
	case "longmemeval":
		return runner.RunLongMemEval(ctx, memSvc, searchFn)
	case "beam_1m":
		return runner.RunBEAM(ctx, memSvc, searchFn, "1m")
	case "beam_10m":
		return runner.RunBEAM(ctx, memSvc, searchFn, "10m")
	case "all":
		return runner.RunAll(ctx, memSvc, searchFn), nil
	default:
		return nil, fmt.Errorf("unknown dataset %q", dataset)
	}
}

func runIngestionSuite(ctx context.Context, memSvc sources.MemoryService) (map[string]any, error) {
	svc := sources.NewService(memSvc, benchmarkBlobStore{}, sources.Config{ChunkMaxBytes: 512})
	start := time.Now()
	inputs := []sources.IngestRequest{
		{
			Type:    "text",
			Title:   "Support handbook",
			Content: strings.Repeat("Escalate enterprise incidents with customer context, source links, and runbook ownership. ", 20),
			OrgID:   "benchmark",
		},
		{
			Type:       "notion",
			Provider:   "notion",
			ExternalID: "notion-page-1",
			Title:      "Product FAQ",
			Content:    strings.Repeat("The product memory graph links users, sessions, files, and decisions for agent recall. ", 18),
			OrgID:      "benchmark",
		},
	}
	totalChunks := 0
	totalMemories := 0
	for _, input := range inputs {
		result, err := svc.Ingest(ctx, input)
		if err != nil {
			return nil, err
		}
		totalChunks += result.ChunksCreated
		totalMemories += result.MemoriesCreated
	}
	upload, err := svc.Upload(ctx, sources.UploadRequest{
		Filename:    "benchmark.txt",
		ContentType: "text/plain",
		Reader:      bytes.NewBufferString(strings.Repeat("Benchmark attachment ingestion keeps file attribution and chunk memory IDs. ", 16)),
		OrgID:       "benchmark",
	})
	if err != nil {
		return nil, err
	}
	totalChunks += upload.ChunksCreated
	totalMemories += upload.MemoriesCreated
	latency := time.Since(start)
	return map[string]any{
		"suite":            "ingestion",
		"sources_ingested": 3,
		"chunks_created":   totalChunks,
		"memories_created": totalMemories,
		"latency_ms":       float64(latency.Microseconds()) / 1000,
		"avg_chunks_per_source": func() float64 {
			if totalChunks == 0 {
				return 0
			}
			return float64(totalChunks) / 3
		}(),
		"storage": "blob-store",
	}, nil
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

// envInt reads an integer environment variable; returns fallback when unset or invalid.
func envInt(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return n
}
