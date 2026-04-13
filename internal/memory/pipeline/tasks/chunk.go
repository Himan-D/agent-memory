package tasks

import (
	"context"
	"fmt"
	"strings"

	"agent-memory/internal/memory/chunking"
	"agent-memory/internal/memory/datapoint"
)

type ChunkTask struct {
	chunkSize    int
	chunkOverlap int
	separator    string
	chunker      chunking.Chunker
}

type ChunkTaskConfig struct {
	ChunkSize    int
	ChunkOverlap int
	Separator    string
	Strategy     string
}

func NewChunkTask(cfg ChunkTaskConfig) *ChunkTask {
	var ch chunking.Chunker

	switch strings.ToLower(cfg.Strategy) {
	case "recursive":
		ch = chunking.NewRecursiveChunker(cfg.ChunkSize, cfg.ChunkOverlap, cfg.Separator)
	case "fixed":
		ch = chunking.NewFixedSizeChunker(cfg.ChunkSize, cfg.ChunkOverlap)
	case "semantic":
		ch = chunking.NewSemanticChunker(nil)
	default:
		ch = chunking.NewRecursiveChunker(cfg.ChunkSize, cfg.ChunkOverlap, cfg.Separator)
	}

	return &ChunkTask{
		chunkSize:    cfg.ChunkSize,
		chunkOverlap: cfg.ChunkOverlap,
		separator:    cfg.Separator,
		chunker:      ch,
	}
}

func (t *ChunkTask) Name() string {
	return "chunk"
}

func (t *ChunkTask) Execute(ctx context.Context, input *datapoint.DataPoint) (*datapoint.DataPoint, error) {
	if input.Type != datapoint.DataPointTypeDocument {
		return input, nil
	}

	chunks := t.chunker.Chunk(input.Content)

	result := datapoint.New(strings.Join(chunks, "\n\n"), datapoint.DataPointTypeChunk)
	result.Metadata["chunk_count"] = len(chunks)
	result.Metadata["chunks"] = chunks
	result.IsPartOf = &datapoint.PartOfInfo{
		ID:   input.ID,
		Name: input.Source.Name,
		Type: string(input.Type),
	}

	return result, nil
}

func (t *ChunkTask) Validate(input *datapoint.DataPoint) error {
	if input == nil {
		return fmt.Errorf("input is nil")
	}
	if input.Content == "" {
		return fmt.Errorf("input content is empty")
	}
	return nil
}

func (t *ChunkTask) ChunkContent(content string) []string {
	return t.chunker.Chunk(content)
}
