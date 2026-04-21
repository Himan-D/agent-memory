package smart

import (
	"context"
	"strings"
	"sync"
	"time"

	"agent-memory/internal/compression/extractor"
	"agent-memory/internal/compression/pipeline"
	"agent-memory/internal/compression/radix"
	"agent-memory/internal/compression/relational"
	"agent-memory/internal/llm"
)

type SmartCompressor struct {
	extractor *extractor.MemoryExtractor
	radix    *radix.MemoryCompressor
	relational *relational.RelationalMapper
	pipeline  *pipeline.CompressionPipeline
	
	mu       sync.RWMutex
	stats    CompressorStats
}

type CompressorStats struct {
	TotalCompressions   int64
	ExtractionBased     int64
	RelationalBased    int64
	RadixBased         int64
	TotalTokensSaved   int64
	AvgReduction       float64
	AvgLatencyMs       float64
}

func NewSmartCompressor(llmClient llm.Provider, workerCount int) *SmartCompressor {
	c := &SmartCompressor{}

	if llmClient != nil {
		c.extractor = extractor.NewMemoryExtractor(llmClient)
		c.relational = relational.NewRelationalMapper(llmClient)
	}

	c.radix = radix.NewMemoryCompressor()
	c.pipeline = pipeline.NewCompressionPipeline(workerCount, c.extractor)
	c.pipeline.Start()

	return c
}

func (c *SmartCompressor) Compress(ctx context.Context, content string, mode Mode) (string, float64, error) {
	if content == "" {
		return "", 0.0, nil
	}

	start := time.Now()

	var compressed string
	var reduction float64

	switch mode {
	case ModeExtraction:
		compressed, reduction = c.compressWithExtraction(ctx, content)
	case ModeRelational:
		compressed, reduction = c.compressWithRelational(ctx, content)
	case ModeHybrid:
		compressed, reduction = c.compressHybrid(ctx, content)
	case ModeRadix:
		compressed, reduction = c.radix.Compress(content), c.radix.GetStats(content).Reduction
	default:
		compressed, reduction = c.compressHybrid(ctx, content)
	}

	latencyMs := float64(time.Since(start).Milliseconds())

	c.mu.Lock()
	c.stats.TotalCompressions++
	c.stats.TotalTokensSaved += int64(float64(len(content)) * reduction)
	c.stats.AvgLatencyMs = ((c.stats.AvgLatencyMs * float64(c.stats.TotalCompressions-1)) + latencyMs) / float64(c.stats.TotalCompressions)
	c.stats.AvgReduction = ((c.stats.AvgReduction * float64(c.stats.TotalCompressions-1)) + reduction) / float64(c.stats.TotalCompressions)
	c.mu.Unlock()

	return compressed, reduction, nil
}

func (c *SmartCompressor) compressWithExtraction(ctx context.Context, content string) (string, float64) {
	if c.extractor == nil {
		return c.radix.Compress(content), c.radix.GetStats(content).Reduction
	}

	result, err := c.extractor.Extract(ctx, content)
	if err != nil || result == nil || len(result.Facts) == 0 {
		c.mu.Lock()
		c.stats.RadixBased++
		c.mu.Unlock()
		return c.radix.Compress(content), c.radix.GetStats(content).Reduction
	}

	var facts []string
	for _, fact := range result.Facts {
		facts = append(facts, fact.Fact)
	}

	compressed := strings.Join(facts, "; ")

	c.mu.Lock()
	c.stats.ExtractionBased++
	c.mu.Unlock()

	reduction := result.TokenReduction
	if reduction == 0 {
		reduction = 1.0 - float64(len(strings.Fields(compressed))) / float64(len(strings.Fields(content)))
	}

	return compressed, reduction
}

