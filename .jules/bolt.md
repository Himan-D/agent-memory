## 2026-06-06 - [Batch Metadata Retrieval]
**Learning:** Identified a classic N+1 query bottleneck in the `SearchMemories` method where metadata for each result was fetched individually. This resulted in significant latency proportional to the number of search results.
**Action:** Use batch retrieval methods like `GetMemoriesByIDs` to fetch all metadata in a single round-trip to the database. Always check for bulk retrieval opportunities when processing collections of identifiers.
