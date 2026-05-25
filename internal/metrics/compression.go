package metrics

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// CompressionMetricsCollector tracks compression engine metrics
type CompressionMetricsCollector struct {
	mu                    sync.RWMutex
	jobsProcessed         int64
	tokensSaved           int64
	accuracyRetention     float64
	latencies             []int64 // milliseconds
	modesExtracted        map[string]int64
	spreadingOpsCount     int64

	// Prometheus metrics
	compressionJobs       prometheus.Gauge
	compressionTokens     prometheus.Gauge
	compressionAccuracy   prometheus.Gauge
	compressionLatencyP95  prometheus.Gauge
	compressionLatencyAvg  prometheus.Gauge
}

// NewCompressionMetricsCollector creates a new collector
func NewCompressionMetricsCollector() *CompressionMetricsCollector {
	return &CompressionMetricsCollector{
		latencies:       make([]int64, 0),
		modesExtracted:  make(map[string]int64),
		compressionJobs: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "agent_memory_compression_jobs_total",
			Help: "Total compression jobs processed",
		}),
		compressionTokens: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "agent_memory_tokens_saved_total",
			Help: "Total tokens saved through compression",
		}),
		compressionAccuracy: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "agent_memory_compression_accuracy_retention",
			Help: "Accuracy retention rate (0-1)",
		}),
		compressionLatencyP95: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "agent_memory_compression_latency_p95_ms",
			Help: "P95 compression latency in milliseconds",
		}),
		compressionLatencyAvg: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "agent_memory_compression_latency_avg_ms",
			Help: "Average compression latency in milliseconds",
		}),
	}
}

// RecordCompressionJob records a completed compression job
func (cmc *CompressionMetricsCollector) RecordCompressionJob(tokensSaved int64, latencyMs int64, accuracyRetention float64) {
	cmc.mu.Lock()
	defer cmc.mu.Unlock()

	atomic.AddInt64(&cmc.jobsProcessed, 1)
	atomic.AddInt64(&cmc.tokensSaved, tokensSaved)
	cmc.accuracyRetention = accuracyRetention
	cmc.latencies = append(cmc.latencies, latencyMs)

	// Update Prometheus metrics
	cmc.compressionJobs.Set(float64(atomic.LoadInt64(&cmc.jobsProcessed)))
	cmc.compressionTokens.Set(float64(atomic.LoadInt64(&cmc.tokensSaved)))
	cmc.compressionAccuracy.Set(accuracyRetention)
	cmc.updateLatencyMetrics()
}

// RecordMemoryExtraction tracks memory extraction per mode
func (cmc *CompressionMetricsCollector) RecordMemoryExtraction(mode string, count int64) {
	cmc.mu.Lock()
	defer cmc.mu.Unlock()

	cmc.modesExtracted[mode] += count
}

// RecordSpreadingActivationOp counts spreading activation operations
func (cmc *CompressionMetricsCollector) RecordSpreadingActivationOp() {
	atomic.AddInt64(&cmc.spreadingOpsCount, 1)
}

// GetJobsProcessed returns total jobs processed
func (cmc *CompressionMetricsCollector) GetJobsProcessed() int64 {
	return atomic.LoadInt64(&cmc.jobsProcessed)
}

// GetTokensSaved returns total tokens saved
func (cmc *CompressionMetricsCollector) GetTokensSaved() int64 {
	return atomic.LoadInt64(&cmc.tokensSaved)
}

// GetAccuracyRetention returns current accuracy retention
func (cmc *CompressionMetricsCollector) GetAccuracyRetention() float64 {
	cmc.mu.RLock()
	defer cmc.mu.RUnlock()
	return cmc.accuracyRetention
}

// GetLatencyStats returns latency statistics
func (cmc *CompressionMetricsCollector) GetLatencyStats() map[string]interface{} {
	cmc.mu.RLock()
	defer cmc.mu.RUnlock()

	if len(cmc.latencies) == 0 {
		return map[string]interface{}{
			"count":     0,
			"p95_ms":    0,
			"avg_ms":    0,
			"min_ms":    0,
			"max_ms":    0,
		}
	}

	sum := int64(0)
	min := cmc.latencies[0]
	max := cmc.latencies[0]

	for _, lat := range cmc.latencies {
		sum += lat
		if lat < min {
			min = lat
		}
		if lat > max {
			max = lat
		}
	}

	avg := sum / int64(len(cmc.latencies))
	p95Index := int(float64(len(cmc.latencies)) * 0.95)
	if p95Index >= len(cmc.latencies) {
		p95Index = len(cmc.latencies) - 1
	}

	return map[string]interface{}{
		"count":     len(cmc.latencies),
		"p95_ms":    cmc.latencies[p95Index],
		"avg_ms":    avg,
		"min_ms":    min,
		"max_ms":    max,
	}
}

// GetModeStats returns extraction stats per mode
func (cmc *CompressionMetricsCollector) GetModeStats() map[string]interface{} {
	cmc.mu.RLock()
	defer cmc.mu.RUnlock()

	modes := make(map[string]int64)
	for mode, count := range cmc.modesExtracted {
		modes[mode] = count
	}

	return map[string]interface{}{
		"modes": modes,
	}
}

// GetSpreadingOpsCount returns total spreading activation operations
func (cmc *CompressionMetricsCollector) GetSpreadingOpsCount() int64 {
	return atomic.LoadInt64(&cmc.spreadingOpsCount)
}

// GetSnapshot returns complete metrics snapshot
func (cmc *CompressionMetricsCollector) GetSnapshot() map[string]interface{} {
	cmc.mu.RLock()
	defer cmc.mu.RUnlock()

	return map[string]interface{}{
		"jobs_processed":           atomic.LoadInt64(&cmc.jobsProcessed),
		"tokens_saved":             atomic.LoadInt64(&cmc.tokensSaved),
		"accuracy_retention":       cmc.accuracyRetention,
		"latency_stats":            cmc.GetLatencyStats(),
		"mode_stats":               cmc.GetModeStats(),
		"spreading_ops_count":      atomic.LoadInt64(&cmc.spreadingOpsCount),
		"timestamp":                time.Now().UTC().Format(time.RFC3339),
	}
}

// updateLatencyMetrics calculates and updates latency prometheus metrics
func (cmc *CompressionMetricsCollector) updateLatencyMetrics() {
	if len(cmc.latencies) == 0 {
		return
	}

	sum := int64(0)
	for _, lat := range cmc.latencies {
		sum += lat
	}

	avg := float64(sum) / float64(len(cmc.latencies))
	cmc.compressionLatencyAvg.Set(avg)

	p95Index := int(float64(len(cmc.latencies)) * 0.95)
	if p95Index >= len(cmc.latencies) {
		p95Index = len(cmc.latencies) - 1
	}
	cmc.compressionLatencyP95.Set(float64(cmc.latencies[p95Index]))
}

// ResetMetrics clears all metrics
func (cmc *CompressionMetricsCollector) ResetMetrics() {
	cmc.mu.Lock()
	defer cmc.mu.Unlock()

	atomic.StoreInt64(&cmc.jobsProcessed, 0)
	atomic.StoreInt64(&cmc.tokensSaved, 0)
	atomic.StoreInt64(&cmc.spreadingOpsCount, 0)
	cmc.accuracyRetention = 0
	cmc.latencies = make([]int64, 0)
	cmc.modesExtracted = make(map[string]int64)
}
