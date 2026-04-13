package pipeline

import (
	"context"
	"fmt"
	"sync"

	"agent-memory/internal/memory/datapoint"
)

type Task interface {
	Name() string
	Execute(ctx context.Context, input *datapoint.DataPoint) (*datapoint.DataPoint, error)
	Validate(input *datapoint.DataPoint) error
}

type Pipeline struct {
	name   string
	tasks  []Task
	status PipelineStatus
	mu     sync.RWMutex
}

type PipelineStatus string

const (
	PipelineStatusIdle      PipelineStatus = "idle"
	PipelineStatusRunning   PipelineStatus = "running"
	PipelineStatusCompleted PipelineStatus = "completed"
	PipelineStatusFailed    PipelineStatus = "failed"
	PipelineStatusCancelled PipelineStatus = "cancelled"
)

type PipelineResult struct {
	Output    *datapoint.DataPoint
	Status    PipelineStatus
	TaskName  string
	Error     error
	TaskIndex int
}

func NewPipeline(name string) *Pipeline {
	return &Pipeline{
		name:   name,
		tasks:  make([]Task, 0),
		status: PipelineStatusIdle,
	}
}

func (p *Pipeline) AddTask(task Task) *Pipeline {
	p.tasks = append(p.tasks, task)
	return p
}

func (p *Pipeline) Execute(ctx context.Context, input *datapoint.DataPoint) (*datapoint.DataPoint, error) {
	p.mu.Lock()
	if p.status == PipelineStatusRunning {
		p.mu.Unlock()
		return nil, fmt.Errorf("pipeline already running")
	}
	p.status = PipelineStatusRunning
	p.mu.Unlock()

	defer func() {
		p.mu.Lock()
		p.status = PipelineStatusIdle
		p.mu.Unlock()
	}()

	current := input
	var err error

	for _, task := range p.tasks {
		select {
		case <-ctx.Done():
			p.mu.Lock()
			p.status = PipelineStatusCancelled
			p.mu.Unlock()
			return nil, ctx.Err()
		default:
		}

		if err = task.Validate(current); err != nil {
			p.mu.Lock()
			p.status = PipelineStatusFailed
			p.mu.Unlock()
			return nil, fmt.Errorf("task %s validation failed: %w", task.Name(), err)
		}

		current, err = task.Execute(ctx, current)
		if err != nil {
			p.mu.Lock()
			p.status = PipelineStatusFailed
			p.mu.Unlock()
			return nil, fmt.Errorf("task %s failed: %w", task.Name(), err)
		}

		if current == nil {
			p.mu.Lock()
			p.status = PipelineStatusFailed
			p.mu.Unlock()
			return nil, fmt.Errorf("task %s returned nil", task.Name())
		}
	}

	p.mu.Lock()
	p.status = PipelineStatusCompleted
	p.mu.Unlock()

	return current, nil
}

func (p *Pipeline) ExecuteBatch(ctx context.Context, inputs []*datapoint.DataPoint) ([]*PipelineResult, error) {
	p.mu.Lock()
	if p.status == PipelineStatusRunning {
		p.mu.Unlock()
		return nil, fmt.Errorf("pipeline already running")
	}
	p.status = PipelineStatusRunning
	p.mu.Unlock()

	defer func() {
		p.mu.Lock()
		p.status = PipelineStatusIdle
		p.mu.Unlock()
	}()

	results := make([]*PipelineResult, len(inputs))
	var wg sync.WaitGroup

	for i, input := range inputs {
		wg.Add(1)
		go func(idx int, inp *datapoint.DataPoint) {
			defer wg.Done()

			result := &PipelineResult{TaskIndex: idx}
			output, err := p.Execute(ctx, inp)
			if err != nil {
				result.Error = err
				result.Status = PipelineStatusFailed
			} else {
				result.Output = output
				result.Status = PipelineStatusCompleted
			}
			results[idx] = result
		}(i, input)
	}

	wg.Wait()
	return results, nil
}

func (p *Pipeline) Status() PipelineStatus {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.status
}

func (p *Pipeline) Name() string {
	return p.name
}

func (p *Pipeline) Tasks() []Task {
	return p.tasks
}

type PipelineRunner struct {
	mu        sync.RWMutex
	pipelines map[string]*Pipeline
}

func NewRunner() *PipelineRunner {
	return &PipelineRunner{
		pipelines: make(map[string]*Pipeline),
	}
}

func (r *PipelineRunner) Register(pipeline *Pipeline) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.pipelines[pipeline.Name()] = pipeline
}

func (r *PipelineRunner) Get(name string) (*Pipeline, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.pipelines[name]
	return p, ok
}

func (r *PipelineRunner) Run(ctx context.Context, pipelineName string, input *datapoint.DataPoint) (*datapoint.DataPoint, error) {
	pipeline, ok := r.Get(pipelineName)
	if !ok {
		return nil, fmt.Errorf("pipeline not found: %s", pipelineName)
	}
	return pipeline.Execute(ctx, input)
}

func (r *PipelineRunner) RunBatch(ctx context.Context, pipelineName string, inputs []*datapoint.DataPoint) ([]*PipelineResult, error) {
	pipeline, ok := r.Get(pipelineName)
	if !ok {
		return nil, fmt.Errorf("pipeline not found: %s", pipelineName)
	}
	return pipeline.ExecuteBatch(ctx, inputs)
}
