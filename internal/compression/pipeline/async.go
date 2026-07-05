package pipeline

import (
	"context"
	"fmt"
	"sync"
	"time"

	"agent-memory/internal/compression/evaluator"
	"agent-memory/internal/compression/extractor"
	"agent-memory/internal/compression/llm"
	"agent-memory/internal/compression/radix"
	"agent-memory/internal/memory/types"
)

type CompressionPipeline struct {
	jobQueue        chan CompressionJob
	workers         int
	extractor       *extractor.MemoryExtractor
	llmRouter       *llm.LLMRouter
	radix           *radix.MemoryCompressor
	fidelityTracker *evaluator.FidelityTracker
	stats           *PipelineStats
	wg              sync.WaitGroup
	ctx             context.Context
	cancel          context.CancelFunc
}

type CompressionJob struct {
	MemoryID string
	Priority int
	Content  string
	Done     chan Result
}

type Result struct {
	Compressed      string
	TokenReduction  float64
	CompressionMode string // "extraction", "extractor", "radix"
	Error           error
}

// ModeRatioStats tracks byte-level compression ratio for a single mode.
type ModeRatioStats struct {
	Count           int64   `json:"count"`
	AvgRatio        float64 `json:"avg_ratio"`
	TotalBytesSaved int64   `json:"total_bytes_saved"`
}

type PipelineStats struct {
	TotalProcessed   int64
	TotalTokensSaved int64
	AvgLatencyMs     float64
	QueueDepth       int64
	ModeRatios       map[string]*ModeRatioStats
	mu               sync.Mutex
}

// GetStats returns a copy of pipeline stats
func (p *PipelineStats) GetStats() (int64, int64, float64, int64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.TotalProcessed, p.TotalTokensSaved, p.AvgLatencyMs, p.QueueDepth
}

