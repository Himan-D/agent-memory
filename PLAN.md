# Hystersis — Master Implementation Plan

> This is the single source of truth for what needs to be built. Every feature maps to exact files, interfaces, and test requirements. An AI agent reading this document has everything needed to implement any feature without additional context.

---

## Codebase Snapshot

| Metric | Value |
|--------|-------|
| Go packages | 40 (internal/) |
| Total lines | 38,000+ |
| API routes | 104 |
| Test files | 19 (5,269 lines) |
| Implemented | ~95% |
| Needs work | See below |

---

## Feature Registry

Each feature has a **priority**, **status**, **owner orchestrator**, and links to its detailed spec.

| # | Feature | Priority | Status | Spec | Owner |
|---|---------|----------|--------|------|-------|
| 1 | [RBAC Enforcement](#1-rbac-enforcement) | P0 | Defined, unenforced | `docs/features/rbac.md` | Memory Space |
| 2 | [Compression Observability](#2-compression-observability) | P0 | Endpoint wired, no persistence | `docs/features/observability.md` | Compression |
| 3 | [Mem0 v3 Parity](#3-mem0-v3-parity) | P1 | Analyzed, not implemented | `docs/features/mem0-v3-parity.md` | Memory Space |
| 4 | [Self-Improvement System](#4-self-improvement-system) | P1 | Empty dir | `docs/features/self-improvement.md` | Memory Space |
| 5 | [Memory Chunking](#5-memory-chunking) | P1 | Empty dir | `docs/features/chunking.md` | Memory Space |
| 6 | [Skill Audit Events](#6-skill-audit-events) | P1 | Missing emitters | `docs/features/skill-audit.md` | Skills |
| 7 | [Azure LLM Provider](#7-azure-llm-provider) | P2 | Embeddings/reranking stubbed | inline below | Compression |
| 8 | [OAuth Refresh Tokens](#8-oauth-refresh-tokens) | P2 | 501 stub | inline below | MCP |
| 9 | [Archive Tier Backend](#9-archive-tier-backend) | P2 | Tier routing done, no storage | inline below | Compression |
| 10 | [SkillSharingEnabled + AgentConfig Domains](#10-skills-group-policy) | P2 | Defined, never checked | inline below | Skills |

---

## 1. RBAC Enforcement

**Status**: `internal/roles/roles.go` exists (89 lines) but is never called. All endpoints are open to any authenticated key.

**Goal**: Gate write/admin endpoints by role. Read-only keys cannot create/delete. Admin keys can manage users and API keys.

**Files to touch:**
- `internal/roles/roles.go` — already has `Role`, `Permission`, `RolePolicy` types
- `cmd/server/api.go` — add `requireRole(role)` middleware chained on handler registration
- `internal/config/config.go` — add `DefaultRole` config

**Implementation steps:**

1. In `internal/roles/roles.go`, add a `CheckPermission(apiKey, endpoint, method) bool` function that looks up the key's role and returns true if allowed.

2. In `cmd/server/api.go`, create a middleware:
   ```go
   func (a *API) requireScope(scope string) func(http.Handler) http.Handler {
       return func(next http.Handler) http.Handler {
           return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
               key := r.Header.Get("X-API-Key")
               if !a.roles.HasScope(key, scope) {
                   safeHTTPError(w, http.StatusForbidden, fmt.Errorf("scope %q required", scope))
                   return
               }
               next.ServeHTTP(w, r)
           })
       }
   }
   ```

3. Apply `requireScope` on route registration:
   - `read` scope: all GET endpoints
   - `write` scope: POST/PUT/DELETE on memories, entities, sessions, skills
   - `admin` scope: /admin/*, /api-keys (creation/deletion)

4. Store scope in the API key record (already has `Scopes []string` field in `AdminAPIKey` type).

**Test file**: `internal/roles/roles_test.go` (292 lines already exists — verify it covers `CheckPermission`).

**Detailed spec**: `docs/features/rbac.md`

---

## 2. Compression Observability

**Status**: `GET /compression/stats` returns in-memory counters. No persistence, no history, no Prometheus metrics.

**Goal**: Persist compression stats to Redis/Neo4j, expose Prometheus metrics, track per-user stats.

**Files to create:**
- `internal/metrics/compression.go` — `CompressionMetrics` struct with `Record()`, `Get()`, `GetHistory()`
- `internal/metrics/prometheus.go` — register Prometheus gauges/counters for compression

**Files to modify:**
- `internal/compression/pipeline/pipeline.go` — call `metrics.Record()` after each compression job
- `cmd/server/api.go` — wire metrics to `/compression/stats` and `/metrics/compression`

**Data model:**
```go
type CompressionStats struct {
    TotalJobs          int64   `json:"total_jobs"`
    TotalTokensSaved   int64   `json:"total_tokens_saved"`
    AvgTokenReduction  float64 `json:"avg_token_reduction"`
    AccuracyRetention  float64 `json:"accuracy_retention"`
    AvgLatencyMS       float64 `json:"avg_latency_ms"`
    P95LatencyMS       float64 `json:"p95_latency_ms"`
    ExtractionsByMode  map[string]int64 `json:"extractions_by_mode"`
    SpreadingActivations int64 `json:"spreading_activations"`
    PeriodStart        time.Time `json:"period_start"`
    PeriodEnd          time.Time `json:"period_end"`
}
```

**Prometheus metrics to register:**
```
hystersis_compression_jobs_total (counter)
hystersis_compression_token_reduction_ratio (gauge)
hystersis_compression_accuracy_retention (gauge)
hystersis_compression_latency_ms (histogram, buckets: 50, 100, 200, 500, 1000)
hystersis_spreading_activations_total (counter)
```

**Detailed spec**: `docs/features/observability.md`

---

## 3. Mem0 v3 Parity

**Status**: Analyzed in `docs/mem0-v3-analysis.md`. None of the 4 innovations are implemented in Hystersis yet.

**Four innovations to implement:**

1. **Single-pass ADD-only extraction** — replace two-pass (extract + merge) with single LLM call + hash dedup
2. **BM25 keyword search** — add to `VectorStore` interface and Qdrant implementation
3. **Agent-generated facts as first-class** — extraction prompt must explicitly capture both user and assistant content
4. **Entity store in vector DB** — store entities in `{userID}_entities` Qdrant collection (reduce Neo4j coupling)

**Files to create:**
- `internal/memory/extraction/v3.go` — `ExtractionV3` struct implementing single-pass ADD-only
- `internal/vector/bm25.go` — BM25 scoring integration with Qdrant sparse vectors
- `internal/memory/entity/extractor.go` — entity extraction (NER + patterns)
- `internal/memory/entity/store.go` — entity collection in vector store

**Files to modify:**
- `internal/vector/provider.go` — add `KeywordSearch()` and `SearchBatch()` to `VectorStore` interface
- `internal/memory/processor.go` — add extraction mode selector (v2 vs v3)
- `cmd/server/api.go` — wire `?extraction_mode=v3` query param

**Detailed spec**: `docs/features/mem0-v3-parity.md`

---

## 4. Self-Improvement System

**Status**: `internal/memory/self_improve/` directory exists but is empty.

**Goal**: When users give feedback, automatically adjust memory importance, learn synonyms, and trigger content correction for negative feedback.

**Files to create:**
- `internal/memory/self_improve/engine.go` — `SelfImprovementEngine` struct
- `internal/memory/self_improve/synonym_store.go` — stores learned synonym pairs in Neo4j
- `internal/memory/self_improve/importance_adjuster.go` — adjusts importance scores based on feedback history

**Core algorithm:**
1. Positive feedback → `memory.ImportanceScore += 0.1` (max 1.0)
2. Negative feedback → `memory.ImportanceScore -= 0.2` + queue for LLM correction
3. `very_negative` → flag for human review + `memory.Status = "needs_correction"`
4. After 3+ positive feedbacks for similar queries → learn synonym pair (query → memory terms)

**Interface:**
```go
type SelfImprovementEngine interface {
    ProcessFeedback(ctx context.Context, memoryID, feedbackType, userID string) error
    GetLearnedSynonyms(ctx context.Context, tenantID string) ([]SynonymPair, error)
    GetMemoriesNeedingCorrection(ctx context.Context, tenantID string) ([]*types.Memory, error)
}
```

**Detailed spec**: `docs/features/self-improvement.md`

---

## 5. Memory Chunking

**Status**: `internal/memory/chunking/` directory exists but is empty.

**Goal**: Split large memory content (>512 tokens) into overlapping chunks. Store each chunk with a `parent_memory_id`. On retrieval, merge top-scoring chunks from same parent.

**Files to create:**
- `internal/memory/chunking/splitter.go` — sentence-aware text splitter with overlap
- `internal/memory/chunking/merger.go` — merges chunks from same parent on retrieval

**Chunking config:**
```go
type ChunkConfig struct {
    MaxTokens      int     // Default: 512
    OverlapTokens  int     // Default: 50
    Strategy       string  // "sentence" | "paragraph" | "fixed"
    MinChunkTokens int     // Default: 100 (don't create tiny chunks)
}
```

**Integration points:**
- `internal/memory/service.go` — call `chunker.Split()` in `CreateMemory()` when content > MaxTokens
- `internal/memory/service.go` — call `merger.Merge()` in search result post-processing

**Detailed spec**: `docs/features/chunking.md`

---

## 6. Skill Audit Events

**Status**: `skill.approved`, `skill.rejected`, `skill.synthesized` events are never emitted. The audit logger exists and works for memory operations.

**Files to modify:**
- `cmd/server/api.go` — add `a.audit.Log(...)` calls in:
  - `handleProcessSkillReview` after approve/reject decision
  - `handleSynthesizeSkills` after synthesis
  - `handleExtractSkills` after extraction
  - `handleExecuteSkill` after execution

**Event types to add in `internal/audit/audit.go`:**
```go
const (
    EventSkillApproved   = "skill.approved"
    EventSkillRejected   = "skill.rejected"
    EventSkillSynthesized = "skill.synthesized"
    EventSkillExtracted  = "skill.extracted"
    EventSkillExecuted   = "skill.executed"
)
```

**Audit payload format** (matches existing memory audit events):
```go
AuditEvent{
    TenantID:   tenantID,
    EntityType: "skill",
    EntityID:   skillID,
    Action:     EventSkillApproved,
    ActorID:    apiKeyID,
    Metadata:   map[string]interface{}{"notes": reviewNotes},
}
```

---

## 7. Azure LLM Provider

**Status**: `internal/llm/providers.go:418` and `:422` return `"not implemented"` for `Embed()` and `Rerank()`.

**Files to modify:** `internal/llm/providers.go`

**Implementation**: Azure OpenAI uses the same API as OpenAI but with different base URL format:
```go
// Embed uses Azure OpenAI embeddings endpoint
// URL: {AZURE_OPENAI_ENDPOINT}/openai/deployments/{EMBEDDING_MODEL}/embeddings?api-version=2023-05-15
func (p *AzureProvider) Embed(ctx context.Context, texts []string) ([][]float32, error) {
    // Set endpoint: config.AzureOpenAIEndpoint + "/openai/deployments/" + p.embedModel + "/embeddings"
    // Header: api-key: config.AzureAPIKey (not Bearer token)
    // Body: {"input": texts}
    // Same response format as OpenAI
}
```

**New env vars needed:**
- `AZURE_OPENAI_ENDPOINT` — base URL like `https://my-resource.openai.azure.com`
- `AZURE_EMBEDDING_DEPLOYMENT` — deployment name for embeddings
- `AZURE_RERANKER_DEPLOYMENT` — deployment name for reranking (optional)

---

## 8. OAuth Refresh Tokens

**Status**: `internal/mcp/oauth/handler.go:189` returns HTTP 501 for the token refresh endpoint.

**Files to modify:** `internal/mcp/oauth/handler.go`

**Implementation**: Standard OAuth2 refresh flow:
```go
func (h *OAuthHandler) handleTokenRefresh(w http.ResponseWriter, r *http.Request) {
    refreshToken := r.FormValue("refresh_token")
    // 1. Validate refresh token against token store
    // 2. Generate new access token (JWT, 1h expiry)
    // 3. Generate new refresh token (rotate on use)
    // 4. Invalidate old refresh token
    // 5. Return new token pair
}
```

**Token store**: Already have JWT infrastructure. Store refresh tokens in Redis with TTL=7d.

---

## 9. Archive Tier Backend

**Status**: Tier routing exists (Working→Hot→Cold), but `TierArchive` has no storage backend.

**Goal**: Move memories inactive for 90+ days to object storage (S3/GCS/local filesystem).

**Files to create:**
- `internal/memory/tier/archive.go` — `ArchiveBackend` interface + S3 implementation
- `internal/memory/tier/archive_worker.go` — background goroutine that scans for archivable memories

**Interface:**
```go
type ArchiveBackend interface {
    Store(ctx context.Context, memory *types.Memory) error
    Retrieve(ctx context.Context, memoryID string) (*types.Memory, error)
    Delete(ctx context.Context, memoryID string) error
}
```

**New env vars:**
- `ARCHIVE_BACKEND` — `s3` | `gcs` | `local`
- `ARCHIVE_BUCKET` — S3/GCS bucket name
- `ARCHIVE_LOCAL_PATH` — path for local filesystem
- `ARCHIVE_AFTER_DAYS` — default 90

---

## 10. Skills Group Policy

**Status**: `SkillSharingEnabled bool` in `GroupPolicy` and `AgentConfig.SkillDomains []string` are defined but never checked.

**Files to modify:**
- `internal/skills/registry.go` — in `ListSkills()`, filter by `AgentConfig.SkillDomains` if set
- `cmd/server/api.go` — in group skill endpoints, check `SkillSharingEnabled` before returning cross-agent skills
- `internal/memory/neo4j/client.go` — in `GetSkillsByDomain()`, accept domain filter slice

**Logic:**
```go
// In handleListSkills, after fetching skills:
if agentConfig.SkillDomains != nil {
    skills = filterByDomains(skills, agentConfig.SkillDomains)
}

// In handleGetGroupSkills:
if !group.Policy.SkillSharingEnabled {
    return emptyList
}
```

---

## Implementation Order

For a fresh agent starting work, tackle in this order:

```
Week 1: P0 blockers
  → Feature 1: RBAC Enforcement (1-2 days)
  → Feature 2: Compression Observability (1-2 days)
  → Feature 6: Skill Audit Events (0.5 days)

Week 2: Core intelligence
  → Feature 4: Self-Improvement System (2-3 days)
  → Feature 5: Memory Chunking (1-2 days)
  → Feature 10: Skills Group Policy (0.5 days)

Week 3: Competitive parity
  → Feature 3: Mem0 v3 Parity — Phase 1: Single-pass extraction (2 days)
  → Feature 3: Mem0 v3 Parity — Phase 2: BM25 + Entity store (2 days)

Week 4: Infrastructure
  → Feature 7: Azure LLM Provider (1 day)
  → Feature 8: OAuth Refresh Tokens (0.5 days)
  → Feature 9: Archive Tier Backend (1-2 days)
```

---

## Before Implementing Any Feature

Always run:
```bash
go build ./...    # Must pass before starting
go test ./...     # Note which tests fail
```

After implementing:
```bash
go build ./...    # Must still pass
go test ./...     # Must not have new failures
go vet ./...      # No new vet warnings
```

Commit format: `feat(feature-name): description`

---

## Key Conventions

| Convention | Rule |
|-----------|------|
| Error wrapping | `fmt.Errorf("package: operation: %w", err)` |
| HTTP errors | `safeHTTPError(w, status, err)` — never `http.Error()` directly |
| Config | All config via `internal/config/config.go`, never `os.Getenv()` directly |
| Batch operations | Use `GetMemoriesByIDs()`, `BatchCreateMemories()` — no N+1 loops |
| Compression code | Anything in `internal/compression/extractor/`, `retrieval/`, `llm/` is PROPRIETARY — do not expose in public APIs or open-source |
| Timeouts | Use `c.queryTimeout()` for Neo4j, configurable via `NEO4J_QUERY_TIMEOUT` |
| Rate limiter | Has cleanup goroutine — call `Stop()` on shutdown |
