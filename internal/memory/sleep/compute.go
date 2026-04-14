package sleep

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"agent-memory/internal/llm"
)

type Engine struct {
	mu          sync.RWMutex
	tasks       map[string]*SleepTask
	completed   map[string]*SleepResult
	scheduler   *Scheduler
	llm         llm.Provider
	maxWorkers  int
	activeCount int
}

type SleepTask struct {
	ID          string            `json:"id"`
	Type        TaskType          `json:"type"`
	Prompt      string            `json:"prompt"`
	Config      TaskConfig        `json:"config"`
	Status      TaskStatus        `json:"status"`
	ScheduledAt time.Time         `json:"scheduled_at"`
	StartedAt   *time.Time        `json:"started_at,omitempty"`
	CompletedAt *time.Time        `json:"completed_at,omitempty"`
	Result      *SleepResult      `json:"result,omitempty"`
	Error       string            `json:"error,omitempty"`
	RetryCount  int               `json:"retry_count"`
	DependsOn   []string          `json:"depends_on,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

type TaskType string

const (
	TaskTypeSummarize TaskType = "summarize"
	TaskTypeExtract   TaskType = "extract"
	TaskTypeClassify  TaskType = "classify"
	TaskTypeGenerate  TaskType = "generate"
	TaskTypeAnalyze   TaskType = "analyze"
	TaskTypeCompress  TaskType = "compress"
	TaskTypeSearch    TaskType = "search"
	TaskTypeReason    TaskType = "reason"
)

type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusScheduled TaskStatus = "scheduled"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusDone      TaskStatus = "done"
	TaskStatusFailed    TaskStatus = "failed"
	TaskStatusCancelled TaskStatus = "cancelled"
)

type TaskConfig struct {
	Model       string        `json:"model"`
	Temperature float64       `json:"temperature"`
	MaxTokens   int           `json:"max_tokens"`
	Priority    int           `json:"priority"`
	MaxRetries  int           `json:"max_retries"`
	Timeout     time.Duration `json:"timeout"`
}

type SleepResult struct {
	Content   string            `json:"content"`
	Artifacts map[string]string `json:"artifacts,omitempty"`
	Insights  []string          `json:"insights,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

type Scheduler struct {
	mu       sync.RWMutex
	tasks    map[string]*ScheduledTask
	triggers map[string]*Trigger
}

type ScheduledTask struct {
	ID       string
	TaskID   string
	Trigger  *Trigger
	NextRun  time.Time
	Schedule Schedule
}

type Trigger struct {
	Type      string
	Condition string
	Time      time.Time
	Interval  time.Duration
}

type Schedule string

const (
	ScheduleOnce   Schedule = "once"
	ScheduleHourly Schedule = "hourly"
	ScheduleDaily  Schedule = "daily"
	ScheduleWeekly Schedule = "weekly"
	ScheduleCustom Schedule = "custom"
)

func NewEngine(llmClient llm.Provider) *Engine {
	return &Engine{
		tasks:      make(map[string]*SleepTask),
		completed:  make(map[string]*SleepResult),
		scheduler:  NewScheduler(),
		llm:        llmClient,
		maxWorkers: 5,
	}
}

func NewScheduler() *Scheduler {
	return &Scheduler{
		tasks:    make(map[string]*ScheduledTask),
		triggers: make(map[string]*Trigger),
	}
}

func (e *Engine) Submit(ctx context.Context, task *SleepTask) error {
	if task.ID == "" {
		task.ID = generateID()
	}

	if task.Config.Model == "" {
		task.Config.Model = "gpt-4o"
	}
	if task.Config.MaxTokens == 0 {
		task.Config.MaxTokens = 2000
	}
	if task.Config.Temperature == 0 {
		task.Config.Temperature = 0.7
	}
	if task.Config.MaxRetries == 0 {
		task.Config.MaxRetries = 3
	}
	if task.Config.Timeout == 0 {
		task.Config.Timeout = 5 * time.Minute
	}

	e.mu.Lock()
	e.tasks[task.ID] = task
	e.mu.Unlock()

	go e.runTask(context.Background(), task)

	return nil
}

func (e *Engine) SubmitBatch(ctx context.Context, tasks []*SleepTask) error {
	for _, task := range tasks {
		if err := e.Submit(ctx, task); err != nil {
			return fmt.Errorf("failed to submit task %s: %w", task.ID, err)
		}
	}
	return nil
}

func (e *Engine) runTask(ctx context.Context, task *SleepTask) {
	e.mu.Lock()
	if e.activeCount >= e.maxWorkers {
		task.Status = TaskStatusScheduled
		e.mu.Unlock()
		time.Sleep(100 * time.Millisecond)
		go e.runTask(ctx, task)
		return
	}
	e.activeCount++
	e.mu.Unlock()

	defer func() {
		e.mu.Lock()
		e.activeCount--
		e.mu.Unlock()
	}()

	now := time.Now()
	task.StartedAt = &now
	task.Status = TaskStatusRunning

	result, err := e.executeTask(ctx, task)
	if err != nil {
		task.Status = TaskStatusFailed
		task.Error = err.Error()
		task.RetryCount++

		if task.RetryCount < task.Config.MaxRetries {
			go func() {
				time.Sleep(time.Duration(task.RetryCount*2) * time.Second)
				e.runTask(ctx, task)
			}()
		}
		return
	}

	completed := time.Now()
	task.CompletedAt = &completed
	task.Status = TaskStatusDone
	task.Result = result

	e.mu.Lock()
	e.completed[task.ID] = result
	e.mu.Unlock()
}

func (e *Engine) executeTask(ctx context.Context, task *SleepTask) (*SleepResult, error) {
	if e.llm == nil {
		return nil, fmt.Errorf("LLM not configured")
	}

	ctx, cancel := context.WithTimeout(ctx, task.Config.Timeout)
	defer cancel()

	switch task.Type {
	case TaskTypeSummarize:
		return e.summarize(ctx, task)
	case TaskTypeExtract:
		return e.extract(ctx, task)
	case TaskTypeClassify:
		return e.classify(ctx, task)
	case TaskTypeGenerate:
		return e.generate(ctx, task)
	case TaskTypeAnalyze:
		return e.analyze(ctx, task)
	case TaskTypeCompress:
		return e.compress(ctx, task)
	case TaskTypeSearch:
		return e.search(ctx, task)
	case TaskTypeReason:
		return e.reason(ctx, task)
	default:
		return nil, fmt.Errorf("unknown task type: %s", task.Type)
	}
}

func (e *Engine) summarize(ctx context.Context, task *SleepTask) (*SleepResult, error) {
	prompt := fmt.Sprintf("Summarize the following concisely:\n\n%s", task.Prompt)

	resp, err := e.llm.Complete(ctx, &llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: "user", Content: prompt},
		},
		Model:       task.Config.Model,
		MaxTokens:   task.Config.MaxTokens,
		Temperature: task.Config.Temperature,
	})
	if err != nil {
		return nil, fmt.Errorf("summarize failed: %w", err)
	}

	return &SleepResult{
		Content:  resp.Content,
		Metadata: map[string]string{"type": "summary"},
	}, nil
}