func NewCompressionPipeline(workers int, ext *extractor.MemoryExtractor, router *llm.LLMRouter) *CompressionPipeline {
	ctx, cancel := context.WithCancel(context.Background())
	return &CompressionPipeline{
		jobQueue:  make(chan CompressionJob, 1000),
		workers:   workers,
		extractor: ext,
		llmRouter: router,
		radix:     radix.NewMemoryCompressor(),
		stats: &PipelineStats{
			TotalProcessed:   0,
			TotalTokensSaved: 0,
			AvgLatencyMs:     0,
			QueueDepth:       0,
			ModeRatios:       make(map[string]*ModeRatioStats),
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
}

func (p *CompressionPipeline) SetFidelityTracker(tracker *evaluator.FidelityTracker) {
	p.fidelityTracker = tracker
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

	// Update queue depth now that we've dequeued the job.
	p.stats.mu.Lock()
	p.stats.QueueDepth = int64(len(p.jobQueue))
	p.stats.mu.Unlock()

	var compressed string
	var tokenReduction float64
	var extractionErr error
	var mode string

	if p.llmRouter != nil {
		result, err := p.llmRouter.Route(p.ctx, job.Content)
		extractionErr = err

		if err == nil && result != nil && len(result.Facts) > 0 {
			facts := result.Facts
			if len(result.VerifiedFacts) > 0 {
				facts = result.VerifiedFacts
			}
			for _, fact := range facts {
				if len(compressed) > 0 {
					compressed += "; "
				}
				compressed += fact.Fact
			}
			tokenReduction = result.TokenReduction
			mode = "extraction"
		} else {
			extractionErr = err
		}
	}

	if compressed == "" && p.extractor != nil {
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
			mode = "extractor"
		} else {
			compressed = ""
			extractionErr = nil
		}
	}

	if compressed == "" {
		compressed = p.radix.Compress(job.Content)
		stats := p.radix.GetStats(job.Content)
		tokenReduction = stats.Reduction
		mode = "radix"
		if tokenReduction == 0 {
			compressed = job.Content
			tokenReduction = 0.0
		}
	}

	// Compute byte-level compression ratio (canonical metric)
	byteRatio := 0.0
	if len(job.Content) > 0 {
		byteRatio = 1.0 - float64(len(compressed))/float64(len(job.Content))
		if byteRatio < 0 {
			byteRatio = 0
		}
	}

	latencyMs := float64(time.Since(start).Milliseconds())

	result := Result{
		Compressed:      compressed,
		TokenReduction:  tokenReduction,
		CompressionMode: mode,
		Error:           extractionErr,
	}

	// Guard against goroutine leak: if the caller abandoned the Done channel,
	// we fall through on pipeline shutdown rather than blocking forever.
	select {
	case job.Done <- result:
	case <-p.ctx.Done():
	}

	// Only count successfully compressed jobs in stats.
	if extractionErr == nil {
		p.stats.mu.Lock()
		p.stats.TotalProcessed++
		p.stats.TotalTokensSaved += int64(float64(len(job.Content)) * tokenReduction)

		oldAvg := p.stats.AvgLatencyMs
		count := float64(p.stats.TotalProcessed)
		p.stats.AvgLatencyMs = ((oldAvg * (count - 1)) + latencyMs) / count

		// Per-mode byte-level ratio tracking
		if mode != "" {
			ms, ok := p.stats.ModeRatios[mode]
			if !ok {
				ms = &ModeRatioStats{}
				p.stats.ModeRatios[mode] = ms
			}
			bytesSaved := int64(float64(len(job.Content)) * byteRatio)
			ms.TotalBytesSaved += bytesSaved
			// Running average ratio
			oldRatio := ms.AvgRatio
			modeCount := float64(ms.Count + 1)
			ms.AvgRatio = ((oldRatio * float64(ms.Count)) + byteRatio) / modeCount
			ms.Count++
		}
		p.stats.mu.Unlock()

		// Sample-based fidelity evaluation
		if p.fidelityTracker != nil {
			p.fidelityTracker.MaybeEvaluate(p.ctx, job.Content, compressed)
		}
	}
}

func (p *CompressionPipeline) CompressAsync(job CompressionJob) {
	if job.Done == nil {
		job.Done = make(chan Result, 1)
	}

	select {
	case p.jobQueue <- job:
	case <-p.ctx.Done():
		job.Done <- Result{
			Error: fmt.Errorf("compression pipeline stopped"),
		}
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

// GetPipelineStats returns pipeline statistics
func (p *CompressionPipeline) GetPipelineStats() (int64, int64, float64, int64) {
	return p.stats.GetStats()
}

// GetModeRatios returns a copy of per-mode byte-level compression ratio stats.
func (p *CompressionPipeline) GetModeRatios() map[string]ModeRatioStats {
	p.stats.mu.Lock()
	defer p.stats.mu.Unlock()
	out := make(map[string]ModeRatioStats, len(p.stats.ModeRatios))
	for k, v := range p.stats.ModeRatios {
		out[k] = *v
	}
	return out
}

// FidelityStats holds fidelity evaluation summary for API responses.
type FidelityStats struct {
	Recall      float64 `json:"recall"`
	Precision   float64 `json:"precision"`
	F1          float64 `json:"f1"`
	SampleCount int     `json:"sample_count"`
	SampleRate  float64 `json:"sample_rate"`
	TotalEvals  int64   `json:"total_evals"`
	TotalCalls  int64   `json:"total_calls"`
}

// GetFidelityStats returns fidelity evaluation statistics, or nil if no tracker is configured.
func (p *CompressionPipeline) GetFidelityStats() *FidelityStats {
	if p.fidelityTracker == nil {
		return nil
	}
	avg := p.fidelityTracker.AverageFidelity()
	evals, calls, sampleCount := p.fidelityTracker.Stats()
	if sampleCount == 0 {
		return nil
	}
	return &FidelityStats{
		Recall:      avg.Recall,
		Precision:   avg.Precision,
		F1:          avg.F1,
		SampleCount: sampleCount,
		SampleRate:  p.fidelityTracker.SampleRate(),
		TotalEvals:  evals,
		TotalCalls:  calls,
	}
}

// RecordPipelineStats allows external recording of pipeline stats to metrics collector
func (p *CompressionPipeline) RecordPipelineStats(processed int64, tokensSaved int64, latencyMs float64) {
	p.stats.mu.Lock()
	defer p.stats.mu.Unlock()
	p.stats.TotalProcessed += processed
	p.stats.TotalTokensSaved += tokensSaved
	if processed > 0 {
		oldAvg := p.stats.AvgLatencyMs
		count := float64(p.stats.TotalProcessed)
		p.stats.AvgLatencyMs = ((oldAvg * (count - 1)) + latencyMs) / count
	}
}

type CompressionMode string

const (
	CompressionModeExtract    CompressionMode = "extract"
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
	Mode         CompressionMode
	TierPolicy   TierPolicy
	Enabled      bool
	AsyncEnabled bool
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
		Memory:  mem,
		Tier:    tier,
		TierKey: fmt.Sprintf("%s:%s", tier, mem.ID),
	}
}
