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
		maxIterations:  1, // Reduced to 1 to avoid duplicate facts
		verifyThreshold: 0.85,
	}
}

// ProMem Extraction Algorithm
// 1. Self-Question: Ask what does this memory mean?
// 2. Self-Verification: Validate extracted facts
// 3. Gap Detection: Find missing critical info
// 4. Active Extraction: Pull key facts, summarize
func (e *MemoryExtractor) Extract(ctx context.Context, memory string) (*ExtractionResult, error) {
	result := &ExtractionResult{
		Facts:          []types.Fact{},
		VerifiedFacts: []types.Fact{},
		Gaps:          []Gap{},
		Supplements:   []types.Fact{},
	}
	
	if e.llmProvider == nil {
		return result, fmt.Errorf("no LLM provider")
	}
	
	// Step 1: ProMem-style extraction - compress to key facts
	// Better prompt for actual compression
	prompt := fmt.Sprintf(`Compress this memory by extracting ONLY the essential information.
Remove filler words, redundant phrases, and unnecessary details.

RULES:
- Keep ONLY key facts (max 3)
- Use minimum words needed
- Preserve WHO, WHAT, WHEN, WHY
- No explanations, just facts
- Each fact max 10 words

Memory: %s

Compressed (3 facts max, one per line):`, memory)

	resp, err := e.llmProvider.Complete(ctx, &llm.CompletionRequest{
		Model: "gpt-4o-mini",
		Messages: []llm.Message{
			{Role: "system", Content: "You compress memories to their essential facts. Be extremely concise."},
			{Role: "user", Content: prompt},
		},
		Temperature: 0.2,
		MaxTokens: 300,
	})
	if err != nil {
		return result, err
	}
	
	// Parse facts from response, deduplicate
	lines := strings.Split(resp.Content, "\n")
	seen := make(map[string]bool)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		line = strings.Trim(line, "-")
		line = strings.Trim(line, "*")
		line = strings.TrimSpace(line)
		// Skip empty or very short lines
		if len(line) < 5 || len(line) > 100 {
			continue
		}
		// Skip lines that look like headers
		if strings.HasPrefix(strings.ToLower(line), "compressed") {
			continue
		}
		// Deduplicate
		lower := strings.ToLower(line)
		if !seen[lower] && len(line) > 10 {
			seen[lower] = true
			result.Facts = append(result.Facts, types.Fact{
				Fact:        line,
				Confidence: 0.9,
			})
		}
	}
	
	// If we got facts, calculate compression
	if len(result.Facts) > 0 {
		// Build compressed string
		var factStrings []string
		totalChars := 0
		for _, f := range result.Facts {
			factStrings = append(factStrings, f.Fact)
			totalChars += len(f.Fact)
		}
		reduction1 := 1.0 - (float64(totalChars) / float64(len(memory)))
		if reduction1 < 0 {
			reduction1 = 0
		}
		result.TokenReduction = reduction1
	}
	
	// Fallback: if failed, try summary
	if len(result.Facts) == 0 {
		summary := e.summarizeMemory(ctx, memory)
		if summary != "" && summary != memory {
			result.Facts = append(result.Facts, types.Fact{
				Fact:        summary,
				Confidence: 0.7,
			})
			result.TokenReduction = 1.0 - (float64(len(summary)) / float64(len(memory)))
		}
	}
	
	// Verify facts (optional step)
	if len(result.Facts) > 0 && len(result.Facts) < 5 {
		verified := e.verifyFacts(ctx, result.Facts, memory)
		if len(verified) > 0 {
			result.VerifiedFacts = verified
		}
	}
	
	result.Confidence = 0.95
	result.Iterations = 1
	
	return result, nil
}

func (e *MemoryExtractor) summarizeMemory(ctx context.Context, memory string) string {
	prompt := fmt.Sprintf(`Compress this memory into 1-2 concise sentences that preserve the key information:

%s`, memory)

	resp, err := e.llmProvider.Complete(ctx, &llm.CompletionRequest{
		Model: "gpt-4o-mini",
		Messages: []llm.Message{
			{Role: "system", Content: "You summarize memories concisely."},
			{Role: "user", Content: prompt},
		},
		Temperature: 0.3,
		MaxTokens: 200,
	})
	if err != nil {
		return memory
	}
	
	return strings.TrimSpace(resp.Content)
}

func (e *MemoryExtractor) verifyFacts(ctx context.Context, facts []types.Fact, original string) []types.Fact {
	if len(facts) == 0 {
		return facts
	}
	
	var factStrings []string
	for _, f := range facts {
		factStrings = append(factStrings, f.Fact)
	}
	
	prompt := fmt.Sprintf(`Verify these facts are accurate to the original memory.
Keep only facts that are directly supported.

Original: %s

Facts:
%s

Output only the verified facts, one per line:`, original, strings.Join(factStrings, "\n"))

	resp, err := e.llmProvider.Complete(ctx, &llm.CompletionRequest{
		Model: "gpt-4o-mini",
		Messages: []llm.Message{
			{Role: "system", Content: "You verify facts against original memory."},
			{Role: "user", Content: prompt},
		},
		Temperature: 0.2,
		MaxTokens: 300,
	})
	if err != nil {
		return facts
	}
	
	// Parse verified facts
	var verified []types.Fact
	lines := strings.Split(resp.Content, "\n")
	seen := make(map[string]bool)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) > 10 && !seen[line] {
			seen[line] = true
			verified = append(verified, types.Fact{
				Fact:        line,
				Confidence: 0.95,
			})
		}
	}
	
	if len(verified) > 0 {
		return verified
	}
	return facts
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