func (e *Engine) extract(ctx context.Context, task *SleepTask) (*SleepResult, error) {
	prompt := fmt.Sprintf("Extract key information from the following:\n\n%s", task.Prompt)

	resp, err := e.llm.Complete(ctx, &llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: "user", Content: prompt},
		},
		Model:       task.Config.Model,
		MaxTokens:   task.Config.MaxTokens,
		Temperature: task.Config.Temperature,
	})
	if err != nil {
		return nil, fmt.Errorf("extract failed: %w", err)
	}

	return &SleepResult{
		Content:  resp.Content,
		Insights: []string{resp.Content},
		Metadata: map[string]string{"type": "extraction"},
	}, nil
}

func (e *Engine) classify(ctx context.Context, task *SleepTask) (*SleepResult, error) {
	prompt := fmt.Sprintf("Classify the following into categories:\n\n%s", task.Prompt)

	resp, err := e.llm.Complete(ctx, &llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: "user", Content: prompt},
		},
		Model:       task.Config.Model,
		MaxTokens:   task.Config.MaxTokens,
		Temperature: task.Config.Temperature,
	})
	if err != nil {
		return nil, fmt.Errorf("classify failed: %w", err)
	}

	return &SleepResult{
		Content:  resp.Content,
		Metadata: map[string]string{"type": "classification"},
	}, nil
}

