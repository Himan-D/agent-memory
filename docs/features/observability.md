# Feature: Compression Observability

**Priority**: P1 — Compression pipeline runs silently; no metrics, no visibility into token savings or latency.
**Status**: `internal/metrics/` directory is empty. `GET /compression/stats` returns stub data.
**Estimated effort**: 1-2 days

---

## What to Build

Two files in `internal/metrics/`:

| File | Responsibility |
|------|---------------|
| `compression.go` | In-memory metrics collector with percentile support |
| `prometheus.go` | Prometheus metric registrations, wired into the collector |

Plus wiring into:
- `internal/compression/extractor/` — record job completion
- `internal/compression/retrieval/` — record spreading activation calls
- `cmd/server/api.go` — serve `/compression/stats` from real data

---

## Component 1: `compression.go`

```go
package metrics

import (
    "sync"
    "sort"
    "time"
)

// CompressionMetrics collects stats about the compression pipeline.
// Thread-safe. Designed for frequent writes (every compression job).
type CompressionMetrics struct {
    mu sync.RWMutex

    // Counters
    JobsTotal          int64
    JobsSucceeded      int64
    JobsFailed         int64
    SpreadingActivations int64
    SynonymHits        int64

    // Token savings (cumulative)
    TokensIn           int64
    TokensOut          int64

    // Latency samples (ms) — kept as ring buffer, capped at maxSamples
    latencySamples []float64
    maxSamples     int

    // Per-mode counters: mode name → count
    modeCounters map[string]int64

    startTime time.Time
}

func NewCompressionMetrics() *CompressionMetrics {
    return &CompressionMetrics{
        maxSamples:   1000,
        modeCounters: make(map[string]int64),
        startTime:    time.Now(),
    }
}

// RecordJob records a completed compression job.
// tokensIn/tokensOut are the token counts before and after compression.
// mode is "aggressive" | "balanced" | "conservative".
// latencyMs is end-to-end job duration in milliseconds.
func (m *CompressionMetrics) RecordJob(tokensIn, tokensOut int64, mode string, latencyMs float64, success bool) {
    m.mu.Lock()
    defer m.mu.Unlock()

    m.JobsTotal++
    if success {
        m.JobsSucceeded++
    } else {
        m.JobsFailed++
    }
    m.TokensIn += tokensIn
    m.TokensOut += tokensOut
    m.modeCounters[mode]++

    // Append latency, evict oldest if at cap
    m.latencySamples = append(m.latencySamples, latencyMs)
    if len(m.latencySamples) > m.maxSamples {
        m.latencySamples = m.latencySamples[len(m.latencySamples)-m.maxSamples:]
    }
}

// RecordSpreadingActivation records a spreading activation retrieval call.
func (m *CompressionMetrics) RecordSpreadingActivation() {
    m.mu.Lock()
    m.SpreadingActivations++
    m.mu.Unlock()
}

// RecordSynonymHit records a synonym expansion hit.
func (m *CompressionMetrics) RecordSynonymHit() {
    m.mu.Lock()
    m.SynonymHits++
    m.mu.Unlock()
}

// Get returns a snapshot of all metrics.
func (m *CompressionMetrics) Get() CompressionStats {
    m.mu.RLock()
    defer m.mu.RUnlock()

    stats := CompressionStats{
        JobsTotal:            m.JobsTotal,
        JobsSucceeded:        m.JobsSucceeded,
        JobsFailed:           m.JobsFailed,
        SpreadingActivations: m.SpreadingActivations,
        SynonymHits:          m.SynonymHits,
        TokensIn:             m.TokensIn,
        TokensOut:            m.TokensOut,
        UptimeSeconds:        int64(time.Since(m.startTime).Seconds()),
        ModeCounters:         make(map[string]int64, len(m.modeCounters)),
    }

    for k, v := range m.modeCounters {
        stats.ModeCounters[k] = v
    }

    if m.TokensIn > 0 {
        stats.CompressionRatio = 1.0 - float64(m.TokensOut)/float64(m.TokensIn)
    }

    if len(m.latencySamples) > 0 {
        sorted := make([]float64, len(m.latencySamples))
        copy(sorted, m.latencySamples)
        sort.Float64s(sorted)

        stats.LatencyP50Ms = percentile(sorted, 50)
        stats.LatencyP95Ms = percentile(sorted, 95)
        stats.LatencyP99Ms = percentile(sorted, 99)
        stats.LatencyAvgMs = average(sorted)
    }

    return stats
}

// CompressionStats is the serializable snapshot returned by GET /compression/stats.
type CompressionStats struct {
    JobsTotal            int64              `json:"jobs_total"`
    JobsSucceeded        int64              `json:"jobs_succeeded"`
    JobsFailed           int64              `json:"jobs_failed"`
    SpreadingActivations int64              `json:"spreading_activations"`
    SynonymHits          int64              `json:"synonym_hits"`
    TokensIn             int64              `json:"tokens_in"`
    TokensOut            int64              `json:"tokens_out"`
    CompressionRatio     float64            `json:"compression_ratio"` // 0.0–1.0 (fraction saved)
    LatencyP50Ms         float64            `json:"latency_p50_ms"`
    LatencyP95Ms         float64            `json:"latency_p95_ms"`
    LatencyP99Ms         float64            `json:"latency_p99_ms"`
    LatencyAvgMs         float64            `json:"latency_avg_ms"`
    UptimeSeconds        int64              `json:"uptime_seconds"`
    ModeCounters         map[string]int64   `json:"mode_counters"` // mode → job count
}

func percentile(sorted []float64, p float64) float64 {
    if len(sorted) == 0 {
        return 0
    }
    idx := int(float64(len(sorted)-1) * p / 100)
    return sorted[idx]
}

func average(vals []float64) float64 {
    if len(vals) == 0 {
        return 0
    }
    sum := 0.0
    for _, v := range vals {
        sum += v
    }
    return sum / float64(len(vals))
}
```

