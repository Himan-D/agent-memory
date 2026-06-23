## 2026-06-08 - [Worker Name Collision — apex serves dashboard]
**Learning:** Renaming root `wrangler.jsonc` worker from `agent-memory` to `hystersis-app` (PR #85) collided with `dashboard/wrangler.jsonc`. Dashboard deploy overwrote landing; `hystersis.com` redirected to `/auth/signin`.
**Action:** Keep worker names unique: `agent-memory` (landing) vs `hystersis-app` (dashboard). Do not deploy dashboard from `build-cloudflare.sh`. See `.github/MULTI_AGENT_COORDINATION.md` before parallel agent edits.

## 2026-06-06 - [Batch Metadata Retrieval]
**Learning:** Identified a classic N+1 query bottleneck in the `SearchMemories` method where metadata for each result was fetched individually. This resulted in significant latency proportional to the number of search results.
**Action:** Use batch retrieval methods like `GetMemoriesByIDs` to fetch all metadata in a single round-trip to the database. Always check for bulk retrieval opportunities when processing collections of identifiers.

## 2026-06-23 - [Parallel Query Expansion Retrieval]
**Learning:** Query expansion (Prospection-guided retrieval) significantly increases recall but also latency when embeddings and vector searches are performed sequentially. In this case, 6 sequential queries took ~58ms (mocked).
**Action:** Use `GenerateBatchEmbeddingsWithContext` for all expanded queries and parallelize `VectorStore.Search` calls using goroutines. This achieved a ~4x speedup (~58ms -> ~14ms) in benchmarks.
