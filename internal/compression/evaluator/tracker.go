package evaluator

import (
	"context"
	"math/rand"
	"sync"
)

const maxTrackedResults = 1000

// FidelityTracker performs sample-based continuous fidelity evaluation.
// Instead of evaluating every compression (too expensive), it evaluates
// a configurable percentage of compressions and maintains a rolling window
// of fidelity scores.
type FidelityTracker struct {
	mu         sync.RWMutex
	evaluator  *FidelityEvaluator
	sampleRate float64 // 0.0–1.0, e.g. 0.05 = evaluate 5% of compressions
	results    []FidelityResult
	totalEvals int64
	totalCalls int64
}

// NewFidelityTracker creates a tracker with the given evaluator and sample rate.
// sampleRate of 0.05 means ~5% of compressions are evaluated for fidelity.
func NewFidelityTracker(evaluator *FidelityEvaluator, sampleRate float64) *FidelityTracker {
	if sampleRate < 0 {
		sampleRate = 0
	}
	if sampleRate > 1 {
		sampleRate = 1
	}
	return &FidelityTracker{
		evaluator:  evaluator,
		sampleRate: sampleRate,
		results:    make([]FidelityResult, 0, maxTrackedResults),
	}
}

// MaybeEvaluate evaluates fidelity for a random sample of compressions.
// Safe to call from hot paths — most calls return immediately.
func (t *FidelityTracker) MaybeEvaluate(ctx context.Context, original, compressed string) {
	if t.evaluator == nil || t.sampleRate <= 0 {
		return
	}

	t.mu.Lock()
	t.totalCalls++
	shouldEval := rand.Float64() < t.sampleRate
	t.mu.Unlock()

	if !shouldEval {
		return
	}

	result, err := t.evaluator.Evaluate(ctx, original, compressed)
	if err != nil || result == nil {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	t.totalEvals++
	if len(t.results) >= maxTrackedResults {
		// Evict oldest 10% to make room
		evict := maxTrackedResults / 10
		if evict < 1 {
			evict = 1
		}
		t.results = t.results[evict:]
	}
	t.results = append(t.results, *result)
}

// AverageFidelity returns the running average fidelity scores.
func (t *FidelityTracker) AverageFidelity() FidelityResult {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if len(t.results) == 0 {
		return FidelityResult{}
	}

	var totalRecall, totalPrecision, totalF1 float64
	for _, r := range t.results {
		totalRecall += r.Recall
		totalPrecision += r.Precision
		totalF1 += r.F1
	}
	n := float64(len(t.results))
	return FidelityResult{
		Recall:    totalRecall / n,
		Precision: totalPrecision / n,
		F1:        totalF1 / n,
	}
}

// Stats returns the tracker's operational statistics.
func (t *FidelityTracker) Stats() (evals int64, calls int64, sampleCount int) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.totalEvals, t.totalCalls, len(t.results)
}

// SampleRate returns the configured sampling rate.
func (t *FidelityTracker) SampleRate() float64 {
	return t.sampleRate
}