---

## Component 2: `prometheus.go`

```go
package metrics

import "github.com/prometheus/client_golang/prometheus"

// PrometheusCollector bridges CompressionMetrics to Prometheus.
// Register once at startup, then call Collect() to push current values.
type PrometheusCollector struct {
    source *CompressionMetrics

    // Counters
    jobsTotal            *prometheus.CounterVec
    tokensProcessedTotal *prometheus.CounterVec
    spreadingTotal       prometheus.Counter
    synonymHitsTotal     prometheus.Counter

    // Gauges (set from snapshot)
    compressionRatio prometheus.Gauge
    latencyP50       prometheus.Gauge
    latencyP95       prometheus.Gauge
    latencyP99       prometheus.Gauge
}

func NewPrometheusCollector(source *CompressionMetrics, reg prometheus.Registerer) *PrometheusCollector {
    c := &PrometheusCollector{
        source: source,

        jobsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
            Name: "hystersis_compression_jobs_total",
            Help: "Total compression jobs processed, labeled by mode and result.",
        }, []string{"mode", "result"}),

        tokensProcessedTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
            Name: "hystersis_compression_tokens_total",
            Help: "Total tokens processed by compression (in/out).",
        }, []string{"direction"}),

        spreadingTotal: prometheus.NewCounter(prometheus.CounterOpts{
            Name: "hystersis_spreading_activations_total",
            Help: "Total spreading activation retrieval calls.",
        }),

        synonymHitsTotal: prometheus.NewCounter(prometheus.CounterOpts{
            Name: "hystersis_synonym_hits_total",
            Help: "Total synonym expansion hits.",
        }),

        compressionRatio: prometheus.NewGauge(prometheus.GaugeOpts{
            Name: "hystersis_compression_ratio",
            Help: "Current token compression ratio (1.0 = 100% saved).",
        }),

        latencyP50: prometheus.NewGauge(prometheus.GaugeOpts{
            Name: "hystersis_compression_latency_p50_ms",
            Help: "P50 compression job latency in milliseconds.",
        }),

        latencyP95: prometheus.NewGauge(prometheus.GaugeOpts{
            Name: "hystersis_compression_latency_p95_ms",
            Help: "P95 compression job latency in milliseconds.",
        }),

        latencyP99: prometheus.NewGauge(prometheus.GaugeOpts{
            Name: "hystersis_compression_latency_p99_ms",
            Help: "P99 compression job latency in milliseconds.",
        }),
    }

    reg.MustRegister(
        c.jobsTotal,
        c.tokensProcessedTotal,
        c.spreadingTotal,
        c.synonymHitsTotal,
        c.compressionRatio,
        c.latencyP50,
        c.latencyP95,
        c.latencyP99,
    )

    return c
}

// Sync pushes the latest snapshot values to Prometheus gauges.
// Call periodically (e.g., every 15s via a goroutine) or on each scrape.
func (c *PrometheusCollector) Sync() {
    stats := c.source.Get()

    c.compressionRatio.Set(stats.CompressionRatio)
    c.latencyP50.Set(stats.LatencyP50Ms)
    c.latencyP95.Set(stats.LatencyP95Ms)
    c.latencyP99.Set(stats.LatencyP99Ms)
}
```

