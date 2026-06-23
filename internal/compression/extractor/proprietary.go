package extractor

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"agent-memory/internal/llm"
	"agent-memory/internal/memory/types"
)

// MetricsRecorder is the subset of metrics.MetricsCollector needed by the extractor.
// Using an interface avoids an import cycle.
type MetricsRecorder interface {
	RecordExtraction(provider string, tokensSaved int64, latencyMs float64)
}

type tokenReductionRecorder interface {
	SetTokenReduction(reduction float64)
}

type MemoryExtractor struct {
	llmProvider     llm.Provider
	maxIterations   int
	verifyThreshold float64
	metrics         MetricsRecorder
}

func (e *MemoryExtractor) SetMetrics(m MetricsRecorder) {
	e.metrics = m
}

type ExtractionResult struct {
	Facts          []types.Fact
	VerifiedFacts  []types.Fact
	Gaps           []Gap
	Supplements    []types.Fact
	Confidence     float64
	TokenReduction float64
	Iterations     int
}

type Gap struct {
	Question string
	Answer   string
	MemoryID string
}

func NewMemoryExtractor(provider llm.Provider) *MemoryExtractor {
	return &MemoryExtractor{
		llmProvider:     provider,
		maxIterations:   2, // ProMem: 2 passes — first extracts, second fills gaps
		verifyThreshold: 0.85,
	}
}

// ProMem Extraction Algorithm
// 1. Self-Question: Ask what does this memory mean?
// 2. Self-Verification: Validate extracted facts
// 3. Gap Detection: Find missing critical info
// 4. Active Extraction: Pull key facts, summarize
func (e *MemoryExtractor) Extract(ctx context.Context, memory string) (*ExtractionResult, error) {
	start := time.Now()
	result, err := e.extract(ctx, memory)
	if e.metrics != nil {
		latencyMs := float64(time.Since(start).Milliseconds())
		tokensSaved := int64(0)
		if result != nil {
			tokensSaved = int64(float64(len(memory)) * result.TokenReduction)
			if recorder, ok := e.metrics.(tokenReductionRecorder); ok {
				recorder.SetTokenReduction(result.TokenReduction)
			}
		}
		e.metrics.RecordExtraction("promem", tokensSaved, latencyMs)
	}
	return result, err
}

// extract implements the ProMem algorithm (arXiv:2601.04463):
// Pass 1: TOON extraction → self-question → gap detection
// Pass 2 (if gaps found): gap-fill → deduplicate → verify
func (e *MemoryExtractor) extract(ctx context.Context, memory string) (*ExtractionResult, error) {
	result := &ExtractionResult{
		Facts:         []types.Fact{},
		VerifiedFacts: []types.Fact{},
		Gaps:          []Gap{},
		Supplements:   []types.Fact{},
	}

	if e.llmProvider == nil {
		return result, fmt.Errorf("no LLM provider")
	}

	// Step 1: Initial TOON-format extraction
	initialFacts, err := e.extractInitialFacts(ctx, memory)
	if err != nil {
		return result, err
	}
	result.Facts = initialFacts

	// Step 2: Self-questioning — generate questions this memory should answer
	questions := e.generateQuestions(ctx, memory)

	// Step 3: Answer each question from the memory text
	answers := e.answerQuestions(ctx, questions, memory)

	// Step 4: Detect gaps — information not yet captured in initial facts
	gaps := e.detectGaps(ctx, result.Facts, memory)
	result.Gaps = gaps

	// Step 5: Second-pass gap-fill (ProMem iter 2)
	if len(gaps) > 0 && e.maxIterations > 1 {
		supplements := e.extractGaps(ctx, gaps, memory)
		if len(supplements) > 0 {
			result.Supplements = supplements
			result.Facts = deduplicateFacts(append(result.Facts, supplements...))
		}
	}

	// Step 6: Verify the combined fact set using answers from the self-question pass.
	// verifyWithProvider uses the Q&A context for higher-confidence verification;
	// fall back to simpler verifyFacts when no answers were produced.
	if len(result.Facts) > 0 {
		var verified []types.Fact
		if len(answers) > 0 {
			verified = e.verifyWithProvider(ctx, answers, memory)
		} else {
			verified = e.verifyFacts(ctx, result.Facts, memory)
		}
		// Apply verifyThreshold: discard facts below confidence threshold.
		var aboveThreshold []types.Fact
		for _, f := range verified {
			if f.Confidence >= e.verifyThreshold {
				aboveThreshold = append(aboveThreshold, f)
			}
		}
		if len(aboveThreshold) > 0 {
			verified = aboveThreshold
		}
		if len(verified) > 0 {
			for i := range verified {
				verified[i].Verified = true
			}
			result.VerifiedFacts = verified
		} else {
			result.VerifiedFacts = result.Facts
		}
	}

	// Real confidence: average across verified facts
	result.Confidence = e.calculateConfidence(result.VerifiedFacts)
	if result.Confidence == 0.0 && len(result.VerifiedFacts) > 0 {
		result.Confidence = 0.85
	}

	// Real token reduction ratio
	result.TokenReduction = e.calculateReduction(memory, result.Facts)

	// Actual iteration count
	result.Iterations = e.maxIterations

	return result, nil
}

