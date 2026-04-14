package cognify

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"agent-memory/internal/embedding"
	"agent-memory/internal/llm"
	"agent-memory/internal/memory/chunking"
	"agent-memory/internal/memory/datapoint"
	"agent-memory/internal/memory/pipeline"
	"agent-memory/internal/memory/pipeline/tasks"
)

type Cognify struct {
	llm         llm.Provider
	embedding   *embedding.OpenAIEmbedding
	graphStore  GraphStore
	vectorStore VectorStore
	config      Config
	pipeline    *pipeline.Pipeline
}

type Config struct {
	ChunkSize      int
	ChunkOverlap   int
	ChunkSeparator string
	ChunkStrategy  string
	EntityTypes    []string
	Model          string
	EmbedModel     string
	UseCache       bool
	MaxConcurrency int
}

type GraphStore interface {
	CreateNode(label string, properties map[string]interface{}) error
	CreateRelationship(fromID, toID, relType string, properties map[string]interface{}) error
	FindNodes(label string, properties map[string]interface{}) ([]map[string]interface{}, error)
	FindEdges(fromID, toID string) ([]map[string]interface{}, error)
}

type VectorStore interface {
	Upsert(collection string, vectors []VectorRecord) error
	Search(collection string, query []float32, limit int) ([]SearchResult, error)
	Delete(collection string, ids []string) error
}

type VectorRecord struct {
	ID      string
	Vector  []float32
	Payload map[string]interface{}
}

type SearchResult struct {
	ID      string
	Score   float64
	Payload map[string]interface{}
}

type CognifyResult struct {
	DocumentID       string
	ChunksCreated    int
	EntitiesCreated  int
	RelationsCreated int
	Summary          string
	Errors           []error
}

func New(llmClient llm.Provider, embedClient *embedding.OpenAIEmbedding, graph GraphStore, vector VectorStore, cfg Config) *Cognify {
	c := &Cognify{
		llm:         llmClient,
		embedding:   embedClient,
		graphStore:  graph,
		vectorStore: vector,
		config:      cfg,
	}

	c.initPipeline()
	return c
}

func (c *Cognify) initPipeline() {
	c.pipeline = pipeline.NewPipeline("cognify").
		AddTask(tasks.NewChunkTask(tasks.ChunkTaskConfig{
			ChunkSize:    c.config.ChunkSize,
			ChunkOverlap: c.config.ChunkOverlap,
			Separator:    c.config.ChunkSeparator,
			Strategy:     c.config.ChunkStrategy,
		})).
		AddTask(tasks.NewSummarizeTask(c.llm, tasks.SummarizeTaskConfig{
			Model: c.config.Model,
		})).
		AddTask(tasks.NewEmbedTask(c.embedding, tasks.EmbedTaskConfig{
			Model: c.config.EmbedModel,
		}))
}

type CognifyInput struct {
	Text       string
	Source     string
	SourceType string
	Metadata   map[string]interface{}
}

func (c *Cognify) Cognify(ctx context.Context, input *CognifyInput) (*CognifyResult, error) {
	if input.Text == "" {
		return nil, fmt.Errorf("input text is empty")
	}

	source := input.Source
	if source == "" {
		source = "unknown"
	}
	sourceType := input.SourceType
	if sourceType == "" {
		sourceType = "text"
	}

	doc := datapoint.New(input.Text, datapoint.DataPointTypeDocument)
	doc.SetSource(sourceType, source, "")
	for k, v := range input.Metadata {
		doc.Metadata[k] = v
	}

	result := &CognifyResult{
		DocumentID: doc.ID,
	}

	chunks := c.chunkDocument(input.Text)

	var mu sync.Mutex
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, c.config.MaxConcurrency)
	if c.config.MaxConcurrency == 0 {
		c.config.MaxConcurrency = 5
	}

	for i, chunkText := range chunks {
		wg.Add(1)
		semaphore <- struct{}{}

		go func(idx int, text string) {
			defer wg.Done()
			defer func() { <-semaphore }()

			chunk := datapoint.NewChunk(text, doc, idx)
			chunk.SetSource(sourceType, source, "")

			extracted, err := c.extractEntities(ctx, text, source)
			if err == nil && extracted != nil {
				mu.Lock()
				result.EntitiesCreated += len(extracted.Entities)
				result.RelationsCreated += len(extracted.Relations)
				mu.Unlock()

				for _, entity := range extracted.Entities {
					c.storeEntity(ctx, entity)
				}
				for _, rel := range extracted.Relations {
					c.storeRelation(ctx, rel)
				}
			}

			embedding, err := c.generateEmbedding(ctx, text)
			if err == nil && embedding != nil {
				c.storeVector(chunk.ID, embedding, chunkText, source)
			}

			mu.Lock()
			result.ChunksCreated++
			mu.Unlock()
		}(i, chunkText)
	}

	wg.Wait()

	summary, err := c.generateSummary(ctx, input.Text)
	if err == nil {
		result.Summary = summary
	}

	return result, nil
}

func (c *Cognify) chunkDocument(text string) []string {
	chunker := chunking.NewRecursiveChunker(c.config.ChunkSize, c.config.ChunkOverlap, c.config.ChunkSeparator)
	return chunker.Chunk(text)
}

func (c *Cognify) extractEntities(ctx context.Context, text, source string) (*datapoint.ExtractedData, error) {
	if c.llm == nil {
		return datapoint.NewExtractedData(), nil
	}

	task := tasks.NewExtractAllTask(c.llm, tasks.ExtractTaskConfig{
		EntityTypes: c.config.EntityTypes,
	})

	chunk := datapoint.New(text, datapoint.DataPointTypeChunk)
	chunk.SetSource("cognify", source, "")

	return task.Execute(ctx, chunk)
}

