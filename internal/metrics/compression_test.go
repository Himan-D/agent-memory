package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestCompressionMetricsLatencyStatsSortBeforeP95(t *testing.T) {
	collector := NewCompressionMetricsCollectorWithRegisterer(prometheus.NewRegistry())

	latencies := []int64{1000, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	for _, latency := range latencies {
		collector.RecordCompressionJob(10, latency, 0.97)
	}

	stats := collector.GetLatencyStats()

	if stats["p95_ms"] != int64(1000) {
		t.Fatalf("expected sorted p95 latency to be 1000, got %v", stats["p95_ms"])
	}
	if stats["min_ms"] != int64(1) {
		t.Fatalf("expected min latency to be 1, got %v", stats["min_ms"])
	}
	if stats["max_ms"] != int64(1000) {
		t.Fatalf("expected max latency to be 1000, got %v", stats["max_ms"])
	}
	if stats["avg_ms"] != int64(104) {
		t.Fatalf("expected average latency to be 104, got %v", stats["avg_ms"])
	}
}

func TestCompressionMetricsPrometheusGaugesUseSortedP95(t *testing.T) {
	registry := prometheus.NewRegistry()
	collector := NewCompressionMetricsCollectorWithRegisterer(registry)

	for _, latency := range []int64{1000, 1, 2, 3, 4, 5, 6, 7, 8, 9} {
		collector.RecordCompressionJob(10, latency, 0.97)
	}

	if got := testutil.ToFloat64(collector.compressionLatencyP95); got != 1000 {
		t.Fatalf("expected Prometheus p95 gauge to be 1000, got %f", got)
	}
	if got := testutil.ToFloat64(collector.compressionLatencyAvg); got != 104 {
		t.Fatalf("expected Prometheus average gauge to be 104, got %f", got)
	}
}
