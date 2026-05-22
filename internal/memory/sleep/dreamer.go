package sleep

import (
	"context"
	"fmt"
	"strings"

	"agent-memory/internal/llm"
	"agent-memory/internal/memory/types"
)

// Dreamer implements cross-session memory consolidation.
// It reads a region of memories, identifies patterns, and produces compact replacements.
// Based on Auto-Dreamer (arXiv:2605.20616).
type Dreamer struct {
	LLMProvider  llm.Provider
	MaxBatchSize int // max memories per consolidation batch (default: 50)
	MinBatchSize int // minimum to trigger consolidation (default: 5)
}

// ConsolidationResult contains the output of a consolidation run.
type ConsolidationResult struct {
	OriginalCount     int
	ConsolidatedCount int
	TokenReduction    float64
	Replacements      []MemoryReplacement
}

// MemoryReplacement maps a set of original memory IDs to their consolidated replacement.
type MemoryReplacement struct {
	OriginalIDs []string
	NewContent  string
	NewMetadata map[string]interface{}
}

// NewDreamer creates a Dreamer with default batch sizes.
func NewDreamer(provider llm.Provider) *Dreamer {
	return &Dreamer{
		LLMProvider:  provider,
		MaxBatchSize: 50,
		MinBatchSize: 5,
	}
}

// ShouldConsolidate returns true if the memory set meets the minimum threshold for consolidation.
func (d *Dreamer) ShouldConsolidate(memories []types.Memory) bool {
	return len(memories) >= d.MinBatchSize
}

// Consolidate runs the consolidation algorithm on a batch of memories.
// It identifies overlapping information, merges duplicates, abstracts patterns,
// preserves unique details, and produces compact replacements.
func (d *Dreamer) Consolidate(ctx context.Context, memories []types.Memory) (*ConsolidationResult, error) {
	if len(memories) == 0 {
		return &ConsolidationResult{}, nil
	}

	if d.LLMProvider == nil {
		return nil, fmt.Errorf("dreamer: consolidate: LLM provider not configured")
	}

	// Cap at MaxBatchSize
	batch := memories
	if len(batch) > d.MaxBatchSize {
		batch = batch[:d.MaxBatchSize]
	}

	// Build the memory listing for the prompt
	var memoryListing strings.Builder
	var totalOriginalTokens int
	for i, mem := range batch {
		memoryListing.WriteString(fmt.Sprintf("[Memory %d] (ID: %s)\n%s\n\n", i+1, mem.ID, mem.Content))
		totalOriginalTokens += len(strings.Fields(mem.Content))
	}

	prompt := fmt.Sprintf(`You are a memory consolidation engine. Your task is to analyze a set of memories and produce compact, consolidated replacements that preserve all unique information while eliminating redundancy.

## Instructions

1. **Identify clusters**: Group memories that discuss the same topic, entity, or event.
2. **Merge duplicates**: Combine memories with overlapping information into single consolidated entries.
3. **Abstract patterns**: When multiple memories describe similar patterns, create a generalized memory.
4. **Preserve unique details**: Never discard information that appears in only one memory.
5. **Maintain accuracy**: Do not infer or add information not present in the originals.

## Input Memories

%s

## Output Format

For each consolidated group, output:
GROUP:
ORIGINAL_IDS: comma-separated list of memory IDs being replaced
CONSOLIDATED: the merged memory content

Produce as few consolidated memories as possible while retaining all unique information.`, memoryListing.String())

	resp, err := d.LLMProvider.Complete(ctx, &llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: "user", Content: prompt},
		},
		Model:       "gpt-4o",
		MaxTokens:   4000,
		Temperature: 0.2,
	})
	if err != nil {
		return nil, fmt.Errorf("dreamer: consolidate: LLM call failed: %w", err)
	}

	replacements := parseConsolidationResponse(resp.Content, batch)

	// Compute token reduction
	var consolidatedTokens int
	for _, r := range replacements {
		consolidatedTokens += len(strings.Fields(r.NewContent))
	}

	var tokenReduction float64
	if totalOriginalTokens > 0 {
		tokenReduction = 1.0 - float64(consolidatedTokens)/float64(totalOriginalTokens)
	}

	return &ConsolidationResult{
		OriginalCount:     len(batch),
		ConsolidatedCount: len(replacements),
		TokenReduction:    tokenReduction,
		Replacements:      replacements,
	}, nil
}

// parseConsolidationResponse extracts MemoryReplacements from the LLM output.
// Falls back to a single replacement if parsing fails.
func parseConsolidationResponse(response string, memories []types.Memory) []MemoryReplacement {
	lines := strings.Split(response, "\n")
	var replacements []MemoryReplacement
	var currentIDs []string
	var contentLines []string
	inContent := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "ORIGINAL_IDS:") {
			// Save previous group if any
			if len(currentIDs) > 0 && len(contentLines) > 0 {
				replacements = append(replacements, MemoryReplacement{
					OriginalIDs: currentIDs,
					NewContent:  strings.TrimSpace(strings.Join(contentLines, "\n")),
					NewMetadata: map[string]interface{}{"source": "consolidation"},
				})
			}
			idsStr := strings.TrimPrefix(trimmed, "ORIGINAL_IDS:")
			currentIDs = parseIDList(strings.TrimSpace(idsStr))
			contentLines = nil
			inContent = false
			continue
		}

		if strings.HasPrefix(trimmed, "CONSOLIDATED:") {
			inContent = true
			rest := strings.TrimPrefix(trimmed, "CONSOLIDATED:")
			rest = strings.TrimSpace(rest)
			if rest != "" {
				contentLines = append(contentLines, rest)
			}
			continue
		}

		if strings.HasPrefix(trimmed, "GROUP:") {
			inContent = false
			continue
		}

		if inContent && trimmed != "" {
			contentLines = append(contentLines, trimmed)
		}
	}

	// Save last group
	if len(currentIDs) > 0 && len(contentLines) > 0 {
		replacements = append(replacements, MemoryReplacement{
			OriginalIDs: currentIDs,
			NewContent:  strings.TrimSpace(strings.Join(contentLines, "\n")),
			NewMetadata: map[string]interface{}{"source": "consolidation"},
		})
	}

	// Fallback: if parsing produced nothing, create a single replacement with all IDs
	if len(replacements) == 0 && len(memories) > 0 {
		allIDs := make([]string, len(memories))
		for i, m := range memories {
			allIDs[i] = m.ID
		}
		replacements = append(replacements, MemoryReplacement{
			OriginalIDs: allIDs,
			NewContent:  response,
			NewMetadata: map[string]interface{}{"source": "consolidation", "fallback": true},
		})
	}

	return replacements
}

// parseIDList splits a comma-separated ID string into a slice.
func parseIDList(s string) []string {
	parts := strings.Split(s, ",")
	var ids []string
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			ids = append(ids, trimmed)
		}
	}
	return ids
}
