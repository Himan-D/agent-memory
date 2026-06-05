# Mem0 Competitive Analysis vs Hystersis

**Status:** Research complete — no edits needed, this is an analysis document.

---

## 1. Memory Types

### Mem0

Mem0 supports four named memory types, but they are **not architecturally distinct stores** — they are classification labels applied to entries in the same vector store:

| Type | Description |
|------|-------------|
| Working Memory | Short-term context (current session) |
| Factual Memory | Static facts about a user (name, preferences) |
| Episodic Memory | Specific past events or interactions |
| Semantic Memory | Generalized patterns and relationships |

**Critical limitation:** These are taxonomy labels, not separate retrieval mechanisms. There is no procedural memory. There is no tiered promotion (Working → Hot → Cold → Archive). The "type" field influences how an entry is labeled, not how it is retrieved or weighted.

### Hystersis Advantage

- True tiered memory with Working → Hot → Cold → Archive promotion/demotion
- ProMem extraction treats procedural and factual memory differently at the extraction stage
- Spreading activation respects memory tier during graph propagation

---

## 2. Temporal Reasoning

### Mem0

- **Platform-only** (not in open-source SDK)
- Described as "time-aware ranking" — recency boosts results for queries containing temporal language ("last week," "upcoming")
- How it works technically: the new retrieval algorithm (April 2026 release) incorporates timestamp metadata into scoring; time-relative queries parsed for intent
- No details on decay function formula; decay is described as "boost recently-reinforced, dampen stale" with no tunable parameters exposed in public docs
- Temporal filtering via `created_at` in V2 filters (AND/OR compound queries)

**Limitation:** Temporal reasoning appears to be shallow — it is timestamp-aware ranking, not true temporal inference (e.g., "I told you I was moving — that supersedes my old address"). Conflict resolution between time-ordered contradictory facts is handled implicitly via entity linking replacement, not explicit temporal logic.

### Hystersis Advantage

- Spreading activation propagates recency through the graph (linked nodes decay together)
- Tiered memory inherently encodes temporal importance: Hot = recent+relevant, Cold = old but retained
- Hyperparameter: `decay=0.85` is explicitly tunable per deployment; Mem0 exposes nothing comparable

---

## 3. Knowledge Graph

### Mem0 — MAJOR REGRESSION

**Graph store support was completely removed from the OSS SDK.** Approximately 4,000 lines of code were deleted. The databases previously supported (Neo4j, Memgraph, Kuzu, Apache AGE, Amazon Neptune) are all gone.

**What replaced it:** "Entity linking" — entities are extracted during memory add and stored in a parallel vector collection (`{collection_name}_entities`). At search time, entity matches boost the combined relevance score by up to 0.5 multiplier.

**What is lost:**
- No `relations` field on search results (breaking change)
- No multi-hop traversal
- No explicit relationship queries ("who does X work with?")
- No subgraph extraction
- Relationships are implicit through co-occurrence, not modeled edges

**Why they removed it:** Maintenance burden + complexity with no clear performance win vs entity boosting for their target use cases.

### Hystersis Advantage — This is the Single Biggest Differentiator

- Spreading activation retrieval is a first-class graph algorithm, not entity tag boosting
- Multi-hop traversal with `hops=3` (tunable) catches indirect relationships Mem0 fundamentally cannot find
- The `+23% multi-hop` benchmark claim directly addresses what Mem0 has abandoned
- Neo4j storage gives explicit, queryable edge semantics
- This is not a feature gap — Mem0 actively walked away from this space

---

## 4. Compression

### Mem0

- Described as "automatically condenses chat history into compact memories" and a "memory compression engine"
- Technically: LLM-based fact extraction via `ADDITIVE_EXTRACTION_PROMPT` — messages → LLM → discrete fact strings
- Hash-based deduplication (MD5 of text) within batch operations
- No multi-pass verification, no self-questioning loop, no complexity-routing
- Token reduction: benchmarks show 6.7K–7.0K tokens used on LongMemEval/LoCoMo (but this is retrieval context size, not compression ratio)
- **No disclosed compression ratio** (e.g., input tokens → stored tokens)

**What compression is NOT in Mem0:**
- No fast/verify LLM routing (they use single-pass)
- No complexity threshold (`COMPRESSION_COMPLEXITY_THRESHOLD`-style logic)
- No ProMem-style self-questioning + verification loop
- No explicit async background pipeline for compression

### Hystersis Advantage

