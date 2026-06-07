# ADR-003: Proprietary Compression Engine Boundary

**Status**: Accepted  
**Date**: 2026-06-07  
**Deciders**: Engineering, Product  

## Context

Hystersis competes with Mem0 on memory quality and compression. The compression engine (ProMem extraction, spreading activation, LLM routing) is the primary differentiator. We must balance open-source community growth with competitive IP protection.

## Options Considered

| Option | Open Source | Competitive Moat | Community Trust |
|--------|------------|------------------|-----------------|
| A. Fully open source everything | 100% | None | Maximum |
| B. Fully closed source | 0% | Maximum | Low adoption |
| C. Open core with proprietary compression | ~70% | Strong | Balanced |

## Decision

**Option C: Open core** with a hard boundary around proprietary algorithms.

### Open Source (MIT)
- Basic `compression.go` (summarization)
- Vector similarity search
- Neo4j/Qdrant storage adapters
- REST API endpoints and configuration
- SDKs and dashboard UI

### Proprietary (not distributed in public repos)
- `internal/compression/extractor/` — ProMem self-questioning extraction
- `internal/compression/retrieval/` — Spreading activation hyperparameters
- `internal/compression/llm/router.go` — Hybrid fast/verify routing
- Benchmark implementation code

## Consequences

**Positive**
- Developers can self-host the platform and extend via APIs
- Marketing can cite benchmark targets without exposing implementation
- Agents must follow boundary rules in `AGENTS.md` and `CLAUDE.md`

**Negative**
- Contributors cannot improve proprietary algorithms via public PRs
- Requires clear documentation of what is and is not open

## Benchmark Targets

| Metric | Target |
|--------|--------|
| Accuracy retention | ≥97% |
| Token reduction | 80–85% |
| Multi-hop reasoning | +23% vs pure vector |
| P95 latency | <200ms |
| Write impact | <5ms (async) |
