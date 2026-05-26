package sleep

import (
	"context"
	"fmt"
	"time"
)

// ConsolidationScheduler triggers consolidation on a configurable interval.
// It wraps a Dreamer and a BackgroundWorker to run periodic memory consolidation.
type ConsolidationScheduler struct {
	Interval time.Duration // default: 24 hours
	Dreamer  *Dreamer
	Worker   *BackgroundWorker
	stopCh   chan struct{}
}

// NewConsolidationScheduler creates a scheduler with the given dreamer, worker, and interval.
func NewConsolidationScheduler(dreamer *Dreamer, worker *BackgroundWorker, interval time.Duration) *ConsolidationScheduler {
	if interval <= 0 {
		interval = 24 * time.Hour
	}

	return &ConsolidationScheduler{
		Interval: interval,
		Dreamer:  dreamer,
		Worker:   worker,
		stopCh:   make(chan struct{}),
	}
}

// Start begins the periodic consolidation loop. It blocks until the context is cancelled
// or Stop is called.
func (s *ConsolidationScheduler) Start(ctx context.Context) error {
	if s.Dreamer == nil {
		return fmt.Errorf("consolidation scheduler: dreamer is nil")
	}

	ticker := time.NewTicker(s.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-s.stopCh:
			return nil
		case <-ticker.C:
			if err := s.TriggerNow(ctx); err != nil {
				// Log but don't stop — next tick will retry.
				_ = err
			}
		}
	}
}

// Stop signals the scheduler to exit its run loop.
func (s *ConsolidationScheduler) Stop() {
	select {
	case <-s.stopCh:
		// Already closed
	default:
		close(s.stopCh)
	}
}

// TriggerNow runs a single consolidation cycle immediately.
// In production, this would fetch memories from the store; the skeleton accepts
// an empty batch since store wiring happens in service.go.
func (s *ConsolidationScheduler) TriggerNow(ctx context.Context) error {
	if s.Dreamer == nil {
		return fmt.Errorf("consolidation scheduler: dreamer is nil")
	}

	// Placeholder: in service.go wiring, this will pull memories from the repository
	// and feed them to the dreamer. For now we just validate the dreamer is ready.
	return nil
}
