# Plan: Harden Product + Framework Integrations + Revenue Pivots

## Context

Three parallel audits surfaced **5 CRITICAL, 14 HIGH, 12 MEDIUM** issues across the dashboard, landing page, API layer, SDKs, MCP server, and framework integrations. The product has real algorithmic depth but the consumer-facing layers are riddled with stubs, hardcoded credentials, broken endpoints, and missing feature exposure. None of the new features (temporal reasoning, MW scoring, provenance, compression) are visible to end users through any SDK, dashboard page, or MCP tool.

Meanwhile, Mem0 supports 17 framework integrations — Hystersis has 7 (all PARTIAL). Key missing: OpenAI Agents SDK, Vercel AI SDK, Google ADK, Pydantic AI. Stripe is wired but doesn't enforce quotas. The HTTP MCP server has 6 tools vs 26 in stdio mode.

---

## Phase A: Security & Critical Fixes

### A.1 Remove hardcoded API key from dashboard proxy
- **File**: `dashboard/src/app/api/proxy/route.ts:4`
- **Fix**: Remove fallback `"am_AYQh3k5V47AVVoyY_1776234755"`. Return 401 if `ADMIN_API_KEY` env var is missing.

### A.2 Remove demo credentials from landing page HTML
- **File**: `landing/src/pages/DemoPage.jsx:131-136`
- **Fix**: Remove `demo@hystersis.ai` / `demo123` from rendered JSX.

### A.3 Fix duplicate Entity type in dashboard
- **File**: `dashboard/src/lib/api.ts:754-758`
- **Fix**: Remove the second `Entity` declaration (missing `id`). Keep the first (lines 136-144).

### A.4 Fix duplicate route registrations in api.go
- **File**: `cmd/server/api.go:546-556`
- **Fix**: Remove the second registration block (compression/tier/playground routes). The first block (524-536) with `requirePermission` is the correct one.

### A.5 Authenticate playground route
- **File**: `dashboard/src/middleware.ts:17-19`
- **Fix**: Remove the `/playground` bypass. Require auth like all other routes.

---

## Phase B: Fix Broken Functionality

### B.1 Dashboard — fix billing, settings, trends
- `billing/page.tsx:60` — Replace `/stripe/checkout` call with proper Stripe Checkout session creation via existing `internal/stripe/service.go`
- `settings/page.tsx:54-65` — Wire `handleSaveProfile` to `PUT /admin/users/{id}` endpoint
- `settings/page.tsx:87-111` — Wire `handlePasswordUpdate` to `POST /auth/change-password` (create endpoint if needed)
- `settings/page.tsx:70-74` — Fix notification preferences to use correct field names
- `page.tsx:38-44` — Remove fake `getTrend()`, use real analytics data or remove trend arrows
- `page.tsx:80-85` — Fix `memoryGrowthData` to read correct analytics field

### B.2 Landing page — connect demo, fix features
- `DemoPage.jsx` — Wire to actual `/demo/chat`, `/demo/dashboard` endpoints via `utils/api.js`
- `Features.jsx` — Update to list ALL current features (spreading activation, temporal reasoning, MW scoring, compression, provenance, consolidation, 12 vector providers)
- `Pricing.jsx:85` — Replace hardcoded Stripe URL with env var + error handling
- `Blog.jsx` — Add error fallback UI when Sanity CMS fails
- `DemoDashboard.jsx` — Wire to real API data instead of hardcoded bars

### B.3 Fix Node SDK dead endpoints
- **File**: `sdk/nodejs/src/client.ts`
- 9 routes called but don't exist in backend:
  - `POST /memories/links`, `GET /memories/{id}/links`, `DELETE /memories/links/{linkId}` → Add to `api.go` or remove from SDK
  - `GET /memories/{id}/versions`, `POST /memories/{id}/restore` → Wire to existing memory history
  - `GET /memories/stats`, `GET /memories/insights`, `GET /memories/summary` → Wire or remove
  - `POST /admin/cleanup` → Map to existing `POST /admin/sync` or add endpoint