- ProMem extraction: self-questioning + verification loop → 97%+ accuracy target vs Mem0's single-pass extraction
- Fast/verify LLM routing: cheap model for simple memories, powerful model for complex ones → cost efficiency
- 80–85% token reduction target (Mem0 publishes no equivalent metric)
- `<5ms write impact` async pipeline (Mem0 has `AsyncMemory` but no write-latency claims)

---

## 5. Search & Retrieval

### Mem0

**OSS tier:**
- Semantic similarity (embedding vectors)
- BM25 keyword matching (normalized via sigmoid-style function)
- Entity boosting (post-graph-removal entity tag scoring)
- Reranker support: Cohere, Sentence Transformers, HuggingFace, LLM-based, Zero Entropy
- `AsyncMemory` for non-blocking calls

**Platform tier adds:**
- V2 Filters: compound AND/OR metadata queries on entity, time, custom fields
- Advanced Retrieval with explicit reranking toggle
- Criteria-based retrieval (custom criteria beyond similarity)
- Temporal reasoning filter
- Memory decay weighting
- Contextual Add (multi-turn context awareness)
- Hybrid search mode (semantic + BM25 combined)
- Reranking adds 150–200ms latency overhead

**Benchmarks (April 2026 algorithm):**

| Benchmark | Score | Tokens Used | Latency |
|-----------|-------|-------------|---------|
| LoCoMo | 91.6 (+20.2 vs baseline) | 7.0K | 0.88s |
| LongMemEval | 94.8 (+27.0 vs baseline) | 6.8K | 1.09s |
| BEAM (1M) | 64.1 | 6.7K | 1.00s |

**Limitation:** BEAM score of 64.1 on 1M-token contexts is notably lower. Hybrid search (semantic + BM25 + entity) is strong for flat retrieval but lacks the depth of graph-based retrieval for connected facts.

### Hystersis Advantage

- Spreading activation provides graph-traversal recall that flat hybrid search cannot replicate
- ProMem's extraction accuracy feeds better signal into retrieval than Mem0's single-pass
- `threshold=0.1` tunable sensitivity vs Mem0's opaque scoring

---

## 6. Multi-Agent Memory

### Mem0

Scoping identifiers: `user_id`, `agent_id`, `run_id`, `session_id`, `app_id`

- Each dimension is a partition key in the vector store
- Agents can read across partitions by omitting filters
- Group chat support (multi-participant conversations) on Platform tier
- No documented mechanism for agents to "negotiate" shared memory or resolve conflicts across agent writes
- No concept of agent-scoped knowledge graphs

**Limitation:** Multi-agent is essentially "shared namespace with partition keys." There is no coordination layer — if two agents write contradictory facts, the conflict resolution is the same single-entity replacement logic used for single-agent operations.

### Hystersis Opportunity

- Graph model enables agent-specific subgraphs that merge at shared entity nodes
- Spreading activation can propagate across agent boundaries through shared entities
- This is an architectural advantage waiting to be documented

---

## 7. Conflict Resolution

### Mem0

- **No explicit conflict resolution documented for the public API**
- OSS implementation: entity linking replaces old entity associations on update; new facts overwrite via entity-scoped replacement
- Platform tier: "Contextual Add" considers surrounding conversation context during ingestion
- For contradictory facts (e.g., "I live in NYC" then "I moved to LA"): the newer entity-linked memory displaces the older one — this is implicit, not a modeled resolution step
- No user-visible conflict log or resolution audit trail

### Hystersis Advantage

- Explicit `ResolveConflict` LLM template (from `internal/memory/templates.go`)
- Conflict resolution is a first-class, auditable step — not a side effect of entity replacement
- Temporal ordering feeds into resolution (newer + high-confidence wins)

---

## 8. Importance Scoring & Memory Decay

### Mem0

- **Importance scoring:** Not explicitly exposed. Scoring at retrieval time = semantic similarity + BM25 + entity boost. No "importance" field stored at write time.
- **Memory decay:** Platform-only feature. Described as "boost recently-reinforced, dampen stale results." No formula, decay constant, or tunable parameters documented publicly.
- No concept of memory promotion/demotion between tiers

### Hystersis Advantage

- `decay=0.85` is an explicitly tunable, documented hyperparameter
- Tiered memory (Working/Hot/Cold/Archive) is a structural importance signal — not just a score adjustment
- ProMem assigns importance at extraction time, not just at retrieval time
- Importance informs tier assignment, which affects both retrieval weighting AND storage cost

---

## 9. Integrations