---

## Integration: Wire into Compression Pipeline

### 1. In `internal/compression/extractor/` (ProMem pipeline)

Find the main extraction function (likely `Extract` or `Compress`). Add timing and metric recording:

```go
func (e *Extractor) Extract(ctx context.Context, content string, mode string) (*Result, error) {
    startTime := time.Now()
    tokensIn := int64(estimateTokens(content))
    
    result, err := e.runExtraction(ctx, content, mode)
    
    latencyMs := float64(time.Since(startTime).Milliseconds())
    tokensOut := int64(0)
    if err == nil && result != nil {
        tokensOut = int64(estimateTokens(result.Compressed))
    }
    
    if e.metrics != nil {
        e.metrics.RecordJob(tokensIn, tokensOut, mode, latencyMs, err == nil)
    }
    
    return result, err
}
```

Add `metrics *metrics.CompressionMetrics` field to the `Extractor` struct. Wire it from `cmd/server/api.go` during startup.

### 2. In `internal/compression/retrieval/` (Spreading Activation)

Find the spreading activation search function. Add:

```go
func (s *SpreadingActivation) Search(ctx context.Context, query string, userID string) ([]Result, error) {
    if s.metrics != nil {
        s.metrics.RecordSpreadingActivation()
    }
    return s.doSearch(ctx, query, userID)
}
```

### 3. In `cmd/server/api.go` — fix `handleGetCompressionStats`

Replace the stub with real data:

```go
func (a *API) handleGetCompressionStats(w http.ResponseWriter, r *http.Request) {
    if a.compressionMetrics == nil {
        jsonOK(w, metrics.CompressionStats{})
        return
    }
    jsonOK(w, a.compressionMetrics.Get())
}
```

Add `compressionMetrics *metrics.CompressionMetrics` to the `API` struct.

Wire at startup in `main.go` or wherever the API is initialized:

```go
compressionMetrics := metrics.NewCompressionMetrics()

// Pass to extractor
extractor.SetMetrics(compressionMetrics)

// Pass to spreading activation retriever
spreadingActivation.SetMetrics(compressionMetrics)

// Pass to API
api := NewAPI(config, ..., compressionMetrics)

// Start Prometheus sync goroutine
if config.PrometheusEnabled {
    collector := metrics.NewPrometheusCollector(compressionMetrics, prometheus.DefaultRegisterer)
    go func() {
        ticker := time.NewTicker(15 * time.Second)
        for range ticker.C {
            collector.Sync()
        }
    }()
}
```

### 4. Prometheus HTTP endpoint

Add to `cmd/server/api.go` router:

```go
import "github.com/prometheus/client_golang/prometheus/promhttp"

// In route registration:
r.Handle("/metrics", promhttp.Handler())
```

This serves the standard Prometheus scrape endpoint at `GET /metrics`.

---

## Config Additions

Add to `internal/config/config.go`:

```go
type MetricsConfig struct {
    PrometheusEnabled   bool `env:"PROMETHEUS_ENABLED" default:"false"`
    LatencySampleCap    int  `env:"METRICS_LATENCY_SAMPLE_CAP" default:"1000"`
    SyncIntervalSeconds int  `env:"METRICS_PROMETHEUS_SYNC_INTERVAL" default:"15"`
}
```

---

## Environment Variables

```bash
PROMETHEUS_ENABLED=false                 # Expose /metrics endpoint
METRICS_LATENCY_SAMPLE_CAP=1000         # Max latency samples kept in memory
METRICS_PROMETHEUS_SYNC_INTERVAL=15     # Seconds between gauge syncs
```