func (e *Engine) generate(ctx context.Context, task *SleepTask) (*SleepResult, error) {
	resp, err := e.llm.Complete(ctx, &llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: "user", Content: task.Prompt},
		},
		Model:       task.Config.Model,
		MaxTokens:   task.Config.MaxTokens,
		Temperature: task.Config.Temperature,
	})
	if err != nil {
		return nil, fmt.Errorf("generate failed: %w", err)
	}

	return &SleepResult{
		Content:  resp.Content,
		Metadata: map[string]string{"type": "generation"},
	}, nil
}

func (e *Engine) analyze(ctx context.Context, task *SleepTask) (*SleepResult, error) {
	prompt := fmt.Sprintf("Analyze the following and provide insights:\n\n%s", task.Prompt)

	resp, err := e.llm.Complete(ctx, &llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: "user", Content: prompt},
		},
		Model:       task.Config.Model,
		MaxTokens:   task.Config.MaxTokens,
		Temperature: task.Config.Temperature,
	})
	if err != nil {
		return nil, fmt.Errorf("analyze failed: %w", err)
	}

	return &SleepResult{
		Content:  resp.Content,
		Insights: []string{resp.Content},
		Metadata: map[string]string{"type": "analysis"},
	}, nil
}

func (e *Engine) compress(ctx context.Context, task *SleepTask) (*SleepResult, error) {
	prompt := fmt.Sprintf("Compress and retain the most important information:\n\n%s", task.Prompt)

	resp, err := e.llm.Complete(ctx, &llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: "user", Content: prompt},
		},
		Model:       task.Config.Model,
		MaxTokens:   task.Config.MaxTokens,
		Temperature: task.Config.Temperature,
	})
	if err != nil {
		return nil, fmt.Errorf("compress failed: %w", err)
	}

	return &SleepResult{
		Content:  resp.Content,
		Metadata: map[string]string{"type": "compression"},
	}, nil
}

func (e *Engine) search(ctx context.Context, task *SleepTask) (*SleepResult, error) {
	prompt := fmt.Sprintf("Search and retrieve relevant information for:\n\n%s", task.Prompt)

	resp, err := e.llm.Complete(ctx, &llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: "user", Content: prompt},
		},
		Model:       task.Config.Model,
		MaxTokens:   task.Config.MaxTokens,
		Temperature: task.Config.Temperature,
	})
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	return &SleepResult{
		Content:  resp.Content,
		Metadata: map[string]string{"type": "search"},
	}, nil
}

func (e *Engine) reason(ctx context.Context, task *SleepTask) (*SleepResult, error) {
	prompt := fmt.Sprintf("Reason through the following step by step:\n\n%s", task.Prompt)

	resp, err := e.llm.Complete(ctx, &llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: "user", Content: prompt},
		},
		Model:       task.Config.Model,
		MaxTokens:   task.Config.MaxTokens,
		Temperature: task.Config.Temperature,
	})
	if err != nil {
		return nil, fmt.Errorf("reason failed: %w", err)
	}

	return &SleepResult{
		Content:  resp.Content,
		Insights: []string{resp.Content},
		Metadata: map[string]string{"type": "reasoning"},
	}, nil
}

func (e *Engine) GetTask(id string) (*SleepTask, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if task, ok := e.tasks[id]; ok {
		return task, nil
	}

	return nil, fmt.Errorf("task not found: %s", id)
}

func (e *Engine) GetResult(id string) (*SleepResult, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if result, ok := e.completed[id]; ok {
		return result, nil
	}

	if task, ok := e.tasks[id]; ok {
		if task.Result != nil {
			return task.Result, nil
		}
		return nil, fmt.Errorf("task not completed: %s", id)
	}

	return nil, fmt.Errorf("result not found: %s", id)
}

