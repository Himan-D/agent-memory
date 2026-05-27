package context

// Budget manages token allocation across context sections.
// Models the MemGPT approach of treating the LLM context window like OS virtual memory,
// with explicit budgets for each section (system prompt, core memory, recalled memories, conversation).
type Budget struct {
	MaxTokens         int // Total model context limit (e.g., 128000)
	SystemReserve     int // Reserved for system prompt (e.g., 2000)
	CoreMemoryReserve int // Reserved for core memory (e.g., 1000)
	MemoryBudget      int // Budget for recalled memories (e.g., 4000)
	// ConversationBudget = MaxTokens - SystemReserve - CoreMemoryReserve - MemoryBudget
}

// NewBudget creates a Budget with sensible defaults for the given max token limit.
// Allocations: 2% system, 1% core memory, 4% recalled memories, remainder for conversation.
func NewBudget(maxTokens int) *Budget {
	if maxTokens <= 0 {
		maxTokens = 128000
	}
	return &Budget{
		MaxTokens:         maxTokens,
		SystemReserve:     maxTokens * 2 / 100,  // 2%
		CoreMemoryReserve: maxTokens * 1 / 100,  // 1%
		MemoryBudget:      maxTokens * 4 / 100,  // 4%
	}
}

// CountTokens estimates the token count for a given text.
// Uses the common approximation of 4 characters per token.
func CountTokens(text string) int {
	if len(text) == 0 {
		return 0
	}
	tokens := len(text) / 4
	if tokens == 0 {
		tokens = 1
	}
	return tokens
}

// Allocate returns how many tokens each section gets: system, core, memory, conversation.
func (b *Budget) Allocate() (system, core, memory, conversation int) {
	system = b.SystemReserve
	core = b.CoreMemoryReserve
	memory = b.MemoryBudget
	conversation = b.MaxTokens - system - core - memory
	if conversation < 0 {
		conversation = 0
	}
	return system, core, memory, conversation
}

// FitsInBudget checks if adding content would exceed the total budget.
func (b *Budget) FitsInBudget(currentTokens, additionalTokens int) bool {
	return (currentTokens + additionalTokens) <= b.MaxTokens
}

// ConversationBudget returns the remaining token budget for conversation turns
// after accounting for system, core memory, and recalled memory reserves.
func (b *Budget) ConversationBudget() int {
	remaining := b.MaxTokens - b.SystemReserve - b.CoreMemoryReserve - b.MemoryBudget
	if remaining < 0 {
		return 0
	}
	return remaining
}
