package evaluator

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"agent-memory/internal/llm"
)

// FidelityResult holds the outcome of a compression fidelity evaluation.
type FidelityResult struct {
	Recall    float64 `json:"recall"`
	Precision float64 `json:"precision"`
	F1        float64 `json:"f1"`
	Reasoning string  `json:"reasoning,omitempty"`
}

// FidelityEvaluator scores compression fidelity using an LLM judge.
// Port of sdk/python/test_evaluator.py — provider-agnostic via llm.Provider.
type FidelityEvaluator struct {
	provider llm.Provider
	model    string
}

// NewFidelityEvaluator creates a fidelity evaluator using the given LLM provider.
func NewFidelityEvaluator(provider llm.Provider, model string) *FidelityEvaluator {
	if model == "" {
		model = "gpt-4o-mini"
	}
	return &FidelityEvaluator{
		provider: provider,
		model:    model,
	}
}

// Evaluate scores how well a compressed output preserves the original text's facts.
// Returns recall (factual retention), precision (no hallucination), and F1.
func (e *FidelityEvaluator) Evaluate(ctx context.Context, original, compressed string) (*FidelityResult, error) {
	if e.provider == nil {
		return nil, fmt.Errorf("fidelity evaluator: no LLM provider configured")
	}
	if original == "" || compressed == "" {
		return &FidelityResult{Recall: 0, Precision: 0, F1: 0}, nil
	}

	prompt := fmt.Sprintf(`You are an expert data QA engineer evaluating an AI memory compression engine.

ORIGINAL TEXT:
"""%s"""

COMPRESSED OUTPUT (FACTS/GRAPH):
"""%s"""

Analyze the compressed output against the original text and calculate two scores between 0.0 and 1.0:
1. Recall (Factual Retention): What percentage of the critical information from the original text was successfully preserved? (1.0 = everything preserved, 0.0 = completely lost).
2. Precision (No Hallucination): Did the compression introduce any false assumptions or hallucinations not present in the original text? (1.0 = completely clean, 0.0 = completely hallucinated).

Respond STRICTLY in the following raw JSON format:
{"recall": 0.95, "precision": 1.0, "reasoning": "Brief narrative explanation."}`, original, compressed)

	resp, err := e.provider.Complete(ctx, &llm.CompletionRequest{
		Model: e.model,
		Messages: []llm.Message{
			{Role: "system", Content: "You evaluate compression fidelity. Respond only with valid JSON."},
			{Role: "user", Content: prompt},
		},
		Temperature: 0.0,
		MaxTokens:   300,
	})
	if err != nil {
		return nil, fmt.Errorf("fidelity evaluation: %w", err)
	}

	var parsed struct {
		Recall    float64 `json:"recall"`
		Precision float64 `json:"precision"`
		Reasoning string  `json:"reasoning"`
	}

	content := resp.Content
	jsonStart := strings.Index(content, "{")
	if jsonStart == -1 {
		return nil, fmt.Errorf("fidelity evaluator: no JSON in response")
	}
	if err := json.Unmarshal([]byte(content[jsonStart:]), &parsed); err != nil {
		return nil, fmt.Errorf("fidelity evaluator: parse JSON: %w", err)
	}

	// Clamp scores to [0, 1]
	parsed.Recall = clamp(parsed.Recall)
	parsed.Precision = clamp(parsed.Precision)

	f1 := 0.0
	if parsed.Recall+parsed.Precision > 0 {
		f1 = 2 * parsed.Recall * parsed.Precision / (parsed.Recall + parsed.Precision)
	}

	return &FidelityResult{
		Recall:    parsed.Recall,
		Precision: parsed.Precision,
		F1:        f1,
		Reasoning: parsed.Reasoning,
	}, nil
}

func clamp(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
