package metrics

import "sync"

type MetricsCollector struct {
	mu                       sync.RWMutex
	ExtractionsTotal         int64
	ExtractionsByProvider    map[string]int64
	SpreadingActivationsTotal int64
	SpreadingActivationHops  map[int]int64
	CompressionLatencyMs    float64
	latencySum               float64
	latencyCount             int64
	TokensSavedTotal         int64
	AccuracyRetention        float64
	CacheHits                int64
	CacheMisses              int64
	TierHits                 map[string]int64
}

type MetricsSnapshot struct {
	ExtractionsTotal          int64
	ExtractionsByProvider     map[string]int64
	SpreadingActivationsTotal int64
	SpreadingActivationHops   map[int]int64
	CompressionLatencyMs      float64
	TokensSavedTotal          int64
	AccuracyRetention         float64
	CacheHits                 int64
	CacheMisses               int64
	TierHits                  map[string]int64
}

func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		ExtractionsByProvider:    make(map[string]int64),
		SpreadingActivationHops: make(map[int]int64),
		TierHits:                make(map[string]int64),
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
		CacheHits:                 m.CacheHits,
		CacheMisses:               m.CacheMisses,
		TierHits:                  tierCopy,
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
	m.TokensSavedTotal = 0
	m.AccuracyRetention = 0
	m.CacheHits = 0
	m.CacheMisses = 0
	m.TierHits = make(map[string]int64)
}