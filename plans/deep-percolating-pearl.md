# Hystersis vs Mem0: Feature Parity Report & Gap Analysis

## Context

This is a comprehensive audit of the Hystersis (agent-memory) codebase compared against Mem0's feature set. The goal: identify what's real, what's stubbed, where Hystersis already exceeds Mem0, and what gaps need closing to reach full parity.

---

## Part 1: How This Code Actually Works

### Architecture Overview

Hystersis is a Go monolith (`cmd/server/api.go` — ~1800+ lines, 100+ routes) backed by:
- **Neo4j** — primary graph store for entities, relations, memories (full Cypher implementation)
- **Qdrant** — vector store via gRPC (real, production-grade)
- **Redis** — tiered memory cache (hot storage layer)
- **OpenAI** — LLM provider for extraction + embeddings

### Memory Processing Pipeline (how data actually flows)

```
Content in → POST /memories or /v3/memories/add
  │
  ├─ Classic Path (MemoryProcessor):
  │   1. shouldStore() — LLM decides if worth keeping (importance + categories)
  │   2. ExtractFacts() — LLM extracts structured facts from content
  │   3. ExtractEntities() — LLM pulls named entities
  │   4. ResolveConflict() — checks graph for contradictions
  │   5. Neo4j: CreateMemory + AddEntity + LinkMemoryEntity
  │   6. Qdrant: embed via OpenAI → upsert vector
  │
  ├─ Pipeline Path (newer, task-based):
  │   1. chunk → split content if CHUNKING_ENABLED
  │   2. embed → OpenAI embeddings per chunk
  │   3. extract → LLM entity/relation extraction
  │   4. graphify → write to Neo4j
  │
  └─ Compression Path (proprietary):
      1. LLMRouter.Route() → complexity heuristic (no LLM call)
      2. Fast path: gpt-4o-mini → JSON fact array
      3. Verify path: gpt-4o-mini first, then claude-3-5-sonnet verifies
      4. Fallback: Huffman/LZ77/radix classical compression
```

### Retrieval Pipeline (how search works)

```
Query in → GET /search or POST /search/hybrid
  │
  ├─ MultiSignalRetrieval (parallel, 3 goroutines):
  │   ├─ Semantic: Qdrant cosine similarity (weight: 0.60)
  │   ├─ Keyword: BM25 over memory corpus (weight: 0.25)
  │   └─ Entity: extract query entities → match (weight: 0.15)
  │
  ├─ Reciprocal Rank Fusion (RRF, k=60):
  │   score(d) = Σ weight_i / (60 + rank_i(d))
  │
  ├─ CompositeScorer (5-signal):
  │   semantic (0.30) + temporal (0.20) + confidence/MW (0.15)
  │   + centrality (0.10) + authority (0.15) - negation penalty (0.10)
  │
  ├─ DecayScorer:
  │   Ebbinghaus forgetting curve: R = e^(-t/S)
  │   Stability extends logarithmically with each access
  │
  └─ Optional: Cohere reranker post-processing
```

### What Is REAL (substantial, working code)