---

## Tests

Create `internal/metrics/compression_test.go`:

```go
func TestRecordJob_UpdatesCounters(t *testing.T) {
    m := NewCompressionMetrics()
    m.RecordJob(1000, 400, "balanced", 50.0, true)
    
    stats := m.Get()
    assert.Equal(t, int64(1), stats.JobsTotal)
    assert.Equal(t, int64(1), stats.JobsSucceeded)
    assert.Equal(t, int64(0), stats.JobsFailed)
    assert.Equal(t, int64(1000), stats.TokensIn)
    assert.Equal(t, int64(400), stats.TokensOut)
    assert.InDelta(t, 0.60, stats.CompressionRatio, 0.001) // 1 - 400/1000
}

func TestRecordJob_FailureTracked(t *testing.T) {
    m := NewCompressionMetrics()
    m.RecordJob(500, 0, "aggressive", 10.0, false)
    
    stats := m.Get()
    assert.Equal(t, int64(1), stats.JobsTotal)
    assert.Equal(t, int64(0), stats.JobsSucceeded)
    assert.Equal(t, int64(1), stats.JobsFailed)
}

func TestLatencyPercentiles(t *testing.T) {
    m := NewCompressionMetrics()
    // Record 100 jobs with latencies 1..100ms
    for i := 1; i <= 100; i++ {
        m.RecordJob(100, 50, "balanced", float64(i), true)
    }
    
    stats := m.Get()
    assert.InDelta(t, 50.0, stats.LatencyP50Ms, 1.0)
    assert.InDelta(t, 95.0, stats.LatencyP95Ms, 1.0)
    assert.InDelta(t, 99.0, stats.LatencyP99Ms, 1.0)
}

func TestLatencySampleCap(t *testing.T) {
    m := NewCompressionMetrics()
    m.maxSamples = 100
    
    // Record 200 jobs
    for i := 0; i < 200; i++ {
        m.RecordJob(100, 50, "balanced", float64(i), true)
    }
    
    // Samples should be capped at 100
    m.mu.RLock()
    defer m.mu.RUnlock()
    assert.Len(t, m.latencySamples, 100)
}

func TestModeCounters(t *testing.T) {
    m := NewCompressionMetrics()
    m.RecordJob(100, 50, "aggressive", 10, true)
    m.RecordJob(100, 50, "aggressive", 10, true)
    m.RecordJob(100, 50, "balanced", 10, true)
    
    stats := m.Get()
    assert.Equal(t, int64(2), stats.ModeCounters["aggressive"])
    assert.Equal(t, int64(1), stats.ModeCounters["balanced"])
}

func TestCompressionRatio_ZeroTokensIn(t *testing.T) {
    // Guard against division by zero
    m := NewCompressionMetrics()
    stats := m.Get()
    assert.Equal(t, 0.0, stats.CompressionRatio)
}

func TestSpreadingActivationCounter(t *testing.T) {
    m := NewCompressionMetrics()
    m.RecordSpreadingActivation()
    m.RecordSpreadingActivation()
    
    stats := m.Get()
    assert.Equal(t, int64(2), stats.SpreadingActivations)
}

func TestConcurrentWrites(t *testing.T) {
    // Verify thread safety under concurrent RecordJob calls
    m := NewCompressionMetrics()
    var wg sync.WaitGroup
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            m.RecordJob(100, 50, "balanced", 10.0, true)
        }()
    }
    wg.Wait()
    
    stats := m.Get()
    assert.Equal(t, int64(100), stats.JobsTotal)
}
```

---

## API Response: `GET /compression/stats`

After wiring, the endpoint returns:

```json
{
  "jobs_total": 1842,
  "jobs_succeeded": 1831,
  "jobs_failed": 11,
  "spreading_activations": 4201,
  "synonym_hits": 93,
  "tokens_in": 924100,
  "tokens_out": 311200,
  "compression_ratio": 0.663,
  "latency_p50_ms": 42.1,
  "latency_p95_ms": 187.4,
  "latency_p99_ms": 312.0,
  "latency_avg_ms": 58.3,
  "uptime_seconds": 86400,
  "mode_counters": {
    "aggressive": 412,
    "balanced": 1209,
    "conservative": 221
  }
}
```
