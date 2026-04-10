package memory

import (
	"context"
	"sync"
	"time"

	"agent-memory/internal/memory/types"
)

type AsyncClient struct {
	service *Service
	workers int
	queue   chan asyncTask
	wg      sync.WaitGroup
}

type asyncTask struct {
	ctx      context.Context
	taskType string
	taskFunc func() (interface{}, error)
	resultCh chan *asyncResult
	timeout  time.Duration
}

type asyncResult struct {
	Value interface{}
	Error error
}

func NewAsyncClient(service *Service, workers int) *AsyncClient {
	if workers <= 0 {
		workers = 10
	}

	client := &AsyncClient{
		service: service,
		workers: workers,
		queue:   make(chan asyncTask, 1000),
	}

	for i := 0; i < workers; i++ {
		go client.worker()
	}

	return client
}

func (c *AsyncClient) worker() {
	for task := range c.queue {
		result := &asyncResult{}

		ctx := task.ctx
		if task.timeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, task.timeout)
			defer cancel()
		}

		value, err := task.taskFunc()
		result.Value = value
		result.Error = err

		select {
		case task.resultCh <- result:
		default:
		}
	}
	c.wg.Done()
}

func (c *AsyncClient) Submit(taskFunc func() (interface{}, error)) <-chan *asyncResult {
	return c.SubmitWithContext(context.Background(), taskFunc, 0)
}

func (c *AsyncClient) SubmitWithContext(ctx context.Context, taskFunc func() (interface{}, error), timeout time.Duration) <-chan *asyncResult {
	resultCh := make(chan *asyncResult, 1)

	select {
	case c.queue <- asyncTask{
		ctx:      ctx,
		taskFunc: taskFunc,
		resultCh: resultCh,
		timeout:  timeout,
	}:
		c.wg.Add(1)
	default:
		resultCh <- &asyncResult{Error: ErrQueueFull}
	}

	return resultCh
}

func (c *AsyncClient) CreateMemoryAsync(ctx context.Context, mem *types.Memory, timeout time.Duration) <-chan *asyncResult {
	return c.SubmitWithContext(ctx, func() (interface{}, error) {
		return c.service.CreateMemory(ctx, mem)
	}, timeout)
}

func (c *AsyncClient) SearchMemoriesAsync(ctx context.Context, req *types.SearchRequest, timeout time.Duration) <-chan *asyncResult {
	return c.SubmitWithContext(ctx, func() (interface{}, error) {
		return c.service.SearchMemories(ctx, req)
	}, timeout)
}

func (c *AsyncClient) GetMemoryAsync(ctx context.Context, id string, timeout time.Duration) <-chan *asyncResult {
	return c.SubmitWithContext(ctx, func() (interface{}, error) {
		return c.service.GetMemory(ctx, id)
	}, timeout)
}

func (c *AsyncClient) UpdateMemoryAsync(ctx context.Context, id string, content string, metadata map[string]interface{}, timeout time.Duration) <-chan *asyncResult {
	return c.SubmitWithContext(ctx, func() (interface{}, error) {
		return nil, c.service.UpdateMemory(ctx, id, content, metadata)
	}, timeout)
}

func (c *AsyncClient) DeleteMemoryAsync(ctx context.Context, id string, timeout time.Duration) <-chan *asyncResult {
	return c.SubmitWithContext(ctx, func() (interface{}, error) {
		return nil, c.service.DeleteMemory(ctx, id)
	}, timeout)
}

func (c *AsyncClient) BatchCreateMemoriesAsync(ctx context.Context, memories []*types.Memory, timeout time.Duration) <-chan *asyncResult {
	return c.SubmitWithContext(ctx, func() (interface{}, error) {
		return c.service.BatchCreateMemories(ctx, memories)
	}, timeout)
}

func (c *AsyncClient) HybridSearchAsync(ctx context.Context, req *types.HybridSearchRequest, timeout time.Duration) <-chan *asyncResult {
	return c.SubmitWithContext(ctx, func() (interface{}, error) {
		return c.service.HybridSearch(ctx, req)
	}, timeout)
}

