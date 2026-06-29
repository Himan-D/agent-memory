## 2026-06-08 - [Worker Name Collision — apex serves dashboard]
**Learning:** Renaming root `wrangler.jsonc` worker from `agent-memory` to `hystersis-app` (PR #85) collided with `dashboard/wrangler.jsonc`. Dashboard deploy overwrote landing; `hystersis.com` redirected to `/auth/signin`.
**Action:** Keep worker names unique: `agent-memory` (landing) vs `hystersis-app` (dashboard). Do not deploy dashboard from `build-cloudflare.sh`. See `.github/MULTI_AGENT_COORDINATION.md` before parallel agent edits.

## 2026-06-06 - [Batch Metadata Retrieval]
**Learning:** Identified a classic N+1 query bottleneck in the `SearchMemories` method where metadata for each result was fetched individually. This resulted in significant latency proportional to the number of search results.
**Action:** Use batch retrieval methods like `GetMemoriesByIDs` to fetch all metadata in a single round-trip to the database. Always check for bulk retrieval opportunities when processing collections of identifiers.

## 2026-06-09 - [Parallel Multi-Query Retrieval]
**Learning:** Sequential processing of expanded search queries (PGR paper) created a major latency bottleneck. By batching embeddings and parallelizing vector searches with goroutines, I reduced the retrieval phase from O(N) to O(1) in terms of round-trip wait time.
**Action:** Always parallelize independent retrieval signals. Use `sync.WaitGroup` for concurrency and `sync.Mutex` to safely collect results into shared slices when performing multi-query or multi-vector searches.
