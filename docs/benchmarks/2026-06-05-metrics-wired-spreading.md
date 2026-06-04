# Benchmark: Metrics Wired + Working Processor + Spreading Activation

**Date**: 2026-06-05  
**Config**: `gpt-4o-mini` (scorer + processor), `text-embedding-3-small` (embeddings),  
  Qdrant vector store, Neo4j graph store, concurrency=3, mode=spreading (3 hops, decay=0.85)

## What Changed

1. **Processor model fix** — All 10 LLM methods in `processor.go` now use `p.defaultModel` instead of `"defaultModel"`  
2. **Metrics wiring** — All 5 recording methods wired into compression pipeline, spreading activation, tier router, and extractor  
3. **Env config fix** — `OPENAI_MODEL=text-embedding-3-small` for embeddings (separate from `LLM_MODEL=gpt-4o-mini`)

## Results

| Dataset | Overall | Single-Hop | Multi-Hop | P50 Latency | P95 Latency | Tokens |
|---------|---------|-----------|-----------|-------------|-------------|--------|
| locomo | **0.82** | **0.89** | 0.65 | 2712ms | 3470ms | 9 |
| longmemeval | **0.82** | **0.90** | 0.00 | 2414ms | 3533ms | 9 |
| es_memeval | **0.82** | **0.86** | **0.75** | 2141ms | 2523ms | 11 |
| beam_1m | **0.89** | **0.87** | **1.00** | 119ms | 3785ms | 8 |

## Comparison vs Baseline (2026-06-04, broken processor)

| Dataset | Baseline | This Run | Δ |
|---------|----------|----------|---|
| locomo | 0.83 | 0.82 | -0.01 |
| longmemeval | 0.82 | 0.82 | 0.00 |
| es_memeval | 0.81 | 0.82 | +0.01 |

## Key Observations

- **Processor works correctly now** — entity extraction, relation extraction, fact extraction all functional  
- **Scores unchanged** — Working processor replaces memory content with extracted facts, but retrieval scores remain essentially the same vs raw content  
- **Latency ~50% higher** — The working processor adds 5-6 LLM calls per memory (extractFacts, extractEntities, extractRelations, extractCategories, etc.), increasing P50 from ~1800ms to ~2400ms  
- **Spreading activation still sparse** — The Neo4j graph has too few entity-entity edges for meaningful propagation; SA benefit is marginal  
- **beam_1m scores high (0.89)** — Only the 1M scale was tested; very fast P50 (119ms)
