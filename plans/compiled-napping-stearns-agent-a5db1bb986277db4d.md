# P0 Gap Implementation Plan

## 1. Smart Dedup / Conflict Check on CreateMemory

### Entry point
`internal/memory/service.go`, function `CreateMemory`, line 496:
```go
func (s *Service) CreateMemory(ctx context.Context, mem *types.Memory) (*types.Memory, error)
```

### Current write flow (in order)
| Line | What happens |
|------|-------------|
| 496–508 | ID, tenant, timestamps, version, status, validity defaults set |
| 519–529 | RawSegment, GraphLayer, SourceType, SourceAuthority set |
| 535–542 | VolatilityScore + PoolType computed |
| 545–555 | LLM ProcessContent — extracts facts/entities/importance |
| 558–572 | Dimension extraction (DimMem keyword heuristic) |
| 575–580 | Safety classifier — returns early on unsafe content |
| 583–587 | Quota check |
| 590 | `s.routeMemoryTier(ctx, mem)` |
| **593–595** | **`s.graph.CreateMemory(mem)` — first write to Neo4j** |
| 596–600 | `s.materializeMemoryEntities` then `s.graph.UpdateMemory` |
| 603–623 | Embedding generated via `s.embedder.GenerateEmbeddingWithContext` |
| **613–621** | **`s.vector.StoreEmbedding(ctx, …)` — write to vector store** |

### Where the dedup/conflict check should be inserted
**After** the LLM processing block (line 555) and **before** the quota check (line 583). Specifically, insert between lines 572 and 575 (after dimension extraction, before safety check). This is the earliest point where the embedding can be generated for similarity search without polluting the hot write path, and before any quota or storage side-effects occur.

Pseudo-code sketch:
```go
// ~line 573, after dimension extraction block
if s.embedder != nil && s.vector != nil && s.processor != nil {
    emb, err := s.embedder.GenerateEmbeddingWithContext(ctx, mem.Content)
    if err == nil && len(emb) > 0 {
        filterMap := map[string]interface{}{}
        if mem.UserID != "" { filterMap["user_id"] = mem.UserID }
        if mem.OrgID  != "" { filterMap["org_id"]  = mem.OrgID  }
        similar, err := s.vector.Search(ctx, emb, 5, 0.90, filterMap)  // threshold TBD
        if err == nil {
            for _, candidate := range similar {
                existing, err := s.graph.GetMemory(candidate.MemoryID)
                if err != nil || existing == nil { continue }
                resolution, err := s.processor.ResolveConflict(
                    ctx,
                    existing.Content,
                    string(existing.Importance),
                    mem.Content,
                )
                if err != nil { continue }
                switch resolution.Action {
                case ConflictActionDiscardNew:
                    return existing, nil  // or return nil, ErrDuplicate — TBD
                case ConflictActionUpdate:
                    existing.Content = resolution.UpdatedContent
                    existing.UpdatedAt = time.Now()
                    existing.Version++
                    _ = s.graph.UpdateMemory(existing)
                    return existing, nil
                // ConflictActionKeepBoth: fall through to normal create
                }
            }
        }
    }
}
```

### ResolveConflict signature (processor.go, line 287)
```go
func (p *MemoryProcessor) ResolveConflict(
    ctx context.Context,
    existingContent string,
    existingImportance string,
    newContent string,
) (*ConflictResolutionResult, error)
```
Returns `*ConflictResolutionResult` (defined at templates.go line 308):
```go
type ConflictResolutionResult struct {
    Action         ConflictResolutionAction `json:"action"`          // "update" | "keep_both" | "discard_new"
    UpdatedContent string                   `json:"updated_content,omitempty"`
    Reason         string                   `json:"reason"`
}
```
Constants (templates.go lines 301–305):
```go
ConflictActionUpdate     ConflictResolutionAction = "update"
ConflictActionKeepBoth   ConflictResolutionAction = "keep_both"
ConflictActionDiscardNew ConflictResolutionAction = "discard_new"
```

### RenderConflictValidity (templates.go)
A richer alternative to `ResolveConflict` is the package-level helper at **line 593**:
```go
func RenderConflictValidity(existingContent, newContent string) (string, string, error)
```
Returns `(systemPrompt, userPrompt, error)`. This is the template that tracks `old_validity`/`new_validity` fields (`ConflictValidityResult`, line 532). It is used by calling `p.llmProvider.Complete` directly after rendering. Consider using this instead of `ResolveConflict` to also set `mem.ValidityStatus` and the existing memory's `ValidityStatus` on update.

### vector.Search call pattern (service.go, line 813)
```go
vectorResults, err := s.vector.Search(ctx, emb, limit*2, 0.0, filterMap)
```
`filterMap` is `map[string]interface{}` built at lines 805–812 with `"org_id"` and `"user_id"` keys. For the dedup check, use a higher similarity threshold (e.g. `0.85`–`0.92`) instead of `0.0`.

---

## 2. MCP Server Protocol Compliance

### Current structure (`cmd/mcp-server/main.go`)

- **Transport**: HTTP only, listening on `:8082` (flag `--port`). No stdin/stdout, no SSE.
- **Server type**: `MCPServer` struct wrapping `*http.Server` with a `mux.ServeMux` (lines 26–85).
- **Tool registration**: Hard-coded `mux.HandleFunc` calls per-tool at lines 36–69. Tool list served by `handleMCP` at lines 155–186 as a plain JSON array.
- **Request dispatch**: `handleMCP` (lines 142–233) reads a JSON body, extracts `method` field, calls `routeByMethod` (lines 235–293). No JSON-RPC 2.0 envelope — no `jsonrpc`, `id`, `params` fields.

