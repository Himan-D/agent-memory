package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	compressionBenchmarks "agent-memory/internal/compression/benchmarks"
)

func (s *APIServer) listCompressionBenchmarkCorporaHandler(w http.ResponseWriter, r *http.Request) {
	if s.compressionBench == nil {
		safeHTTPError(w, r, fmt.Errorf("compression benchmark runner is not configured"), http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"corpora":    s.compressionBench.Corpora(),
		"algorithms": []string{"radix", "smart_radix", "smart_hybrid", "real_best", "gzip"},
		"defaults": map[string]interface{}{
			"corpus":        "agent_memory",
			"iterations":    3,
			"min_retention": 0.9,
		},
	})
}

func (s *APIServer) runCompressionBenchmarkHandler(w http.ResponseWriter, r *http.Request) {
	if s.compressionBench == nil {
		safeHTTPError(w, r, fmt.Errorf("compression benchmark runner is not configured"), http.StatusServiceUnavailable)
		return
	}

	var req compressionBenchmarks.Config
	if r.Body != nil && r.ContentLength != 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			safeHTTPError(w, r, fmt.Errorf("invalid benchmark request: %w", err), http.StatusBadRequest)
			return
		}
	}
	result, err := s.compressionBench.Run(r.Context(), req)
	if err != nil {
		safeHTTPError(w, r, fmt.Errorf("compression benchmark failed: %w", err), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