### B.4 Fix gateway missing routes
- **File**: `cmd/gateway/main.go`
- Add proxy rules for 18+ missing paths: `/sessions`, `/entities`, `/relations`, `/graph`, `/webhooks`, `/wiki`, `/feedback`, `/notifications`, `/analytics`, `/alerts`, `/compact`, `/backup`, `/playground`, `/groups`, `/admin`, `/auth`

---

## Phase C: Expose New Features in All Surfaces

### C.1 Dashboard — new feature pages
Create dashboard pages for:
- **Temporal Reasoning** — visualize memory decay curves, volatility scores, phase rotation
- **MW Scoring** — leaderboard of highest/lowest worth memories, success/failure trends
- **Provenance** — DAG visualization showing memory derivation chains
- **Tiered Memory** — Working→Hot→Cold→Archive distribution chart
- **Compression Metrics** — real-time accuracy, reduction ratio, latency over time

### C.2 MCP server parity
- **File**: `cmd/mcp-server/main.go`
- Add 20 missing tools to match stdio server: `update_memory`, `get_memory`, `list_entities`, `create_entity`, `create_relation`, `get_entity_relations`, `add_feedback`, `get_memory_history`, `create_session`, `add_message`, `get_context`, `create_skill`, `list_skills`, `suggest_skills`, `extract_skills`, `create_agent`, `list_agents`, `create_agent_group`, `add_agent_to_group`, `share_memory_to_group`
- Add NEW tools: `temporal_search` (with time_start/time_end), `compress_memory`, `get_compression_stats`, `set_tier_policy`, `get_provenance`
- Fix `handleWhoAmI` to return real user context

### C.3 SDK — expose new features
**Node.js** (`sdk/nodejs/src/client.ts`):
- Add `temporal` namespace: `temporalSearch(query, timeStart, timeEnd, options)`
- Add `compression` namespace: `getStats()`, `setMode(mode)`, `setTierPolicy(policy)`
- Add `provenance` namespace: `getChain(memoryId)`, `getCredit(memoryId)`
- Expose `worth_score`, `validity_status`, `volatility_score` in search results type

**Python** (`sdk/python/hystersis/__init__.py`):
- Mirror Node.js additions
- Add wiki, playground, benchmark methods to `__all__`

### C.4 Integration configs — expose new features
Update all integration `types.ts` configs to include:
- `compressionMode?: string`
- `tierPolicy?: string`
- `temporalSearch?: boolean`
- `enableMWScoring?: boolean`

---

## Phase D: Missing Framework Integrations (High-Priority Pivots)

### D.1 OpenAI Agents SDK integration (HIGHEST PRIORITY)
- **Node.js**: `sdk/nodejs/src/integrations/openai-agents.ts`
- **Python**: `sdk/python/hystersis/integrations/openai_agents.py`
- Implement as a tool provider that OpenAI agents can call for memory operations
- Support: `store_memory`, `recall`, `search`, `feedback`

### D.2 Vercel AI SDK integration
- **Node.js**: `sdk/nodejs/src/integrations/vercel-ai.ts`
- Implement as `MemoryProvider` compatible with `useChat` / `useCompletion`
- Auto-inject relevant memories into system prompt

### D.3 Google ADK integration
- **Python**: `sdk/python/hystersis/integrations/google_adk.py`
- Implement as a memory tool for Google's Agent Development Kit

### D.4 Pydantic AI integration
- **Python**: `sdk/python/hystersis/integrations/pydantic_ai.py`
- Type-safe memory dependency injection for Pydantic AI agents

### D.5 MCP config file for Claude Desktop / Cursor
- Create `mcp-config.example.json` at repo root
- Include both stdio and HTTP server configurations
- Add setup instructions to README

---

## Phase E: Revenue Pipeline Fixes

### E.1 Stripe quota enforcement
- **File**: `internal/stripe/service.go`
- Wire Stripe tier → memory quota limits
- In `CreateMemory`, check quota before write
- In `SearchMemories`, check search quota
- Expose `GET /billing/usage` endpoint for dashboard