func (e *Engine) ListTasks(status TaskStatus) []*SleepTask {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var result []*SleepTask
	for _, task := range e.tasks {
		if status == "" || task.Status == status {
			result = append(result, task)
		}
	}

	return result
}

func (e *Engine) Cancel(id string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	task, ok := e.tasks[id]
	if !ok {
		return fmt.Errorf("task not found: %s", id)
	}

	if task.Status == TaskStatusRunning || task.Status == TaskStatusDone {
		return fmt.Errorf("cannot cancel task in state: %s", task.Status)
	}

	task.Status = TaskStatusCancelled
	return nil
}

func (e *Engine) Schedule(taskID string, schedule Schedule, trigger *Trigger) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	task, ok := e.tasks[taskID]
	if !ok {
		return fmt.Errorf("task not found: %s", taskID)
	}

	scheduled := &ScheduledTask{
		ID:       generateID(),
		TaskID:   taskID,
		Trigger:  trigger,
		Schedule: schedule,
		NextRun:  e.calculateNextRun(schedule, trigger),
	}

	e.scheduler.tasks[scheduled.ID] = scheduled
	task.Status = TaskStatusScheduled

	return nil
}

func (e *Engine) calculateNextRun(schedule Schedule, trigger *Trigger) time.Time {
	now := time.Now()

	switch schedule {
	case ScheduleHourly:
		return now.Add(1 * time.Hour)
	case ScheduleDaily:
		return now.Add(24 * time.Hour)
	case ScheduleWeekly:
		return now.Add(7 * 24 * time.Hour)
	case ScheduleOnce:
		if trigger != nil && !trigger.Time.IsZero() {
			return trigger.Time
		}
		return now.Add(1 * time.Hour)
	case ScheduleCustom:
		if trigger != nil && trigger.Interval > 0 {
			return now.Add(trigger.Interval)
		}
		return now.Add(1 * time.Hour)
	default:
		return now.Add(1 * time.Hour)
	}
}

func (e *Engine) Wait(ctx context.Context, taskID string) (*SleepResult, error) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			task, err := e.GetTask(taskID)
			if err != nil {
				return nil, err
			}

			switch task.Status {
			case TaskStatusDone:
				return task.Result, nil
			case TaskStatusFailed:
				return nil, fmt.Errorf("task failed: %s", task.Error)
			case TaskStatusCancelled:
				return nil, fmt.Errorf("task cancelled")
			}
		}
	}
}

func (e *Engine) ProcessPipeline(ctx context.Context, tasks []*SleepTask) ([]*SleepResult, error) {
	if len(tasks) == 0 {
		return nil, nil
	}

	var results []*SleepResult
	for _, task := range tasks {
		if err := e.Submit(ctx, task); err != nil {
			return nil, fmt.Errorf("pipeline submit error: %w", err)
		}

		result, err := e.Wait(ctx, task.ID)
		if err != nil {
			return nil, fmt.Errorf("pipeline wait error: %w", err)
		}

		results = append(results, result)
	}

	return results, nil
}

func generateID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

type BackgroundWorker struct {
	engine     *Engine
	interval   time.Duration
	stopCh     chan struct{}
	maxRetries int
}

func NewBackgroundWorker(interval time.Duration) *BackgroundWorker {
	return &BackgroundWorker{
		engine:     NewEngine(nil),
		interval:   interval,
		stopCh:     make(chan struct{}),
		maxRetries: 3,
	}
}

func (w *BackgroundWorker) Start() {
	go func() {
		ticker := time.NewTicker(w.interval)
		defer ticker.Stop()

		for {
			select {
			case <-w.stopCh:
				return
			case <-ticker.C:
				w.processPendingTasks()
			}
		}
	}()
}

func (w *BackgroundWorker) Stop() {
	close(w.stopCh)
}

func (w *BackgroundWorker) processPendingTasks() {
	w.engine.mu.RLock()
	pending := w.engine.ListTasks(TaskStatusScheduled)
	w.engine.mu.RUnlock()

	for _, task := range pending {
		if task.RetryCount < w.maxRetries {
			go w.engine.runTask(context.Background(), task)
		}
	}
}

func (w *BackgroundWorker) SetEngine(engine *Engine) {
	w.engine = engine
}
