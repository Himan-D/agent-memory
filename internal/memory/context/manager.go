package context

import (
	"context"
	"fmt"
	"strings"
)

// MemorySnippet represents a single recalled memory fragment with relevance score.
type MemorySnippet struct {
	ID      string
	Content string
	Score   float64
	Tokens  int
}

// Message represents a single conversation turn.
type Message struct {
	Role    string
	Content string
	Tokens  int
}

// ComposedContext is the fully assembled LLM context, built within the token budget.
type ComposedContext struct {
	SystemPrompt     string
	CoreMemory       string
	RecalledMemories []MemorySnippet
	RecentMessages   []Message
	TotalTokens      int
}

// ContextManager implements MemGPT-style virtual context management.
// It treats the LLM context window like OS virtual memory — paging data in and out
// to keep the working set within the token budget while maximizing relevance.
type ContextManager struct {
	core         *CoreMemory
	budget       *Budget
	memorySearch func(ctx context.Context, query string, userID string, limit int) ([]MemorySnippet, error)
}

// NewContextManager creates a ContextManager with the given core memory, budget, and
// a search callback for retrieving relevant memories. The search function is a callback
// to avoid importing internal/memory directly (cycle prevention).
func NewContextManager(
	core *CoreMemory,
	budget *Budget,
	searchFn func(ctx context.Context, query, userID string, limit int) ([]MemorySnippet, error),
) *ContextManager {
	if budget == nil {
		budget = NewBudget(128000)
	}
	return &ContextManager{
		core:         core,
		budget:       budget,
		memorySearch: searchFn,
	}
}

// ComposeContext builds the full LLM context within the token budget.
// Assembly order (highest priority = always included):
//  1. System prompt + core memory (always included, reserved budget)
//  2. Recall relevant memories for the query via semantic search
//  3. Recent conversation turns (as many as the remaining budget allows)
//  4. Evict oldest turns if over budget
func (cm *ContextManager) ComposeContext(ctx context.Context, userID, query string, recentMessages []Message) (*ComposedContext, error) {
	_, _, memBudget, convBudget := cm.budget.Allocate()

	composed := &ComposedContext{}

	// 1. Core memory — always included
	if cm.core != nil {
		composed.CoreMemory = cm.core.FormatForPrompt()
	}
	coreTokens := CountTokens(composed.CoreMemory)

	// 2. System prompt placeholder (caller sets this after composition)
	composed.SystemPrompt = ""

	// 3. Recall relevant memories via semantic search
	if cm.memorySearch != nil && query != "" {
		// Estimate how many snippets we can fit in the memory budget
		maxSnippets := 20
		snippets, err := cm.memorySearch(ctx, query, userID, maxSnippets)
		if err != nil {
			return nil, fmt.Errorf("context: compose: memory search: %w", err)
		}

		usedMemTokens := 0
		for _, s := range snippets {
			tokensForSnippet := s.Tokens
			if tokensForSnippet == 0 {
				tokensForSnippet = CountTokens(s.Content)
			}
			if usedMemTokens+tokensForSnippet > memBudget {
				break // memory budget exhausted
			}
			s.Tokens = tokensForSnippet
			composed.RecalledMemories = append(composed.RecalledMemories, s)
			usedMemTokens += tokensForSnippet
		}
	}

	recalledTokens := 0
	for _, s := range composed.RecalledMemories {
		recalledTokens += s.Tokens
	}

	// 4. Recent messages — fill remaining conversation budget, newest first
	remainingBudget := convBudget
	// Adjust if core memory was larger than its reserve
	overflowFromCore := coreTokens - cm.budget.CoreMemoryReserve
	if overflowFromCore > 0 {
		remainingBudget -= overflowFromCore
	}
	// Adjust if recalled memories exceeded their budget
	overflowFromMemory := recalledTokens - memBudget
	if overflowFromMemory > 0 {
		remainingBudget -= overflowFromMemory
	}
	if remainingBudget < 0 {
		remainingBudget = 0
	}

	// Walk messages from newest to oldest, keeping as many as fit
	usedConvTokens := 0
	var keptMessages []Message
	for i := len(recentMessages) - 1; i >= 0; i-- {
		msg := recentMessages[i]
		if msg.Tokens == 0 {
			msg.Tokens = CountTokens(msg.Content)
		}
		if usedConvTokens+msg.Tokens > remainingBudget {
			break // conversation budget exhausted
		}
		keptMessages = append([]Message{msg}, keptMessages...)
		usedConvTokens += msg.Tokens
	}
	composed.RecentMessages = keptMessages

	// Calculate total tokens
	composed.TotalTokens = CountTokens(composed.SystemPrompt) + coreTokens + recalledTokens + usedConvTokens

	return composed, nil
}

// ArchiveMessages splits messages into kept (recent) and archived (old) sets.
// The most recent keepLast messages are retained; the rest are returned for archival
// to external memory storage.
func (cm *ContextManager) ArchiveMessages(messages []Message, keepLast int) (kept []Message, archived []Message) {
	if keepLast < 0 {
		keepLast = 0
	}
	if keepLast >= len(messages) {
		return messages, nil
	}
	splitPoint := len(messages) - keepLast
	return messages[splitPoint:], messages[:splitPoint]
}

// FormatComposedContext renders the composed context as a single string suitable
// for use as a system prompt or context block.
func FormatComposedContext(composed *ComposedContext) string {
	if composed == nil {
		return ""
	}
	var sb strings.Builder

	if composed.SystemPrompt != "" {
		sb.WriteString(composed.SystemPrompt)
		sb.WriteString("\n\n")
	}

	if composed.CoreMemory != "" {
		sb.WriteString(composed.CoreMemory)
		sb.WriteString("\n\n")
	}

	if len(composed.RecalledMemories) > 0 {
		sb.WriteString("<recalled_memories>\n")
		for _, m := range composed.RecalledMemories {
			sb.WriteString(fmt.Sprintf("- %s\n", m.Content))
		}
		sb.WriteString("</recalled_memories>\n\n")
	}

	if len(composed.RecentMessages) > 0 {
		sb.WriteString("<recent_conversation>\n")
		for _, m := range composed.RecentMessages {
			sb.WriteString(fmt.Sprintf("[%s]: %s\n", m.Role, m.Content))
		}
		sb.WriteString("</recent_conversation>")
	}

	return sb.String()
}
