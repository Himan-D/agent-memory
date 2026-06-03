package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"agent-memory/internal/config"
	"agent-memory/internal/evaluation"
	"agent-memory/internal/llm"
	"agent-memory/internal/memory"
	"agent-memory/internal/memory/types"
)

type memServiceAdapter struct {
	svc *memory.Service
}

func (m *memServiceAdapter) CreateMemory(ctx context.Context, content, userID string) (string, error) {
	mem := &types.Memory{
		Content: content,
		UserID:  userID,
	}
	created, err := m.svc.CreateMemory(ctx, mem)
	if err != nil {
		return "", err
	}
	return created.ID, nil
}

func (m *memServiceAdapter) GetMemories(ctx context.Context, sessionID string) ([]evaluation.MemoryResult, error) {
	req := &types.SearchRequest{
		Query: sessionID,
		Limit: 10,
	}
	results, err := m.svc.SearchMemories(ctx, req)
	if err != nil {
		return nil, err
	}
	var out []evaluation.MemoryResult
	for _, r := range results {
		content := ""
		if r.Metadata != nil {
			content = r.Metadata.Content
		}
		out = append(out, evaluation.MemoryResult{
			ID:      r.MemoryID,
			Content: content,
			Score:   r.Score,
		})
	}
	return out, nil
}

func main() {
	dataset := flag.String("dataset", "all", "Benchmark dataset: longmemeval, locomo, beam, all")
	mode := flag.String("mode", "hybrid", "Search mode: vector, spreading, hybrid")
	output := flag.String("output", "results", "Output directory for results")
	flag.Parse()

	cfg := config.Load()
	svc, err := memory.NewService(cfg)
	if err != nil {
		log.Fatalf("Failed to create memory service: %v", err)
	}
	defer svc.Close()

	llmProvider, err := llm.NewProvider(&llm.Config{
		Provider: llm.ProviderType(cfg.LLM.Provider),
		APIKey:   cfg.LLM.APIKey,
		BaseURL:  cfg.LLM.BaseURL,
	})
	if err != nil {
		log.Fatalf("Failed to create LLM provider: %v", err)
	}

	benchCfg := evaluation.BenchmarkConfig{
		Model:         cfg.LLM.Model,
		MaxTokens:     cfg.LLM.MaxTokens,
		ParallelLimit: 10,
	}

	scorer := evaluation.NewScorer(llmProvider, benchCfg)
	runner := evaluation.NewBenchmarkRunner(scorer, benchCfg)

	memSvc := &memServiceAdapter{svc: svc}
	searchFn := func(ctx context.Context, sessionID, query string) ([]evaluation.MemoryResult, error) {
		req := &types.SearchRequest{
			Query: query,
			Limit: 10,
		}
		results, err := svc.SearchMemories(ctx, req)
		if err != nil {
			return nil, err
		}
		var out []evaluation.MemoryResult
		for _, r := range results {
			content := ""
			if r.Metadata != nil {
				content = r.Metadata.Content
			}
			out = append(out, evaluation.MemoryResult{
				ID:      r.MemoryID,
				Content: content,
				Score:   r.Score,
			})
		}
		return out, nil
	}

	fmt.Printf("Running benchmarks: dataset=%s mode=%s\n", *dataset, *mode)
	start := time.Now()

	var results []*evaluation.BenchmarkResult
	ctx := context.Background()

	switch *dataset {
	case "longmemeval":
		r, err := runner.RunLongMemEval(ctx, memSvc, searchFn)
		if err != nil {
			log.Fatalf("LongMemEval failed: %v", err)
		}
		results = append(results, r)
	case "locomo":
		r, err := runner.RunLoCoMo(ctx, memSvc, searchFn)
		if err != nil {
			log.Fatalf("LoCoMo failed: %v", err)
		}
		results = append(results, r)
	case "beam":
		r, err := runner.RunBEAM(ctx, memSvc, searchFn, "1m")
		if err != nil {
			log.Fatalf("BEAM failed: %v", err)
		}
		results = append(results, r)
	case "all":
		all := runner.RunAll(ctx, memSvc, searchFn)
		if all.LoCoMo != nil {
			results = append(results, all.LoCoMo)
		}
		if all.LongMemEval != nil {
			results = append(results, all.LongMemEval)
		}
		if all.ESMemEval != nil {
			results = append(results, all.ESMemEval)
		}
		if all.BEAM1M != nil {
			results = append(results, all.BEAM1M)
		}
		if all.BEAM10M != nil {
			results = append(results, all.BEAM10M)
		}
	default:
		log.Fatalf("Unknown dataset: %s", *dataset)
	}

	elapsed := time.Since(start)
	fmt.Printf("\nBenchmarks completed in %s\n\n", elapsed)

	if err := os.MkdirAll(*output, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal results: %v", err)
	}

	outFile := fmt.Sprintf("%s/benchmark_%s_%s_%d.json", *output, *dataset, *mode, time.Now().Unix())
	if err := os.WriteFile(outFile, data, 0644); err != nil {
		log.Fatalf("Failed to write results: %v", err)
	}

	fmt.Printf("Results written to %s\n\n", outFile)
	fmt.Println("| Dataset | Mode | Overall Score | Single-Hop | Multi-Hop | P50 Latency | P95 Latency | Tokens Retrieved |")
	fmt.Println("|----------|------|--------------|------------|-----------|-------------|-------------|-------------------|")
	for _, r := range results {
		fmt.Printf("| %s | %s | %.2f | %.2f | %.2f | %.0fms | %.0fms | %d |\n",
			r.Dataset, *mode, r.OverallScore, r.SingleHopScore, r.MultiHopScore, r.LatencyP50Ms, r.LatencyP95Ms, r.TokensRetrieved)
	}
}