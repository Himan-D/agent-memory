package search

import (
	"context"
	"fmt"
	"strings"

	"agent-memory/internal/llm"
)

// Distiller rewrites retrieved memories into query-conditioned evidence.
// DeferMem-style post-retrieval distillation: each candidate is rewritten as a
// self-contained evidence statement, irrelevant details are removed, and
// confidence markers are added.
type Distiller struct {
	LLMProvider llm.Provider
}

// NewDistiller creates a Distiller with the given LLM provider.
func NewDistiller(provider llm.Provider) *Distiller {
	return &Distiller{
		LLMProvider: provider,
	}
}

// Distill rewrites retrieved candidates into query-conditioned evidence statements.
// Each candidate is distilled to be self-contained, accurate, and directly relevant to the query.
func (d *Distiller) Distill(ctx context.Context, query string, candidates []AdaptiveResult) ([]AdaptiveResult, error) {
	if d.LLMProvider == nil {
		return candidates, nil
	}

	if len(candidates) == 0 {
		return candidates, nil
	}

	// Build candidate listing for the prompt
	var candidateListing strings.Builder
	for i, c := range candidates {
		candidateListing.WriteString(fmt.Sprintf("[%d] (score: %.3f) %s\n\n", i+1, c.Score, c.Content))
	}

	prompt := fmt.Sprintf(`You are a memory distillation engine. Given a user query and a list of retrieved memory candidates, rewrite each candidate as a self-contained evidence statement.

## Rules
1. Rewrite each candidate as a concise, self-contained evidence statement that directly addresses the query.
2. Remove details that are irrelevant to the query.
3. Preserve all factual accuracy — never add information not in the original.
4. Add a confidence marker at the start: [HIGH], [MEDIUM], or [LOW] based on how directly the candidate answers the query.
5. If a candidate is completely irrelevant, output [IRRELEVANT] and skip it.

## Query
%s

## Candidates
%s

## Output Format
For each candidate, output one line:
[N] [CONFIDENCE] distilled evidence statement

Where N is the candidate number and CONFIDENCE is HIGH/MEDIUM/LOW/IRRELEVANT.`, query, candidateListing.String())

	resp, err := d.LLMProvider.Complete(ctx, &llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: "user", Content: prompt},
		},
		Model:       "gpt-4o",
		MaxTokens:   2000,
		Temperature: 0.1,
	})
	if err != nil {
		return nil, fmt.Errorf("distiller: LLM call failed: %w", err)
	}

	return parseDistillationResponse(resp.Content, candidates), nil
}

// parseDistillationResponse parses the LLM output and updates candidate content.
// Falls back to original content if parsing fails for a candidate.
func parseDistillationResponse(response string, candidates []AdaptiveResult) []AdaptiveResult {
	lines := strings.Split(response, "\n")
	distilled := make([]AdaptiveResult, 0, len(candidates))

	// Build a map of distilled content by index
	distilledMap := make(map[int]string)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Parse lines like "[1] [HIGH] distilled content"
		if len(line) < 3 || line[0] != '[' {
			continue
		}

		// Find the index
		closeIdx := strings.Index(line, "]")
		if closeIdx < 0 {
			continue
		}

		idxStr := strings.TrimSpace(line[1:closeIdx])
		var idx int
		if _, err := fmt.Sscanf(idxStr, "%d", &idx); err != nil {
			continue
		}

		rest := strings.TrimSpace(line[closeIdx+1:])
		if strings.HasPrefix(rest, "[IRRELEVANT]") {
			continue // Skip irrelevant candidates
		}

		// Remove confidence marker prefix for the content
		for _, marker := range []string{"[HIGH] ", "[MEDIUM] ", "[LOW] "} {
			if strings.HasPrefix(rest, marker) {
				rest = strings.TrimPrefix(rest, marker)
				break
			}
		}

		if rest != "" {
			distilledMap[idx] = rest
		}
	}

	// Rebuild candidates with distilled content
	for i, c := range candidates {
		if content, ok := distilledMap[i+1]; ok {
			distilled = append(distilled, AdaptiveResult{
				MemoryID: c.MemoryID,
				Content:  content,
				Score:    c.Score,
				Source:   "distilled",
			})
		} else {
			// Keep original if distillation didn't produce output for this candidate
			distilled = append(distilled, c)
		}
	}

	return distilled
}
