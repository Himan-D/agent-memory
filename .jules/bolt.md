## 2026-06-08 - [Worker Name Collision — apex serves dashboard]
**Learning:** Renaming root `wrangler.jsonc` worker from `agent-memory` to `hystersis-app` (PR #85) collided with `dashboard/wrangler.jsonc`. Dashboard deploy overwrote landing; `hystersis.com` redirected to `/auth/signin`.
**Action:** Keep worker names unique: `agent-memory` (landing) vs `hystersis-app` (dashboard). Do not deploy dashboard from `build-cloudflare.sh`. See `.github/MULTI_AGENT_COORDINATION.md` before parallel agent edits.

## 2026-06-06 - [Batch Metadata Retrieval]
**Learning:** Identified a classic N+1 query bottleneck in the `SearchMemories` method where metadata for each result was fetched individually. This resulted in significant latency proportional to the number of search results.
**Action:** Use batch retrieval methods like `GetMemoriesByIDs` to fetch all metadata in a single round-trip to the database. Always check for bulk retrieval opportunities when processing collections of identifiers.

## 2026-06-10 - [Batch Embedding Index Mapping Bug]
**Learning:** `GenerateBatchEmbeddingsWithContext` in `openai.go` was incorrectly mapping API results back to the original request indices when cache hits were involved. It was appending results dynamically, which scrambled the association between text and embedding.
**Action:** Pre-allocate the results slice to the full request size and use an explicit index map (e.g., `fetchIndices[i+j]`) to place each newly fetched embedding in its correct stable position alongside cached ones.

## 2026-06-10 - [Redundant Embedding Generation in CreateMemory]
**Learning:** `CreateMemory` was calling `GenerateEmbeddingWithContext` twice: once for semantic deduplication and again for final vector storage. This doubled LLM latency for the write path.
**Action:** Generate the content embedding once at the start of the creation flow and pass it through as a variable to both the search/dedup and store stages.
