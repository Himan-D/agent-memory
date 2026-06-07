package metrics

import (
	"sort"
	"sync"
)

type MetricsCollector struct {
	mu                        sync.RWMutex
	ExtractionsTotal          int64
	ExtractionsByProvider     map[string]int64
	SpreadingActivationsTotal int64
	SpreadingActivationHops   map[int]int64
	CompressionLatencyMs      float64
	latencySum                float64
	latencyCount              int64
	latencies                 []float64
	TokensSavedTotal          int64
	AccuracyRetention         float64
	TokenReduction            float64
	CacheHits                 int64
	CacheMisses               int64
	TierHits                  map[string]int64
	P95LatencyMs              float64
	CompressionErrors         int64
}

type MetricsSnapshot struct {
	ExtractionsTotal          int64
	ExtractionsByProvider     map[string]int64
	SpreadingActivationsTotal int64
	SpreadingActivationHops   map[int]int64
	CompressionLatencyMs      float64
	TokensSavedTotal          int64
	AccuracyRetention         float64
	TokenReduction            float64
	CacheHits                 int64
	CacheMisses               int64
	TierHits                  map[string]int64
	P95LatencyMs              float64
	CompressionErrors         int64
}

func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		ExtractionsByProvider:   make(map[string]int64),
		SpreadingActivationHops: make(map[int]int64),
		TierHits:                make(map[string]int64),
		latencies:               make([]float64, 0, 10000),
	}
}

func (m *MetricsCollector) RecordExtraction(provider string, tokensSaved int64, latencyMs float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ExtractionsTotal++
	m.ExtractionsByProvider[provider]++
	m.TokensSavedTotal += tokensSaved
	m.latencySum += latencyMs
	m.latencyCount++
	m.CompressionLatencyMs = m.latencySum / float64(m.latencyCount)
	m.latencies = append(m.latencies, latencyMs)
	if len(m.latencies) > 10000 {
		m.latencies = m.latencies[len(m.latencies)-10000:]
	}
	m.recalcP95Locked()
}

func (m *MetricsCollector) RecordSpreadingActivation(hops int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.SpreadingActivationsTotal++
	m.SpreadingActivationHops[hops]++
}

func (m *MetricsCollector) RecordTierHit(tier string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.TierHits[tier]++
}

func (m *MetricsCollector) RecordCacheHit(hit bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if hit {
		m.CacheHits++
	} else {
		m.CacheMisses++
	}
}

func (m *MetricsCollector) SetAccuracyRetention(retention float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.AccuracyRetention = retention
}

func (m *MetricsCollector) SetTokenReduction(reduction float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.TokenReduction = reduction
}

func (m *MetricsCollector) RecordCompressionError() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CompressionErrors++
}

func (m *MetricsCollector) recalcP95Locked() {
	n := len(m.latencies)
	if n == 0 {
		m.P95LatencyMs = 0
		return
	}
	sorted := make([]float64, n)
	copy(sorted, m.latencies)
	sort.Float64s(sorted)
	idx := int(float64(n) * 0.95)
	if idx >= n {
		idx = n - 1
	}
	m.P95LatencyMs = sorted[idx]
}

func (m *MetricsCollector) GetSnapshot() MetricsSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()

	providerCopy := make(map[string]int64, len(m.ExtractionsByProvider))
	for k, v := range m.ExtractionsByProvider {
		providerCopy[k] = v
	}

	hopsCopy := make(map[int]int64, len(m.SpreadingActivationHops))
	for k, v := range m.SpreadingActivationHops {
		hopsCopy[k] = v
	}

	tierCopy := make(map[string]int64, len(m.TierHits))
	for k, v := range m.TierHits {
		tierCopy[k] = v
	}

	return MetricsSnapshot{
		ExtractionsTotal:          m.ExtractionsTotal,
		ExtractionsByProvider:     providerCopy,
		SpreadingActivationsTotal: m.SpreadingActivationsTotal,
		SpreadingActivationHops:   hopsCopy,
		CompressionLatencyMs:      m.CompressionLatencyMs,
		TokensSavedTotal:          m.TokensSavedTotal,
		AccuracyRetention:         m.AccuracyRetention,
		TokenReduction:            m.TokenReduction,
		CacheHits:                 m.CacheHits,
		CacheMisses:               m.CacheMisses,
		TierHits:                  tierCopy,
		P95LatencyMs:              m.P95LatencyMs,
		CompressionErrors:         m.CompressionErrors,
	}
}

func (m *MetricsCollector) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ExtractionsTotal = 0
	m.ExtractionsByProvider = make(map[string]int64)
	m.SpreadingActivationsTotal = 0
	m.SpreadingActivationHops = make(map[int]int64)
	m.CompressionLatencyMs = 0
	m.latencySum = 0
	m.latencyCount = 0
	m.latencies = make([]float64, 0, 10000)
	m.TokensSavedTotal = 0
	m.AccuracyRetention = 0
	m.TokenReduction = 0
	m.CacheHits = 0
	m.CacheMisses = 0
	m.TierHits = make(map[string]int64)
	m.P95LatencyMs = 0
	m.CompressionErrors = 0
}

func (m *MetricsCollector) RestoreFromSnapshot(snap MetricsSnapshot) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ExtractionsTotal = snap.ExtractionsTotal
	m.ExtractionsByProvider = snap.ExtractionsByProvider
	m.SpreadingActivationsTotal = snap.SpreadingActivationsTotal
	m.SpreadingActivationHops = snap.SpreadingActivationHops
	m.CompressionLatencyMs = snap.CompressionLatencyMs
	m.TokensSavedTotal = snap.TokensSavedTotal
	m.AccuracyRetention = snap.AccuracyRetention
	m.TokenReduction = snap.TokenReduction
	m.CacheHits = snap.CacheHits
	m.CacheMisses = snap.CacheMisses
	m.TierHits = snap.TierHits
	m.P95LatencyMs = snap.P95LatencyMs
	m.CompressionErrors = snap.CompressionErrors
	m.latencySum = snap.CompressionLatencyMs * float64(snap.ExtractionsTotal)
	m.latencyCount = snap.ExtractionsTotal
	m.latencies = make([]float64, 0, 10000)
}
