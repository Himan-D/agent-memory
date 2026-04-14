package resilience

import (
	"context"
	"errors"
	"sync"
	"time"
)

var ErrCircuitOpen = errors.New("circuit breaker: circuit is open")

type State int

const (
	StateClosed State = iota
	StateOpen
	StateHalfOpen
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

type CircuitBreaker struct {
	mu sync.RWMutex

	name string

	failureThreshold int
	successThreshold int
	timeout          time.Duration

	state State

	failures    int
	successes   int
	lastFailure time.Time

	onStateChange func(name string, from, to State)
}

type CircuitBreakerConfig struct {
	Name             string
	FailureThreshold int
	SuccessThreshold int
	Timeout          time.Duration
	OnStateChange    func(name string, from, to State)
}

func NewCircuitBreaker(cfg CircuitBreakerConfig) *CircuitBreaker {
	if cfg.FailureThreshold == 0 {
		cfg.FailureThreshold = 5
	}
	if cfg.SuccessThreshold == 0 {
		cfg.SuccessThreshold = 2
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}

	return &CircuitBreaker{
		name:             cfg.Name,
		failureThreshold: cfg.FailureThreshold,
		successThreshold: cfg.SuccessThreshold,
		timeout:          cfg.Timeout,
		state:            StateClosed,
		onStateChange:    cfg.OnStateChange,
	}
}

func (cb *CircuitBreaker) Name() string {
	return cb.name
}

func (cb *CircuitBreaker) State() State {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

func (cb *CircuitBreaker) Execute(ctx context.Context, fn func() error) error {
	if err := cb.allowRequest(ctx); err != nil {
		return err
	}

	err := fn()

	cb.recordResult(err)

	return err
}

func (cb *CircuitBreaker) allowRequest(ctx context.Context) error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		return nil

	case StateOpen:
		if time.Since(cb.lastFailure) > cb.timeout {
			cb.setState(StateHalfOpen)
			return nil
		}
		return ErrCircuitOpen

	case StateHalfOpen:
		return nil
	}

	return nil
}

func (cb *CircuitBreaker) recordResult(err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		cb.onFailure()
	} else {
		cb.onSuccess()
	}
}

func (cb *CircuitBreaker) onFailure() {
	cb.failures++
	cb.successes = 0
	cb.lastFailure = time.Now()

	if cb.state == StateHalfOpen {
		cb.setState(StateOpen)
		return
	}

	if cb.failures >= cb.failureThreshold {
		cb.setState(StateOpen)
	}
}

func (cb *CircuitBreaker) onSuccess() {
	cb.successes++

	if cb.state == StateHalfOpen && cb.successes >= cb.successThreshold {
		cb.setState(StateClosed)
		return
	}

	if cb.state == StateClosed {
		cb.failures = 0
	}
}

func (cb *CircuitBreaker) setState(state State) {
	if cb.state == state {
		return
	}

	oldState := cb.state
	cb.state = state

	if oldState == StateClosed && state == StateOpen {
		cb.lastFailure = time.Now()
	}

	if state == StateClosed {
		cb.failures = 0
		cb.successes = 0
	}

	if cb.onStateChange != nil {
		cb.onStateChange(cb.name, oldState, state)
	}
}

func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.setState(StateClosed)
}
