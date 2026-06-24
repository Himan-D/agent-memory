## 2026-06-08 - [Worker Name Collision — apex serves dashboard]
**Learning:** Renaming root `wrangler.jsonc` worker from `agent-memory` to `hystersis-app` (PR #85) collided with `dashboard/wrangler.jsonc`. Dashboard deploy overwrote landing; `hystersis.com` redirected to `/auth/signin`.
**Action:** Keep worker names unique: `agent-memory` (landing) vs `hystersis-app` (dashboard). Do not deploy dashboard from `build-cloudflare.sh`. See `.github/MULTI_AGENT_COORDINATION.md` before parallel agent edits.

## 2026-06-06 - [Batch Metadata Retrieval]
**Learning:** Identified a classic N+1 query bottleneck in the `SearchMemories` method where metadata for each result was fetched individually. This resulted in significant latency proportional to the number of search results.
**Action:** Use batch retrieval methods like `GetMemoriesByIDs` to fetch all metadata in a single round-trip to the database. Always check for bulk retrieval opportunities when processing collections of identifiers.

## 2026-06-17 - [Parallel Retrieval and Batch Embedding]
**Learning:** Sequential embedding and vector search for expanded queries (PGR) created a significant bottleneck. Parallelizing these operations reduces retrieval latency from O(N) to O(1) round-trips.
**Action:** Always use batch embedding methods and parallelize independent vector searches. Ensure shared mutable state (like filter maps) is defensively copied for each goroutine to prevent race conditions.
