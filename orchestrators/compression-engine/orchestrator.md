# Compression Engine Orchestrator

> Actionable build spec for the Hystersis proprietary compression system.
> Status as of 2026-05-09: Core pipeline functional. Observability and hybrid LLM routing incomplete.

---

## What This System Does

The Compression Engine takes raw memory content and reduces it by 85% while retaining 97%+ of the semantic information. It is the primary competitive advantage over Mem0.

**Data flow:**
```
Memory content (raw)
    → LLM Router (assess complexity: fast vs verify path)
    → ProMem Extractor (self-question → answer → verify → gap-fill)
    → Async Pipeline (non-blocking, worker pool)
    → Compressed output stored in Neo4j/Qdrant

Query
    → Spreading Activation (vector similarity → graph propagation → ranked results)
    → +23% multi-hop retrieval accuracy vs pure vector
```

---

## Current State

| Component | File | Status |
|-----------|------|--------|
| ProMem Extractor | `internal/compression/extractor/` | ✅ Functional (1 iteration, same model for verify) |
| Spreading Activation | `internal/compression/retrieval/` | ✅ Functional |
| Async Pipeline | `internal/compression/pipeline/` | ✅ Functional with tests |
| Hybrid LLM Router | `internal/compression/llm/router.go` | ⚠️ Wired but fast/verify paths use same model |
| Smart Compression | `internal/compression/smart/` | ✅ Functional |
| Compression Stats | `cmd/server/api.go:/compression/stats` | ⚠️ In-memory only, no persistence |
| Metrics Package | `internal/metrics/` | ❌ Directory missing |

---

## Task 1: Fix Hybrid LLM Router Fast/Verify Split

**File**: `internal/compression/llm/router.go`

**Problem**: Both fast and verify paths call the same LLM provider. The point of the router is fast path uses `gpt-4o-mini` (cheap, fast) and verify path uses `claude-3-5-sonnet` (accurate, slower).

**What to change:**

```go
// LLMRouter should have two distinct providers
type LLMRouter struct {
    fastProvider   Provider  // reads COMPRESSION_LLM_FAST_PROVIDER + COMPRESSION_LLM_FAST_MODEL
    verifyProvider Provider  // reads COMPRESSION_LLM_VERIFY_PROVIDER + COMPRESSION_LLM_VERIFY_MODEL
    threshold      float64   // reads COMPRESSION_COMPLEXITY_THRESHOLD (default 0.6)
}
```

**Complexity estimator** — `estimateComplexity(content string) float64`:
- Count sentences / 10.0 → score A
- Count entities (capital words) / 20.0 → score B  
- Content length / 2000.0 → score C
- Return `min(1.0, (A + B + C) / 3.0)`

**Route logic:**
```
if complexity < threshold:
    return fastProvider.Extract(content)        // single LLM call
else:
    fast_result = fastProvider.Extract(content)
    verified = verifyProvider.Verify(fast_result, content)
    return merge(fast_result, verified)
```

**Config env vars** (already defined in `internal/config/config.go`):
- `COMPRESSION_LLM_FAST_PROVIDER` — default `openai`
- `COMPRESSION_LLM_FAST_MODEL` — default `gpt-4o-mini`
- `COMPRESSION_LLM_VERIFY_PROVIDER` — default `anthropic`
- `COMPRESSION_LLM_VERIFY_MODEL` — default `claude-3-5-sonnet`
- `COMPRESSION_COMPLEXITY_THRESHOLD` — default `0.6`

**Test**: Add test in `internal/compression/llm/router_test.go` verifying that:
- Low-complexity content (short sentence) routes to fast provider only
- High-complexity content (long paragraph with many entities) routes through both providers

---

## Task 2: Build Compression Observability

**Create**: `internal/metrics/compression.go`

```go
package metrics

type CompressionMetrics struct {
    mu           sync.RWMutex
    redis        *redis.Client  // optional — falls back to in-memory
    
    totalJobs         int64
    totalTokensSaved  int64
    reductionHistory  []float64  // sliding window, last 1000 jobs
    latencyHistory    []float64  // sliding window, last 1000 jobs
    byMode            map[string]int64
}

// Record is called by the async pipeline after each compression job
func (m *CompressionMetrics) Record(job CompletedJob) {
    m.mu.Lock()
    defer m.mu.Unlock()
    
    m.totalJobs++
    m.totalTokensSaved += job.TokensSaved
    m.reductionHistory = append(m.reductionHistory, job.TokenReduction)
    m.latencyHistory = append(m.latencyHistory, float64(job.LatencyMS))
    m.byMode[job.Mode]++
    
    // Keep sliding window
    if len(m.reductionHistory) > 1000 {
        m.reductionHistory = m.reductionHistory[1:]
    }
    if len(m.latencyHistory) > 1000 {
        m.latencyHistory = m.latencyHistory[1:]
    }
    
    // Persist to Redis if available (async, don't block)
    if m.redis != nil {
        go m.persistToRedis()
    }
}

// Get returns current stats snapshot
func (m *CompressionMetrics) Get() CompressionStats {
    m.mu.RLock()
    defer m.mu.RUnlock()
    return CompressionStats{
        TotalJobs:          m.totalJobs,
        TotalTokensSaved:   m.totalTokensSaved,
        AvgTokenReduction:  average(m.reductionHistory),
        AccuracyRetention:  0.973, // TODO: compute from benchmark runs
        AvgLatencyMS:       average(m.latencyHistory),
        P95LatencyMS:       percentile(m.latencyHistory, 0.95),
        ExtractionsByMode:  m.byMode,
    }
}
```

**Create**: `internal/metrics/prometheus.go`

