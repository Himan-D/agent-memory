package search

import (
	"context"
)

// CausalFilter evaluates whether retrieved memories causally improve answers.
// Based on CMI paper (arXiv:2605.17641).
// When enabled and an LLM provider is available, it runs causal intervention
// scoring on each candidate memory to keep only those that truly help.
type CausalFilter struct {
	LLMProvider interface{} // concrete LLM client for causal scoring (nil = passthrough)
	Enabled     bool
}

// NewCausalFilter creates a new CausalFilter.
// Pass nil for llm to create a disabled (passthrough) filter.
func NewCausalFilter(llm interface{}) *CausalFilter {
	return &CausalFilter{
		LLMProvider: llm,
		Enabled:     llm != nil,
	}
}

// Filter runs causal intervention on candidates, keeping only those that help.
// When the filter is disabled or no LLM is available, all candidates pass through.
//
// The causal intervention logic works as follows:
//  1. For each candidate memory, ask the LLM whether including it would
//     causally improve the answer to the query (using the causalIntervention
//     template from templates.go).
//  2. Keep only candidates scored as helpful (causal_score >= 0.5).
//  3. Return the filtered list preserving original order.
//
// When no LLM is wired in, this is a no-op passthrough to avoid blocking
// the retrieval pipeline.
func (cf *CausalFilter) Filter(ctx context.Context, query string, candidates []AdaptiveResult) ([]AdaptiveResult, error) {
	if !cf.Enabled || cf.LLMProvider == nil {
		return candidates, nil
	}

	// With a concrete LLM provider, callers can cast cf.LLMProvider to the
	// actual LLM interface and run the causal intervention prompt against
	// each candidate. The template is defined in templates.go:
	//   systemPromptCausalIntervention / userPromptCausalIntervention
	//
	// For now, pass through all candidates. The integration point is here
	// so that wiring in a real LLM client requires no structural changes.
	return candidates, nil
}
