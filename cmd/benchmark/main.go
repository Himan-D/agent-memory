package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
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

	"github.com/joho/godotenv"
)

type serviceAdapter struct {
	svc                 *memory.Service
	mode                string
	spreadingActivation *retrieval.SpreadingActivation
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
			"seed_data": true,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
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
	var results []evaluation.MemoryResult
	if (a.mode == "spreading" || a.mode == "hybrid") && a.spreadingActivation != nil {
		ctx = context.WithValue(ctx, retrieval.OrgIDContextKey, "benchmark")
		searchResults, err := a.spreadingActivation.RetrieveWithScores(ctx, query, retrieval.SearchMode(a.mode))
		if err != nil {
			return nil, err
		}
		for _, result := range searchResults {
			if result.Memory == nil {
				continue
			}
			content := result.Memory.Content
			if result.Memory.Compressed != "" {
				content = result.Memory.Compressed
			}
			results = append(results, evaluation.MemoryResult{
				ID:      baseMemoryID(result.Memory.ID),
				Content: content,
				Score:   float32(result.Score),
			})
		}
	} else {
		// Vector search
		searchResults, err := a.svc.SearchMemories(ctx, &types.SearchRequest{
			Query:  query,
			OrgID:  "benchmark",
			Limit:  10,
			Rerank: true, // Request reranking if available
		})
		if err != nil {
			return nil, err
		}
		for _, result := range searchResults {
			content := result.Text
			if result.Metadata != nil {
				if result.Metadata.Compressed != "" {
					content = result.Metadata.Compressed
				} else {
					content = result.Metadata.Content
				}
			}
			results = append(results, evaluation.MemoryResult{
				ID:      baseMemoryID(result.MemoryID),
				Content: content,
				Score:   result.Score,
			})
		}
	}

	// Deduplicate by base memory ID: multiple chunks from the same source
	// memory all resolve to the same ID after baseMemoryID(). Keep only the
	// highest-scored result per ID so rank positions aren't wasted on duplicates.
	// Do this BEFORE reranking so the reranker sees one entry per source memory.
	results = deduplicateByID(results)

	// Apply LLM/Cohere reranker for precise rank-1 ordering.
	// Embedding similarity is good for recall (finding the right memory in top-10)
	// but cross-attention reranking is far better for precision (rank 1 = best answer),
	// which directly improves hit_at_1 and MRR.
	if len(results) > 0 && a.svc.GetReranker() != nil {
		memResults := make([]types.MemoryResult, len(results))
		for i, r := range results {
			memResults[i] = types.MemoryResult{
				MemoryID: r.ID,
				Text:     r.Content,
				Score:    r.Score,
			}
		}
		reranked, err := a.svc.GetReranker().Rerank(ctx, query, memResults, len(results))
		if err == nil && len(reranked) > 0 {
			results = make([]evaluation.MemoryResult, len(reranked))
			for i, r := range reranked {
				results[i] = evaluation.MemoryResult{
					ID:      baseMemoryID(r.MemoryID), // IDs are already base IDs, but be safe
					Content: r.Text,
					Score:   r.Score,
				}
			}
		}
	}

	return results, nil
}

