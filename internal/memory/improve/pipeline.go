// Package improve implements Hystersis's six-stage memory improvement pipeline,
// modeled after Cognee's improve(dataset, session_ids). The stages are:
//  1. FeedbackWeights   - update edge feedback_weight from session feedback
//  2. PersistSessions   - cognify session Q&A into the permanent graph
//  3. DistillSessions   - run the session distiller to produce durable lessons
//  4. MemifyEnrichment  - create triplet embeddings for graph edges
//  5. GlobalContext     - build retrieval-ready high-level summaries
//  6. SyncToCache       - copy new graph summaries back to the session cache
//
// The real stage implementations live in their own files
// (feedback_weights.go, persist_sessions.go, distill_sessions.go,
// memify.go, global_context.go, sync.go). This file holds the shared
// Pipeline/Stage/Input/Output types.
package improve

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Input describes what the pipeline should improve.
type Input struct {
	UserID             string
	DatasetID          string
	SessionIDs         []string
	BuildGlobalContext bool
	RunSyncToCache     bool
}

// Output summarizes the stages that ran and their per-stage status.
type Output struct {
	Stages     map[string]StageResult
	Started    time.Time
	DurationMs int64
}

// StageResult is one stage's outcome.
type StageResult struct {
	Name    string
	Started time.Time
	Ended   time.Time
	Error   error
	Items   int
}

// Stage is one improvement stage.
type Stage interface {
	Name() string
	Run(ctx context.Context, in Input) (StageResult, error)
}

// Pipeline executes the configured stages in order.
type Pipeline struct {
	stages []Stage
	mu     sync.Mutex
	ran    map[string]bool
}

// NewPipeline returns a Pipeline pre-configured with the default six
// stages as zero-value instances. Callers replace specific stages with
// real implementations via WithStage when graph/vector/session backends
// are available.
func NewPipeline() *Pipeline {
	return &Pipeline{
		ran: make(map[string]bool),
		stages: []Stage{
			FeedbackWeights{},
			PersistSessions{},
			DistillSessions{},
			MemifyEnrichment{},
			GlobalContextIndex{},
			SyncToCache{},
		},
	}
}

// WithStage replaces or appends a stage by name.
func (p *Pipeline) WithStage(s Stage) *Pipeline {
	p.mu.Lock()
	defer p.mu.Unlock()
	for i, existing := range p.stages {
		if existing.Name() == s.Name() {
			p.stages[i] = s
			p.ran[s.Name()] = true
			return p
		}
	}
	p.stages = append(p.stages, s)
	p.ran[s.Name()] = true
	return p
}

// Stages returns a copy of the current stage list.
func (p *Pipeline) Stages() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]string, len(p.stages))
	for i, s := range p.stages {
		out[i] = s.Name()
	}
	return out
}

// Run executes the configured stages in order and returns a per-stage
// summary. The pipeline is fail-open: a stage error is recorded and the
// pipeline continues.
func (p *Pipeline) Run(ctx context.Context, in Input) Output {
	start := time.Now().UTC()
	out := Output{
		Stages:  make(map[string]StageResult, len(p.stages)),
		Started: start,
	}
	var firstErr error
	for _, s := range p.stages {
		// Skip global-context if not requested.
		if s.Name() == "global_context_index" && !in.BuildGlobalContext {
			continue
		}
		if s.Name() == "sync_to_cache" && !in.RunSyncToCache {
			continue
		}
		res, err := s.Run(ctx, in)
		out.Stages[s.Name()] = res
		if err != nil && firstErr == nil {
			firstErr = fmt.Errorf("stage %s: %w", s.Name(), err)
		}
	}
	out.DurationMs = time.Since(start).Milliseconds()
	return out
}