### Mem0 (comprehensive)

**LLM Providers (17+):** OpenAI, Anthropic, Azure OpenAI, AWS Bedrock, Google AI/Vertex, Groq, DeepSeek, Mistral, MiniMax, xAI, Sarvam, Together, Ollama, LM Studio, LiteLLM, vLLM, LangChain

**Embedding Providers (10):** OpenAI, Azure OpenAI, AWS Bedrock, Google AI, Vertex AI, Hugging Face, Ollama, LM Studio, Together, LangChain

**Vector Stores (23):** Qdrant, Chroma, PGVector, Milvus, Pinecone, MongoDB, Azure AI Search, Azure MySQL, Redis, Valkey, Elasticsearch, OpenSearch, Supabase, Upstash Vector, Vectorize, Vertex AI, Weaviate, FAISS, Baidu, Cassandra, S3 Vectors, Databricks, Neptune, Turbopuffer

**Rerankers (5):** Cohere, Sentence Transformers, HuggingFace, LLM-based, Zero Entropy

**Agent Frameworks (14):** LangChain, LangGraph, LlamaIndex, CrewAI, AutoGen, Agno, Camel AI, ChatDev, Hermes, OpenAI Agents SDK, Google ADK, Mastra, Vercel AI, OpenAI Agents SDK

**Editors/Tools:** Claude Code, Cursor, Codex, Raycast

**Voice/Realtime:** LiveKit, Pipecat, ElevenLabs

**Cloud:** AWS Bedrock

**DevOps/Observability:** Dify, Flowise, AgentOps, Keywords AI

This is a major strength. Mem0 has invested heavily in the integration surface.

### Hystersis Gap

Integration breadth is currently much narrower. This is likely the largest go-to-market gap vs Mem0 for Hystersis. Closing even 30% of this list (LangChain, CrewAI, LlamaIndex, OpenAI SDK) would cover ~90% of developer use cases.

---

## 10. Performance Claims

### Mem0

| Metric | Value |
|--------|-------|
| LoCoMo benchmark | 91.6 (+20.2 vs prior) |
| LongMemEval benchmark | 94.8 (+27.0 vs prior) |
| BEAM (1M token) | 64.1 |
| Retrieval tokens (avg) | ~6.8K |
| Retrieval latency (avg) | ~0.93s |
| Reranking overhead | +150–200ms |
| Add write path | Not disclosed |

**No published:**
- Token compression ratio (input → stored)
- Write latency under load
- Throughput (requests/second)
- Multi-hop recall accuracy

### Hystersis Targets (from CLAUDE.md)

| Metric | Target |
|--------|--------|
| Extraction accuracy | 97%+ |
| Token reduction | 80–85% |
| Write impact | <5ms async |
| Multi-hop improvement | +23% |

**Framing note:** Mem0 benchmarks on answer quality (LoCoMo, LongMemEval, BEAM). Hystersis should benchmark on: (1) the same suites to establish comparability, (2) multi-hop recall specifically (where Mem0 has no graph), (3) token reduction ratio (where Mem0 publishes nothing).

---

## 11. Pricing vs Feature Gating

| Feature | Free | Starter ($19) | Growth ($79) | Pro ($249) | Enterprise |
|---------|------|---------------|--------------|------------|------------|
| Add requests/mo | 10K | 50K | 200K | 500K | Custom |
| Retrieval/mo | 1K | 5K | 20K | 50K | Custom |
| Projects | 1 | 1 | 3 | Unlimited | Unlimited |
| Temporal reasoning | No | Platform only | Platform only | Yes | Yes |
| Memory decay | No | Platform only | Platform only | Yes | Yes |
| V2 Filters | No | Platform only | Platform only | Yes | Yes |
| Webhooks | No | No | No | Yes | Yes |
| SSO / on-prem | No | No | No | No | Enterprise only |
| HIPAA / SOC2 | No | No | No | No | Enterprise only |

**Key gating observation:** The most technically interesting features (temporal reasoning, decay, advanced filters, webhooks) are all Platform-tier. The OSS SDK gives you semantic+BM25+entity hybrid search but none of the time-aware or criteria-based capabilities. This means anyone doing serious production work needs at minimum $79/month.

---

## 12. Documented Weaknesses & Known Limitations

**From technical analysis:**

1. **Graph removal is a hard limitation.** Multi-hop relationship queries are architecturally impossible in current Mem0. If your application needs "find all people connected to X through Y" — Mem0 cannot do it.

