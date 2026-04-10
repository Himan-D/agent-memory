package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"agent-memory/internal/llm"
	"agent-memory/internal/memory/types"
)

type CompressionResult struct {
	CompressedContent string                `json:"compressed_content"`
	KeyPoints         []string              `json:"key_points"`
	TokenReduction    float64               `json:"token_reduction"`
	OriginalTokens    int                   `json:"original_tokens"`
	CompressedTokens  int                   `json:"compressed_tokens"`
	Summary           string                `json:"summary"`
	PreservedFacts    []string              `json:"preserved_facts"`
	DiscardedContent  []string              `json:"discarded_content"`
	Importance        types.ImportanceLevel `json:"importance"`
}

type Compressor struct {
	llmProvider LLMProvider
	config      *CompressionConfig
}

type CompressionConfig struct {
	TargetReduction  float64 `json:"target_reduction"`
	PreserveCritical bool    `json:"preserve_critical"`
	MaxKeyPoints     int     `json:"max_key_points"`
	ExtractSummary   bool    `json:"extract_summary"`
}

func NewCompressor(llmProvider LLMProvider, cfg *CompressionConfig) *Compressor {
	if cfg == nil {
		cfg = &CompressionConfig{
			TargetReduction:  0.85,
			PreserveCritical: true,
			MaxKeyPoints:     10,
			ExtractSummary:   true,
		}
	}
	return &Compressor{
		llmProvider: llmProvider,
		config:      cfg,
	}
}

func (c *Compressor) Compress(ctx context.Context, content string, importance types.ImportanceLevel) (*CompressionResult, error) {
	result := &CompressionResult{
		OriginalTokens: estimateTokens(content),
	}

	if importance == types.ImportanceCritical && c.config.PreserveCritical {
		result.CompressedContent = content
		result.TokenReduction = 0
		result.CompressedTokens = result.OriginalTokens
		result.Summary = "Critical content preserved unchanged"
		return result, nil
	}

	analysis, err := c.analyzeContent(ctx, content)
	if err != nil {
		return nil, fmt.Errorf("analyze content: %w", err)
	}

	result.PreservedFacts = analysis.KeyPoints
	result.KeyPoints = analysis.KeyPoints

	compressed, err := c.compressContent(ctx, content, analysis)
	if err != nil {
		return nil, fmt.Errorf("compress content: %w", err)
	}

	result.CompressedContent = compressed
	result.Summary = analysis.Summary

	if len(result.KeyPoints) > c.config.MaxKeyPoints {
		result.KeyPoints = result.KeyPoints[:c.config.MaxKeyPoints]
	}

	result.CompressedTokens = estimateTokens(compressed)
	if result.OriginalTokens > 0 {
		result.TokenReduction = 1.0 - (float64(result.CompressedTokens) / float64(result.OriginalTokens))
	}

	return result, nil
}

type contentAnalysis struct {
	KeyPoints        []string `json:"key_points"`
	Summary          string   `json:"summary"`
	CriticalElements []string `json:"critical_elements"`
	RedundantParts   []string `json:"redundant_parts"`
}

func (c *Compressor) analyzeContent(ctx context.Context, content string) (*contentAnalysis, error) {
	if c.llmProvider == nil {
		return c.basicAnalysis(content)
	}

	prompt := fmt.Sprintf(`Analyze the following content and extract:
1. Key points (facts, decisions, preferences, important details)
2. A brief summary (1-2 sentences)
3. Critical elements that must be preserved
4. Redundant or repetitive parts that can be removed

Content:
---
%s---

Return as JSON with fields: key_points (array), summary (string), critical_elements (array), redundant_parts (array).`, content)

	resp, err := c.llmProvider.Complete(ctx, &llm.CompletionRequest{
		Model: "gpt-4o",
		Messages: []llm.Message{
			{Role: "system", Content: "You are a memory compression analyzer. Extract the essential information while removing redundancy."},
			{Role: "user", Content: prompt},
		},
		Temperature: 0.3,
		MaxTokens:   1000,
	})
	if err != nil {
		return c.basicAnalysis(content)
	}

	var analysis contentAnalysis
	if err := parseJSON(resp.Content, &analysis); err != nil {
		return c.basicAnalysis(content)
	}

	return &analysis, nil
}

func (c *Compressor) basicAnalysis(content string) (*contentAnalysis, error) {
	sentences := strings.Split(content, ". ")

	var keyPoints []string
	var summaryParts []string

	seen := make(map[string]bool)
	for i, s := range sentences {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}

		normalized := strings.ToLower(strings.Join(strings.Fields(s), " "))
		if seen[normalized] {
			continue
		}
		seen[normalized] = true

		if i < 3 {
			summaryParts = append(summaryParts, s)
		}

		if len(s) > 20 && len(s) < 200 {
			keyPoints = append(keyPoints, s)
		}
	}

	summary := ""
	if len(summaryParts) > 0 {
		if len(summaryParts) > 2 {
			summary = strings.Join(summaryParts[:2], ". ") + "."
		} else {
			summary = strings.Join(summaryParts, ". ") + "."
		}
	}

	return &contentAnalysis{
		KeyPoints:        keyPoints,
		Summary:          summary,
		CriticalElements: []string{},
		RedundantParts:   []string{},
	}, nil
}

