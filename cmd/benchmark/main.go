package main

import (
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
)

type serviceAdapter struct {
	svc  *memory.Service
	mode string
}

func (a *serviceAdapter) CreateMemory(ctx context.Context, content, userID string) (string, error) {
	mem := &types.Memory{
		ID:        uuid.New().String(),
		Content:   content,
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
	id := uuid.New().String()
	m.memories = append(m.memories, evaluation.MemoryResult{ID: id, Content: content, Score: 1})
	return id, nil
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
	var closeFn func()
	if *mock {
		mockSvc := &mockMemoryService{}
		memSvc = mockSvc
		searchFn = mockSvc.Search
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
	}
	if closeFn != nil {
		defer closeFn()
	}

	var result any
	var err error
	switch *dataset {
	case "locomo":
		result, err = runner.RunLoCoMo(ctx, memSvc, searchFn)
	case "longmemeval":
		result, err = runner.RunLongMemEval(ctx, memSvc, searchFn)
	case "beam_1m":
		result, err = runner.RunBEAM(ctx, memSvc, searchFn, "1m")
	case "beam_10m":
		result, err = runner.RunBEAM(ctx, memSvc, searchFn, "10m")
	case "all":
		result = runner.RunAll(ctx, memSvc, searchFn)
	default:
		err = fmt.Errorf("unknown dataset %q", *dataset)
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
