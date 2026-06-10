package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// MemoryCreatedTotal counts memories successfully created, by type and tenant.
	MemoryCreatedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "hystersis_memory_created_total",
		Help: "Total memories created",
	}, []string{"type", "tenant_id"})

	// MemoryDedupTotal counts dedup decisions made during create.
	// action: add, update, discard
	MemoryDedupTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "hystersis_memory_dedup_total",
		Help: "Total dedup decisions during memory creation",
	}, []string{"action"})

	// SearchTotal is a simple counter of all search operations.
	SearchTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "hystersis_search_total",
		Help: "Total search operations",
	})

	// SearchLatency tracks end-to-end search duration in seconds.
	SearchLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "hystersis_search_duration_seconds",
		Help:    "Search latency distribution",
		Buckets: []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5},
	})

	// SearchTokensReturned tracks the number of tokens returned per search
	// after token-budget enforcement.
	SearchTokensReturned = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "hystersis_search_tokens_returned",
		Help:    "Number of estimated tokens returned per search after budget enforcement",
		Buckets: []float64{500, 1000, 2000, 5000, 7000, 10000, 20000},
	})

	// CacheHitTotal counts cache hits in the tier router.
	CacheHitTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "hystersis_cache_hit_total",
		Help: "Total cache hits",
	})

	// CacheMissTotal counts cache misses in the tier router.
	CacheMissTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "hystersis_cache_miss_total",
		Help: "Total cache misses",
	})

	// ConflictResolutionTotal counts conflict-resolution actions during dedup.
	// action: keep_both, update, discard_new
	ConflictResolutionTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "hystersis_conflict_resolution_total",
		Help: "Total conflict resolutions by action",
	}, []string{"action"})

	// TierDistribution tracks current memory count by storage tier.
	TierDistribution = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "hystersis_memory_tier_count",
		Help: "Current memory count by tier",
	}, []string{"tier"})

	// CompressionTokensSaved counts the cumulative tokens saved by compression.
	CompressionTokensSaved = promauto.NewCounter(prometheus.CounterOpts{
		Name: "hystersis_compression_tokens_saved_total",
		Help: "Total tokens saved by compression",
	})
)
