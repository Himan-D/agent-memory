# Codebase Gap Audit — Hystersis / agent-memory

**Date**: 2026-05-22  
**Scope**: All Go backend, dashboard (Next.js), connectors, CI, billing, connectors, wiki, evaluation, skills-npm

---

## P0 — Security Vulnerabilities (Fix Before Ship)

### 1. GitHub webhook signature verification is a no-op
**File**: `internal/connectors/github.go:317-319`
```go
func verifyGitHubSignature(payload []byte, signature, secret string) bool {
    return true   // <-- always trusts all payloads
}
```
Anyone can POST arbitrary events to the GitHub webhook endpoint. Must implement HMAC-SHA256 comparison (same pattern as Slack's `verifySignature` which is correctly implemented in `slack.go`).

### 2. Hardcoded secrets in AGENTS.md / dashboard CLAUDE.md
**File**: `dashboard/AGENTS.md` (checked into repo)
- `ADMIN_API_KEY=am_AYQh3k5V47AVVoyY_1776234755` — plaintext admin key
- `NEXTAUTH_SECRET=0g0XXNo7EKz2AYmTVIk/Ma0EqYptwkP8mjNterPENZs=` — plaintext NextAuth secret
- `demo@hystersis.ai / demo123` — demo credentials

These are committed to the codebase. Keys must be rotated and removed from all tracked files.

---

## P1 — Stub Methods in Core Service (Will Return Silent Nulls in Production)

### 3. `internal/memory/service.go` — 50+ stub `return nil, nil` methods
These are not errors; they silently return empty results. The most impactful:

| Method | Line | Impact |
|--------|------|--------|
| `GetContext()` | 133 | Sessions always return empty message history |
| `RunCompaction()` | 141 | Compaction UI triggers do nothing |
| `RunTargetedCompaction()` | 144 | Same |
| `CompactNegativeFeedback()` | 147 | Same |
| `GetEntitiesByMemory()` | 568 | Entity→Memory graph link is dead; TODO comment present |
| `GetMemoriesPaginated()` | 601 | Paginated memory listing returns nothing; TODO comment present |
| `AddToContext()` / `GetMessages()` | 132, 135 | In-memory session context fully stubbed |
| Skills methods (lines 742–760) | 742-760 | `CreateSkill`, `GetSkill`, `UpdateSkill` etc. all `return nil, nil` |
| Tier/archive methods (lines 818-891) | 818-891 | Tiered memory operations are no-ops |

**Root cause**: Service implements a large interface but many methods were scaffolded without backing store calls.

### 4. `internal/metrics/neo4j_store.go:97-108` — three metric store methods return `nil, nil`
GetMetrics, ListMetrics, AggregateMetrics all stub. The analytics dashboard pulls from these via the analytics service. If neo4j is the backing store for metrics, these will silently return empty data.

### 5. `internal/memory/search/strategies.go:32,78,150` — three search strategies stub
These are the concrete search strategy implementations (likely keyword, semantic, hybrid variants). All return `nil, nil`. Whatever code dispatches to these strategies gets empty results silently.

### 6. `internal/memory/search/search.go:280,312` — two search coordinator returns stub

---

## P2 — Connector Stubs (Features Advertised but Not Working)

### 7. `internal/connectors/gdrive.go` — completely hollow
Only two methods exist: `NewGoogleDriveClient` (doesn't even store `accessToken`) and `ListFiles` (returns empty slice). No OAuth, no file reading, no content extraction. The constructor silently discards the `accessToken` parameter it receives.

### 8. `internal/connectors/s3.go` — not read yet, but pattern suggests stub
(Listed alongside gdrive in the same connectors package; needs verification before claim.)

### 9. Stripe service does not enforce quotas
`internal/stripe/service.go` handles webhook events with `fmt.Printf` logging only — `handleCheckoutComplete`, `handlePaymentSuccess`, `handlePaymentFailed`, `handleSubscriptionDeleted` all just print to stdout. No database writes, no quota updates, no user plan changes. A successful Stripe checkout does nothing to the user's account.

---

## P3 — Dashboard Pages with Fake/Missing Data

### 10. `dashboard/src/app/(dashboard)/page.tsx` — fake trend calculations
```tsx
const getTrend = (value: number, multiplier: number = 0.8) => {
  const base = Math.ceil(value * multiplier);
  // trend = synthetic ratio of current vs (current * 0.8)
```
The "trend" shown on the main dashboard is always exactly `~25%` (since 100/80=1.25) for any non-zero value. It is entirely fabricated from the current value; no historical data is fetched or compared.

### 11. `dashboard/src/app/(dashboard)/billing/page.tsx` — no current plan state
The "Current Subscription" card always shows "Free Tier" hardcoded. No API call is made to fetch the actual plan. The Upgrade button does nothing. Plan prices ($29, $99) don't match Stripe's `GetPlans()` definition (which says $29/seat and $99/seat respectively but uses seat quantity).

### 12. `dashboard/src/app/(dashboard)/settings/page.tsx` — profile save is a no-op
`handleSaveProfile` reads DOM elements by ID and calls `toast.success("Profile settings saved")` with no API call. The profile is never actually persisted.

### 13. No dashboard pages for new features
The following backend systems have API handlers but no corresponding dashboard pages:
- **Wiki** (`/wiki/ingest`, `/wiki/query`, `/wiki/pages`) — no `/wiki` dashboard page
- **Temporal memory** — no dedicated view
- **Provenance/memory lineage** — no dashboard view
- **Benchmark results** — no `/benchmarks` page to view last run results

---

## P4 — CI/CD Disabled Steps (Pipeline has Silent Gaps)

**File**: `.github/workflows/ci.yml`

| Disabled step | `if: false` reason noted | Risk |
|---|---|---|
| Lighthouse CI | "Temporarily disabled" | Perf regressions go undetected |
| Node.js SDK ESLint | "not configured" | Lint errors in SDK are never caught |
| Node.js SDK tests | "npm test not configured" | SDK has zero CI test coverage |
| Python SDK mypy | "type errors in SDK" | Type errors known but suppressed |
| Go security scan (gosec) | "version issues" | Security scan silently skipped |
| Trivy SARIF upload | "permissions issue" | Vuln scan runs but results never uploaded |
| Production deploy job | `if: false` | Deploy is entirely manual/webhook-based |
| Docker job `if` condition | `refs/heads main` (missing `/`) | Docker push **never triggers** (syntax bug) |

The Docker `if` condition on line 219 reads:
```yaml
if: github.ref == 'refs/heads main'
```
This should be `'refs/heads/main'` (with a slash). The Docker image is never published to Docker Hub from CI.

---

## P5 — Benchmark System: Real Framework, Missing Data Files

`internal/evaluation/benchmark.go` is a real, functional implementation (parallel execution, LLM scoring, percentile calculation). **However**, it requires dataset files at runtime:
- `evaluation/locomo/dataset.json`
- `evaluation/longmemeval/dataset.json`
- `evaluation/beam_1m/dataset.json`
- `evaluation/beam_10m/dataset.json`

These files are not present in the repo. `RunLoCoMo`, `RunLongMemEval`, `RunBEAM` all call `os.ReadFile` and will return errors immediately. The `runBenchmarkHandler` swallows these errors silently (via `if err == nil` checks in `RunAll`). The benchmark API endpoint appears to work but all results are nil.

---

## P6 — Wiki System: Functional but Keyword-Only

`internal/wiki/service.go` is substantially implemented (LLM-powered ingestion, entity extraction, stub page creation). **Gaps**:
- `rankPagesByRelevance` uses only substring keyword matching (no vector similarity)
- `Lint()` method returns empty `LintResult{}` — no validation logic
- `wiki.Store` interface — need to verify if `store.go` has real or in-memory backing

---

## P7 — skills-npm Package: Works, But Has Issues

`skills-npm/src/index.js` is a real HTTP client wrapping all skills API endpoints. Gaps:
- `package.json` name needs verification (check if it's published as `@hystersis/skills`)
- No authentication header support (no API key is passed in requests — anyone who can reach the API can use the SDK without auth)
- `executeSkill` calls `/skills/${id}/execute` — this endpoint needs to be verified on the backend
- No TypeScript types (pure JS, no `.d.ts`)

---

## P8 — `internal/recommendation/` adapters return `nil, nil`

`adapter.go:27,42` and `neo4j_adapter.go:36` all return `nil, nil`. If the recommendation service is wired in and called, it silently returns nothing. Check if `recommendation` is actually wired to any API route.

---

## Summary by Priority

| # | Area | Severity | Effort |
|---|------|----------|--------|
| 1 | GitHub webhook signature bypass | P0 Security | Small (1h) |
| 2 | Secrets committed in AGENTS.md | P0 Security | Immediate rotation |
| 3 | 50+ stub methods in memory/service.go | P1 Correctness | Large (multi-day) |
| 4 | Stripe handlers don't persist plan changes | P1 Business logic | Medium |
| 5 | Neo4j metrics store stubs | P1 Analytics | Medium |
| 6 | Search strategy stubs | P1 Search | Medium |
| 7 | GDrive connector is hollow | P2 Feature | Medium |
| 8 | Dashboard trend numbers are fabricated | P2 Trust | Small |
| 9 | Billing page hardcodes "Free Tier" | P2 UX | Small |
| 10 | Settings profile save is no-op | P2 UX | Small |
| 11 | Docker CI `refs/heads main` syntax bug | P3 DevOps | Trivial fix |
| 12 | CI has 7 disabled steps | P3 DevOps | Medium |
| 13 | Benchmark datasets missing | P3 Testing | Data work |
| 14 | No wiki/benchmark dashboard pages | P4 UX | Medium |
| 15 | skills-npm lacks auth headers | P4 SDK | Small |
