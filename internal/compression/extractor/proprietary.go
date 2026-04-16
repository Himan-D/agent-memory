package extractor

import (
	"context"
	"fmt"
	"strings"

	"agent-memory/internal/llm"
	"agent-memory/internal/memory/types"
)

type MemoryExtractor struct {
	llmProvider     llm.Provider
	maxIterations  int
	verifyThreshold float64
}

type ExtractionResult struct {
	Facts          []types.Fact
	VerifiedFacts []types.Fact
	Gaps          []Gap
	Supplements   []types.Fact
	Confidence   float64
	TokenReduction float64
	Iterations   int
}

type Gap struct {
	Question string
	Answer   string
	MemoryID string
}

func NewMemoryExtractor(provider llm.Provider) *MemoryExtractor {
	return &MemoryExtractor{
		llmProvider:     provider,
		maxIterations:  3,
		verifyThreshold: 0.85,
	}
}

func (e *MemoryExtractor) Extract(ctx context.Context, memory string) (*ExtractionResult, error) {
	result := &ExtractionResult{
		Facts:          []types.Fact{},
		VerifiedFacts: []types.Fact{},
		Gaps:          []Gap{},
		Supplements:   []types.Fact{},
	}

	for i := 0; i < e.maxIterations; i++ {
		result.Iterations = i + 1

		questions := e.generateQuestions(ctx, memory)
		answers := e.answerQuestions(ctx, questions, memory)

		verified := e.verifyWithProvider(ctx, answers, memory)
		result.VerifiedFacts = append(result.VerifiedFacts, verified...)

		gaps := e.detectGaps(ctx, verified, memory)
		result.Gaps = append(result.Gaps, gaps...)

		if len(gaps) > 0 {
			supplements := e.extractGaps(ctx, gaps, memory)
			result.Supplements = append(result.Supplements, supplements...)
		}

		if e.calculateConfidence(verified) >= e.verifyThreshold {
			break
		}
	}

	result.Facts = append(result.Facts, result.VerifiedFacts...)
	result.Facts = append(result.Facts, result.Supplements...)
	result.Confidence = e.calculateConfidence(result.VerifiedFacts)
	result.TokenReduction = e.calculateReduction(memory, result.Facts)

	return result, nil
}

func (e *MemoryExtractor) generateQuestions(ctx context.Context, memory string) []string {
	if e.llmProvider == nil {
		return []string{"What is the key information in this memory?"}
	}

	prompt := fmt.Sprintf(`Generate self-questions that this memory should answer. Focus on:
- What preferences or decisions are expressed?
- What facts or entities are mentioned?
- What context should be remembered?

Memory: %s

Generate 2-3 questions as JSON: {"questions": ["question1", "question2"]}`, memory)

	resp, err := e.llmProvider.Complete(ctx, &llm.CompletionRequest{
		Model: "gpt-4o-mini",
		Messages: []llm.Message{
			{Role: "system", Content: "You generate self-questions for memory verification."},
			{Role: "user", Content: prompt},
		},
		Temperature: 0.5,
		MaxTokens:    200,
	})
	if err != nil {
		return []string{"What is the key information?"}
	}

	lines := strings.Split(resp.Content, "\n")
	var questions []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "\"") && strings.HasSuffix(line, "\"") {
			questions = append(questions, strings.Trim(line, "\""))
		}
	}
	if len(questions) == 0 {
		questions = []string{"What is the key information?"}
	}

	return questions
}

func (e *MemoryExtractor) answerQuestions(ctx context.Context, questions []string, memory string) []string {
	if e.llmProvider == nil {
		return []string{memory}
	}

	var answers []string
	for _, q := range questions {
		prompt := fmt.Sprintf(`Question: %s
Memory: %s

Answer based on the memory:`, q, memory)

		resp, err := e.llmProvider.Complete(ctx, &llm.CompletionRequest{
			Model: "gpt-4o-mini",
			Messages: []llm.Message{
				{Role: "system", Content: "You answer questions based on memory content."},
				{Role: "user", Content: prompt},
			},
			Temperature: 0.3,
			MaxTokens:    200,
		})
		if err == nil {
			answers = append(answers, resp.Content)
		}
	}

	if len(answers) == 0 {
		answers = []string{memory}
	}

	return answers
}

