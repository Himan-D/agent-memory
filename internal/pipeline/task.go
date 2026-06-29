// Package pipeline provides a lightweight task composition framework modeled
// after Cogney's get_default_tasks + run_pipeline. Stages are typed units of
// work that take a PipelineContext and an input, and produce an output that
// the next stage consumes. The framework is intentionally small and free of
// dependencies on the rest of the codebase so it can be wired up
// incrementally without breaking existing call sites.
package pipeline

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// StageResult carries the outcome of one stage. Error is fail-open metadata;
// callers decide whether to abort the pipeline.
type StageResult struct {
	StageName string
	Started   time.Time
	Ended     time.Time
	Error     error
	Output    any
}

// PipelineContext carries provenance + runtime state through the pipeline.
// It mirrors Cognee's PipelineContext (user, dataset, pipeline_name,
// pipeline_run_id) but stays small and additive.
type PipelineContext struct {
	UserID         string
	DatasetID      string
	TenantID       string
	PipelineRunID  string
	PipelineName   string
	StageOutputs   map[string]any
	Metadata       map[string]string
	Started        time.Time
}

// NewPipelineContext returns an initialized context with sensible empty
// values for the map fields.
func NewPipelineContext(name string) *PipelineContext {
	return &PipelineContext{
		PipelineName: name,
		StageOutputs: make(map[string]any),
		Metadata:     make(map[string]string),
		Started:      time.Now().UTC(),
	}
}

// Set stores a stage output by stage name.
func (c *PipelineContext) Set(stage string, output any) {
	if c.StageOutputs == nil {
		c.StageOutputs = make(map[string]any)
	}
	c.StageOutputs[stage] = output
}

// Get returns a previously stored stage output and whether it was found.
func (c *PipelineContext) Get(stage string) (any, bool) {
	if c.StageOutputs == nil {
		return nil, false
	}
	v, ok := c.StageOutputs[stage]
	return v, ok
}

// Task is the unit of work in a pipeline. Input is whatever the previous
// stage produced (or the initial data for the first stage).
type Task interface {
	Name() string
	Run(ctx context.Context, pctx *PipelineContext, input any) (any, error)
}

// TaskFunc adapts a function to the Task interface for terse inline tasks.
type TaskFunc struct {
	N string
	F func(ctx context.Context, pctx *PipelineContext, input any) (any, error)
}

// Name returns the task name.
func (t *TaskFunc) Name() string { return t.N }

// Run invokes the underlying function.
func (t *TaskFunc) Run(ctx context.Context, pctx *PipelineContext, input any) (any, error) {
	return t.F(ctx, pctx, input)
}

// Pipeline executes a sequence of tasks. Each task receives the previous
// task's output as its input. Failures are recorded but do not abort
// subsequent stages unless FailFast is set.
type Pipeline struct {
	Name     string
	Tasks    []Task
	FailFast bool
	OnStage  func(pctx *PipelineContext, result StageResult)
}

// New constructs a Pipeline with the given tasks.
func New(name string, tasks ...Task) *Pipeline {
	return &Pipeline{Name: name, Tasks: tasks}
}

// Run executes every task in order. It returns the final output (from the
// last task) and a slice of per-stage results.
func (p *Pipeline) Run(ctx context.Context, pctx *PipelineContext, initial any) (any, []StageResult, error) {
	if pctx == nil {
		pctx = NewPipelineContext(p.Name)
	}
	if pctx.PipelineName == "" {
		pctx.PipelineName = p.Name
	}

	results := make([]StageResult, 0, len(p.Tasks))
	current := initial
	var firstErr error

	for _, t := range p.Tasks {
		start := time.Now().UTC()
		out, err := t.Run(ctx, pctx, current)
		end := time.Now().UTC()
		res := StageResult{
			StageName: t.Name(),
			Started:   start,
			Ended:     end,
			Error:     err,
			Output:    out,
		}
		results = append(results, res)
		if p.OnStage != nil {
			p.OnStage(pctx, res)
		}
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			if p.FailFast {
				return out, results, err
			}
			// Fail-open: keep the previous output so subsequent stages can
			// still produce useful work.
			continue
		}
		pctx.Set(t.Name(), out)
		current = out
	}
	return current, results, firstErr
}

// Concurrent runs tasks in parallel with a shared input. It is useful for
// stages that produce independent summaries (e.g. graph extraction +
// summarization in Cognee's extract_graph_and_summarize). All tasks share
// the same input; results are written to PipelineContext under their task
// names. Errors are aggregated.
func Concurrent(ctx context.Context, pctx *PipelineContext, limit int, tasks ...Task) error {
	if pctx == nil {
		pctx = NewPipelineContext("concurrent")
	}
	if limit <= 0 {
		limit = 4
	}
	sem := make(chan struct{}, limit)
	var (
		wg     sync.WaitGroup
		mu     sync.Mutex
		errs   []error
	)
	for _, t := range tasks {
		wg.Add(1)
		go func(t Task) {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				return
			}
			out, err := t.Run(ctx, pctx, nil)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				errs = append(errs, fmt.Errorf("%s: %w", t.Name(), err))
				return
			}
			pctx.Set(t.Name(), out)
		}(t)
	}
	wg.Wait()
	if len(errs) > 0 {
		return fmt.Errorf("concurrent stage: %d error(s): %v", len(errs), errs)
	}
	return nil
}
