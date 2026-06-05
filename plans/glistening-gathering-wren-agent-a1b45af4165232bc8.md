# Hystersis Audit — Findings

## Plan Mode: Audit Results

This document records all findings from the landing page, API integration, and SDK audit.

---

## LANDING PAGE

### [CRITICAL] Demo page is a static mockup — no live API connection
- File: `landing/src/pages/DemoPage.jsx` (lines 50–85)
- The "Live Playground" page renders hardcoded static bars (`91%`, `78%`, `85%` reduction).
- All three CTAs link to `https://dashboard.hystersis.ai` — an external domain not confirmed to exist.
- The `utils/api.js` (which has real `/demo/chat`, `/demo/dashboard`, `/demo/session` calls) is never imported by this page.
- The backend `cmd/server/api.go` does register `/demo/chat`, `/demo/dashboard`, `/demo/session` routes (lines 539–543) — but the landing page never calls them.
- The `DemoDashboard.jsx` component (which wires up the api.js calls) is never rendered on this page.

### [CRITICAL] Demo credentials exposed in plain HTML
- File: `landing/src/pages/DemoPage.jsx` (lines 131–136)
- Username `demo@hystersis.ai` and password `demo123` are hardcoded in JSX visible to all visitors.
- These will appear in the rendered HTML source and search engine caches.

### [HIGH] Stripe checkout hits a hardcoded absolute API URL
- File: `landing/src/components/Pricing.jsx` (line 85)
- `https://api.hystersis.ai/stripe/checkout` is hardcoded — no env variable.
- There is no `/stripe/checkout` route anywhere in `cmd/server/api.go`.
- Clicking "Get Pro" or "Get Team" will always fail with a network error or show the fallback alert.
- The `alert()` fallback (`alert('Payment not available yet...')`) will appear in production.

### [HIGH] Features page is missing all new proprietary features
- File: `landing/src/components/Features.jsx`
- Only 6 features listed: Semantic Search, Knowledge Graph, Memory Compaction, Skill Extraction, Multi-Agent Sync, Memory Versioning.
- Not listed: Spreading Activation retrieval, LLM Router (fast/verify), Tiered Memory (Working→Hot→Cold→Archive), ProMem Extraction, Async Compression Pipeline, Benchmarking (LoCoMo, LongMemEval, BEAM), Wiki feature, Consolidation/MemGPT-style summarization, Playground.
- The section header says "Six memory systems" — but the system has far more.

### [MEDIUM] Blog section silently hides itself on CMS failure
- File: `landing/src/components/Blog.jsx` (line 22)
- `if (loading) return null` — but the final state after a fetch error is also `loading=false, blogs=[]`.
- If Sanity CMS is unreachable or `VITE_SANITY_PROJECT_ID` is unset, the blog section renders as completely empty with no fallback, no error message, and no static content.
- `VITE_SANITY_PROJECT_ID` and `VITE_SANITY_READ_TOKEN` are NOT in the committed `.env` file (only PostHog key is there), so any fresh deploy without a `.env.local` will silently break the blog.

### [MEDIUM] "Read the Docs" links to an unconfirmed external domain
- File: `landing/src/components/Features.jsx` (line 132)
- `href="https://docs.hystersis.ai"` — no routing to `/docs` which exists as a local page.
- DocsPage (`/docs`) exists as a local route but is not linked from the Features CTA.

### [MEDIUM] DemoDashboard sparkline is 100% hardcoded static data
- File: `landing/src/components/DemoDashboard.jsx` (lines 81–95)
- Memory growth chart uses hardcoded percentage heights (20%, 35%, 50%, 40%, 60%, 45%, 55%, 30%).
- The "Avg Compression" stat is hardcoded as `85%` (line 75), not from the API.
- The component is wired to receive `data` from the API but falls back to these statics.

### [LOW] PostHog key is committed to `.env`
- File: `landing/.env` (line 2)
- `VITE_POSTHOG_KEY=phc_sbFTQyvxSFLMEJV8Y3c4ogwSkKGrFyZkZxAprhTaPFxm` is in a committed env file.
- PostHog analytics key is semi-public (client-side by design) but committing it to git is poor hygiene.

### [LOW] Amplitude analytics: VITE_AMPLITUDE_API_KEY never defined anywhere
- File: `landing/src/utils/analytics.js` (line 4)
- Amplitude initialization is gated on `VITE_AMPLITUDE_API_KEY` which is absent from `.env` and `.env.example`.
- Amplitude is silently never initialized in any deployment.

---

## API INTEGRATION AUDIT

### [CRITICAL] Duplicate route registrations in api.go will silently shadow each other
- File: `cmd/server/api.go` (lines 524–556)
- `GET /compression/stats`, `GET /compression/mode`, `PUT /compression/mode`, `GET /tier/policy`, `PUT /tier/policy`, `POST /playground/compress`, `POST /playground/search`, `GET /playground/stats` are all registered twice.
- gorilla/mux will use the first match; the second registration is dead code that will confuse future maintainers.
- The first registration (lines 524–536) wraps writes with `requirePermission` but the second (lines 546–556) does not — so the permission wrapper is bypassed on the duplicated GET routes.

### [HIGH] memory-api stub handlers
- File: `cmd/memory-api/main.go`
- `handleBenchmark` (line 143) returns only `{"status": "benchmark endpoint"}` — stub.
- `handleMetrics` (lines 148–156) returns hardcoded static metric strings with all-zero values, not real Prometheus metrics.
- `hybridSearch` (line 137) is referenced but only `searchMemories` is shown as complete — needs confirmation of the full implementation.

