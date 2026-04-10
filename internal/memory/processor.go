package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"agent-memory/internal/llm"
)

type LLMProvider interface {
	Complete(ctx context.Context, req *llm.CompletionRequest) (*llm.CompletionResponse, error)
}

type MemoryProcessor struct {
	llmProvider    LLMProvider
	promptRenderer *PromptRenderer
	config         *Config
}

func NewMemoryProcessor(llmProvider LLMProvider) *MemoryProcessor {
	return &MemoryProcessor{
		llmProvider:    llmProvider,
		promptRenderer: NewPromptRenderer(),
		config:         DefaultConfig(),
	}
}

func NewMemoryProcessorWithConfig(llmProvider LLMProvider, cfg *Config) *MemoryProcessor {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	return &MemoryProcessor{
		llmProvider:    llmProvider,
		promptRenderer: NewPromptRenderer(),
		config:         cfg,
	}
}

func (p *MemoryProcessor) SetConfig(cfg *Config) {
	if cfg != nil {
		p.config = cfg
	}
}

func (p *MemoryProcessor) ProcessContent(ctx context.Context, content, userID string, memType MemoryType) (*MemoryProcessingResult, error) {
	if !p.config.Enabled {
		return &MemoryProcessingResult{
			ProcessedContent: content,
			Importance:       p.config.DefaultImportance,
			ShouldStore:      true,
			Categories:       []string{},
		}, nil
	}

	var result MemoryProcessingResult
	var err error

	result.ShouldStore, result.Importance, result.Reason, err = p.shouldStore(ctx, content)
	if err != nil {
		return nil, fmt.Errorf("should store check: %w", err)
	}

	importanceScore := 5
	if result.Importance == ImportanceHigh {
		importanceScore = 8
	} else if result.Importance == ImportanceLow {
		importanceScore = 2
	}

	if !result.ShouldStore && importanceScore < 5 {
		result.ProcessedContent = content
		return &result, nil
	}

	if p.config.AutoExtractFacts {
		result.Facts, err = p.extractFacts(ctx, content, userID, string(memType))
		if err != nil {
			return nil, fmt.Errorf("extract facts: %w", err)
		}

		if len(result.Facts) > 0 {
			var facts []string
			for _, f := range result.Facts {
				facts = append(facts, f.Fact)
			}
			result.ProcessedContent = strings.Join(facts, "; ")
		} else {
			result.ProcessedContent = content
		}
	} else {
		result.ProcessedContent = content
	}

	if p.config.AutoExtractEntities {
		result.Entities, err = p.extractEntities(ctx, content)
		if err != nil {
			return nil, fmt.Errorf("extract entities: %w", err)
		}
	}

	result.Categories, err = p.extractCategories(ctx, content)
	if err != nil {
		return nil, fmt.Errorf("extract categories: %w", err)
	}

	if result.ProcessedContent == "" {
		result.ProcessedContent = content
	}

	if result.Importance == "" {
		result.Importance = p.config.DefaultImportance
	}

	return &result, nil
}

func (p *MemoryProcessor) shouldStore(ctx context.Context, content string) (bool, string, string, error) {
	if p.llmProvider == nil {
		return true, ImportanceMedium, "no_llm_provider", nil
	}

	userPrompt, err := p.promptRenderer.RenderShouldStore(content)
	if err != nil {
		return true, ImportanceMedium, fmt.Sprintf("prompt error: %v", err), nil
	}

	resp, err := p.llmProvider.Complete(ctx, &llm.CompletionRequest{
		Model: "gpt-4o",
		Messages: []llm.Message{
			{Role: "system", Content: p.promptRenderer.GetSystemPromptShouldStore()},
			{Role: "user", Content: userPrompt},
		},
		Temperature: 0.3,
		MaxTokens:   500,
	})
	if err != nil {
		return true, ImportanceMedium, fmt.Sprintf("llm error: %v", err), nil
	}

	content = strings.TrimSpace(resp.Content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var result ShouldStoreResult
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return true, ImportanceMedium, fmt.Sprintf("parse error: %v", err), nil
	}

	importance := ImportanceMedium
	if result.Importance >= 7 {
		importance = ImportanceHigh
	} else if result.Importance <= 3 {
		importance = ImportanceLow
	}

	return result.Store, importance, result.Reason, nil
}

func (p *MemoryProcessor) extractFacts(ctx context.Context, content, userID, memType string) ([]ExtractedFact, error) {
	if p.llmProvider == nil {
		return nil, nil
	}

	userPrompt, err := p.promptRenderer.RenderExtractFacts(content, userID, memType)
	if err != nil {
		return nil, err
	}

	resp, err := p.llmProvider.Complete(ctx, &llm.CompletionRequest{
		Model: "gpt-4o",
		Messages: []llm.Message{
			{Role: "system", Content: p.promptRenderer.GetSystemPromptExtractFacts()},
			{Role: "user", Content: userPrompt},
		},
		Temperature: 0.3,
		MaxTokens:   1000,
	})
	if err != nil {
		return nil, err
	}

	resultContent := strings.TrimSpace(resp.Content)
	resultContent = strings.TrimPrefix(resultContent, "```json")
	resultContent = strings.TrimPrefix(resultContent, "```")
	resultContent = strings.TrimSuffix(resultContent, "```")
	resultContent = strings.TrimSpace(resultContent)

	var facts []ExtractedFact
	if err := json.Unmarshal([]byte(resultContent), &facts); err != nil {
		return nil, fmt.Errorf("parse facts: %w, content: %s", err, resultContent)
	}

	return facts, nil
}