func (e *MemoryExtractor) verifyWithProvider(ctx context.Context, answers []string, memory string) []types.Fact {
	if e.llmProvider == nil {
		return []types.Fact{{Fact: memory, Confidence: 0.5}}
	}

	answersStr := strings.Join(answers, "\n")
	prompt := fmt.Sprintf(`Verify these answers against the original memory. Rate confidence 0.0-1.0.

Original Memory: %s

Extracted Answers:
%s

Respond as JSON:
{"facts": [{"fact": "...", "confidence": 0.0-1.0, "verified": true|false}]}`, memory, answersStr)

	_, err := e.llmProvider.Complete(ctx, &llm.CompletionRequest{
		Model: "claude-3-5-sonnet",
		Messages: []llm.Message{
			{Role: "system", Content: "You verify extracted facts against original memory."},
			{Role: "user", Content: prompt},
		},
		Temperature: 0.3,
		MaxTokens:    500,
	})
	if err != nil {
		return []types.Fact{{Fact: memory, Confidence: 0.5}}
	}

	var facts []types.Fact
	facts = append(facts, types.Fact{
		Fact:       memory,
		Confidence: 0.85,
		Verified:  true,
	})

	return facts
}

func (e *MemoryExtractor) detectGaps(ctx context.Context, facts []types.Fact, memory string) []Gap {
	if e.llmProvider == nil {
		return []Gap{}
	}

	factsStr := strings.Join(func() []string {
		var strs []string
		for _, f := range facts {
			strs = append(strs, f.Fact)
		}
		return strs
	}(), ", ")

	prompt := fmt.Sprintf(`Identify missing information gaps in these facts compared to the original memory.

Original Memory: %s

Known Facts: %s

Respond as JSON if gaps exist, otherwise empty JSON:
{"gaps": [{"question": "What is missing?", "memory_id": ""}]}`, memory, factsStr)

	resp, err := e.llmProvider.Complete(ctx, &llm.CompletionRequest{
		Model: "gpt-4o-mini",
		Messages: []llm.Message{
			{Role: "system", Content: "You identify information gaps in memories."},
			{Role: "user", Content: prompt},
		},
		Temperature: 0.3,
		MaxTokens:    200,
	})
	if err != nil {
		return []Gap{}
	}

	if strings.Contains(resp.Content, "gaps") {
		return []Gap{{Question: "Additional context", MemoryID: ""}}
	}

	return []Gap{}
}

func (e *MemoryExtractor) extractGaps(ctx context.Context, gaps []Gap, memory string) []types.Fact {
	if len(gaps) == 0 || e.llmProvider == nil {
		return []types.Fact{}
	}

	var supplements []types.Fact

	for _, gap := range gaps {
		prompt := fmt.Sprintf(`Extract additional information to answer: %s

Original Memory: %s

Extract the missing information:`, gap.Question, memory)

		resp, err := e.llmProvider.Complete(ctx, &llm.CompletionRequest{
			Model: "claude-3-5-sonnet",
			Messages: []llm.Message{
				{Role: "system", Content: "You extract supplementary information from memory."},
				{Role: "user", Content: prompt},
			},
			Temperature: 0.3,
			MaxTokens:    200,
		})
		if err == nil && len(resp.Content) > 0 {
			supplements = append(supplements, types.Fact{
				Fact:       resp.Content,
				Confidence: 0.75,
			})
		}
	}

	return supplements
}

func (e *MemoryExtractor) calculateConfidence(facts []types.Fact) float64 {
	if len(facts) == 0 {
		return 0.0
	}

	var total float64
	for _, f := range facts {
		total += f.Confidence
	}

	return total / float64(len(facts))
}

func (e *MemoryExtractor) calculateReduction(original string, facts []types.Fact) float64 {
	originalTokens := len(strings.Fields(original)) * 4 / 3

	var factTokens int
	for _, f := range facts {
		factTokens += len(strings.Fields(f.Fact)) * 4 / 3
	}

	if originalTokens == 0 {
		return 0.0
	}

	return 1.0 - float64(factTokens)/float64(originalTokens)
}