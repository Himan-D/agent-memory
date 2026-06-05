package main

import (
	"encoding/json"
	"net/http"

	"agent-memory/internal/metrics"
)

// getCompressionStatsHandler returns persisted compression/metrics snapshot when available.
func (s *APIServer) getCompressionStatsHandler(w http.ResponseWriter, r *http.Request) {
	// Prefer persisted store (Neo4j) when available
	var snap metrics.MetricsSnapshot
	if s.metricsStore != nil {
		if stored, err := s.metricsStore.LoadSnapshot(r.Context()); err == nil && stored != nil {
			snap = *stored
		} else {
			snap = s.metricsCollector.GetSnapshot()
		}
	} else {
		snap = s.metricsCollector.GetSnapshot()
	}

	resp := map[string]interface{}{
		"accuracy_retention":          snap.AccuracyRetention,
		"tokens_saved_total":          snap.TokensSavedTotal,
		"p95_latency_ms":              snap.P95LatencyMs,
		"extractions_total":           snap.ExtractionsTotal,
		"extractions_by_provider":     snap.ExtractionsByProvider,
		"spreading_activations_total": snap.SpreadingActivationsTotal,
		"spreading_activation_hops":   snap.SpreadingActivationHops,
		"compression_latency_ms_avg":  snap.CompressionLatencyMs,
		"cache_hits":                  snap.CacheHits,
		"cache_misses":                snap.CacheMisses,
		"tier_hits":                   snap.TierHits,
		"compression_errors":          snap.CompressionErrors,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
