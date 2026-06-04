package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
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
		content := r.Text
		if content == "" && r.Metadata != nil {
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

func getGitSHA() string {
	sha, err := os.ReadFile(".git/HEAD")
	if err != nil {
		return "unknown"
	}
	ref := strings.TrimSpace(string(sha))
	if strings.HasPrefix(ref, "ref: ") {
		refPath := strings.TrimPrefix(ref, "ref: ")
		refData, err := os.ReadFile(filepath.Join(".git", refPath))
		if err != nil {
			return "unknown"
		}
		return strings.TrimSpace(string(refData))
	}
	return ref
}

func main() {
	dataset := flag.String("dataset", "all", "Benchmark dataset: longmemeval, locomo, es_memeval, beam, all")
	mode := flag.String("mode", "hybrid", "Search mode: vector, spreading, hybrid")
	output := flag.String("output", "benchmarks", "Output directory for results")
	concurrency := flag.Int("concurrency", 10, "Parallel question limit")
	maxQuestions := flag.Int("max-questions", 0, "Max questions per dataset (0 = all)")
	deterministic := flag.Bool("deterministic", false, "Use fixed seed and temperature 0 for reproducibility")
	synthesize := flag.Bool("synthesize", false, "Synthesize answer from top-K retrieved memories via LLM")
	synthesisTopK := flag.Int("synthesis-topk", 5, "Number of memories to feed into answer synthesizer")
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
		Model:          cfg.LLM.Model,
		MaxTokens:      cfg.LLM.MaxTokens,
		ParallelLimit:  *concurrency,
		MaxQuestions:   *maxQuestions,
		Deterministic:  *deterministic,
		UseSynthesis:   *synthesize,
		SynthesisTopK:  *synthesisTopK,
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
			content := r.Text
			if content == "" && r.Metadata != nil {
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

	fmt.Printf("Running benchmarks: dataset=%s mode=%s synthesize=%v\n", *dataset, *mode, *synthesize)
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
	case "es_memeval":
		r, err := runner.RunESMemEval(ctx, memSvc, searchFn)
		if err != nil {
			log.Fatalf("ESMemEval failed: %v", err)
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

	date := time.Now().Format("2006-01-02")
	fullSHA := getGitSHA()
	shortSHA := fullSHA
	if len(shortSHA) > 12 {
		shortSHA = shortSHA[:12]
	}
	runDir := filepath.Join(*output, fmt.Sprintf("%s-%s", date, shortSHA))
	if err := os.MkdirAll(runDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal results: %v", err)
	}

	outFile := filepath.Join(runDir, fmt.Sprintf("results_%s_%s.json", *dataset, *mode))
	if err := os.WriteFile(outFile, data, 0644); err != nil {
		log.Fatalf("Failed to write results: %v", err)
	}

	mdPath := filepath.Join(runDir, fmt.Sprintf("results_%s_%s.md", *dataset, *mode))
	mdContent := fmt.Sprintf("# Benchmark Results: %s\n- **Dataset**: %s\n- **Mode**: %s\n- **Date**: %s\n- **Git SHA**: %s\n- **Concurrency**: %d\n\n| Dataset | Mode | Overall Score | Single-Hop | Multi-Hop | P50 Latency | P95 Latency | Tokens Retrieved |\n|----------|------|--------------|------------|-----------|-------------|-------------|-------------------|\n",
		time.Now().Format(time.RFC3339), *dataset, *mode, time.Now().Format(time.RFC3339), shortSHA, *concurrency)
	for _, r := range results {
		mdContent += fmt.Sprintf("| %s | %s | %.2f | %.2f | %.2f | %.0fms | %.0fms | %d |\n",
			r.Dataset, *mode, r.OverallScore, r.SingleHopScore, r.MultiHopScore, r.LatencyP50Ms, r.LatencyP95Ms, r.TokensRetrieved)
	}
	if err := os.WriteFile(mdPath, []byte(mdContent), 0644); err != nil {
		log.Fatalf("Failed to write markdown: %v", err)
	}

	fmt.Printf("Results written to %s\n\n", runDir)
	fmt.Println("| Dataset | Mode | Overall Score | Single-Hop | Multi-Hop | P50 Latency | P95 Latency | Tokens Retrieved |")
	fmt.Println("|----------|------|--------------|------------|-----------|-------------|-------------|-------------------|")
	for _, r := range results {
		fmt.Printf("| %s | %s | %.2f | %.2f | %.2f | %.0fms | %.0fms | %d |\n",
			r.Dataset, *mode, r.OverallScore, r.SingleHopScore, r.MultiHopScore, r.LatencyP50Ms, r.LatencyP95Ms, r.TokensRetrieved)
	}
}