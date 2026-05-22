# Plan: Final Hardening — Enterprise, Revenue, & Remaining Gaps

## Context

After three major implementation passes, the algorithmic core is solid and consumer surfaces are connected. What remains is the **enterprise plumbing** — features that exist in code but aren't enforced at runtime. The pattern is consistent: validators/checkers/middleware exist but are never wired into the HTTP layer.

---

## Phase E: Revenue Pipeline (Stripe Quota Enforcement)

### E.1 Stripe tier-to-quota mapping + enforcement
- **Files**: `internal/stripe/service.go`, `internal/memory/service.go`, `cmd/server/api.go`
- **Problem**: Stripe webhook handlers are `fmt.Printf` only — no customer record, no quota, no enforcement
- **Fix**:
  1. In `stripe/service.go`: Define tier→quota map (Free: 1K memories/10K searches, Pro: 50K/100K, Team: 200K/500K, Enterprise: unlimited)
  2. In `stripe/service.go`: `handleCheckoutComplete` and `handlePaymentSuccess` → persist customer tier to Neo4j via a `CustomerStore` interface
  3. Add `GetCustomerTier(tenantID)` and `CheckQuota(tenantID, operation)` methods
  4. In `memory/service.go` → `CreateMemory`: call `stripeService.CheckQuota(tenantID, "memory_create")` before writing
  5. In `memory/service.go` → `SearchMemories`: call `stripeService.CheckQuota(tenantID, "search")` before searching
  6. Add `GET /billing/usage` endpoint in `api.go` returning current usage vs limits
  7. Add `GET /billing/subscription` endpoint returning current plan tier

### E.2 Wire RecordSearch into analytics
- **File**: `cmd/server/api.go` — search handler
- **Problem**: `analytics.RecordSearch()` exists but is never called
- **Fix**: In the search endpoint handler, call `s.analyticsSvc.RecordSearch(resultCount)` after returning results

---

## Phase F: Tenant Isolation Fixes

### F.1 Add tenant filter to all Neo4j read queries
- **File**: `internal/memory/neo4j/client.go`
- **Problem**: `GetMemory(id)`, `GetMemoriesByIDs(ids)`, `ListAllMemories()` have NO tenant filter — cross-tenant data leakage
- **Fix**:
  1. Change `GetMemory(id string)` → `GetMemory(id, tenantID string)` — add `WHERE m.tenant_id = $tenantID` to Cypher
  2. Change `GetMemoriesByIDs` similarly
  3. Change `ListAllMemories` → require tenantID parameter
  4. Update `memory/service.go` to pass tenantID through all methods
  5. Add tenantID field to Service struct, populated from config or request context

### F.2 Service-layer tenant scoping
- **File**: `internal/memory/service.go`
- **Problem**: Zero references to TenantID — service layer adds no scoping
- **Fix**: Add `tenantID string` parameter to `CreateMemory`, `SearchMemories`, `GetMemory` or extract from context. Pass through to neo4j client.

---

## Phase G: License Enforcement

### G.1 Wire license middleware into API routes
- **File**: `cmd/server/api.go`
- **Problem**: `license.Middleware` with `RequireValidLicense` and `RequireFeature` exists but is never applied
- **Fix**:
  1. In server startup, create `licenseMiddleware := license.NewMiddleware(validator)`
  2. Apply `licenseMiddleware.RequireValidLicense()` as global middleware
  3. Apply `licenseMiddleware.RequireFeature("compression")` on compression endpoints
  4. Apply `licenseMiddleware.RequireFeature("skills")` on skills endpoints
  5. Apply `licenseMiddleware.RequireFeature("knowledge_graph")` on graph endpoints

---

## Phase H: Dashboard Remaining Fixes

### H.1 Fix settings page (profile + password save)
- **File**: `dashboard/src/app/(dashboard)/settings/page.tsx`
- **Problem**: `handleSaveProfile` and `handlePasswordUpdate` are no-ops — show success but do nothing
- **Fix**: Wire to `PUT /admin/users/{id}` for profile, add `POST /auth/change-password` endpoint for password

### H.2 Fix billing page
- **File**: `dashboard/src/app/(dashboard)/billing/page.tsx`
- **Problem**: Calls nonexistent `/stripe/checkout`, hardcodes "Free Tier"
- **Fix**: Wire to new `GET /billing/subscription` endpoint for current tier, use `POST /stripe/create-checkout-session` for upgrades

### H.3 Fix home page fake trends + broken chart
- **File**: `dashboard/src/app/(dashboard)/page.tsx`
- **Problem**: `getTrend()` computes fake percentages, `memoryGrowthData` reads wrong field
- **Fix**: Remove fake trend calculation, use real analytics delta or remove trend arrows. Fix chart to use time-series data from analytics endpoint.

### H.4 Create new feature dashboard pages
- **Temporal Reasoning page** — show memory decay curves, volatility distribution, phase angles
- **MW Scoring page** — memory worth leaderboard, success/failure trends
- **Provenance page** — DAG visualization of memory derivation chains
- **Compression Metrics page** — real-time accuracy, reduction ratio, latency

---

## Phase I: Audit + Notification Persistence

### I.1 Persist audit logs to Neo4j
- **File**: `internal/audit/logger.go`
- **Problem**: Defaults to `InMemoryStorage` — all events lost on restart
- **Fix**: Create `Neo4jAuditStorage` implementing the `Storage` interface. Wire in server startup.

