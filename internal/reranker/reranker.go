package reranker

import (
	"context"
	"fmt"

	"agent-memory/internal/config"
	"agent-memory/internal/llm"
	"agent-memory/internal/memory/types"
)

type Provider interface {
	Name() string
	Rerank(ctx context.Context, query string, results []types.MemoryResult, limit int) ([]types.MemoryResult, error)
	Close() error
}

type Config struct {
	Provider string
	APIKey   string
	BaseURL  string
	Model    string
}

func NewProvider(cfg config.RerankerConfig, llmClient llm.Provider) (Provider, error) {
	switch cfg.Provider {
	case "cohere":
		return NewCohereReranker(cfg.APIKey, cfg.BaseURL, cfg.Model)
	case "llm":
		if llmClient == nil {
			return nil, fmt.Errorf("llm provider required for LLM reranker")
		}
		return NewLLMReranker(llmClient), nil
	case "disabled", "":
		return &disabledReranker{}, nil
	default:
		return nil, fmt.Errorf("unknown reranker provider: %s", cfg.Provider)
	}
}

type disabledReranker struct{}

func (d *disabledReranker) Name() string { return "disabled" }
func (d *disabledReranker) Rerank(ctx context.Context, query string, results []types.MemoryResult, limit int) ([]types.MemoryResult, error) {
	if limit > 0 && len(results) > limit {
		return results[:limit], nil
	}
	return results, nil
}
func (d *disabledReranker) Close() error { return nil }