| Component | File | Lines | Verdict |
|---|---|---|---|
| Config | `internal/config/config.go` | 517 | Complete — all env vars |
| API Server | `cmd/server/api.go` | 1800+ | Complete — 100+ routes |
| Memory Service | `internal/memory/service.go` | 700+ | Complete — full business logic |
| Memory Processor | `internal/memory/processor.go` | Real | LLM-backed extraction |
| Neo4j Client | `internal/memory/neo4j/client.go` | Large | Full Cypher, multi-tenant |
| Qdrant Client | `internal/memory/qdrant/client.go` | Real | gRPC, collection mgmt |
| LLM Provider | `internal/llm/providers.go` | Real | OpenAI HTTP calls |
| BM25 | `internal/retrieval/bm25.go` | 150 | Full BM25 with TF-IDF |
| Multi-Signal Retrieval | `internal/retrieval/multisignal.go` | 198 | Parallel RRF fusion |
| Composite Scorer | `internal/memory/scoring/composite.go` | 169 | 5-signal + Wilson CI |
| Decay Scorer | `internal/memory/decay/scorer.go` | 153 | Ebbinghaus curve |
| ProMem Extractor | `internal/compression/extractor/proprietary.go` | 616 | TOON + self-questioning |
| Spreading Activation | `internal/compression/retrieval/proprietary.go` | 661 | Graph walk + decay |
| LLM Router | `internal/compression/llm/router.go` | 347 | Fast/verify dual-path |
| Async Pipeline | `internal/compression/pipeline/async.go` | 303 | Worker pool + job queue |
| Classical Compression | `internal/compression/algorithm/compressor.go` | 497 | Huffman + LZ77 + dictionary |
| Tier Router | `internal/memory/tier/router.go` | Real | Working/Hot/Cold/Archive |
| Benchmarks | `internal/compression/benchmarks/benchmarks.go` | Real | Multi-algorithm comparison |
| Python SDK | `sdk/python/` | Real | LangChain, CrewAI, LlamaIndex, LangGraph, AutoGen |
| Node SDK | `sdk/nodejs/` | Real | LangChain, CrewAI, Vercel AI, Mastra, Agno, OpenAI Agents |

### What Is STUBBED or Partial

| Component | Status |
|---|---|
| Vector providers (Pinecone, Weaviate, Chroma, etc.) | Stubs — only Qdrant is real |
| OpenSearch | Config exists, no client implementation found |
| LLM providers beyond OpenAI | Router references Anthropic but only OpenAI HTTP client confirmed |
| Sleep/dreaming consolidation | Files exist, depth uncertain |
| Self-improvement loop | Files exist, depth uncertain |
| Ontology system | Config-gated, likely partial |
| Cognify / Memify | Files exist, depth uncertain |

---

## Part 2: Feature Parity Matrix — Hystersis vs Mem0

### Legend: existing = already built, partial = exists but incomplete, missing = not implemented, **ahead** = exceeds Mem0

| Feature | Mem0 | Hystersis | Status |
|---|---|---|---|
| **Core Memory CRUD** | add/get/update/delete/get_all | Full CRUD + batch + versions + history | **AHEAD** |
| **Fact Extraction via LLM** | Single structured-output call | Multi-step: shouldStore → ExtractFacts → ExtractEntities | **AHEAD** |
| **Entity Extraction** | LLM extracts, stored in vector collection | LLM extracts, stored in Neo4j graph | **AHEAD** |
| **Conflict Resolution** | LLM decides ADD/UPDATE/DELETE/NONE | ResolveConflict template + graph check | Existing |
| **Hash Deduplication** | MD5 before write | Dedup logic exists in service.go | Existing |
| **Memory History** | Immutable event log | `/memories/{id}/history` + `/versions` endpoints | Existing |
| **Batch Operations** | batch update/delete | batch create/update/delete/bulk-delete | **AHEAD** |
| **Memory Feedback** | feedback endpoint | `/memories/{id}/feedback` | Existing |
| **Memory Export** | export endpoint | Export exists in sync/backup.go | Existing |
| **Session Management** | run_id scoping | Full session CRUD + messages + context | **AHEAD** |
| **Entity Scoping** | user_id, agent_id, run_id, app_id | user_id, agent_id, org_id + memory_type | Existing |
| **Semantic Search** | Cosine similarity (4x overfetch) | Qdrant cosine similarity | Existing |
| **BM25 Keyword Search** | Built-in hybrid | `internal/retrieval/bm25.go` — real | Existing |
| **Hybrid Retrieval** | semantic + BM25 + entity boost | RRF fusion: semantic(0.60) + keyword(0.25) + entity(0.15) | Existing |
| **Memory Decay** | Opt-in, search-time reorder | Ebbinghaus curve + access boost + recency | **AHEAD** |
| **Score Explanation** | `explain` param returns breakdown | CompositeScorer returns full signal breakdown | Existing |
| **Reranking** | Cohere, SentenceTransformers, etc. | `internal/reranker/` exists (Cohere) | Existing |
| **`infer=False` Verbatim** | Skip LLM, store raw | Config flag `processing_enabled` / `auto_extract_facts` | Existing |
| **Custom Instructions** | Per-request prompt override | Templates system, less dynamic per-request | **PARTIAL** |
| **Graph Memory** | **REMOVED** (entity linking only) | **Full Neo4j** — Cypher queries, traversal, relations | **AHEAD** |
| **Knowledge Graph Traversal** | Not available (deprecated) | `/graph/traverse/{entityID}`, `/graph/query` | **AHEAD** |
| **Tiered Memory** | Single collection | Working → Hot → Cold → Archive | **AHEAD** |
| **Compression Engine** | None | ProMem + spreading activation + classical | **AHEAD** |
| **Async Processing** | Platform only (event polling) | Async compression pipeline | Existing |
| **MCP Server** | Built-in MCP server | `.well-known/mcp/server-card.json` endpoint | Existing |
| **Agent Discovery** | Not available | `.well-known/agent-skills`, `llms.txt`, `agents.md` | **AHEAD** |
| **Temporal Reasoning** | `reference_date` for queries | `internal/memory/temporal/reasoning.go` | Existing |
| **Privacy Filtering** | Not prominent | `internal/memory/privacy/filter.go` | **AHEAD** |
| **Observability** | Basic | Prometheus + OpenTelemetry + Pyroscope profiling | **AHEAD** |
| **Auth** | API key | JWT + API key + LDAP + SSO (Google/GitHub OAuth) | **AHEAD** |
| **Billing** | Platform SaaS | Stripe integration | **AHEAD** |
| **Helm/K8s** | Docker | Helm chart in `deploy/helm/` | **AHEAD** |

