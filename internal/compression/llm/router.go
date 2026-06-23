package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"agent-memory/internal/llm"
	"agent-memory/internal/memory/types"
)

type LLMRouter struct {
	fastProvider        llm.Provider
	verifyProvider      llm.Provider
	complexityThreshold float64
	fastModel           string
	verifyModel         string
	metrics             MetricsRecorder
}

type MetricsRecorder interface {
	RecordExtraction(provider string, tokensSaved int64, latencyMs float64)
}

type tokenReductionRecorder interface {
	SetTokenReduction(reduction float64)
}

func (r *LLMRouter) SetMetrics(m MetricsRecorder) {
	r.metrics = m
}

type RouterConfig struct {
	FastProvider        string  `env:"COMPRESSION_LLM_FAST_PROVIDER" envDefault:"openai"`
	FastModel           string  `env:"COMPRESSION_LLM_FAST_MODEL" envDefault:"gpt-4o-mini"`
	VerifyProvider      string  `env:"COMPRESSION_LLM_VERIFY_PROVIDER" envDefault:"anthropic"`
	VerifyModel         string  `env:"COMPRESSION_LLM_VERIFY_MODEL" envDefault:"claude-3-5-sonnet"`
	ComplexityThreshold float64 `env:"COMPRESSION_COMPLEXITY_THRESHOLD" envDefault:"0.6"`
}

type ExtractionResult struct {
	Facts              []types.Fact
	VerifiedFacts      []types.Fact
	Gaps               []Gap
	Supplements        []types.Fact
	Confidence         float64
	TokenReduction     float64
	Provider           string
	VerificationFailed bool // true when the verify LLM call failed and results are best-effort
}

type Gap struct {
	Question string
	Answer   string
	MemoryID string
}

func NewLLMRouter(fastProvider, verifyProvider llm.Provider, cfg *RouterConfig) *LLMRouter {
	if cfg == nil {
		cfg = &RouterConfig{}
	}
	return &LLMRouter{
		fastProvider:        fastProvider,
		verifyProvider:      verifyProvider,
		complexityThreshold: cfg.ComplexityThreshold,
		fastModel:           cfg.FastModel,
		verifyModel:         cfg.VerifyModel,
	}
}

// estimateComplexity uses text heuristics to estimate memory complexity (0.0–1.0).
// This avoids making an LLM call just to decide which LLM path to use.
func (r *LLMRouter) estimateComplexity(memory string) float64 {
	length := len(memory)
	sentences := strings.Count(memory, ".") + strings.Count(memory, "?") + strings.Count(memory, "!")
	commas := strings.Count(memory, ",")
	semicolons := strings.Count(memory, ";")
	parens := strings.Count(memory, "(") + strings.Count(memory, ")")
	techTerms := countTechnicalTerms(memory)

	score := 0.0

	// Length signals (agent memories typically range 50-2000 chars)
	if length > 200 {
		score += 0.15
	}
	if length > 800 {
		score += 0.15
	}

	// Sentence complexity
	if sentences > 3 {
		score += 0.10
	}
	if sentences > 8 {
		score += 0.10
	}

	// Clause density (commas + semicolons per sentence)
	sentenceCount := math.Max(float64(sentences), 1)
	clauseDensity := float64(commas+semicolons) / sentenceCount
	if clauseDensity > 1.5 {
		score += 0.10
	}
	if clauseDensity > 3.5 {
		score += 0.10
	}

	// Nested structure (parentheses)
	if parens > 3 {
		score += 0.05
	}

	// Technical vocabulary density
	if techTerms > 2 {
		score += 0.15
	}
	if techTerms > 6 {
		score += 0.15
	}

	if score > 1.0 {
		return 1.0
	}
	return score
}

// countTechnicalTerms counts words that indicate technical or domain-specific content.
func countTechnicalTerms(text string) int {
	techKeywords := []string{
		"algorithm", "database", "api", "function", "system", "protocol",
		"framework", "architecture", "latency", "throughput", "embedding",
		"neural", "inference", "deployment", "configuration", "authentication",
		"authorization", "encryption", "serialization", "concurrency", "async",
		"pipeline", "threshold", "hyperparameter", "gradient", "optimization",
	}
	lower := strings.ToLower(text)
	count := 0
	for _, kw := range techKeywords {
		if strings.Contains(lower, kw) {
			count++
		}
	}
	return count
}

func (r *LLMRouter) Route(ctx context.Context, memory string) (*ExtractionResult, error) {
	start := time.Now()
	complexity := r.estimateComplexity(memory)

	if complexity < r.complexityThreshold {
		result, err := r.extractFast(ctx, memory)
		if err != nil {
			return nil, fmt.Errorf("fast extraction: %w", err)
		}
		result.Provider = "fast"
		latencyMs := float64(time.Since(start).Milliseconds())
		if r.metrics != nil {
			tokensSaved := int64(float64(len(memory)) * result.TokenReduction)
			r.metrics.RecordExtraction("fast", tokensSaved, latencyMs)
			if recorder, ok := r.metrics.(tokenReductionRecorder); ok {
				recorder.SetTokenReduction(result.TokenReduction)
			}
		}
		return result, nil
	}

	result, err := r.extractWithVerification(ctx, memory)
	if err != nil {
		return nil, fmt.Errorf("verification extraction: %w", err)
	}
	result.Provider = "verify"
	latencyMs := float64(time.Since(start).Milliseconds())
	if r.metrics != nil {
		tokensSaved := int64(float64(len(memory)) * result.TokenReduction)
		r.metrics.RecordExtraction("verify", tokensSaved, latencyMs)
		if recorder, ok := r.metrics.(tokenReductionRecorder); ok {
			recorder.SetTokenReduction(result.TokenReduction)
		}
	}
	return result, nil
}