### What needs to change for JSON-RPC 2.0 + SSE compliance

**Current request format** (informal):
```json
{"method": "addMemory", "params": {...}}
```

**Required JSON-RPC 2.0 format**:
```json
{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": {"name": "addMemory", "arguments": {...}}}
```

**Gap analysis:**

| Area | Current state | Required |
|------|-------------|----------|
| Request envelope | `{"method": "...", "params": {...}}` | `{"jsonrpc":"2.0","id":N,"method":"...","params":{...}}` |
| Response envelope | Raw tool output or HTTP error | `{"jsonrpc":"2.0","id":N,"result":{...}}` or `{"jsonrpc":"2.0","id":N,"error":{"code":N,"message":"..."}}` |
| Initialize handshake | Missing | `initialize` method returning server capabilities + `initialized` notification |
| Tool listing | `GET /mcp` returns `{"tools":[...]}` | `tools/list` JSON-RPC method returning tools with input schemas |
| Tool call | `POST /mcp` with `method` field | `tools/call` JSON-RPC method with `name` + `arguments` |
| SSE transport | Not present | `GET /sse` endpoint emitting `data: {...}\n\n`, `POST /message` for client→server |
| Error codes | HTTP status codes | JSON-RPC codes: -32700 parse error, -32600 invalid request, -32601 method not found, -32602 invalid params, -32603 internal error |

**Minimal changes to `handleMCP`** (single endpoint path):

1. Parse `{"jsonrpc":"2.0","id":...,"method":"...","params":{...}}` in `handleMCP`.
2. Add `initialize` method handler returning:
   ```json
   {"protocolVersion":"2024-11-05","capabilities":{"tools":{}},"serverInfo":{"name":"hystersis-mcp","version":"1.0.0"}}
   ```
3. Map `tools/list` → return tool array with `inputSchema` (JSON Schema) per tool.
4. Map `tools/call` → dispatch via `routeByMethod` using `params.name` and `params.arguments`.
5. Wrap all responses in `{"jsonrpc":"2.0","id":N,"result":{...}}`.
6. Wrap errors as `{"jsonrpc":"2.0","id":N,"error":{"code":-32603,"message":"..."}}`.
7. Add `GET /sse` handler that upgrades to SSE (`Content-Type: text/event-stream`) and streams results; add `POST /message` that posts JSON-RPC requests to be dispatched through the SSE channel.

**Key structural change**: `routeByMethod` (line 235) can be kept as the internal dispatch table — it just needs to be called from the JSON-RPC layer instead of directly from `handleMCP`.

---

## 3. Prometheus Metrics

### Is `/metrics` already registered?
**Yes — fully implemented.** `cmd/server/api.go` line **590**:
```go
s.router.Handle("/metrics", promhttp.Handler()).Methods("GET")
```
This is in `registerRoutes()`, called from `NewAPIServer` at line 582.

### Is `prometheus/client_golang` in go.mod?
**Yes.** `go.mod` line **17**:
```
github.com/prometheus/client_golang v1.23.2
```
It is a direct dependency.

### Existing Prometheus metric definitions (api.go lines 184–221)
```go
httpRequestsTotal = promauto.NewCounterVec(
    prometheus.CounterOpts{Name: "agent_memory_http_requests_total", ...},
    []string{"method", "endpoint", "status"},
)
httpRequestDuration = promauto.NewHistogramVec(
    prometheus.HistogramOpts{Name: "agent_memory_http_request_duration_seconds", ...},
    []string{"method", "endpoint"},
)
benchmarkScore        = promauto.NewGaugeVec(...)
benchmarkLatency      = promauto.NewHistogramVec(...)
benchmarkTokensRetrieved = promauto.NewGauge(...)
```

### Metrics middleware (api.go lines 1041–1051)
```go
func metricsMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
        next.ServeHTTP(rw, r)
        duration := time.Since(start).Seconds()
        httpRequestsTotal.WithLabelValues(r.Method, r.URL.Path, fmt.Sprintf("%d", rw.statusCode)).Inc()
        httpRequestDuration.WithLabelValues(r.Method, r.URL.Path).Observe(duration)
    })
}
```
Applied globally at `NewAPIServer` line 284: `router.Use(metricsMiddleware)`.

### `internal/metrics/` package
Custom in-memory collector (`metrics.go`): `MetricsCollector` struct tracking compression-specific metrics (extractions, spreading activation hops, cache hits, tier hits, P95 latency). This is **not** Prometheus-native — it's a manual counter store. It is wired into `APIServer` at line 239 (`metricsCollector *metrics.MetricsCollector`) and exposed at `/metrics/compression` (line 747, behind `requireScope("read")`).

### What is actually missing for Prometheus
The `/metrics` endpoint and middleware are **already there**. What may be a gap:

1. **Business-level counters** — `CreateMemory`, `SearchMemories`, dedup hits, conflict resolutions, etc. are not instrumented. These would be new `promauto.NewCounterVec` definitions alongside the existing ones at lines 184–221.
2. **Compression metrics bridge** — `MetricsCollector.GetSnapshot()` data is not fed into Prometheus gauges. A bridging `prometheus.Collector` implementing `Describe`/`Collect` over the `MetricsCollector` snapshot would expose these to `/metrics`.
3. **`/metrics` is unauthenticated** — consistent with Prometheus scrape conventions, but worth noting. The bypass list confirms this at line 1208.

### Where to add new metrics
- New `promauto` declarations: alongside lines 184–221 in `api.go`.
- Increment calls: inside `CreateMemory` (service.go) and `SearchMemories` (service.go). Either pass a counter reference into the service or use a package-level registration in a new file (e.g. `internal/metrics/memory_metrics.go`) and call from service methods.