### Where Hystersis is BEHIND Mem0

| Gap | Mem0 Has | Hystersis Has | Severity |
|---|---|---|---|
| **Vector Store Backends** | 20+ (Qdrant, Pinecone, Chroma, PGVector, Milvus, Weaviate, FAISS, Redis, MongoDB, Supabase, etc.) | Only Qdrant is real | **HIGH** — limits adoption |
| **LLM Provider Breadth** | 20+ (OpenAI, Anthropic, Groq, Mistral, Ollama, LiteLLM, vLLM, Bedrock, Vertex, etc.) | Only OpenAI confirmed real | **HIGH** — vendor lock-in |
| **Embedding Provider Breadth** | OpenAI, Azure, Bedrock, Vertex, HuggingFace, Ollama, LM Studio, Together AI | Only OpenAI confirmed | **MEDIUM** |
| **Per-Request Custom Instructions** | `custom_instructions` param overrides extraction prompt per add() call | Templates are static, no per-request override | **MEDIUM** |
| **v3 Additive Extraction** | Single-pass ADD-only prompt with per-fact attribution + `linked_memory_ids` | Older multi-step pattern | **LOW** — Hystersis approach is arguably better |
| **Entity Dedup Threshold** | Cosine 0.95 auto-merges similar entities | Unknown if implemented | **MEDIUM** |
| **CLI Tools** | `mem0` CLI (Python + Node) | `cmd/cli/commands.go` exists but depth unknown | **LOW** |
| **`reset()` Method** | Wipe all memories for a scope | Bulk delete exists, no single reset endpoint | **LOW** |

---

## Part 3: Recommended Implementation Plan (Parity Gaps)

### Phase 1: Vector Store Backends (HIGH priority)

Add at least 3 more production-grade vector providers to break Qdrant lock-in:

**Files to create/modify:**
- `internal/vector/provider.go` — shared interface (may already exist as `internal/memory/store.go` VectorStore interface)
- `internal/vector/pinecone/client.go` — Pinecone via REST API
- `internal/vector/pgvector/client.go` — pgvector via `pgx`
- `internal/vector/chroma/client.go` — Chroma via REST
- `internal/config/config.go` — add provider configs

**Pattern to follow:** Mirror `internal/memory/qdrant/client.go` structure. Each provider implements the existing `VectorStore` interface from `internal/memory/store.go`.