func (c *AsyncClient) GetMemoryStatsAsync(ctx context.Context, userID, orgID string, timeout time.Duration) <-chan *asyncResult {
	return c.SubmitWithContext(ctx, func() (interface{}, error) {
		return c.service.GetMemoryStats(ctx, userID, orgID)
	}, timeout)
}

func (c *AsyncClient) GenerateMemorySummaryAsync(ctx context.Context, userID string, timeout time.Duration) <-chan *asyncResult {
	return c.SubmitWithContext(ctx, func() (interface{}, error) {
		return c.service.GenerateMemorySummary(ctx, userID)
	}, timeout)
}

func (c *AsyncClient) Close() error {
	close(c.queue)
	c.wg.Wait()
	return nil
}

var ErrQueueFull = &AsyncError{Code: "QUEUE_FULL", Message: "async queue is full"}

type AsyncError struct {
	Code    string
	Message string
}

func (e *AsyncError) Error() string {
	return e.Message
}

type BatchAsyncResult struct {
	Results []interface{}
	Errors  []error
	Count   int
}

func (c *AsyncClient) SubmitBatch(tasks []func() (interface{}, error), timeout time.Duration) *BatchAsyncResult {
	result := &BatchAsyncResult{
		Results: make([]interface{}, len(tasks)),
		Errors:  make([]error, len(tasks)),
		Count:   len(tasks),
	}

	var wg sync.WaitGroup
	mu := sync.Mutex{}

	for i, task := range tasks {
		wg.Add(1)
		go func(idx int, t func() (interface{}, error)) {
			defer wg.Done()
			r := <-c.SubmitWithContext(context.Background(), t, timeout)
			mu.Lock()
			if r.Error != nil {
				result.Errors[idx] = r.Error
			} else {
				result.Results[idx] = r.Value
			}
			mu.Unlock()
		}(i, task)
	}

	wg.Wait()
	return result
}

type RateLimiter struct {
	tokens     float64
	maxTokens  float64
	refillRate float64
	lastRefill time.Time
	mu         sync.Mutex
}

func NewRateLimiter(maxTokens float64, refillRate float64) *RateLimiter {
	return &RateLimiter{
		tokens:     maxTokens,
		maxTokens:  maxTokens,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

func (rl *RateLimiter) Allow(tokens float64) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.refill()

	if rl.tokens >= tokens {
		rl.tokens -= tokens
		return true
	}
	return false
}

func (rl *RateLimiter) refill() {
	now := time.Now()
	elapsed := now.Sub(rl.lastRefill).Seconds()
	refilled := elapsed * rl.refillRate

	rl.tokens += refilled
	if rl.tokens > rl.maxTokens {
		rl.tokens = rl.maxTokens
	}

	rl.lastRefill = now
}

func (rl *RateLimiter) Wait(tokens float64) {
	for !rl.Allow(tokens) {
		time.Sleep(time.Millisecond * 10)
	}
}

type CircuitBreaker struct {
	failures    int
	maxFailures int
	timeout     time.Duration
	lastFailure time.Time
	state       CircuitState
	mu          sync.RWMutex
}

type CircuitState int

const (
	CircuitClosed CircuitState = iota
	CircuitOpen
	CircuitHalfOpen
)

func NewCircuitBreaker(maxFailures int, timeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		maxFailures: maxFailures,
		timeout:     timeout,
		state:       CircuitClosed,
	}
}

func (cb *CircuitBreaker) Execute(task func() (interface{}, error)) (interface{}, error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == CircuitOpen {
		if time.Since(cb.lastFailure) > cb.timeout {
			cb.state = CircuitHalfOpen
		} else {
			return nil, &CircuitOpenError{}
		}
	}

	result, err := task()

	if err != nil {
		cb.failures++
		cb.lastFailure = time.Now()
		if cb.failures >= cb.maxFailures {
			cb.state = CircuitOpen
		}
		return nil, err
	}

	if cb.state == CircuitHalfOpen {
		cb.state = CircuitClosed
		cb.failures = 0
	}

	return result, nil
}

func (cb *CircuitBreaker) GetState() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

type CircuitOpenError struct{}

func (e *CircuitOpenError) Error() string {
	return "circuit breaker is open"
}