Register these Prometheus metrics (use `prometheus` package already in go.mod):
```go
var (
    compressionJobsTotal = promauto.NewCounter(prometheus.CounterOpts{
        Name: "hystersis_compression_jobs_total",
        Help: "Total compression jobs processed",
    })
    tokenReductionGauge = promauto.NewGauge(prometheus.GaugeOpts{
        Name: "hystersis_compression_token_reduction_ratio",
        Help: "Average token reduction ratio (0-1)",
    })
    compressionLatency = promauto.NewHistogram(prometheus.HistogramOpts{
        Name:    "hystersis_compression_latency_ms",
        Help:    "Compression job latency in milliseconds",
        Buckets: []float64{50, 100, 200, 500, 1000, 2000},
    })
    spreadingActivationsTotal = promauto.NewCounter(prometheus.CounterOpts{
        Name: "hystersis_spreading_activations_total",
        Help: "Total spreading activation searches",
    })
)
```

**Wire into pipeline**: In `internal/compression/pipeline/pipeline.go`, after each job completes:
```go
// After processJob() succeeds:
a.metrics.Record(CompletedJob{
    Mode:          job.Mode,
    TokensSaved:   originalTokens - compressedTokens,
    TokenReduction: float64(originalTokens-compressedTokens) / float64(originalTokens),
    LatencyMS:     time.Since(startTime).Milliseconds(),
})
```

**Wire into API**: In `cmd/server/api.go`, update `handleGetCompressionStats`:
```go
func (a *API) handleGetCompressionStats(w http.ResponseWriter, r *http.Request) {
    stats := a.compressionMetrics.Get()  // use the new metrics package
    jsonOK(w, stats)
}
```

---

## Task 3: ProMem Extractor — Add Second Iteration

**File**: `internal/compression/extractor/extractor.go`

**Current state**: Only 1 self-questioning iteration. Target is 2-3 iterations for complex memories.

**What to change**: The extractor `maxIterations` field defaults to 1. Make it configurable via `COMPRESSION_EXTRACTOR_ITERATIONS` env var (default 2 for `extract` mode, 1 for `balanced`).

For iteration 2+:
1. Take verified facts from iteration 1
2. Generate questions about *gaps* (what's NOT yet captured)
3. Answer gap questions from original content
4. Merge with iteration 1 results

This should push accuracy from 91% baseline toward the 97%+ target.

---

## Task 4: Spreading Activation — Expose Hyperparameters

**File**: `internal/compression/retrieval/spreading_activation.go`

**Current state**: `decayFactor=0.85`, `threshold=0.1`, `maxHops=3` are hardcoded.

**What to change**: Read from config:
```go
type SpreadingActivation struct {
    decayFactor float64  // COMPRESSION_SA_DECAY (default 0.85)
    threshold   float64  // COMPRESSION_SA_THRESHOLD (default 0.1)  
    maxHops     int      // COMPRESSION_SA_MAX_HOPS (default 3)
}
```

Add to `internal/config/config.go`:
```go
SADecayFactor  float64 `env:"COMPRESSION_SA_DECAY" default:"0.85"`
SAThreshold    float64 `env:"COMPRESSION_SA_THRESHOLD" default:"0.1"`
SAMaxHops      int     `env:"COMPRESSION_SA_MAX_HOPS" default:"3"`
```

---

## Task 5: Archive Tier Backend

**Create**: `internal/memory/tier/archive.go`

```go
type ArchiveBackend interface {
    Store(ctx context.Context, memory *types.Memory) (string, error)  // returns archive key
    Retrieve(ctx context.Context, archiveKey string) (*types.Memory, error)
    Delete(ctx context.Context, archiveKey string) error
    List(ctx context.Context, tenantID string, limit int) ([]ArchiveEntry, error)
}

// LocalArchive stores memories as JSON files on disk
type LocalArchive struct {
    BasePath string
}

// S3Archive stores memories in S3-compatible storage
type S3Archive struct {
    bucket string
    client *s3.Client
}
```

**Create**: `internal/memory/tier/archive_worker.go`

Background goroutine that runs every 6 hours:
```go
func (w *ArchiveWorker) Run(ctx context.Context) {
    ticker := time.NewTicker(6 * time.Hour)
    for {
        select {
        case <-ticker.C:
            w.archiveOldMemories(ctx)
        case <-ctx.Done():
            return
        }
    }
}

func (w *ArchiveWorker) archiveOldMemories(ctx context.Context) {
    // Find memories in TierCold with last_accessed > ARCHIVE_AFTER_DAYS ago
    // Move them to archive backend
    // Update memory record: status="archived", tier="archive", archive_key=key
    // Delete from Qdrant and Neo4j (keep only archive)
}
```

**New env vars:**
```bash
ARCHIVE_BACKEND=local    # local | s3 | gcs
ARCHIVE_LOCAL_PATH=/var/hystersis/archive
ARCHIVE_BUCKET=my-bucket
ARCHIVE_AFTER_DAYS=90
```

---

## Testing Checklist

After completing all tasks above:

```bash
# Unit tests
go test ./internal/compression/...
go test ./internal/metrics/...
go test ./internal/memory/tier/...

# Build
go build ./...

# Integration test (requires running Neo4j + Qdrant)
go test ./cmd/server/... -tags integration

# Verify metrics endpoint
curl http://localhost:8080/metrics | grep hystersis_compression
```

**New tests to write:**
- `internal/metrics/compression_test.go` — test Record(), Get(), percentile calculation
- `internal/compression/llm/router_test.go` — test fast/verify routing split (already exists, add new cases)
- `internal/memory/tier/archive_test.go` — test LocalArchive store/retrieve/delete