### Phase 2: LLM Provider Breadth (HIGH priority)

Extend `internal/llm/providers.go` to support at minimum:

1. **Anthropic** — the router already references claude-3-5-sonnet; formalize the HTTP client
2. **Ollama** — local model support (OpenAI-compatible endpoint)
3. **LiteLLM** — proxy that gives you 100+ providers for free (OpenAI-compatible)
4. **Google Vertex AI** — enterprise demand

**Files to modify:**
- `internal/llm/providers.go` — add `anthropicProvider`, `ollamaProvider`
- `internal/llm/provider.go` — ensure interface covers streaming if needed
- `internal/config/config.go` — add provider selection logic

**Shortcut:** Since most providers are OpenAI-compatible, the existing `openaiProvider` just needs configurable `base_url` + model override. Anthropic requires its own HTTP format.

### Phase 3: Embedding Provider Breadth (MEDIUM priority)

**Files to modify:**
- `internal/embedding/` — currently only OpenAI; add Ollama, Cohere, HuggingFace
- Same pattern: interface + implementations

### Phase 4: Per-Request Custom Instructions (MEDIUM priority)

Allow `custom_instructions` field on POST `/memories` and POST `/v3/memories/add` to override the extraction prompt.

**Files to modify:**
- `internal/memory/types/types.go` — add `CustomInstructions` field to request type
- `internal/memory/processor.go` — inject custom instructions into extraction template
- `internal/memory/templates.go` — template should accept optional override section
- `cmd/server/api.go` — parse `custom_instructions` from request body

### Phase 5: Entity Dedup Threshold (MEDIUM priority)

Before inserting a new entity, check Qdrant for cosine similarity > 0.95 against existing entities; if found, merge instead of creating duplicate.

**Files to modify:**
- `internal/memory/service.go` or entity extraction pipeline
- `internal/memory/qdrant/client.go` — add entity similarity search method

### Phase 6: Quality of Life (LOW priority)

- **`reset()` endpoint** — `DELETE /memories/reset?user_id=X` wipes all for a scope
- **CLI depth** — verify `cmd/cli/commands.go` covers add/search/delete/export
- **SDK tests** — `sdk/python/tests/test_hystersis.py` exists but coverage unknown

---

## Part 4: Test Coverage Gaps

| Area | Has Tests | Needs Tests |
|---|---|---|
| Compression (radix, router, pipeline, smart, benchmarks) | Yes | Good coverage |
| Neo4j validation (injection prevention) | Yes | Needs CRUD operation tests |
| Config loading | Yes | Good |
| Memory decay scorer | Yes | Good |
| Temporal reasoning | Yes | Good |
| Privacy filter | Yes | Good |
| **Memory Service** | **No** | **Critical — largest file, untested** |
| **API handlers** | **Partial** (in worktrees) | **Need integration tests on main** |
| **Memory Processor** | **No** | **High priority — LLM extraction logic** |
| **Multi-signal retrieval** | Yes (test file exists) | Verify depth |
| **BM25** | **No dedicated test** | Add unit tests |

---

## Verification Plan

After implementing the parity gaps:

```bash
# Build
go build ./...

# Unit tests
go test ./...

# Verify new vector providers
go test ./internal/vector/...

# Verify LLM providers
go test ./internal/llm/...

# Integration test (requires running services)
# Start server: go run cmd/server/api.go
# Hit endpoints:
curl -X POST localhost:8080/memories -d '{"content":"test","type":"user"}'
curl localhost:8080/search?q=test
curl localhost:8080/memories
```

---

## Summary

**Hystersis is already significantly ahead of Mem0 in core capabilities.** The compression engine, graph memory, tiered storage, composite scoring, and observability stack are features Mem0 doesn't have at all. The real gaps are **ecosystem breadth** — vector store backends and LLM provider support — which are integration work, not algorithmic challenges. Closing the top 2 gaps (vector stores + LLM providers) would bring Hystersis to full parity while maintaining its existing advantages.