### I.2 Persist notifications
- **File**: `internal/notification/service.go`
- **Problem**: All notifications in `map[string]*Notification` — lost on restart
- **Fix**: Add Neo4j or Redis backend for notification persistence

### I.3 Wire analytics RecordSearch + persist counters
- **File**: `internal/analytics/service.go`
- **Problem**: Atomic counters reset on restart, `RecordSearch` never called
- **Fix**: Persist counters to Redis with periodic flush. Wire `RecordSearch` into search handler.

---

## Phase J: Wire Remaining Service Stubs

### J.0 Wire remaining service.go methods to graph store
- **File**: `internal/memory/service.go`
- **Problem**: ~35 methods still return `nil, nil`. These are mostly CRUD that should delegate to `s.graph` or `s.neo4jClient`.
- **Fix** (delegate each to neo4j client):

**Skills CRUD** — `CreateSkill`, `ListSkills`, `SearchSkillsByTrigger`, `GetSkillsByDomain`, `GetSkill`, `UpdateSkill`, `DeleteSkill`, `ExecuteSkill`, `UseSkill`, `IncrementSkillUsage`, `GetSimilarSkills`, `SuggestSkills`, `ExtractSkills`

**Chains CRUD** — `CreateChain`, `ListChains`, `GetChain`, `UpdateChain`, `DeleteChain`, `ExecuteChain`, `GetChainExecutions`, `ExtractChains`

**Agents/Groups** — `CreateAgent`, `GetAgent`, `UpdateAgent`, `DeleteAgent`, `ListAgents`, `CreateAgentGroup`, `GetAgentGroup`, `UpdateAgentGroup`, `DeleteAgentGroup`, `ListAgentGroups`, `AddAgentToGroup`, `RemoveAgentFromGroup`, `GetGroupSkills`, `GetGroupMemories`, `ShareMemoryToGroup`

**Reviews** — `CreateSkillReview`, `ListSkillReviews`, `GetSkillReview`, `ListPendingReviews`, `GetReview`

**Sessions** — `ListSessions`, `GetContext`, `AddToContext`

**Batch/Export** — `BatchCreateMemories`, `BatchUpdateMemories`, `BatchDeleteMemories`, `ExportMemories`, `ImportMemories`

**Compaction** — `RunCompaction`, `RunTargetedCompaction`, `CompactNegativeFeedback`

**Other** — `ArchiveMemory`, `HybridSearch`, `AdvancedSearch`, `GetMemoryStats`, `CleanupExpiredMemories`, `SetMemoryExpiration`

Each should: nil-check `s.graph`/`s.neo4jClient`, delegate to the matching method, wrap errors with `fmt.Errorf("service: ...: %w", err)`.

### J.1 Implement the 6 stub API handlers
- **File**: `cmd/server/api.go`
- The 6 `not_implemented` stubs (memory links, insights, summary, admin/cleanup) should be wired to real service methods or removed from the SDK.

---

## Phase K: CI/CD + Documentation + README

### J.1 Fix CI pipeline
- **File**: `.github/workflows/ci.yml`
- Check what's disabled and re-enable: security scanning, Node.js tests, mypy

### J.2 Update README with new features
- **File**: `README.md`
- Add: temporal reasoning, MW scoring, provenance DAG, adaptive retrieval, sleep consolidation, UCB bandit, dual pools
- Add: framework integration table (11 frameworks)
- Add: MCP config instructions
- Update benchmark targets table

### J.3 Add SAML signature verification
- **File**: `internal/sso/saml.go`
- **Problem**: `verifyAssertionSignature()` returns nil always — SAML assertions accepted without crypto verification
- **Fix**: Wire `parseSamlCertificate` into `NewSAMLProvider`, verify signatures properly

---

## Implementation Order

```
Phase E (Revenue — highest business impact):
  ├── E.1: Stripe quota enforcement [stripe/service.go + service.go + api.go]
  └── E.2: Wire analytics RecordSearch [api.go]

Phase F (Security — tenant isolation):
  ├── F.1: Neo4j tenant filters [neo4j/client.go]
  └── F.2: Service-layer tenant scoping [service.go]

Phase G (Enterprise gate):
  └── G.1: License middleware wiring [api.go]

Phase H (Dashboard completeness):
  ├── H.1: Settings page [parallel]
  ├── H.2: Billing page [parallel]
  ├── H.3: Home page trends [parallel]
  └── H.4: New feature pages [parallel]

Phase I (Persistence):
  ├── I.1: Audit to Neo4j [parallel]
  ├── I.2: Notifications persistence [parallel]
  └── I.3: Analytics to Redis [parallel]

Phase J (Wire remaining stubs):
  ├── J.0: Wire ~35 remaining service.go methods [service.go]
  └── J.1: Implement 6 stub API handlers [api.go]

Phase K (Polish):
  ├── K.1: CI pipeline fixes [parallel]
  ├── K.2: README update with all new features [parallel]
  └── K.3: SAML signature verification fix [parallel]
```

## Verification

```bash
# Go
go build ./... && go vet ./... && go test ./internal/memory/... -v

# Dashboard
cd dashboard && npm run build

# Landing
cd landing && npm run build

# Node SDK
cd sdk/nodejs && npx tsc --noEmit
```