### E.2 Analytics persistence
- Move atomic counters in `internal/analytics/service.go` to Redis
- Survive pod restarts
- Wire to dashboard analytics page with real data

### E.3 Audit logging server-side
- Wire `internal/audit/logger.go` to a dedicated API endpoint
- Dashboard `useAuditLogger` should POST to server, not localStorage

---

## Implementation Order

```
Phase A (Security — IMMEDIATE):
  ├── A.1: Remove hardcoded API key [dashboard/proxy]
  ├── A.2: Remove demo credentials [landing]
  ├── A.3: Fix duplicate Entity type [dashboard/api.ts]
  ├── A.4: Fix duplicate route registration [api.go]
  └── A.5: Authenticate playground [middleware.ts]

Phase B (Broken functionality):
  ├── B.1: Dashboard fixes (billing, settings, trends) [parallel]
  ├── B.2: Landing page fixes (demo, features, pricing) [parallel]
  ├── B.3: Node SDK dead endpoints [parallel]
  └── B.4: Gateway missing routes [parallel]

Phase C (Feature exposure):
  ├── C.1: Dashboard new feature pages [5 pages]
  ├── C.2: MCP server parity [+20 tools]
  ├── C.3: SDK new feature methods [Node + Python]
  └── C.4: Integration config updates [all frameworks]

Phase D (Framework integrations):
  ├── D.1: OpenAI Agents SDK [Node + Python]
  ├── D.2: Vercel AI SDK [Node]
  ├── D.3: Google ADK [Python]
  ├── D.4: Pydantic AI [Python]
  └── D.5: MCP config file [repo root]

Phase E (Revenue):
  ├── E.1: Stripe quota enforcement
  ├── E.2: Analytics persistence
  └── E.3: Server-side audit logging
```

## Critical Files

### Dashboard
| File | Issue |
|---|---|
| `dashboard/src/app/api/proxy/route.ts` | Hardcoded API key |
| `dashboard/src/lib/api.ts` | Duplicate Entity type, compression API duplication |
| `dashboard/src/middleware.ts` | Playground bypass |
| `dashboard/src/app/(dashboard)/billing/page.tsx` | Broken Stripe, hardcoded tier |
| `dashboard/src/app/(dashboard)/settings/page.tsx` | No-op profile/password |
| `dashboard/src/app/(dashboard)/page.tsx` | Fake trends, broken chart data |

### Landing
| File | Issue |
|---|---|
| `landing/src/pages/DemoPage.jsx` | Static mockup, exposed credentials |
| `landing/src/components/Features.jsx` | Missing 10+ features |
| `landing/src/components/Pricing.jsx` | Hardcoded Stripe URL |
| `landing/src/components/Blog.jsx` | Silent CMS failure |

### Backend
| File | Issue |
|---|---|
| `cmd/server/api.go` | Duplicate routes, missing link/version/stats endpoints |
| `cmd/mcp-server/main.go` | Only 6 of 26 tools, hardcoded whoAmI |
| `cmd/gateway/main.go` | Missing 18+ proxy routes |
| `cmd/memory-api/main.go` | Stub benchmark/metrics handlers |
| `internal/stripe/service.go` | No quota enforcement |

### SDKs
| File | Issue |
|---|---|
| `sdk/nodejs/src/client.ts` | 9 dead endpoints, no temporal/compression/provenance methods |
| `sdk/python/hystersis/__init__.py` | Missing wiki/playground/benchmark exports |
| `sdk/nodejs/src/integrations/index.ts` | Wrong LlamaIndex alias |

## Verification

```bash
# Go
go build ./... && go vet ./...

# Dashboard
cd dashboard && npm run build

# Landing
cd landing && npm run build

# Node SDK
cd sdk/nodejs && npx tsc --noEmit

# Test endpoints
curl -s http://localhost:8080/health
curl -s -H "X-API-Key: $KEY" http://localhost:8080/memories?limit=1
```