### [HIGH] Node SDK calls routes that don't exist on the server
- File: `sdk/nodejs/src/client.ts`
- `createMemoryLink` (line 863) calls `POST /memories/links` — not registered in `api.go`.
- `getMemoryLinks` (line 868) calls `GET /memories/{id}/links` — not registered in `api.go`.
- `deleteMemoryLink` (line 872) calls `DELETE /memories/links/{linkId}` — not registered in `api.go`.
- `getMemoryVersions` (line 876) calls `GET /memories/{id}/versions` — not registered in `api.go`.
- `restoreMemoryVersion` (line 880) calls `POST /memories/{id}/restore` — not registered in `api.go`.
- `getMemoryStats` (line 888) calls `GET /memories/stats` — not registered in `api.go`.
- `getMemoryInsights` (line 894) calls `GET /memories/insights` — not registered in `api.go`.
- `getMemorySummary` (line 901) calls `GET /memories/summary` — not registered in `api.go`.
- `adminCleanup` (line 766) calls `POST /admin/cleanup` — not registered; server has `POST /admin/sync` only.
- All of these will return 404 or 405 in production.

### [HIGH] MCP server exposes only 6 basic tools — all new features are missing
- File: `cmd/mcp-server/main.go` (lines 135–145)
- Exposed: `addMemory`, `recall`, `search`, `whoAmI`, `getMemories`, `deleteMemory`.
- Missing: spreading activation search, compression stats, skill operations, wiki queries, agent/group management, knowledge graph traversal, tier policy, consolidation.
- The `handleWhoAmI` handler (line 268) returns a hardcoded static JSON: `{"userId": "default", "role": "user", "status": "active"}` — it never consults user state from the memory API.

### [MEDIUM] Gateway log output references wrong URLs
- File: `cmd/gateway/main.go` (lines 206–212)
- The log prints `-> %s" g.memoryAPIURL` for `/api/v1/*`, `/memories`, `/search` — but the proxy config maps these to `*monolithURL`, not `*memoryAPIURL`. Both default to `localhost:8081` so the behavior is correct, but the log message is misleading if someone changes one flag and not the other.

### [MEDIUM] Gateway missing routes for new monolith endpoints
- File: `cmd/gateway/main.go`
- Routes proxied to monolith: `/api/v1/`, `/memories`, `/search`, `/skills`, `/agents`, `/projects`, `/chains`, `/reviews`, `/compression/`, `/tier/`.
- NOT proxied: `/sessions`, `/entities`, `/relations`, `/graph`, `/webhooks`, `/wiki`, `/feedback`, `/notifications`, `/analytics`, `/alerts`, `/compact`, `/backup`, `/playground`, `/groups`, `/admin`, `/auth`.
- Any client connecting via the gateway cannot reach these features.

### [LOW] MCP OAuth token endpoint leaks the raw API key as access_token
- File: `cmd/mcp-server/main.go` (lines 382–394)
- `handleOAuthToken` responds with `"access_token": validAPIKey()` and `"refresh_token": validAPIKey()` — the raw API key is the OAuth token, which is not standard OAuth 2.0 behavior and will confuse clients expecting opaque tokens.

---

## SDK INTEGRATION AUDIT

### [HIGH] Python SDK re-exports from `_async.py` but `__init__.py` lacks Playground, Wiki, and Benchmark classes
- File: `sdk/python/hystersis/__init__.py`
- The `__all__` list (lines 65–88) exports: client classes, errors, enums, configs.
- No mention of Wiki, Playground, Spreading Activation, or Benchmark-related types/methods.
- These features exist in the backend (full API routes) but have no Python SDK surface.

### [MEDIUM] Node SDK integrations only define types, no implementations for new features
- File: `sdk/nodejs/src/integrations/types.ts`
- `LangChainMemoryConfig`, `MastraMemoryConfig`, `AgnoMemoryConfig`, `AutoGenMemoryConfig`, `LlamaIndexReaderConfig`, `CrewMemoryConfig` all have `baseUrl` and `apiKey` — but none have fields for compression mode, tier policy, spreading activation, or skills.
- Integration adapters (langchain.ts, llamaindex.ts, etc.) cannot expose these new features to framework users.

### [LOW] Node SDK `LlamaIndexRetriever` alias is incorrect
- File: `sdk/nodejs/src/integrations/index.ts` (line 15)
- `export { HystersisRetriever as LlamaIndexRetriever } from './langchain.js'` — exporting the LangChain retriever aliased as the LlamaIndex one. The actual LlamaIndex retriever is `HystersisLlamaRetriever` exported on line 22. This creates a confusing duplicate that doesn't use LlamaIndex's reader config.

---

## Summary

| Severity | Count | Areas |
|----------|-------|-------|
| CRITICAL | 3 | Demo page (no API), exposed credentials, duplicate route shadows |
| HIGH | 7 | Stripe stub, missing features page content, MCP stubs, Node SDK dead endpoints, memory-api stubs, gateway missing routes, Python SDK gaps |
| MEDIUM | 6 | Blog silent failure, docs link external, static dashboard, gateway log mismatch, MCP whoAmI static, integration configs lack new fields |
| LOW | 4 | PostHog key committed, Amplitude never init, OAuth token = raw key, wrong LlamaIndex alias |
