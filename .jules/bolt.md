## 2026-06-08 - [Worker Name Collision — apex serves dashboard]
**Learning:** Renaming root `wrangler.jsonc` worker from `agent-memory` to `agent-memorydash` (PR #85) collided with `dashboard/wrangler.jsonc`. Dashboard deploy overwrote landing; `hystersis.com` redirected to `/auth/signin`.
**Action:** Keep worker names unique: `agent-memory` (landing) vs `agent-memorydash` (dashboard). Do not deploy dashboard from `build-cloudflare.sh`. See `.github/MULTI_AGENT_COORDINATION.md` before parallel agent edits.

## 2026-06-06 - [Batch Metadata Retrieval]
**Learning:** Identified a classic N+1 query bottleneck in the `SearchMemories` method where metadata for each result was fetched individually. This resulted in significant latency proportional to the number of search results.
**Action:** Use batch retrieval methods like `GetMemoriesByIDs` to fetch all metadata in a single round-trip to the database. Always check for bulk retrieval opportunities when processing collections of identifiers.

## 2026-06-10 - [Parallel Retrieval & Batch Embeddings in Search]
**Learning:** For multi-query search (prospection/query expansion), sequential LLM API calls for embeddings and sequential vector searches are the primary bottlenecks. Latency was $O(N)$ where $N$ is the number of expanded queries.
**Action:** Use `GenerateBatchEmbeddingsWithContext` to collapse $N$ API calls into 1. Parallelize vector searches using goroutines. Ensure thread-safety for shared state (like deduplication maps) with a `sync.Mutex`.

## 2026-07-01 - [Batch Embedding Integrity & O(1) LRU Cache]
**Learning:** Found a critical data integrity bug in `GenerateBatchEmbeddingsWithContext` where mixing cache hits and misses scrambled the result order. Also identified an $O(N)$ bottleneck in the LRU cache due to slice-based list management.
**Action:** Use pre-allocated slices and direct indexing to ensure input-output alignment in batch operations. Implement LRU caches using `container/list` for $O(1)$ eviction and move-to-front performance. Always verify cache hit/miss interleaving with tests.

## 2026-07-01 - [Node.js 22 Requirement for Cloudflare Tools]
**Learning:** Modern versions of `wrangler` (v4+), `kysely`, and Cloudflare asset handlers now require Node.js >= 22.0.0. Using Node.js 20 in CI triggers `EBADENGINE` warnings and build failures.
**Action:** Ensure `NODE_VERSION` is set to at least '22' in all GitHub Action workflows (`ci.yml`, `deploy-cloudflare.yml`) to maintain compatibility with the latest deployment tooling.

## 2026-07-02 - [Batch Message Flushing in Neo4j]
**Learning:** The `MessageBuffer` was flushing session messages to Neo4j sequentially, resulting in (N)$ round-trips per flush. This pattern significantly increases latency during buffer drain or shutdown.
**Action:** Implement batch insertion using Neo4j's `UNWIND` clause. Introduce `AddMessages` to the `GraphStore` interface to support bulk writes, reducing round-trips to (1)$ per session flush.