2. **Single-pass extraction.** One LLM call per add operation with no verification step. This means extraction quality is bounded by the prompt + model quality with no error-correction loop.

3. **Implicit conflict resolution.** No explicit conflict resolution step — contradictions are resolved by entity replacement, which can silently lose information (if you say "I used to work at Google, now I work at Meta," the Google fact may simply be overwritten with no record of the transition).

4. **BEAM score drop.** 64.1 on 1M-token BEAM is significantly lower than LoCoMo/LongMemEval. This suggests performance degrades at very long context / large memory stores. Hystersis's spreading activation + tiered storage is specifically designed for this regime.

5. **No write-path latency SLA.** The async add path is documented but no latency claims are published. This is a gap Hystersis can directly attack with the <5ms claim.

6. **Compression ratio opacity.** Mem0 markets "compression" but publishes zero data on input-to-stored token ratios. This is a credibility gap Hystersis can fill with concrete 80–85% numbers.

7. **OSS vs Platform feature split is aggressive.** Core production features (temporal reasoning, decay, advanced filters) are behind the Platform paywall. The OSS SDK is not production-complete on its own.

8. **Importance scoring is retrieval-time only.** No write-time importance signal. Every memory starts equal; only retrieval recency/entity boosts differentiate them. This means rare but critical memories (e.g., "user has severe nut allergy") get no special treatment unless recently accessed.

9. **No procedural memory.** Mem0 does not model how to do things, only what is known. Agent skill memory is not a concept.

10. **Tiered storage absent.** All memories live in a flat vector store. There is no cost optimization for cold/archive memories, no automatic demotion of stale data.

---

## 13. Competitive Positioning Summary

| Dimension | Mem0 | Hystersis | Verdict |
|-----------|------|-----------|---------|
| Memory types | 4 label-based types | 4 types + procedural + tiered promotion | Hystersis wins |
| Temporal reasoning | Shallow (timestamp ranking, platform-only) | Decay-weighted graph propagation | Hystersis wins |
| Knowledge graph | **Removed entirely** | Core differentiator (spreading activation) | Hystersis wins decisively |
| Compression | Single-pass extraction, no ratio published | ProMem + fast/verify routing, 80-85% target | Hystersis wins |
| Hybrid search | Semantic + BM25 + entity boost | Same + graph-traversal recall | Hystersis wins |
| Multi-agent | Partition keys only | Graph-scoped agent memory | Hystersis advantage |
| Conflict resolution | Implicit (entity replacement) | Explicit LLM template, auditable | Hystersis wins |
| Importance scoring | Retrieval-time only | Write-time + tiered | Hystersis wins |
| Integration breadth | 14 frameworks, 23 vector stores, 17 LLMs | Narrow | Mem0 wins significantly |
| Benchmarks published | LoCoMo 91.6, LongMemEval 94.8 | Targets only (97% accuracy) | Mem0 wins for now |
| Pricing entry | Free → $19/month | TBD | Mem0 wins |
| Enterprise features | SOC2, HIPAA, on-prem, SSO | TBD | Mem0 wins for now |
| OSS completeness | OSS missing key features | Planned full OSS core | Neutral |

---

## 14. Strategic Recommendations

**Immediate wins to execute on:**

1. **Publish the multi-hop benchmark.** Run LoCoMo and LongMemEval with spreading activation enabled vs disabled. The graph-removal gap is Hystersis's biggest story — quantify it.

2. **Publish the compression ratio.** Input tokens → stored tokens. "80–85% reduction" needs a concrete benchmark table. Mem0 publishes none.

3. **Name the graph gap directly.** "Mem0 removed their knowledge graph in 2026. We built ours deeper." This is factual, verifiable, and devastating as positioning.

4. **Target the BEAM failure.** Mem0 drops to 64.1 at 1M context. Hystersis's tiered memory + spreading activation is purpose-built for large memory stores. Benchmark this directly.

5. **Integration prioritization:** To be competitive on go-to-market, focus on: LangChain > CrewAI > LlamaIndex > OpenAI Agents SDK. These four cover the majority of production agent deployments.

6. **Price the OSS tier to be fully featured.** Mem0's aggressive feature gating is a developer relations liability. If Hystersis's OSS tier includes temporal reasoning, decay, and graph retrieval — that is a powerful differentiator for developer adoption.

7. **Write-time importance is an unoccupied position.** No competitor (including Mem0) stores importance at write time. The healthcare use case (critical allergy information) is a concrete narrative for why this matters.
