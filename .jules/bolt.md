## 2026-06-08 - [Worker Name Collision — apex serves dashboard]
**Learning:** Renaming root `wrangler.jsonc` worker from `agent-memory` to `agent-memorydash` (PR #85) collided with `dashboard/wrangler.jsonc`. Dashboard deploy overwrote landing; `hystersis.com` redirected to `/auth/signin`.
**Action:** Keep worker names unique: `agent-memory` (landing) vs `agent-memorydash` (dashboard). Do not deploy dashboard from `build-cloudflare.sh`. See `.github/MULTI_AGENT_COORDINATION.md` before parallel agent edits.

## 2026-06-06 - [Batch Metadata Retrieval]
**Learning:** Identified a classic N+1 query bottleneck in the `SearchMemories` method where metadata for each result was fetched individually. This resulted in significant latency proportional to the number of search results.
**Action:** Use batch retrieval methods like `GetMemoriesByIDs` to fetch all metadata in a single round-trip to the database. Always check for bulk retrieval opportunities when processing collections of identifiers.

## 2026-06-09 - [Batch Embedding & Parallel Search]
**Learning:** Sequential processing of expanded queries (PGR paper) created a latency bottleneck proportional to the number of expansions (up to 5x). Even when using batching, a subtle bug in mapping cached vs newly fetched results can cause data corruption.
**Action:** Use `GenerateBatchEmbeddingsWithContext` to collapse all query embeddings into one API round-trip. Execute vector searches in parallel using goroutines. Always pre-allocate slices for batch operations to ensure deterministic index mapping.

## 2026-06-09 - [Worker Naming & Build Stability]
**Learning:** Cloudflare Workers Builds expects a specific worker name (`agent-memorydash` in this case). Mismatches between the codebase and Cloudflare's project settings cause CI failures. Additionally, Next.js build-time initialization of auth libraries (like Better-Auth) can crash builds if secrets are missing.
**Action:** Always align worker names across wrangler.jsonc, scripts, and documentation. Provide safe placeholders for mandatory environment variables in build scripts (`scripts/deploy-dashboard-builds.sh`) and fallback logic in code (`dashboard/src/lib/auth.ts`) to ensure build-time stability.
