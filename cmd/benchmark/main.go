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

	"agent-memory/internal/config"
	"agent-memory/internal/evaluation"
	"agent-memory/internal/llm"
	"agent-memory/internal/memory"
	"agent-memory/internal/memory/types"
	"agent-memory/internal/sources"
)

type serviceAdapter struct {
	svc  *memory.Service
	mode string
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
		ID:        memID,
		Content:   benchmarkMem.Content,
		UserID:    userID,
		OrgID:     "benchmark",
		TenantID:  "benchmark",
		Type:      types.MemoryTypeUser,
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
	results, err := a.svc.SearchMemories(ctx, &types.SearchRequest{
		Query: query,
		OrgID: "benchmark",
		Limit: 10,
		Mode:  a.mode,
	})
	if err != nil {
		return nil, err
	}
	out := make([]evaluation.MemoryResult, 0, len(results))
	for _, result := range results {
		content := result.Text
		if content == "" && result.Metadata != nil {
			content = result.Metadata.Content
		}
		out = append(out, evaluation.MemoryResult{
			ID:      result.MemoryID,
			Content: content,
			Score:   result.Score,
		})
	}
	return out, nil
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
	mock := flag.Bool("mock", false, "run against in-memory lexical mock instead of live stores")
	output := flag.String("output", "", "optional path to write JSON result")
	flag.Parse()

	ctx := context.Background()
	cfg := config.Load()
	benchCfg := evaluation.BenchmarkConfig{
		Model:         cfg.LLM.Model,
		MaxTokens:     16,
		ParallelLimit: 4,
	}

	var llmProvider llm.Provider
	if cfg.LLM.APIKey != "" {
		provider, err := llm.NewProvider(&llm.Config{
			Provider: llm.ProviderType(cfg.LLM.Provider),
			APIKey:   cfg.LLM.APIKey,
			BaseURL:  cfg.LLM.BaseURL,
			OpenAI: llm.OpenAIConfig{
				Model:     cfg.LLM.Model,
				MaxTokens: cfg.LLM.MaxTokens,
			},
			Anthropic: llm.AnthropicConfig{
				Model:     cfg.LLM.Model,
				MaxTokens: cfg.LLM.MaxTokens,
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
	if *mock {
		mockSvc := &mockMemoryService{}
		memSvc = mockSvc
		searchFn = mockSvc.Search
		sourceMemSvc = newBenchmarkSourceMemoryService()
	} else {
		svc, err := memory.NewService(cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "init memory service: %v\n", err)
			os.Exit(1)
		}
		closeFn = func() { _ = svc.Close() }
		adapter := &serviceAdapter{svc: svc, mode: *mode}
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