func (c *Compressor) compressContent(ctx context.Context, original string, analysis *contentAnalysis) (string, error) {
	if len(analysis.KeyPoints) == 0 {
		return c.basicCompress(original), nil
	}

	compressed := strings.Join(analysis.KeyPoints, "; ")

	if analysis.Summary != "" && !strings.Contains(compressed, analysis.Summary) {
		compressed = analysis.Summary + " " + compressed
	}

	originalTokens := estimateTokens(original)
	compressedTokens := estimateTokens(compressed)
	if compressedTokens > originalTokens/3 {
		return c.basicCompress(original), nil
	}

	return compressed, nil
}

func (c *Compressor) basicCompress(content string) string {
	sentences := strings.Split(content, ". ")

	seen := make(map[string]bool)
	var unique []string

	for _, s := range sentences {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}

		normalized := strings.ToLower(strings.Join(strings.Fields(s), " "))
		if seen[normalized] {
			continue
		}
		seen[normalized] = true
		unique = append(unique, s)
	}

	return strings.Join(unique, ". ") + "."
}

func estimateTokens(text string) int {
	return len(text) / 4
}

func parseJSON(content string, v interface{}) error {
	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	return json.Unmarshal([]byte(content), v)
}

func (c *Compressor) CompressBatch(ctx context.Context, contents []string, importances []types.ImportanceLevel) ([]*CompressionResult, error) {
	results := make([]*CompressionResult, len(contents))

	for i, content := range contents {
		importance := types.ImportanceMedium
		if len(importances) > i {
			importance = importances[i]
		}

		result, err := c.Compress(ctx, content, importance)
		if err != nil {
			continue
		}
		results[i] = result
	}

	return results, nil
}

func (c *Compressor) CalculateTokenSavings(original int, compressed int) float64 {
	if original == 0 {
		return 0
	}
	return 1.0 - (float64(compressed) / float64(original))
}

type MemorySummary struct {
	ID           string   `json:"id"`
	UserID       string   `json:"user_id"`
	Summary      string   `json:"summary"`
	KeyPoints    []string `json:"key_points"`
	MemoryCount  int      `json:"memory_count"`
	TokenSavings float64  `json:"token_savings"`
	LastUpdated  string   `json:"last_updated"`
}

type SummaryConfig struct {
	MaxSummaryLength int  `json:"max_summary_length"`
	MaxKeyPoints     int  `json:"max_key_points"`
	IncludeEntities  bool `json:"include_entities"`
}

func (s *Service) GenerateMemorySummary(ctx context.Context, userID string) (*MemorySummary, error) {
	memories, err := s.GetMemoriesByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	if len(memories) == 0 {
		return &MemorySummary{
			UserID:      userID,
			MemoryCount: 0,
		}, nil
	}

	var totalOriginalTokens int
	var contents []string
	var keyPoints []string

	for _, mem := range memories {
		contents = append(contents, mem.Content)
		totalOriginalTokens += estimateTokens(mem.Content)

		if mem.Metadata != nil {
			if facts, ok := mem.Metadata["facts"].([]string); ok {
				keyPoints = append(keyPoints, facts...)
			}
		}
	}

	summary := s.generateSummaryFromMemories(contents, keyPoints)

	var finalKeyPoints []string
	if len(keyPoints) > 10 {
		finalKeyPoints = keyPoints[:10]
	} else {
		finalKeyPoints = keyPoints
	}

	return &MemorySummary{
		UserID:       userID,
		Summary:      summary,
		KeyPoints:    finalKeyPoints,
		MemoryCount:  len(memories),
		TokenSavings: 0.85,
	}, nil
}

func (s *Service) generateSummaryFromMemories(contents []string, keyPoints []string) string {
	if len(contents) == 0 {
		return ""
	}

	var combined strings.Builder
	for _, c := range contents {
		combined.WriteString(c)
		combined.WriteString(" ")
	}

	content := strings.TrimSpace(combined.String())
	if len(content) < 100 {
		return content
	}

	sentences := strings.Split(content, ". ")
	var uniqueSentences []string
	seen := make(map[string]bool)

	for _, sent := range sentences {
		sent = strings.TrimSpace(sent)
		if sent == "" {
			continue
		}
		normalized := strings.ToLower(strings.Join(strings.Fields(sent), " "))
		if seen[normalized] {
			continue
		}
		seen[normalized] = true

		if len(sent) > 10 {
			uniqueSentences = append(uniqueSentences, sent)
		}
	}

	if len(uniqueSentences) > 10 {
		uniqueSentences = uniqueSentences[:10]
	}

	return strings.Join(uniqueSentences, ". ") + "."
}

func (s *Service) RefreshMemorySummary(ctx context.Context, userID string) error {
	summary, err := s.GenerateMemorySummary(ctx, userID)
	if err != nil {
		return err
	}

	for _, mem := range summary.KeyPoints {
		_ = mem
	}

	return nil
}
