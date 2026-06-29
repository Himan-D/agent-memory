// Package resilience provides utilities for building fault-tolerant systems.
// gather.go implements concurrency-limited fan-out for goroutines, modeled after
// Cognee's gather_with_concurrency_limit. It bounds the number of in-flight
// tasks and supports both fail-open and fail-strict semantics.
package resilience

import (
	"context"
	"sync"

	"golang.org/x/sync/semaphore"
)

// Task is a unit of work that may fail. Tasks take a context so they can
// observe cancellation.
type Task func(ctx context.Context) error

// GatherOptions controls GatherWithConcurrencyLimit behavior.
type GatherOptions struct {
	// Limit is the maximum number of tasks executing concurrently. Must be > 0.
	Limit int
	// FailFast cancels remaining tasks on the first error. When false (default),
	// GatherWithConcurrencyLimit is fail-open: all tasks run to completion and
	// the first error observed is returned.
	FailFast bool
}

// GatherWithConcurrencyLimit runs every task in fns with at most opts.Limit
// tasks in flight at any time. It returns when all tasks complete or when the
// context is canceled.
//
// In fail-open mode (FailFast == false), all tasks run to completion. The
// returned error is the first non-nil task error observed, or ctx.Err() if
// the context is canceled.
//
// In fail-fast mode (FailFast == true), the first error cancels the context
// for the remaining tasks; tasks that observe ctx.Done() should return
// ctx.Err(). The returned error is that first error.
func GatherWithConcurrencyLimit(ctx context.Context, opts GatherOptions, fns []Task) error {
	if len(fns) == 0 {
		return nil
	}
	limit := opts.Limit
	if limit <= 0 {
		limit = 1
	}

	sem := semaphore.NewWeighted(int64(limit))

	var (
		wg      sync.WaitGroup
		errOnce sync.Once
		firstErr error
	)

	taskCtx := ctx
	if opts.FailFast {
		var cancel context.CancelFunc
		taskCtx, cancel = context.WithCancel(ctx)
		defer cancel()
	}

	setErr := func(err error) {
		errOnce.Do(func() { firstErr = err })
	}

	for _, fn := range fns {
		wg.Add(1)
		go func(fn Task) {
			defer wg.Done()

			if err := sem.Acquire(taskCtx, 1); err != nil {
				setErr(err)
				return
			}
			defer sem.Release(1)

			if err := fn(taskCtx); err != nil {
				setErr(err)
			}
		}(fn)
	}

	wg.Wait()
	return firstErr
}

// Result is a typed result for GatherTyped.
type Result[T any] struct {
	Value T
	Err   error
}

// GatherTyped is the generic variant of GatherWithConcurrencyLimit. It runs
// fns concurrently with at most limit in flight and returns one Result per
// input fn, preserving order. A failing task does not abort the others in
// fail-open mode.
func GatherTyped[T any](ctx context.Context, opts GatherOptions, fns []func(ctx context.Context) (T, error)) []Result[T] {
	if len(fns) == 0 {
		return nil
	}
	limit := opts.Limit
	if limit <= 0 {
		limit = 1
	}

	results := make([]Result[T], len(fns))
	sem := semaphore.NewWeighted(int64(limit))
	var wg sync.WaitGroup

	for i, fn := range fns {
		wg.Add(1)
		go func(idx int, fn func(ctx context.Context) (T, error)) {
			defer wg.Done()
			if err := sem.Acquire(ctx, 1); err != nil {
				results[idx] = Result[T]{Err: err}
				return
			}
			defer sem.Release(1)

			v, err := fn(ctx)
			results[idx] = Result[T]{Value: v, Err: err}
		}(i, fn)
	}

	wg.Wait()
	return results
}