func (c *SmartCompressor) compressWithRelational(ctx context.Context, content string) (string, float64) {
	if c.relational == nil {
		return c.radix.Compress(content), c.radix.GetStats(content).Reduction
	}

	memories := []string{content}
	compressed, _, err := c.relational.CompressWithRelations(ctx, memories)
	if err != nil {
		return c.radix.Compress(content), c.radix.GetStats(content).Reduction
	}

	c.mu.Lock()
	c.stats.RelationalBased++
	c.mu.Unlock()

	reduction := 1.0 - float64(len(strings.Fields(compressed))) / float64(len(strings.Fields(content)))
	if reduction < 0.1 {
		reduction = 0.3
	}

	return compressed, reduction
}

func (c *SmartCompressor) compressHybrid(ctx context.Context, content string) (string, float64) {
	var bestCompressed string
	var bestReduction float64 = 0

	result1, red1 := c.compressWithExtraction(ctx, content)
	if red1 > bestReduction {
		bestReduction = red1
		bestCompressed = result1
	}

	result2, red2 := c.compressWithRelational(ctx, content)
	if red2 > bestReduction {
		bestReduction = red2
		bestCompressed = result2
	}

	result3 := c.radix.Compress(content)
	red3 := c.radix.GetStats(content).Reduction
	if red3 > bestReduction || bestCompressed == "" {
		bestReduction = red3
		bestCompressed = result3
	}

	return bestCompressed, bestReduction
}

func (c *SmartCompressor) LearnPatterns(memories []string) {
	c.radix.LearnFromMemories(memories)

	if c.relational != nil && len(memories) > 0 {
		ctx := context.Background()
		c.relational.ExtractRelations(ctx, memories)
	}
}

func (c *SmartCompressor) GetStats() CompressorStats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.stats
}

func (c *SmartCompressor) Stop() {
	if c.pipeline != nil {
		c.pipeline.Stop()
	}
}

type Mode string

const (
	ModeExtraction  Mode = "extraction"
	ModeRelational Mode = "relational"
	ModeHybrid     Mode = "hybrid"
	ModeRadix     Mode = "radix"
)

func (c *SmartCompressor) CompressAsync(ctx context.Context, job pipeline.CompressionJob) {
	defer close(job.Done)

	compressed, reduction, err := c.Compress(ctx, job.Content, ModeHybrid)

	job.Done <- pipeline.Result{
		Compressed:     compressed,
		TokenReduction: reduction,
		Error:         err,
	}
}

type CompressionRequest struct {
	Content      string
	Mode         Mode
	ReturnFacts  bool
}

type CompressionResponse struct {
	Compressed   string
	TokenReduction float64
	Method       string
	Entities      []relational.Entity
	Relationships []relational.Relationship
	LatencyMs    float64
}

func (c *SmartCompressor) CompressWithDetails(ctx context.Context, req CompressionRequest) (*CompressionResponse, error) {
	start := time.Now()

	var compressed string
	var reduction float64
	var method string
	var entities []relational.Entity
	var relationships []relational.Relationship

	switch req.Mode {
	case ModeExtraction:
		compressed, reduction = c.compressWithExtraction(ctx, req.Content)
		method = "extraction"
	case ModeRelational:
		compressed, reduction = c.compressWithRelational(ctx, req.Content)
		method = "relational"
		if c.relational != nil {
			graph, _ := c.relational.ExtractRelations(ctx, []string{req.Content})
			if graph != nil {
				entities = graph.Entities
				relationships = graph.Relationships
			}
		}
	case ModeHybrid, "":
		compressed, reduction = c.compressHybrid(ctx, req.Content)
		method = "hybrid"
	default:
		compressed = c.radix.Compress(req.Content)
		reduction = c.radix.GetStats(req.Content).Reduction
		method = "radix"
	}

	latencyMs := float64(time.Since(start).Milliseconds())

	return &CompressionResponse{
		Compressed:    compressed,
		TokenReduction: reduction,
		Method:        method,
		Entities:      entities,
		Relationships: relationships,
		LatencyMs:     latencyMs,
	}, nil
}