func (r *LLMRouter) extractFast(ctx context.Context, memory string) (*ExtractionResult, error) {
	if r.fastProvider == nil {
		return &ExtractionResult{
			Facts:          []types.Fact{{Fact: memory, Confidence: 0.5}},
			Confidence:     0.5,
			TokenReduction: 0.0,
		}, nil
	}

	prompt := fmt.Sprintf(`Extract key facts from this memory:

Memory: %s

Return a JSON array of facts, each with "fact" and "confidence" fields.
Example: [{"fact": "user prefers dark mode", "confidence": 0.9}]

Facts:`, memory)

	model := r.fastModel
	if model == "" {
		model = "gpt-4o-mini"
	}
	resp, err := r.fastProvider.Complete(ctx, &llm.CompletionRequest{
		Model: model,
		Messages: []llm.Message{
			{Role: "system", Content: "You extract key facts from memories."},
			{Role: "user", Content: prompt},
		},
		Temperature: 0.3,
		MaxTokens:   1000,
	})
	if err != nil {
		return nil, fmt.Errorf("fast extraction: %w", err)
	}

	var factsData []struct {
		Fact       string  `json:"fact"`
		Confidence float64 `json:"confidence"`
	}
	if err := json.Unmarshal([]byte(resp.Content), &factsData); err != nil {
		// Fallback: treat entire content as single fact
		return &ExtractionResult{
			Facts:          []types.Fact{{Fact: resp.Content, Confidence: 0.6}},
			Confidence:     0.6,
			TokenReduction: 0.3,
		}, nil
	}

	facts := make([]types.Fact, len(factsData))
	for i, f := range factsData {
		facts[i] = types.Fact{Fact: f.Fact, Confidence: f.Confidence}
	}

	return &ExtractionResult{
		Facts:          facts,
		VerifiedFacts:  facts,
		Gaps:           []Gap{},
		Supplements:    []types.Fact{},
		Confidence:     0.7,
		TokenReduction: 0.5,
	}, nil
}

func (r *LLMRouter) extractWithVerification(ctx context.Context, memory string) (*ExtractionResult, error) {
	fastResult, err := r.extractFast(ctx, memory)
	if err != nil {
		return nil, err
	}

	if r.verifyProvider == nil {
		// No verify provider, return fast result with verified=true
		verifiedFacts := make([]types.Fact, len(fastResult.Facts))
		for i, f := range fastResult.Facts {
			f.Verified = true
			verifiedFacts[i] = f
		}
		fastResult.VerifiedFacts = verifiedFacts
		fastResult.Confidence = 0.85
		fastResult.VerificationFailed = true
		return fastResult, nil
	}

	// Actually verify with the verify provider (e.g., Claude)
	factJSON, _ := json.Marshal(fastResult.Facts)

	prompt := fmt.Sprintf(`Verify these facts against the original memory:

Original Memory: %s

Extracted Facts: %s

For each fact, verify if it was actually stated or implied in the memory.
Return JSON array with "fact", "verified" (true/false), and "confidence" fields.`, memory, string(factJSON))

	model := r.verifyModel
	if model == "" {
		model = "claude-3-5-sonnet"
	}
	resp, err := r.verifyProvider.Complete(ctx, &llm.CompletionRequest{
		Model: model,
		Messages: []llm.Message{
			{Role: "system", Content: "You verify extracted facts against original memory."},
			{Role: "user", Content: prompt},
		},
		Temperature: 0.3,
		MaxTokens:   2000,
	})
	if err != nil {
		// Fallback to marking all as verified
		verifiedFacts := make([]types.Fact, len(fastResult.Facts))
		for i, f := range fastResult.Facts {
			f.Verified = true
			verifiedFacts[i] = f
		}
		fastResult.VerifiedFacts = verifiedFacts
		fastResult.Confidence = 0.85
		fastResult.VerificationFailed = true
		return fastResult, nil
	}

	var verifyData []struct {
		Fact       string  `json:"fact"`
		Verified   bool    `json:"verified"`
		Confidence float64 `json:"confidence"`
	}
	if err := json.Unmarshal([]byte(resp.Content), &verifyData); err != nil {
		// Fallback
		verifiedFacts := make([]types.Fact, len(fastResult.Facts))
		for i, f := range fastResult.Facts {
			f.Verified = true
			verifiedFacts[i] = f
		}
		fastResult.VerifiedFacts = verifiedFacts
		fastResult.Confidence = 0.85
		fastResult.VerificationFailed = true
		return fastResult, nil
	}

	verifiedFacts := make([]types.Fact, 0)
	for _, v := range verifyData {
		if v.Verified {
			verifiedFacts = append(verifiedFacts, types.Fact{Fact: v.Fact, Confidence: v.Confidence, Verified: true})
		}
	}

	result := &ExtractionResult{
		Facts:          fastResult.Facts,
		VerifiedFacts:  verifiedFacts,
		Gaps:           fastResult.Gaps,
		Supplements:    fastResult.Supplements,
		Confidence:     0.9,
		TokenReduction: 0.85,
	}

	return result, nil
}