func (c *Cognify) generateEmbedding(ctx context.Context, text string) ([]float32, error) {
	if c.embedding == nil {
		return nil, fmt.Errorf("embedding client not configured")
	}
	return c.embedding.GenerateEmbeddingWithContext(ctx, text)
}

func (c *Cognify) generateSummary(ctx context.Context, text string) (string, error) {
	if c.llm == nil {
		if len(text) > 200 {
			return text[:200] + "...", nil
		}
		return text, nil
	}

	prompt := fmt.Sprintf("Summarize the following text concisely in 2-3 sentences:\n\n%s", text)

	resp, err := c.llm.Complete(ctx, &llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: "user", Content: prompt},
		},
		Model:       c.config.Model,
		MaxTokens:   500,
		Temperature: 0.3,
	})
	if err != nil {
		return "", err
	}

	return resp.Content, nil
}

func (c *Cognify) storeEntity(ctx context.Context, entity *datapoint.DataPoint) {
	if c.graphStore == nil {
		return
	}

	properties := map[string]interface{}{
		"id":     entity.ID,
		"name":   entity.Content,
		"type":   entity.Properties["entity_type"],
		"source": entity.Source.Name,
	}
	for k, v := range entity.Properties {
		properties[k] = v
	}

	c.graphStore.CreateNode("Entity", properties)
}

func (c *Cognify) storeRelation(ctx context.Context, rel *datapoint.DataPoint) {
	if c.graphStore == nil {
		return
	}

	sourceID := rel.Properties["source_id"]
	targetID := rel.Properties["target_id"]
	relType := rel.Properties["relation_type"]

	properties := map[string]interface{}{
		"id":   rel.ID,
		"type": relType,
	}
	for k, v := range rel.Properties {
		if k != "source_id" && k != "target_id" && k != "relation_type" {
			properties[k] = v
		}
	}

	c.graphStore.CreateRelationship(sourceID, targetID, relType, properties)
}

func (c *Cognify) storeVector(id string, vector []float32, content, source string) {
	if c.vectorStore == nil {
		return
	}

	record := VectorRecord{
		ID:     id,
		Vector: vector,
		Payload: map[string]interface{}{
			"content": content,
			"source":  source,
		},
	}

	c.vectorStore.Upsert("chunks", []VectorRecord{record})
}

func (c *Cognify) CognifyBatch(ctx context.Context, inputs []*CognifyInput) ([]*CognifyResult, error) {
	results := make([]*CognifyResult, len(inputs))
	var wg sync.WaitGroup

	for i, input := range inputs {
		wg.Add(1)
		go func(idx int, inp *CognifyInput) {
			defer wg.Done()
			result, err := c.Cognify(ctx, inp)
			if err != nil {
				results[idx] = &CognifyResult{
					DocumentID: inp.Source,
					Errors:     []error{err},
				}
			} else {
				results[idx] = result
			}
		}(i, input)
	}

	wg.Wait()
	return results, nil
}

type PipelineRunner struct {
	llm       llm.Provider
	embedding *embedding.OpenAIEmbedding
	pipelines map[string]*pipeline.Pipeline
}

func NewPipelineRunner(llmClient llm.Provider, embedClient *embedding.OpenAIEmbedding) *PipelineRunner {
	return &PipelineRunner{
		llm:       llmClient,
		embedding: embedClient,
		pipelines: make(map[string]*pipeline.Pipeline),
	}
}

func (r *PipelineRunner) Register(name string, p *pipeline.Pipeline) {
	r.pipelines[name] = p
}

func (r *PipelineRunner) Run(ctx context.Context, name string, input *datapoint.DataPoint) (*datapoint.DataPoint, error) {
	p, ok := r.pipelines[name]
	if !ok {
		return nil, fmt.Errorf("pipeline not found: %s", name)
	}
	return p.Execute(ctx, input)
}

func BuildDefaultPipeline(llmClient llm.Provider, embedClient *embedding.OpenAIEmbedding) *pipeline.Pipeline {
	cfg := Config{
		ChunkSize:      1000,
		ChunkOverlap:   200,
		ChunkStrategy:  "recursive",
		EntityTypes:    []string{"PERSON", "ORGANIZATION", "LOCATION", "CONCEPT", "EVENT", "OBJECT"},
		Model:          "gpt-4o-mini",
		EmbedModel:     "text-embedding-3-small",
		MaxConcurrency: 5,
	}

	return pipeline.NewPipeline("cognify_default").
		AddTask(tasks.NewChunkTask(tasks.ChunkTaskConfig{
			ChunkSize:    cfg.ChunkSize,
			ChunkOverlap: cfg.ChunkOverlap,
			Strategy:     cfg.ChunkStrategy,
		})).
		AddTask(tasks.NewSummarizeTask(llmClient, tasks.SummarizeTaskConfig{
			Model: cfg.Model,
		})).
		AddTask(tasks.NewEmbedTask(embedClient, tasks.EmbedTaskConfig{
			Model: cfg.EmbedModel,
		}))
}

func ParseTextToDataPoints(text string, source string) []*datapoint.DataPoint {
	lines := strings.Split(text, "\n\n")
	points := make([]*datapoint.DataPoint, 0, len(lines))

	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		dp := datapoint.New(line, datapoint.DataPointTypeChunk)
		dp.SetSource("text", source, "")
		dp.Metadata["chunk_index"] = i
		points = append(points, dp)
	}

	return points
}
