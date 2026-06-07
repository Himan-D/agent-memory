package consolidation

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"agent-memory/internal/llm"
	"agent-memory/internal/memory/types"
)

// MemoryService is the subset of memory.Service needed by consolidation.
type MemoryService interface {
	GetMemoriesByUser(ctx context.Context, userID string) ([]*types.Memory, error)
	CreateMemory(ctx context.Context, mem *types.Memory) (*types.Memory, error)
	ArchiveMemory(ctx context.Context, id string) error
}

// Config holds consolidation thresholds.
type Config struct {
	MinMemories int // Minimum memories before consolidation runs (default: 100)
	ChunkSize   int // Memories per summary chunk (default: 10)
	MaxOldest   int // Max memories to consolidate per run (default: 50)
}

func DefaultConfig() *Config {
	return &Config{
		MinMemories: 100,
		ChunkSize:   10,
		MaxOldest:   50,
	}
}

// Service consolidates old episodic memories into semantic summaries
// using the MemGPT/Letta recursive summarization approach.
type Service struct {
	memSvc MemoryService
	llm    llm.Provider
	cfg    *Config
}

func NewService(memSvc MemoryService, llmProvider llm.Provider, cfg *Config) *Service {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	return &Service{
		memSvc: memSvc,
		llm:    llmProvider,
		cfg:    cfg,
	}
}

// ConsolidateUser runs the consolidation algorithm for a single user:
//  1. Fetch all memories sorted by last_updated ascending (oldest first)
//  2. Take the oldest N episodic memories (up to MaxOldest)
//  3. Group into chunks of ChunkSize
//  4. Summarize each chunk into a semantic memory via LLM
//  5. Archive the original episodic memories
func (s *Service) ConsolidateUser(ctx context.Context, userID string) error {
	memories, err := s.memSvc.GetMemoriesByUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("get memories: %w", err)
	}

	// Consolidate conversation/session-type memories (episodic in nature)
	var episodic []*types.Memory
	for _, m := range memories {
		if m.Type == types.MemoryTypeConversation || m.Type == types.MemoryTypeSession || m.Type == "" {
			episodic = append(episodic, m)
		}
	}

	if len(episodic) < s.cfg.MinMemories {
		return nil // not enough memories to warrant consolidation
	}

	// Sort oldest first
	sort.Slice(episodic, func(i, j int) bool {
		return episodic[i].UpdatedAt.Before(episodic[j].UpdatedAt)
	})

	// Take oldest subset
	toConsolidate := episodic
	if len(toConsolidate) > s.cfg.MaxOldest {
		toConsolidate = toConsolidate[:s.cfg.MaxOldest]
	}

	// Process in chunks
	consolidated := 0
	for i := 0; i < len(toConsolidate); i += s.cfg.ChunkSize {
		end := i + s.cfg.ChunkSize
		if end > len(toConsolidate) {
			end = len(toConsolidate)
		}
		chunk := toConsolidate[i:end]

		summary, err := s.summarizeChunk(ctx, chunk)
		if err != nil {
			log.Printf("consolidation: summarize chunk %d-%d for user %s: %v", i, end, userID, err)
			continue
		}

		sourceIDs := make([]string, len(chunk))
		for j, m := range chunk {
			sourceIDs[j] = m.ID
		}

		// Store the summary as a user memory (long-term semantic fact)
		semantic := &types.Memory{
			UserID:     userID,
			Content:    summary,
			Type:       types.MemoryTypeUser,
			Importance: types.ImportanceMedium,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
			Metadata: map[string]interface{}{
				"consolidated_from": len(chunk),
				"source":            "consolidation",
				"source_ids":        strings.Join(sourceIDs, ","),
			},
		}
		if _, err := s.memSvc.CreateMemory(ctx, semantic); err != nil {
			log.Printf("consolidation: create semantic memory for user %s: %v", userID, err)
			continue
		}

		// Archive the original episodic memories
		for _, m := range chunk {
			if err := s.memSvc.ArchiveMemory(ctx, m.ID); err != nil {
				log.Printf("consolidation: archive memory %s: %v", m.ID, err)
			}
		}
		consolidated += len(chunk)
	}

	if consolidated > 0 {
		log.Printf("consolidation: user %s consolidated %d episodic memories into semantic summaries", userID, consolidated)
	}
	return nil
}

// summarizeChunk asks the LLM to summarize a batch of memories into key semantic facts.
func (s *Service) summarizeChunk(ctx context.Context, memories []*types.Memory) (string, error) {
	if s.llm == nil {
		// Fallback: simple concatenation
		parts := make([]string, len(memories))
		for i, m := range memories {
			parts[i] = m.Content
		}
		return strings.Join(parts, ". "), nil
	}

	var contents []string
	for _, m := range memories {
		if m.Content != "" {
			contents = append(contents, m.Content)
		}
	}

	prompt := fmt.Sprintf(`Summarize the following episodic memories into 2-4 key semantic facts that capture the most important information. Be concise.

Memories:
%s

Summary of key facts:`, strings.Join(contents, "\n---\n"))

	resp, err := s.llm.Complete(ctx, &llm.CompletionRequest{
		Model: "gpt-4o-mini",
		Messages: []llm.Message{
			{Role: "system", Content: "You summarize episodic memories into concise semantic facts for long-term storage."},
			{Role: "user", Content: prompt},
		},
		Temperature: 0.3,
		MaxTokens:   300,
	})
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(resp.Content), nil
}
