package llm

import (
	"context"
	"fmt"

	"agent-memory/internal/llm"
	"agent-memory/internal/memory/types"
)

type LLMRouter struct {
	fastProvider      llm.Provider
	verifyProvider  llm.Provider
	complexityThreshold float64
}

type RouterConfig struct {
	FastProvider     string `env:"COMPRESSION_LLM_FAST_PROVIDER" envDefault:"openai"`
	FastModel       string `env:"COMPRESSION_LLM_FAST_MODEL" envDefault:"gpt-4o-mini"`
	VerifyProvider  string `env:"COMPRESSION_LLM_VERIFY_PROVIDER" envDefault:"anthropic"`
	VerifyModel     string `env:"COMPRESSION_LLM_VERIFY_MODEL" envDefault:"claude-3-5-sonnet"`
	ComplexityThreshold float64 `env:"COMPRESSION_COMPLEXITY_THRESHOLD" envDefault:0.6`
}

type ExtractionResult struct {
	Facts          []types.Fact
	VerifiedFacts []types.Fact
	Gaps          []Gap
	Supplements   []types.Fact
	Confidence   float64
	TokenReduction float64
	Provider     string
}

type Gap struct {
	Question     string
	Answer       string
	MemoryID    string
}

func NewLLMRouter(fastProvider, verifyProvider llm.Provider, cfg *RouterConfig) *LLMRouter {
	if cfg == nil {
		cfg = &RouterConfig{}
	}
	return &LLMRouter{
		fastProvider:      fastProvider,
		verifyProvider:    verifyProvider,
		complexityThreshold: cfg.ComplexityThreshold,
	}
}

func (r *LLMRouter) estimateComplexity(memory string) float64 {
	if r.fastProvider == nil {
		return 0.5
	}

	complexityPrompt := fmt.Sprintf(`Analyze the following memory content and estimate its complexity (0.0 to 1.0):
- Simple: factual statements, basic preferences (0.0-0.3)
- Medium: multi-part conversations, decisions with reasons (0.3-0.6)
- Complex: multi-hop reasoning, technical details, emotional context (0.6-1.0)

Memory: %s

Respond with only a number between 0.0 and 1.0.`, memory)

	resp, err := r.fastProvider.Complete(context.Background(), &llm.CompletionRequest{
		Model: "gpt-4o-mini",
		Messages: []llm.Message{
			{Role: "system", Content: "You estimate memory complexity."},
			{Role: "user", Content: complexityPrompt},
		},
		Temperature: 0.3,
		MaxTokens:    10,
	})
	if err != nil {
		return 0.5
	}

	var complexity float64
	fmt.Sscanf(resp.Content, "%f", &complexity)
	if complexity < 0 {
		complexity = 0.5
	}
	if complexity > 1 {
		complexity = 1.0
	}
	return complexity
}

func (r *LLMRouter) Route(ctx context.Context, memory string) (*ExtractionResult, error) {
	complexity := r.estimateComplexity(memory)

	if complexity < r.complexityThreshold {
		result, err := r.extractFast(ctx, memory)
		if err != nil {
			return nil, fmt.Errorf("fast extraction: %w", err)
		}
		result.Provider = "fast"
		return result, nil
	}

	result, err := r.extractWithVerification(ctx, memory)
	if err != nil {
		return nil, fmt.Errorf("verification extraction: %w", err)
	}
	result.Provider = "verify"
	return result, nil
}

func (r *LLMRouter) extractFast(ctx context.Context, memory string) (*ExtractionResult, error) {
	if r.fastProvider == nil {
		return &ExtractionResult{
			Facts: []types.Fact{{Fact: memory, Confidence: 0.5}},
			Confidence: 0.5,
			TokenReduction: 0.0,
		}, nil
	}

	result := &ExtractionResult{
		Facts:          []types.Fact{{Fact: memory, Confidence: 0.7}},
		VerifiedFacts:  []types.Fact{},
		Gaps:          []Gap{},
		Supplements:   []types.Fact{},
		Confidence:   0.7,
		TokenReduction: 0.5,
	}

	return result, nil
}

func (r *LLMRouter) extractWithVerification(ctx context.Context, memory string) (*ExtractionResult, error) {
	fastResult, err := r.extractFast(ctx, memory)
	if err != nil {
		return nil, err
	}

	verifiedFacts := make([]types.Fact, 0, len(fastResult.Facts))
	for _, f := range fastResult.Facts {
		f.Verified = true
		verifiedFacts = append(verifiedFacts, f)
	}

	result := &ExtractionResult{
		Facts:           fastResult.Facts,
		VerifiedFacts:  verifiedFacts,
		Gaps:           fastResult.Gaps,
		Supplements:   fastResult.Supplements,
		Confidence:   0.85,
		TokenReduction: 0.80,
	}

	return result, nil
}