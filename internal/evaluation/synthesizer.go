package evaluation

import (
	"context"
	"fmt"
	"strings"

	"agent-memory/internal/llm"
)

type AnswerSynthesizer struct {
	llmClient llm.Provider
	config    BenchmarkConfig
	topK      int
}

func NewAnswerSynthesizer(llmClient llm.Provider, config BenchmarkConfig, topK int) *AnswerSynthesizer {
	if topK <= 0 {
		topK = 5
	}
	return &AnswerSynthesizer{
		llmClient: llmClient,
		config:    config,
		topK:      topK,
	}
}

func (as *AnswerSynthesizer) Synthesize(ctx context.Context, question string, memories []MemoryResult) (string, error) {
	k := as.topK
	if k > len(memories) {
		k = len(memories)
	}

	var sb strings.Builder
	for i, mem := range memories[:k] {
		sb.WriteString(fmt.Sprintf("[%d] %s\n", i+1, mem.Content))
	}

	prompt := fmt.Sprintf(`You are a memory retrieval assistant. Given a question and retrieved memories, synthesize the most accurate answer.

Question: %s

Retrieved Memories:
%s

Provide a concise and accurate answer based on the memories above. If the memories lack sufficient information, state what is known.`,
		question, sb.String())

	temperature := 0.1
	if as.config.Deterministic {
		temperature = 0.0
	}

	resp, err := as.llmClient.Complete(ctx, &llm.CompletionRequest{
		Model:       as.config.Model,
		Messages:    []llm.Message{{Role: "system", Content: prompt}},
		Temperature: temperature,
		MaxTokens:   as.config.MaxTokens,
	})
	if err != nil {
		return "", err
	}
	return resp.Content, nil
}
