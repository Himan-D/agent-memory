package injection

import (
	"context"
	"fmt"
	"strings"
)

// Interfaces to avoid import cycles — each references the minimal surface area needed.

// SavedStore provides prompt injection from the saved/explicit memory layer.
type SavedStore interface {
	InjectIntoPrompt(userID string) string
}

// HistoryStore provides retrieval and formatting of past conversation references.
type HistoryStore interface {
	FindRelevant(userID, query string, limit int) interface{}
	FormatForPrompt(refs interface{}) string
}

// CoreStore provides the always-loaded core memory formatted for prompts.
type CoreStore interface {
	FormatForPrompt() string
}

// Composer auto-injects relevant memories into LLM system prompts.
// It combines all memory layers (core, saved facts, chat history) into a
// single coherent prompt, respecting a token budget.
type Composer struct {
	savedStore   SavedStore
	historyStore HistoryStore
	coreMemory   CoreStore
	tokenBudget  int
}

// NewComposer creates a Composer that merges memory layers into system prompts.
// Any store parameter may be nil; that layer will simply be skipped during composition.
// tokenBudget limits the total injected memory size (0 = no limit).
func NewComposer(saved SavedStore, history HistoryStore, core CoreStore, tokenBudget int) *Composer {
	return &Composer{
		savedStore:   saved,
		historyStore: history,
		coreMemory:   core,
		tokenBudget:  tokenBudget,
	}
}

// ComposeSystemPrompt builds the full memory-augmented system prompt by layering:
//
//	[Base System Instructions]
//	[Core Memory — always-loaded user/agent facts]
//	[Saved Facts — explicit extracted knowledge]
//	[Chat History References — relevant past conversations]
//	[Session Context — current query context]
//
// Each section is included only if the corresponding store is configured and has data.
// The total output respects the token budget (approximate, using 4-char-per-token estimate).
func (c *Composer) ComposeSystemPrompt(ctx context.Context, userID, query, basePrompt string) (string, error) {
	var sections []string
	totalTokens := 0

	// 1. Base system prompt — always included first
	if basePrompt != "" {
		sections = append(sections, basePrompt)
		totalTokens += estimateTokens(basePrompt)
	}

	// 2. Core memory — always-loaded persistent facts
	if c.coreMemory != nil {
		coreBlock := c.coreMemory.FormatForPrompt()
		if coreBlock != "" {
			coreTokens := estimateTokens(coreBlock)
			if c.fitsInBudget(totalTokens, coreTokens) {
				sections = append(sections, coreBlock)
				totalTokens += coreTokens
			}
		}
	}

	// 3. Saved facts — explicit extracted knowledge
	if c.savedStore != nil {
		savedBlock := c.savedStore.InjectIntoPrompt(userID)
		if savedBlock != "" {
			savedTokens := estimateTokens(savedBlock)
			if c.fitsInBudget(totalTokens, savedTokens) {
				sections = append(sections, savedBlock)
				totalTokens += savedTokens
			}
		}
	}

	// 4. Chat history references — relevant past conversations
	if c.historyStore != nil && query != "" {
		refs := c.historyStore.FindRelevant(userID, query, 5)
		if refs != nil {
			historyBlock := c.historyStore.FormatForPrompt(refs)
			if historyBlock != "" {
				historyTokens := estimateTokens(historyBlock)
				if c.fitsInBudget(totalTokens, historyTokens) {
					sections = append(sections, historyBlock)
					totalTokens += historyTokens
				}
			}
		}
	}

	if len(sections) == 0 {
		return "", nil
	}

	result := strings.Join(sections, "\n\n")
	return result, nil
}

// fitsInBudget checks whether adding more tokens would stay within the budget.
// Returns true if no budget is set (0 = unlimited).
func (c *Composer) fitsInBudget(current, additional int) bool {
	if c.tokenBudget <= 0 {
		return true
	}
	return (current + additional) <= c.tokenBudget
}

// estimateTokens approximates token count using the common 4-chars-per-token heuristic.
func estimateTokens(text string) int {
	if len(text) == 0 {
		return 0
	}
	tokens := len(text) / 4
	if tokens == 0 {
		tokens = 1
	}
	return tokens
}

// FormatMemoryBlock wraps content in XML-style tags for clear prompt delineation.
func FormatMemoryBlock(tag, content string) string {
	if content == "" {
		return ""
	}
	return fmt.Sprintf("<%s>\n%s\n</%s>", tag, content, tag)
}
