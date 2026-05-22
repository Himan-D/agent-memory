package pool

import (
	"context"
	"fmt"
	"strings"

	"agent-memory/internal/llm"
)

// Judge uses LLM-as-judge to evaluate memory quality for pool promotion/demotion.
// It assesses quality, relevance, and actionability of memories to inform pool management decisions.
type Judge struct {
	LLMProvider llm.Provider
}

// JudgeResult contains the evaluation outcome for a memory.
type JudgeResult struct {
	Quality    float64 // [0, 1] overall quality score
	Relevance  float64 // [0, 1] relevance to the given context
	Actionable bool    // whether the memory contains actionable information
	ShouldKeep bool    // final recommendation
	Reason     string  // explanation for the decision
}

// NewJudge creates a Judge with the given LLM provider.
func NewJudge(provider llm.Provider) *Judge {
	return &Judge{
		LLMProvider: provider,
	}
}

// Evaluate assesses a memory's quality and relevance using LLM-as-judge.
// content is the memory text to evaluate. evalContext is the surrounding context
// (e.g., the agent's current task or query) used to judge relevance.
func (j *Judge) Evaluate(ctx context.Context, content string, evalContext string) (*JudgeResult, error) {
	if j.LLMProvider == nil {
		return nil, fmt.Errorf("judge: LLM provider not configured")
	}

	prompt := fmt.Sprintf(`You are a memory quality judge. Evaluate the following memory for quality, relevance, and usefulness.

## Memory Content
%s

## Current Context
%s

## Evaluation Criteria
1. **Quality** (0.0-1.0): Is the memory well-formed, accurate, and specific? Vague or poorly written memories score low.
2. **Relevance** (0.0-1.0): How relevant is this memory to the current context? Completely unrelated scores 0.0.
3. **Actionable** (true/false): Does the memory contain information that could directly inform a decision or action?
4. **Should Keep** (true/false): Overall, should this memory be retained in the active pool?

## Output Format (strict)
QUALITY: <float>
RELEVANCE: <float>
ACTIONABLE: <true|false>
SHOULD_KEEP: <true|false>
REASON: <one-line explanation>`, content, evalContext)

	resp, err := j.LLMProvider.Complete(ctx, &llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: "user", Content: prompt},
		},
		Model:       "gpt-4o",
		MaxTokens:   300,
		Temperature: 0.1,
	})
	if err != nil {
		return nil, fmt.Errorf("judge: LLM call failed: %w", err)
	}

	return parseJudgeResponse(resp.Content), nil
}

// parseJudgeResponse extracts structured evaluation from the LLM response.
// Falls back to neutral defaults if parsing fails.
func parseJudgeResponse(response string) *JudgeResult {
	result := &JudgeResult{
		Quality:    0.5,
		Relevance:  0.5,
		Actionable: false,
		ShouldKeep: true,
		Reason:     "unable to parse evaluation",
	}

	lines := strings.Split(response, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "QUALITY:") {
			val := strings.TrimSpace(strings.TrimPrefix(line, "QUALITY:"))
			var f float64
			if _, err := fmt.Sscanf(val, "%f", &f); err == nil {
				if f >= 0 && f <= 1 {
					result.Quality = f
				}
			}
		} else if strings.HasPrefix(line, "RELEVANCE:") {
			val := strings.TrimSpace(strings.TrimPrefix(line, "RELEVANCE:"))
			var f float64
			if _, err := fmt.Sscanf(val, "%f", &f); err == nil {
				if f >= 0 && f <= 1 {
					result.Relevance = f
				}
			}
		} else if strings.HasPrefix(line, "ACTIONABLE:") {
			val := strings.TrimSpace(strings.TrimPrefix(line, "ACTIONABLE:"))
			result.Actionable = strings.EqualFold(val, "true")
		} else if strings.HasPrefix(line, "SHOULD_KEEP:") {
			val := strings.TrimSpace(strings.TrimPrefix(line, "SHOULD_KEEP:"))
			result.ShouldKeep = strings.EqualFold(val, "true")
		} else if strings.HasPrefix(line, "REASON:") {
			result.Reason = strings.TrimSpace(strings.TrimPrefix(line, "REASON:"))
		}
	}

	return result
}