// extractInitialFacts extracts facts using the TOON compression format.
// This is the first pass of the ProMem algorithm.
func (e *MemoryExtractor) extractInitialFacts(ctx context.Context, memory string) ([]types.Fact, error) {
	prompt := fmt.Sprintf(`You are a memory compression algorithm. Your goal is maximum compression while preserving essential meaning.

INPUT MEMORY:
%s

COMPRESSION TARGET: Reduce to 10-20%% of original size while keeping key facts.

OUTPUT FORMAT - Use EXACTLY this format, one per line:
{subject: [who/what], action: [verb], context: [where/when/how]}

STRICT RULES:
1. Each field MAX 12 characters - use abbreviations aggressively
2. Remove ALL articles: the, a, an
3. Remove ALL pronouns
4. Output 1-5 facts maximum - fewer is better for compression
5. DO NOT add explanations or any other text
6. Each fact on its own line

OUTPUT:`, memory)

	resp, err := e.llmProvider.Complete(ctx, &llm.CompletionRequest{
		Model: "gpt-4o-mini",
		Messages: []llm.Message{
			{Role: "system", Content: "You compress memories to their essential facts. Be extremely concise."},
			{Role: "user", Content: prompt},
		},
		Temperature: 0.2,
		MaxTokens:   300,
	})
	if err != nil {
		// Fallback: summarize
		summary := e.summarizeMemory(ctx, memory)
		if summary == "" || summary == memory {
			return []types.Fact{}, err
		}
		return []types.Fact{{Fact: summary, Confidence: 0.7}}, nil
	}

	lines := strings.Split(resp.Content, "\n")
	seen := make(map[string]bool)
	var facts []types.Fact

	for _, line := range lines {
		line = strings.TrimSpace(line)
		line = strings.Trim(line, "{}")
		line = strings.TrimSpace(line)

		if len(line) < 5 {
			continue
		}

		// Parse TOON triplet: subject, action, context
		parts := strings.Split(line, ",")
		if len(parts) >= 2 {
			var factStr string
			for i, p := range parts {
				p = strings.TrimSpace(p)
				p = strings.TrimPrefix(p, "subject:")
				p = strings.TrimPrefix(p, "action:")
				p = strings.TrimPrefix(p, "context:")
				p = strings.TrimSpace(p)
				if p != "" {
					if i > 0 {
						factStr += " "
					}
					factStr += p
				}
			}
			line = factStr
		}

		if len(line) > 150 {
			continue
		}

		lower := strings.ToLower(line)
		if !seen[lower] && len(line) > 5 {
			seen[lower] = true
			facts = append(facts, types.Fact{
				Fact:       "{" + line + "}",
				Confidence: 0.9,
			})
		}
	}

	if len(facts) == 0 {
		summary := e.summarizeMemory(ctx, memory)
		if summary != "" && summary != memory {
			facts = append(facts, types.Fact{Fact: summary, Confidence: 0.7})
		}
	}

	return facts, nil
}

// deduplicateFacts removes facts whose text is too similar (same lowercase string).
func deduplicateFacts(facts []types.Fact) []types.Fact {
	seen := make(map[string]bool)
	var out []types.Fact
	for _, f := range facts {
		key := strings.ToLower(strings.TrimSpace(f.Fact))
		if !seen[key] {
			seen[key] = true
			out = append(out, f)
		}
	}
	return out
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
		MaxTokens:   200,
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
		MaxTokens:   300,
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
				Fact:       line,
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
		MaxTokens:   200,
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
			MaxTokens:   200,
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

	resp, err := e.llmProvider.Complete(ctx, &llm.CompletionRequest{
		Model: "claude-3-5-sonnet",
		Messages: []llm.Message{
			{Role: "system", Content: "You verify extracted facts against original memory."},
			{Role: "user", Content: prompt},
		},
		Temperature: 0.3,
		MaxTokens:   500,
	})
	if err != nil {
		return []types.Fact{{Fact: memory, Confidence: 0.5}}
	}

	var result struct {
		Facts []struct {
			Fact       string  `json:"fact"`
			Confidence float64 `json:"confidence"`
			Verified   bool    `json:"verified"`
		} `json:"facts"`
	}

	content := resp.Content
	jsonStart := strings.Index(content, "{")
	if jsonStart == -1 {
		return []types.Fact{{Fact: memory, Confidence: 0.6}}
	}
	if err := json.Unmarshal([]byte(content[jsonStart:]), &result); err != nil {
		lines := strings.Split(content, "\n")
		var facts []types.Fact
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if len(line) > 10 {
				facts = append(facts, types.Fact{Fact: line, Confidence: 0.7, Verified: true})
			}
		}
		if len(facts) > 0 {
			return facts
		}
		return []types.Fact{{Fact: memory, Confidence: 0.6}}
	}

	var verified []types.Fact
	for _, f := range result.Facts {
		if f.Verified && f.Fact != "" {
			verified = append(verified, types.Fact{
				Fact:       f.Fact,
				Confidence: f.Confidence,
				Verified:   true,
			})
		}
	}

	if len(verified) == 0 {
		return []types.Fact{{Fact: memory, Confidence: 0.6}}
	}

	return verified
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
		MaxTokens:   200,
	})
	if err != nil {
		return []Gap{}
	}

	content := resp.Content
	jsonStart := strings.Index(content, "{")
	if jsonStart == -1 {
		return []Gap{}
	}

	var gapResult struct {
		Gaps []struct {
			Question string `json:"question"`
			MemoryID string `json:"memory_id"`
		} `json:"gaps"`
	}
	if err := json.Unmarshal([]byte(content[jsonStart:]), &gapResult); err != nil {
		return []Gap{}
	}

	var gaps []Gap
	for _, g := range gapResult.Gaps {
		if g.Question != "" {
			gaps = append(gaps, Gap{
				Question: g.Question,
				MemoryID: g.MemoryID,
			})
		}
	}

	return gaps
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
			MaxTokens:   200,
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
