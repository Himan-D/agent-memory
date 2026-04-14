package tasks

import (
	"context"
	"fmt"

	"agent-memory/internal/embedding"
	"agent-memory/internal/llm"
	"agent-memory/internal/memory/datapoint"
)

type EmbedTask struct {
	embeddingClient *embedding.OpenAIEmbedding
	model           string
}

type EmbedTaskConfig struct {
	Model string
}

func NewEmbedTask(embedClient *embedding.OpenAIEmbedding, cfg EmbedTaskConfig) *EmbedTask {
	model := cfg.Model
	if model == "" {
		model = "text-embedding-3-small"
	}
	return &EmbedTask{
		embeddingClient: embedClient,
		model:           model,
	}
}

func (t *EmbedTask) Name() string {
	return "embed"
}

func (t *EmbedTask) Execute(ctx context.Context, input *datapoint.DataPoint) (*datapoint.DataPoint, error) {
	if t.embeddingClient == nil {
		input.Metadata["embedding"] = []float32{0}
		return input, nil
	}

	text := input.Content
	if input.Summary != "" {
		text = input.Summary
	}

	emb, err := t.embeddingClient.GenerateEmbeddingWithContext(ctx, text)
	if err != nil {
		return nil, fmt.Errorf("embedding failed: %w", err)
	}

	output := input.Clone()
	output.SetEmbedding(emb)
	output.Metadata["embedding_model"] = t.model
	output.Metadata["embedding_dimensions"] = len(emb)

	return output, nil
}

func (t *EmbedTask) Validate(input *datapoint.DataPoint) error {
	if input == nil {
		return fmt.Errorf("input is nil")
	}
	if input.Content == "" {
		return fmt.Errorf("input content is empty")
	}
	return nil
}

type BatchEmbedTask struct {
	embeddingClient *embedding.OpenAIEmbedding
	model           string
	batchSize       int
}

func NewBatchEmbedTask(embedClient *embedding.OpenAIEmbedding, cfg EmbedTaskConfig) *BatchEmbedTask {
	model := cfg.Model
	if model == "" {
		model = "text-embedding-3-small"
	}
	return &BatchEmbedTask{
		embeddingClient: embedClient,
		model:           model,
		batchSize:       100,
	}
}

func (t *BatchEmbedTask) Name() string {
	return "batch_embed"
}

func (t *BatchEmbedTask) Execute(ctx context.Context, inputs []*datapoint.DataPoint) ([]*datapoint.DataPoint, error) {
	if len(inputs) == 0 {
		return inputs, nil
	}

	texts := make([]string, len(inputs))
	for i, input := range inputs {
		texts[i] = input.Content
		if input.Summary != "" {
			texts[i] = input.Summary
		}
	}

	var allEmbeddings [][]float32
	for i := 0; i < len(texts); i += t.batchSize {
		end := i + t.batchSize
		if end > len(texts) {
			end = len(texts)
		}

		batch := texts[i:end]
		embeddings, err := t.embeddingClient.GenerateBatchEmbeddingsWithContext(ctx, batch)
		if err != nil {
			return nil, fmt.Errorf("batch embedding failed at %d: %w", i, err)
		}
		allEmbeddings = append(allEmbeddings, embeddings...)
	}

	outputs := make([]*datapoint.DataPoint, len(inputs))
	for i, input := range inputs {
		output := input.Clone()
		if i < len(allEmbeddings) {
			output.SetEmbedding(allEmbeddings[i])
			output.Metadata["embedding_model"] = t.model
		}
		outputs[i] = output
	}

	return outputs, nil
}

type SummarizeTask struct {
	llmClient llm.Provider
	model     string
}

type SummarizeTaskConfig struct {
	Model string
}

func NewSummarizeTask(llmClient llm.Provider, cfg SummarizeTaskConfig) *SummarizeTask {
	model := cfg.Model
	if model == "" {
		model = "gpt-4o-mini"
	}
	return &SummarizeTask{
		llmClient: llmClient,
		model:     model,
	}
}

func (t *SummarizeTask) Name() string {
	return "summarize"
}

func (t *SummarizeTask) Execute(ctx context.Context, input *datapoint.DataPoint) (*datapoint.DataPoint, error) {
	if t.llmClient == nil {
		summary := input.Content
		if len(summary) > 200 {
			summary = summary[:200] + "..."
		}
		output := input.Clone()
		output.SetSummary(summary)
		return output, nil
	}

	prompt := fmt.Sprintf("Summarize the following text concisely in 2-3 sentences:\n\n%s", input.Content)

	resp, err := t.llmClient.Complete(ctx, &llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: "user", Content: prompt},
		},
		Model:       t.model,
		MaxTokens:   500,
		Temperature: 0.3,
	})
	if err != nil {
		return nil, fmt.Errorf("summarization failed: %w", err)
	}

	output := input.Clone()
	output.SetSummary(resp.Content)

	return output, nil
}

func (t *SummarizeTask) Validate(input *datapoint.DataPoint) error {
	if input == nil {
		return fmt.Errorf("input is nil")
	}
	if input.Content == "" {
		return fmt.Errorf("input content is empty")
	}
	return nil
}
