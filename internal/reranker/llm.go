package reranker

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"agent-memory/internal/llm"
	"agent-memory/internal/memory/types"
)

type LLLMReranker struct {
	client llm.Provider
}

func NewLLMReranker(client llm.Provider) *LLLMReranker {
	return &LLLMReranker{
		client: client,
	}
}

func (l *LLLMReranker) Name() string {
	return "llm"
}

func (l *LLLMReranker) Rerank(ctx context.Context, query string, results []types.MemoryResult, limit int) ([]types.MemoryResult, error) {
	if len(results) == 0 {
		return results, nil
	}

	if limit <= 0 || limit > len(results) {
		limit = len(results)
	}

	scored := make([]struct {
		result types.MemoryResult
		score  float64
	}, len(results))

	for i, r := range results {
		score, err := l.scoreDocument(ctx, query, r.Text)
		if err != nil {
			score = 0.5
		}
		scored[i] = struct {
			result types.MemoryResult
			score  float64
		}{result: r, score: score}
	}

	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	reranked := make([]types.MemoryResult, 0, limit)
	for i := 0; i < limit && i < len(scored); i++ {
		reranked = append(reranked, scored[i].result)
	}

	return reranked, nil
}

func (l *LLLMReranker) scoreDocument(ctx context.Context, query, document string) (float64, error) {
	prompt := fmt.Sprintf(`You are a relevance scorer. Rate how relevant this document is to the query on a scale of 0.0 to 1.0.

Query: %s

Document: %s

Respond with only a number between 0.0 and 1.0 (e.g., 0.85).`, query, document)

	resp, err := l.client.Complete(ctx, &llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: "user", Content: prompt},
		},
		MaxTokens:   10,
		Temperature: 0.0,
	})

	if err != nil {
		return 0.5, err
	}

	scoreStr := strings.TrimSpace(resp.Content)
	score, err := strconv.ParseFloat(scoreStr, 64)
	if err != nil {
		scoreStr = strings.TrimPrefix(scoreStr, ".")
		score, err = strconv.ParseFloat(scoreStr, 64)
		if err != nil {
			return 0.5, nil
		}
	}

	if score < 0 {
		score = 0
	}
	if score > 1 {
		score = 1
	}

	return score, nil
}

func (l *LLLMReranker) Close() error {
	return nil
}
