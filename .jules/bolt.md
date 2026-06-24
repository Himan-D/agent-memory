## 2026-06-08 - [Worker Name Collision — apex serves dashboard]
**Learning:** Renaming root `wrangler.jsonc` worker from `agent-memory` to `hystersis-app` (PR #85) collided with `dashboard/wrangler.jsonc`. Dashboard deploy overwrote landing; `hystersis.com` redirected to `/auth/signin`.
**Action:** Keep worker names unique: `agent-memory` (landing) vs `hystersis-app` (dashboard). Do not deploy dashboard from `build-cloudflare.sh`. See `.github/MULTI_AGENT_COORDINATION.md` before parallel agent edits.

## 2026-06-06 - [Batch Metadata Retrieval]
**Learning:** Identified a classic N+1 query bottleneck in the `SearchMemories` method where metadata for each result was fetched individually. This resulted in significant latency proportional to the number of search results.
**Action:** Use batch retrieval methods like `GetMemoriesByIDs` to fetch all metadata in a single round-trip to the database. Always check for bulk retrieval opportunities when processing collections of identifiers.

## 2026-06-12 - [Parallel Vector Search & Batch Embeddings]
**Learning:** Sequential embedding generation and vector searches for query expansion (PGR) were causing O(N) latency bottlenecks. Batching embeddings reduces API round-trips, and parallelizing vector searches with goroutines brings retrieval latency down to O(1) relative to expansion steps.
**Action:** Always batch embedding requests when multiple texts are known upfront, and use parallel goroutines with proper synchronization (Mutex/WaitGroup) for independent vector database lookups.