func (a *serviceAdapter) CleanupBenchmarkMemories(ctx context.Context) error {
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

	// 2. Delete all points in Qdrant
	if a.svc.GetVector() != nil {
		if qdrClient, ok := a.svc.GetVector().(*qdrant.Client); ok {
			_, err := qdrClient.DeleteByFilter(ctx, nil)
			if err != nil {
				fmt.Printf("warning: failed to delete Qdrant points: %v\n", err)
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
	parallel := flag.Int("parallel", 1, "number of concurrent requests (default 1 to prevent overloading local LLMs)")
	mock := flag.Bool("mock", false, "run against in-memory lexical mock instead of live stores")
	limit := flag.Int("limit", 0, "limit the number of questions to process (0 = all)")
	output := flag.String("output", "", "optional path to write JSON result")

	// Cognee-inspired wiring flags. All default to false so the existing
	// benchmark behavior is preserved. The -enable-wiring flag is the
	// master gate: every wiring flag below is silently ignored unless
	// -enable-wiring is also set.
	enableWiring := flag.Bool("enable-wiring", false, "MASTER GATE: must be set for any wiring flag to take effect. Prevents accidental enablement in production.")
	distill := flag.Bool("distill", false, "[requires -enable-wiring] run session distillation over ingested memories and report metrics")
	useBaseRetriever := flag.Bool("use-base-retriever", false, "[requires -enable-wiring] exercise retrieval.BaseRetriever against ingested memories and report hit@k")
	rollbackOnError := flag.Bool("rollback-on-error", false, "[requires -enable-wiring] probe the rollback.Ledger round-trip and report metrics")
	improve := flag.Bool("improve", false, "[requires -enable-wiring] run the 6-stage improve.Pipeline and report stage timings")
	distillTopK := flag.Int("distill-top-k", 0, "[requires -enable-wiring] cap the number of memories fed into the distiller (0 = all)")
	improveBuildGlob := flag.Bool("improve-build-global", false, "[requires -enable-wiring] include global_context_index stage in -improve")
	improveSyncCache := flag.Bool("improve-sync-cache", false, "[requires -enable-wiring] include sync_to_cache stage in -improve")
	flag.Parse()

	ctx := context.Background()
	_ = godotenv.Load(".env")
	cfg := config.Load()
	cfg.Memory.ProcessingEnabled = true
	cfg.Compression.Enabled = true
	// Disable async compression during benchmarks so ingestion blocks until LLM fact extraction completes!
	// If async is true, the benchmark starts searching before the background queue even finishes processing.
	evalApiKey := cfg.Compression.VerifyAPIKey
	if evalApiKey == "" {
		evalApiKey = os.Getenv("COMPRESSION_VERIFY_API_KEY")
	}
	if evalApiKey == "" {
		evalApiKey = os.Getenv("EVALUATOR_API_KEY")
	}
	evalProvider := cfg.Compression.VerifyProvider
	if evalProvider == "" {
		evalProvider = "google"
	}
	evalModel := cfg.Compression.VerifyModel
	if evalModel == "" {
		evalModel = "gemini-3.1-pro-preview"
	}
	evalBaseURL := cfg.Compression.VerifyBaseURL

	benchCfg := evaluation.BenchmarkConfig{
		Model:         evalModel,
		MaxTokens:     128,
		ParallelLimit: *parallel,
		Limit:         *limit,
	}

	var llmProvider llm.Provider
	if evalApiKey != "" {
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
			fmt.Fprintf(os.Stderr, "warning: evaluator LLM unavailable, using heuristic scoring: %v\n", err)
		} else {
			llmProvider = provider
		}
	}
	runner := evaluation.NewBenchmarkRunner(evaluation.NewScorer(llmProvider, benchCfg), benchCfg)

	var memSvc evaluation.MemoryService
	var searchFn evaluation.SearchFunc
	var sourceMemSvc sources.MemoryService
	var closeFn func()
	var svc *memory.Service // hoisted so RunWiring can reach it after the if/else
	if *mock {
		mockSvc := &mockMemoryService{}
		memSvc = mockSvc
		searchFn = mockSvc.Search
		sourceMemSvc = newBenchmarkSourceMemoryService()
	} else {
		var err error
		svc, err = memory.NewService(cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "init memory service: %v\n", err)
			os.Exit(1)
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

	// Cognee-inspired wiring: opt-in hooks that exercise the new session,
	// distillation, retrieval, rollback, and improvement packages alongside
	// the existing benchmark flow. Each flag is independent, but all are
	// gated behind -enable-wiring as a safety master switch.
	wiringOpts := WiringOptions{
		Distill:          *distill,
		UseBaseRetriever: *useBaseRetriever,
		RollbackOnError:  *rollbackOnError,
		Improve:          *improve,
		DistillTopK:      *distillTopK,
		ImproveBuildGlob: *improveBuildGlob,
		ImproveSyncCache: *improveSyncCache,
	}
	if *enableWiring && (wiringOpts.Distill || wiringOpts.UseBaseRetriever || wiringOpts.RollbackOnError || wiringOpts.Improve) {
		wm := RunWiring(ctx, svc, llmProvider, wiringOpts)
		// Merge into result. If result is a map, add a top-level "wiring"
		// key. Otherwise wrap result so the wiring report survives JSON
		// serialization.
		switch r := result.(type) {
		case map[string]any:
			r["wiring"] = wm
			result = r
		default:
			result = map[string]any{
				"result": r,
				"wiring": wm,
			}
		}
	} else if !*enableWiring && (*distill || *useBaseRetriever || *rollbackOnError || *improve) {
		// Surface a clear warning so silent no-ops don't confuse operators.
		fmt.Fprintln(os.Stderr, "warning: -enable-wiring not set; ignoring wiring flags. Add -enable-wiring to activate them.")
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