func (p *MemoryProcessor) extractEntities(ctx context.Context, content string) ([]ExtractedEntity, error) {
	if p.llmProvider == nil {
		return nil, nil
	}

	userPrompt, err := p.promptRenderer.RenderExtractEntities(content)
	if err != nil {
		return nil, err
	}

	resp, err := p.llmProvider.Complete(ctx, &llm.CompletionRequest{
		Model: "gpt-4o",
		Messages: []llm.Message{
			{Role: "system", Content: p.promptRenderer.GetSystemPromptExtractEntities()},
			{Role: "user", Content: userPrompt},
		},
		Temperature: 0.3,
		MaxTokens:   500,
	})
	if err != nil {
		return nil, err
	}

	resultContent := strings.TrimSpace(resp.Content)
	resultContent = strings.TrimPrefix(resultContent, "```json")
	resultContent = strings.TrimPrefix(resultContent, "```")
	resultContent = strings.TrimSuffix(resultContent, "```")
	resultContent = strings.TrimSpace(resultContent)

	var entities []ExtractedEntity
	if err := json.Unmarshal([]byte(resultContent), &entities); err != nil {
		return nil, fmt.Errorf("parse entities: %w, content: %s", err, resultContent)
	}

	return entities, nil
}

func (p *MemoryProcessor) extractCategories(ctx context.Context, content string) ([]string, error) {
	if p.llmProvider == nil {
		return nil, nil
	}

	userPrompt, err := p.promptRenderer.RenderExtractCategories(content)
	if err != nil {
		return nil, err
	}

	resp, err := p.llmProvider.Complete(ctx, &llm.CompletionRequest{
		Model: "gpt-4o",
		Messages: []llm.Message{
			{Role: "system", Content: p.promptRenderer.GetSystemPromptExtractCategories()},
			{Role: "user", Content: userPrompt},
		},
		Temperature: 0.3,
		MaxTokens:   200,
	})
	if err != nil {
		return nil, err
	}

	resultContent := strings.TrimSpace(resp.Content)
	resultContent = strings.TrimPrefix(resultContent, "```json")
	resultContent = strings.TrimPrefix(resultContent, "```")
	resultContent = strings.TrimSuffix(resultContent, "```")
	resultContent = strings.TrimSpace(resultContent)

	var categories []string
	if err := json.Unmarshal([]byte(resultContent), &categories); err != nil {
		return nil, fmt.Errorf("parse categories: %w, content: %s", err, resultContent)
	}

	return categories, nil
}

func (p *MemoryProcessor) ResolveConflict(ctx context.Context, existingContent, existingImportance, newContent string) (*ConflictResolutionResult, error) {
	if p.llmProvider == nil {
		return &ConflictResolutionResult{
			Action: ConflictActionKeepBoth,
			Reason: "no_llm_provider",
		}, nil
	}

	userPrompt, err := p.promptRenderer.RenderResolveConflict(existingContent, existingImportance, newContent)
	if err != nil {
		return nil, err
	}

	resp, err := p.llmProvider.Complete(ctx, &llm.CompletionRequest{
		Model: "gpt-4o",
		Messages: []llm.Message{
			{Role: "system", Content: p.promptRenderer.GetSystemPromptResolveConflict()},
			{Role: "user", Content: userPrompt},
		},
		Temperature: 0.3,
		MaxTokens:   500,
	})
	if err != nil {
		return nil, err
	}

	resultContent := strings.TrimSpace(resp.Content)
	resultContent = strings.TrimPrefix(resultContent, "```json")
	resultContent = strings.TrimPrefix(resultContent, "```")
	resultContent = strings.TrimSuffix(resultContent, "```")
	resultContent = strings.TrimSpace(resultContent)

	var result ConflictResolutionResult
	if err := json.Unmarshal([]byte(resultContent), &result); err != nil {
		return &ConflictResolutionResult{
			Action: ConflictActionKeepBoth,
			Reason: fmt.Sprintf("parse error: %v", err),
		}, nil
	}

	return &result, nil
}

func (p *MemoryProcessor) InferFromMessages(ctx context.Context, messages []MessageInput) (*MemoryProcessingResult, error) {
	if len(messages) == 0 {
		return &MemoryProcessingResult{}, nil
	}

	var contentBuilder strings.Builder
	for _, msg := range messages {
		contentBuilder.WriteString(fmt.Sprintf("%s: %s\n", msg.Role, msg.Content))
	}

	return p.ProcessContent(ctx, contentBuilder.String(), "", MemoryTypeConversation)
}

type MessageInput struct {
	Role    string
	Content string
}

func (p *MemoryProcessor) IsEnabled() bool {
	return p.config.Enabled
}

func (p *MemoryProcessor) GetConfig() *Config {
	return p.config
}
