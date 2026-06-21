## 2026-06-08 - [Worker Name Collision — apex serves dashboard]
**Learning:** Renaming root `wrangler.jsonc` worker from `agent-memory` to `hystersis-app` (PR #85) collided with `dashboard/wrangler.jsonc`. Dashboard deploy overwrote landing; `hystersis.com` redirected to `/auth/signin`.
**Action:** Keep worker names unique: `agent-memory` (landing) vs `hystersis-app` (dashboard). Do not deploy dashboard from `build-cloudflare.sh`. See `.github/MULTI_AGENT_COORDINATION.md` before parallel agent edits.

## 2026-06-06 - [Batch Metadata Retrieval]
**Learning:** Identified a classic N+1 query bottleneck in the `SearchMemories` method where metadata for each result was fetched individually. This resulted in significant latency proportional to the number of search results.
**Action:** Use batch retrieval methods like `GetMemoriesByIDs` to fetch all metadata in a single round-trip to the database. Always check for bulk retrieval opportunities when processing collections of identifiers.

## 2026-06-21 - [Prospective Search Latency]
**Learning:** Prospection-guided retrieval (PGR) expands a single query into multiple variants, creating a significant latency bottleneck when processed sequentially. Each variant originally required its own embedding API call and vector DB search.
**Action:** Use batch embedding for all expanded queries and execute vector searches in parallel. This reduces latency from O(N) to roughly O(1) retrieval round-trips.

## 2026-06-21 - [Batch Index Panic]
**Learning:** In `GenerateBatchEmbeddingsWithContext`, mapping batch results back to original indices via a sidecar `indices` slice panics if the external API returns more results than requested (e.g. a misbehaving mock or provider).
**Action:** Always pre-allocate results with the exact requested length and use bounds-checking when mapping indices from external API responses.
