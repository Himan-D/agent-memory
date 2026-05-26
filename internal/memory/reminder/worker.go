package reminder

import (
	"context"
	"log"
	"sync"
	"time"

	"agent-memory/internal/memory/types"
)

// Worker polls for due reminders and surfaces them.
// Implements prospective memory — "remembering to do things in the future."
type Worker struct {
	interval     time.Duration
	memoryGetter MemoryGetter
	callbacks    []func(memory *types.Memory)
	stopCh       chan struct{}
	mu           sync.Mutex
}

// MemoryGetter is the interface for fetching memories with reminders.
type MemoryGetter interface {
	GetDueReminders(ctx context.Context, before time.Time) ([]*types.Memory, error)
}

// NewWorker creates a reminder worker that checks for due reminders.
func NewWorker(interval time.Duration, getter MemoryGetter) *Worker {
	return &Worker{
		interval:     interval,
		memoryGetter: getter,
		stopCh:       make(chan struct{}),
	}
}

// OnReminder registers a callback for when a reminder is due.
func (w *Worker) OnReminder(fn func(memory *types.Memory)) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.callbacks = append(w.callbacks, fn)
}

// Start begins the reminder polling loop.
func (w *Worker) Start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(w.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				w.checkReminders(ctx)
			case <-w.stopCh:
				return
			case <-ctx.Done():
				return
			}
		}
	}()
}

// Stop halts the reminder worker.
func (w *Worker) Stop() {
	close(w.stopCh)
}

func (w *Worker) checkReminders(ctx context.Context) {
	if w.memoryGetter == nil {
		return
	}

	memories, err := w.memoryGetter.GetDueReminders(ctx, time.Now())
	if err != nil {
		log.Printf("reminder: error checking reminders: %v", err)
		return
	}

	w.mu.Lock()
	cbs := make([]func(*types.Memory), len(w.callbacks))
	copy(cbs, w.callbacks)
	w.mu.Unlock()

	for _, mem := range memories {
		for _, cb := range cbs {
			cb(mem)
		}
		// Clear the reminder after firing
		mem.RemindAt = nil
	}
}
