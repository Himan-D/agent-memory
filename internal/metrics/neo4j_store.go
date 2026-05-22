package metrics

import (
	"context"
	"encoding/json"
	"fmt"

	"agent-memory/internal/memory/neo4j"

	neo4jdriver "github.com/neo4j/neo4j-go-driver/v6/neo4j"
)

type Neo4jMetricsStore struct {
	client *neo4j.Client
}

func NewNeo4jMetricsStore(client *neo4j.Client) *Neo4jMetricsStore {
	return &Neo4jMetricsStore{client: client}
}

func (s *Neo4jMetricsStore) Init(ctx context.Context) error {
	session, cleanup := s.client.GetSession(ctx)
	defer cleanup()
	_, err := session.Run(ctx,
		"CREATE INDEX metrics_snapshot_idx IF NOT EXISTS FOR (m:MetricsSnapshot) ON (m.id)",
		nil,
	)
	if err != nil {
		return fmt.Errorf("metrics neo4j init: %w", err)
	}
	return nil
}

func (s *Neo4jMetricsStore) SaveSnapshot(ctx context.Context, snap MetricsSnapshot) error {
	extractionsByProviderJSON, err := json.Marshal(snap.ExtractionsByProvider)
	if err != nil {
		return fmt.Errorf("metrics save snapshot marshal extractions_by_provider: %w", err)
	}
	spreadingActivationHopsJSON, err := json.Marshal(snap.SpreadingActivationHops)
	if err != nil {
		return fmt.Errorf("metrics save snapshot marshal spreading_activation_hops: %w", err)
	}
	tierHitsJSON, err := json.Marshal(snap.TierHits)
	if err != nil {
		return fmt.Errorf("metrics save snapshot marshal tier_hits: %w", err)
	}

	session, cleanup := s.client.GetSession(ctx)
	defer cleanup()
	_, err = session.Run(ctx, `
		MERGE (m:MetricsSnapshot {id: "current"})
		SET m.extractions_total = $extractions_total,
		    m.extractions_by_provider_json = $extractions_by_provider_json,
		    m.spreading_activations_total = $spreading_activations_total,
		    m.spreading_activation_hops_json = $spreading_activation_hops_json,
		    m.compression_latency_ms = $compression_latency_ms,
		    m.tokens_saved_total = $tokens_saved_total,
		    m.accuracy_retention = $accuracy_retention,
		    m.cache_hits = $cache_hits,
		    m.cache_misses = $cache_misses,
		    m.tier_hits_json = $tier_hits_json,
		    m.p95_latency_ms = $p95_latency_ms,
		    m.compression_errors = $compression_errors`,
		map[string]interface{}{
			"extractions_total":              snap.ExtractionsTotal,
			"extractions_by_provider_json":   string(extractionsByProviderJSON),
			"spreading_activations_total":    snap.SpreadingActivationsTotal,
			"spreading_activation_hops_json": string(spreadingActivationHopsJSON),
			"compression_latency_ms":         snap.CompressionLatencyMs,
			"tokens_saved_total":             snap.TokensSavedTotal,
			"accuracy_retention":             snap.AccuracyRetention,
			"cache_hits":                     snap.CacheHits,
			"cache_misses":                   snap.CacheMisses,
			"tier_hits_json":                 string(tierHitsJSON),
			"p95_latency_ms":                 snap.P95LatencyMs,
			"compression_errors":             snap.CompressionErrors,
		},
	)
	if err != nil {
		return fmt.Errorf("metrics save snapshot: %w", err)
	}
	return nil
}

func (s *Neo4jMetricsStore) LoadSnapshot(ctx context.Context) (*MetricsSnapshot, error) {
	session, cleanup := s.client.GetSession(ctx)
	defer cleanup()
	result, err := session.Run(ctx,
		"MATCH (m:MetricsSnapshot {id: $id}) RETURN m",
		map[string]interface{}{"id": "current"},
	)
	if err != nil {
		return nil, fmt.Errorf("metrics load snapshot: %w", err)
	}

	if !result.Next(ctx) {
		return nil, nil
	}

	record := result.Record()
	raw, ok := record.Get("m")
	if !ok {
		return nil, nil
	}

	node, ok := raw.(neo4jdriver.Node)
	if !ok {
		return nil, nil
	}

	props := node.Props

	snap := &MetricsSnapshot{
		ExtractionsTotal:          getInt64(props, "extractions_total"),
		SpreadingActivationsTotal: getInt64(props, "spreading_activations_total"),
		CompressionLatencyMs:      getFloat64(props, "compression_latency_ms"),
		TokensSavedTotal:          getInt64(props, "tokens_saved_total"),
		AccuracyRetention:         getFloat64(props, "accuracy_retention"),
		CacheHits:                 getInt64(props, "cache_hits"),
		CacheMisses:               getInt64(props, "cache_misses"),
		P95LatencyMs:              getFloat64(props, "p95_latency_ms"),
		CompressionErrors:         getInt64(props, "compression_errors"),
	}

	if v, ok := props["extractions_by_provider_json"].(string); ok {
		_ = json.Unmarshal([]byte(v), &snap.ExtractionsByProvider)
	}
	if v, ok := props["spreading_activation_hops_json"].(string); ok {
		_ = json.Unmarshal([]byte(v), &snap.SpreadingActivationHops)
	}
	if v, ok := props["tier_hits_json"].(string); ok {
		_ = json.Unmarshal([]byte(v), &snap.TierHits)
	}

	return snap, nil
}

func getInt64(props map[string]interface{}, key string) int64 {
	v, ok := props[key]
	if !ok {
		return 0
	}
	switch val := v.(type) {
	case int64:
		return val
	case float64:
		return int64(val)
	case int:
		return int64(val)
	}
	return 0
}

func getFloat64(props map[string]interface{}, key string) float64 {
	v, ok := props[key]
	if !ok {
		return 0
	}
	switch val := v.(type) {
	case float64:
		return val
	case int64:
		return float64(val)
	case int:
		return float64(val)
	}
	return 0
}
