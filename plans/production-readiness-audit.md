# Production Readiness Plan: Hystersis Agent-Memory

## Context

Hystersis implements 20+ research paper specifications competing with Mem0. A comprehensive audit found:
- **~70% of features are genuinely implemented** (not stubs)
- **Core algorithms work**: ProMem extraction, spreading activation, temporal reasoning, MW scoring, tiered memory
- **GitHub repo is mostly set up** but missing LICENSE, no releases, install script has wrong repo URL
- **Build passes, tests pass**, but coverage is <5% on critical paths
- **Security is solid** (RBAC, SSO, rate limiting) with minor SSRF gap

The goal: verify everything works, fix broken pieces, and make it release-ready.

---

## Phase 1: Critical GitHub & Repo Fixes

### 1a. Fix install script repo URL
**File:** `scripts/install.sh:7`
```
REPO_URL="${REPO_URL:-https://github.com/agent-memory/agent-memory}"
→ REPO_URL="${REPO_URL:-https://github.com/Himan-D/agent-memory}"
```

### 1b. Add LICENSE file
- Create `LICENSE` at repo root (Apache 2.0 or MIT — user to decide)
- README badge already references it

### 1c. Add CODE_OF_CONDUCT.md
- Standard Contributor Covenant

### 1d. Create first release
```bash
git tag v1.0.0
gh release create v1.0.0 --title "v1.0.0 - Production Release" --notes "..."
```

### 1e. Fix README badges
- Go version badge says `1.21+` but go.mod says `1.25` — update badge

---

## Phase 2: Wire Unwired Features

These features have code but aren't connected to the service:

### 2a. Auto-Dreamer Sleep Consolidation
**Files:** `internal/memory/sleep/scheduler_consolidation.go`
- Scheduler exists but never started in `service.go`
- Wire: start consolidation scheduler as background goroutine in `NewService()`

### 2b. Archive Tier Backend
**Files:** `internal/memory/tier/router.go`
- `ArchiveStore` interface + `FilesystemArchive` exist but never instantiated
- Wire: create archive store in service init, add archive trigger to tier router

### 2c. Compression Metrics Persistence
**Files:** `internal/metrics/`
- In-memory counters exist but not persisted to Neo4j/Redis
- Wire: add periodic flush of compression stats to storage

---

## Phase 3: Fix Known Bugs & Gaps

### 3a. SSRF Protection for Webhooks
**File:** `internal/webhook/`
- Webhook URLs are user-provided but not validated against private IP ranges
- Add URL validation: block 127.0.0.0/8, 10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16

### 3b. N+1 Query in SearchMemories
**File:** `internal/memory/service.go:649-686`
- Two loops make individual `GetMemory()` calls per search result
- Add `GetMemoriesByIDs()` batch method to GraphStore interface
- Replace loops with single batch fetch

### 3c. Wiki Persistence
**File:** `internal/wiki/`
- Currently in-memory only — data lost on restart
- Add file-based or Neo4j persistence backend

### 3d. MCP Auth Forwarding
**File:** `internal/mcp/server.go`
- OAuth discovery exists but token not validated
- Add Bearer token extraction and tenant context propagation

---

## Phase 4: CI/CD Fixes

### 4a. Re-enable disabled CI checks
**File:** `.github/workflows/ci.yml`
- gosec (Go security scanning) — fix version issue
- mypy (Python SDK type checking) — fix type errors
- eslint (Node SDK linting) — configure

### 4b. Add required status checks to branch protection
```bash
gh api repos/Himan-D/agent-memory/branches/main/protection -X PUT ...
```

---

## Phase 5: Documentation Polish

### 5a. Fix Mintlify deployment
- PR #19 shows Mintlify preview failed
- Check `docs/mint.json` for configuration issues
- Verify all referenced MDX files exist

### 5b. Verify OpenAPI spec matches actual endpoints
- Cross-reference `docs/openapi.json` with registered routes in `api.go`

---

## Verification

```bash
# Build
go build ./...

# Tests
go test ./...

# Integration (with Docker services)
docker compose up -d neo4j qdrant redis
go test -count=1 ./tests/

# Install script
bash scripts/install.sh --help

# GitHub
gh repo view
gh release list
```

---

## Out of Scope (Future Work)

- OpenTelemetry distributed tracing (nice-to-have, not blocking)
- Migrate from `log` to `slog`/`zap` structured logging
- Test coverage increase to 70%+ (large effort, separate initiative)
- Exploitation/Exploration dual pool + UCB retrieval bandit (not yet designed)
- Single-pass ADD-only extraction (Mem0 v3 parity)
- Load testing at scale

---

## Research Paper Implementation Status (Verified)

| # | Paper/Feature | Status | Location |
|---|---------------|--------|----------|
| 1 | ProMem Extraction (arXiv:2601.04463) | ✅ Implemented | `compression/extractor/proprietary.go` |
| 2 | Spreading Activation (arXiv:2601.02744) | ✅ Implemented | `compression/retrieval/proprietary.go` |
| 3 | Temporal Phase Rotation (RoMem) | ✅ Implemented | `memory/temporal/rotation.go` |
| 4 | Ebbinghaus Decay | ✅ Implemented | `memory/decay/scorer.go` |
| 5 | MW Composite Scoring | ✅ Implemented | `memory/scoring/composite.go` |
| 6 | Provenance DAG + TD(λ) Credit | ✅ Implemented | `memory/provenance/dag.go` |
| 7 | Tiered Memory (W→H→C→A) | ⚠️ 3/4 tiers wired | `memory/tier/router.go` |
| 8 | Async Compression Pipeline | ✅ Implemented | `compression/pipeline/async.go` |
| 9 | Hybrid LLM Router | ✅ Implemented | `compression/llm/router.go` |
| 10 | Knowledge Graph / Entity Linking | ✅ Implemented | `memory/neo4j/` (3500+ LOC) |
| 11 | Semantic Vector Search | ✅ Implemented | `memory/qdrant/` |
| 12 | Memory Consolidation | ✅ Implemented | `memory/consolidation/` |
| 13 | Sleep/Dreamer Consolidation | ⚠️ Not wired | `memory/sleep/` |
| 14 | Privacy/PII Detection | ✅ Implemented | `memory/privacy/filter.go` |
| 15 | Safety/Content Filtering | ✅ Implemented | `memory/safety/classifier.go` |
| 16 | Feedback Loops | ✅ Implemented | `memory/feedback/feedback.go` |
| 17 | Self-Improvement | ✅ Implemented | `memory/self_improve/improver.go` |
| 18 | Ontology Management | ✅ Implemented | `memory/ontology/` |
| 19 | Multi-Agent Sync | ✅ Implemented | `memory/sync/` |
| 20 | Cognee/Cognify Integration | ✅ Implemented | `memory/cognee/`, `memory/cognify/` |
| 21 | Reranking (Cohere + LLM) | ✅ Implemented | `reranker/` |
| 22 | Recommendation Engine | ✅ Implemented | `recommendation/` (14 files) |
| 23 | Skills System | ✅ Implemented | `skills/registry.go` |
| 24 | SSO (OIDC/SAML/LDAP) | ✅ Implemented | `sso/` |
| 25 | RBAC | ✅ Implemented | `roles/` |
| 26 | LLM Wiki (Karpathy) | ⚠️ In-memory only | `wiki/` |
| 27 | Conflict Resolution | ✅ Implemented | `memory/types/` + service |

**Score: 23/27 fully implemented, 4 partially implemented, 0 missing**
