package pipeline

import (
	"context"
	"fmt"
	"sync"
	"time"

	"agent-memory/internal/compression/extractor"
	"agent-memory/internal/compression/radix"
	"agent-memory/internal/memory/types"
)

type CompressionPipeline struct {
	jobQueue    chan CompressionJob
	workers     int
	extractor   *extractor.MemoryExtractor
	radix       *radix.MemoryCompressor
	stats       *PipelineStats
	wg          sync.WaitGroup
	ctx         context.Context
	cancel      context.CancelFunc
}

type CompressionJob struct {
	MemoryID     string
	Priority    int
	Content     string
	Done        chan Result
}

type Result struct {
	Compressed   string
	TokenReduction float64
	Error       error
}

type PipelineStats struct {
	TotalProcessed    int64
	TotalTokensSaved int64
	AvgLatencyMs   float64
	QueueDepth    int64
	mu          sync.Mutex
}

func NewCompressionPipeline(workers int, ext *extractor.MemoryExtractor) *CompressionPipeline {
	ctx, cancel := context.WithCancel(context.Background())
	return &CompressionPipeline{
		jobQueue:  make(chan CompressionJob, 1000),
		workers:  workers,
		extractor: ext,
		radix:    radix.NewMemoryCompressor(),
		stats: &PipelineStats{
			TotalProcessed:  0,
			TotalTokensSaved: 0,
			AvgLatencyMs:    0,
			QueueDepth:      0,
		},
		ctx:    ctx,
		cancel: cancel,
	}
}

func (p *CompressionPipeline) Start() {
	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}
}

func (p *CompressionPipeline) Stop() {
	p.cancel()
	p.wg.Wait()
	close(p.jobQueue)
}

func (p *CompressionPipeline) worker(id int) {
	defer p.wg.Done()

	for {
		select {
		case <-p.ctx.Done():
			return
		case job, ok := <-p.jobQueue:
			if !ok {
				return
			}
			p.processJob(job)
		}
	}
}

func (p *CompressionPipeline) processJob(job CompressionJob) {
	start := time.Now()

	var compressed string
	var tokenReduction float64
	var extractionErr error

	useExtractor := p.extractor != nil

	if useExtractor {
		result, err := p.extractor.Extract(p.ctx, job.Content)
		extractionErr = err

		if err == nil && result != nil && len(result.Facts) > 0 {
			for _, fact := range result.Facts {
				if len(compressed) > 0 {
					compressed += "; "
				}
				compressed += fact.Fact
			}
			tokenReduction = result.TokenReduction
		} else {
			useExtractor = false
		}
	}

	if !useExtractor {
		compressed = p.radix.Compress(job.Content)
		stats := p.radix.GetStats(job.Content)
		tokenReduction = stats.Reduction
		if tokenReduction == 0 {
			compressed = job.Content
			tokenReduction = 0.0
		}
	}

	latencyMs := float64(time.Since(start).Milliseconds())

	job.Done <- Result{
		Compressed:     compressed,
		TokenReduction: tokenReduction,
		Error:       extractionErr,
	}

	p.stats.mu.Lock()
	p.stats.TotalProcessed++
	p.stats.TotalTokensSaved += int64(float64(len(job.Content)) * tokenReduction)

	oldAvg := p.stats.AvgLatencyMs
	count := float64(p.stats.TotalProcessed)
	p.stats.AvgLatencyMs = ((oldAvg * (count - 1)) + latencyMs) / count

	p.stats.mu.Unlock()
}

func (p *CompressionPipeline) CompressAsync(job CompressionJob) {
	if job.Done == nil {
		job.Done = make(chan Result, 1)
	}

	select {
	case p.jobQueue <- job:
		p.stats.mu.Lock()
		p.stats.QueueDepth = int64(len(p.jobQueue))
		p.stats.mu.Unlock()
	default:
		job.Done <- Result{
			Error: fmt.Errorf("compression queue full"),
		}
	}
}

func (p *CompressionPipeline) GetStats() (int64, int64, float64) {
	p.stats.mu.Lock()
	defer p.stats.mu.Unlock()
	return p.stats.TotalProcessed, p.stats.TotalTokensSaved, p.stats.AvgLatencyMs
}

func (p *CompressionPipeline) GetQueueDepth() int64 {
	p.stats.mu.Lock()
	defer p.stats.mu.Unlock()
	return p.stats.QueueDepth
}

func (p *CompressionPipeline) LearnPatterns(memories []string) {
	p.radix.LearnFromMemories(memories)
}

func (p *CompressionPipeline) AddPattern(key, value string) {
	p.radix.AddPattern(key, value)
}

func (p *CompressionPipeline) GetCompressionStats(text string) radix.CompressionStats {
	return p.radix.GetStats(text)
}

type CompressionMode string

const (
	CompressionModeExtract     CompressionMode = "extract"
	CompressionModeBalanced   CompressionMode = "balanced"
	CompressionModeAggressive CompressionMode = "aggressive"
)

type TierPolicy string

const (
	TierPolicyAggressive   TierPolicy = "aggressive"
	TierPolicyBalanced     TierPolicy = "balanced"
	TierPolicyConservative TierPolicy = "conservative"
)

type CompressionConfig struct {
	Mode          CompressionMode
	TierPolicy     TierPolicy
	Enabled       bool
	AsyncEnabled  bool
	WorkerCount  int
}

func DefaultCompressionConfig() *CompressionConfig {
	return &CompressionConfig{
		Mode:         CompressionModeExtract,
		TierPolicy:   TierPolicyBalanced,
		Enabled:      true,
		AsyncEnabled: true,
		WorkerCount:  4,
	}
}

func (c *CompressionConfig) SetMode(mode CompressionMode) {
	c.Mode = mode
}

func (c *CompressionConfig) GetMode() CompressionMode {
	return c.Mode
}

func (c *CompressionConfig) SetTierPolicy(policy TierPolicy) {
	c.TierPolicy = policy
}

func (c *CompressionConfig) GetTierPolicy() TierPolicy {
	return c.TierPolicy
}

type MemoryWithTier struct {
	*types.Memory
	Tier    string
	TierKey string
}

func NewMemoryWithTier(mem *types.Memory, tier string) *MemoryWithTier {
	return &MemoryWithTier{
		Memory: mem,
		Tier:    tier,
		TierKey: fmt.Sprintf("%s:%s", tier, mem.ID),
	}
}