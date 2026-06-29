package resilience

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestGatherWithConcurrencyLimit_NoTasks(t *testing.T) {
	if err := GatherWithConcurrencyLimit(context.Background(), GatherOptions{Limit: 4}, nil); err != nil {
		t.Fatalf("expected nil error for empty task list, got %v", err)
	}
}

func TestGatherWithConcurrencyLimit_AllSucceed(t *testing.T) {
	const total = 20
	tasks := make([]Task, total)
	for i := range tasks {
		tasks[i] = func(ctx context.Context) error { return nil }
	}
	if err := GatherWithConcurrencyLimit(context.Background(), GatherOptions{Limit: 4}, tasks); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestGatherWithConcurrencyLimit_BoundsConcurrency(t *testing.T) {
	const (
		total       = 30
		concurrency = 5
	)
	var inFlight, peak int64
	tasks := make([]Task, total)
	for i := range tasks {
		tasks[i] = func(ctx context.Context) error {
			cur := atomic.AddInt64(&inFlight, 1)
			for {
				p := atomic.LoadInt64(&peak)
				if cur <= p || atomic.CompareAndSwapInt64(&peak, p, cur) {
					break
				}
			}
			time.Sleep(5 * time.Millisecond)
			atomic.AddInt64(&inFlight, -1)
			return nil
		}
	}
	if err := GatherWithConcurrencyLimit(context.Background(), GatherOptions{Limit: concurrency}, tasks); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := atomic.LoadInt64(&peak); got > concurrency {
		t.Fatalf("peak in-flight %d exceeded limit %d", got, concurrency)
	}
}

func TestGatherWithConcurrencyLimit_FailOpenReturnsFirstError(t *testing.T) {
	want := errors.New("boom")
	tasks := []Task{
		func(ctx context.Context) error { return nil },
		func(ctx context.Context) error { return want },
		func(ctx context.Context) error { return nil },
	}
	if err := GatherWithConcurrencyLimit(context.Background(), GatherOptions{Limit: 2}, tasks); !errors.Is(err, want) {
		t.Fatalf("expected %v, got %v", want, err)
	}
}

func TestGatherTyped_PreservesOrder(t *testing.T) {
	fns := []func(ctx context.Context) (int, error){
		func(ctx context.Context) (int, error) { return 1, nil },
		func(ctx context.Context) (int, error) { return 2, errors.New("e2") },
		func(ctx context.Context) (int, error) { return 3, nil },
	}
	results := GatherTyped(context.Background(), GatherOptions{Limit: 2}, fns)
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	if results[0].Value != 1 || results[0].Err != nil {
		t.Fatalf("result[0] = %+v", results[0])
	}
	if results[1].Value != 2 || results[1].Err == nil || results[1].Err.Error() != "e2" {
		t.Fatalf("result[1] = %+v", results[1])
	}
	if results[2].Value != 3 || results[2].Err != nil {
		t.Fatalf("result[2] = %+v", results[2])
	}
}

func TestGatherWithConcurrencyLimit_RespectsZeroLimit(t *testing.T) {
	// Limit 0 should default to 1, not deadlock.
	tasks := []Task{
		func(ctx context.Context) error { return nil },
		func(ctx context.Context) error { return nil },
	}
	if err := GatherWithConcurrencyLimit(context.Background(), GatherOptions{Limit: 0}, tasks); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}
