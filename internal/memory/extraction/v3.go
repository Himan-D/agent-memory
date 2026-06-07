package extraction

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"agent-memory/internal/llm"
	"agent-memory/internal/memory/types"
)

type Fact struct {
	Content string `json:"content"`
	Hash    string `json:"hash"`
}

type ExtractionResult struct {
	Facts []Fact `json:"facts"`
}

type contextSearcher interface {
	SearchMemories(ctx context.Context, req *types.SearchRequest) ([]types.MemoryResult, error)
}

type memoryCreator interface {
	CreateMemory(ctx context.Context, mem *types.Memory) (*types.Memory, error)
}

// ExtractionV3 implements Mem0 v3 single-pass ADD-only extraction.
type ExtractionV3 struct {
	llmProvider llm.Provider
	searcher    contextSearcher
	dedup       map[string]bool
	dedupMu     sync.RWMutex
	model       string
}

func NewExtractionV3(provider llm.Provider, searcher contextSearcher) *ExtractionV3 {
	return &ExtractionV3{
		llmProvider: provider,
		searcher:    searcher,
		dedup:       make(map[string]bool),
		model:       "gpt-4o-mini",
	}
}

func (e *ExtractionV3) Extract(ctx context.Context, userID, input string) (*ExtractionResult, error) {
	if e.llmProvider == nil {
		return nil, fmt.Errorf("extraction v3: llm provider required")
	}
	if input == "" {
		return &ExtractionResult{}, nil
	}

	var contextLines []string
	if e.searcher != nil {
		results, err := e.searcher.SearchMemories(ctx, &types.SearchRequest{
			Query:  input,
			UserID: userID,
			Limit:  10,
		})
		if err == nil {
			for _, r := range results {
				if r.Text != "" {
					contextLines = append(contextLines, r.Text)
				}
			}
		}
	}

	prompt := buildV3ExtractionPrompt(input, contextLines)
	resp, err := e.llmProvider.Complete(ctx, &llm.CompletionRequest{
		Model: e.model,
		Messages: []llm.Message{
			{Role: "system", Content: "Extract atomic facts from input. Return JSON array of strings only."},
			{Role: "user", Content: prompt},
		},
		Temperature: 0.1,
		MaxTokens:   1024,
	})
	if err != nil {
		return nil, fmt.Errorf("extraction v3: llm complete: %w", err)
	}

	rawFacts := parseV3Facts(resp.Content)
	var facts []Fact
	for _, content := range rawFacts {
		content = strings.TrimSpace(content)
		if content == "" {
			continue
		}
		hash := md5Hash(content)
		if e.exists(hash) {
			continue
		}
		e.put(hash)
		facts = append(facts, Fact{Content: content, Hash: hash})
	}

	return &ExtractionResult{Facts: facts}, nil
}

func (e *ExtractionV3) exists(hash string) bool {
	e.dedupMu.RLock()
	defer e.dedupMu.RUnlock()
	return e.dedup[hash]
}

func (e *ExtractionV3) put(hash string) {
	e.dedupMu.Lock()
	defer e.dedupMu.Unlock()
	e.dedup[hash] = true
}

func buildV3ExtractionPrompt(input string, context []string) string {
	var b strings.Builder
	b.WriteString("Extract all discrete facts from the following input.\n")
	b.WriteString("Each fact should be atomic and self-contained.\n\n")
	if len(context) > 0 {
		b.WriteString("Related existing memories (context only):\n")
		for _, line := range context {
			b.WriteString("- ")
			b.WriteString(line)
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}
	b.WriteString("Input:\n")
	b.WriteString(input)
	b.WriteString("\n\nReturn a JSON array of fact strings.")
	return b.String()
}

func parseV3Facts(content string) []string {
	content = strings.TrimSpace(content)
	if strings.HasPrefix(content, "```") {
		lines := strings.Split(content, "\n")
		if len(lines) > 2 {
			content = strings.Join(lines[1:len(lines)-1], "\n")
		}
	}

	var facts []string
	if err := json.Unmarshal([]byte(content), &facts); err == nil {
		return facts
	}

	var wrapped struct {
		Facts []string `json:"facts"`
	}
	if err := json.Unmarshal([]byte(content), &wrapped); err == nil && len(wrapped.Facts) > 0 {
		return wrapped.Facts
	}

	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "- "))
		if line != "" {
			facts = append(facts, line)
		}
	}
	return facts
}

func md5Hash(s string) string {
	sum := md5.Sum([]byte(s))
	return hex.EncodeToString(sum[:])
}
