## 2026-06-08 - [Worker Name Collision — apex serves dashboard]
**Learning:** Renaming root `wrangler.jsonc` worker from `agent-memory` to `hystersis-app` (PR #85) collided with `dashboard/wrangler.jsonc`. Dashboard deploy overwrote landing; `hystersis.com` redirected to `/auth/signin`.
**Action:** Keep worker names unique: `agent-memory` (landing) vs `hystersis-app` (dashboard). Do not deploy dashboard from `build-cloudflare.sh`. See `.github/MULTI_AGENT_COORDINATION.md` before parallel agent edits.

## 2026-06-06 - [Batch Metadata Retrieval]
**Learning:** Identified a classic N+1 query bottleneck in the `SearchMemories` method where metadata for each result was fetched individually. This resulted in significant latency proportional to the number of search results.
**Action:** Use batch retrieval methods like `GetMemoriesByIDs` to fetch all metadata in a single round-trip to the database. Always check for bulk retrieval opportunities when processing collections of identifiers.

## 2026-06-12 - [Batch Retrieval for Skills and Chains]
**Learning:** Extended the batch retrieval pattern to 'Skills' and 'Chains' in 'internal/memory/service.go'. Sequential retrieval in 'SynthesizeSkills' and 'ExtractChains' was causing linear latency growth ((N)$ round-trips). Batching reduced this to (1)$ round-trips, yielding a ~10x speedup in benchmarks for 10 items.
**Action:** Always prefer batch variants (e.g., 'GetSkillsByIDs') when fetching multiple entities by ID. Ensure result order matches input order and check 'result.Err()' when using Neo4j driver iteration.
