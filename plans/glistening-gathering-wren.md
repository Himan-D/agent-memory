# Plan: Beat Mem0 — Full Feature Implementation

## Context

Hystersis has real algorithmic subsystems (ProMem extraction, spreading activation, temporal scoring, decay, consolidation, multi-agent sync) that are **disconnected from the service layer**. `internal/memory/service.go` has ~60 methods that return `nil, nil`. The API endpoints call service methods that do nothing.

Meanwhile, Mem0 **removed their entire knowledge graph** (4000 lines deleted), has no write-time importance scoring, no consolidation, no causal reasoning, and gates most features behind their paid platform. Their moat is benchmark velocity on single-turn QA, not architectural depth.

**Strategy**: Wire what exists → add features Mem0 architecturally can't replicate → benchmark and publish.

---

## Phase 0: Wire Existing Algorithms to Service Layer (CRITICAL FOUNDATION)

Everything else depends on this. The algorithms are built; they just need plumbing.

### 0.1 Wire CreateMemory Pipeline
- **File**: `internal/memory/service.go` — `CreateMemory()`
- Currently calls `graph.CreateMemory()` directly, bypassing the processor
- **Wire**: Content → `MemoryProcessor.ProcessContent()` → fact extraction → entity extraction → importance scoring → conflict check → graph write → vector write → async compression pipeline
- **Dependencies**: `processor.go`, `templates.go`, `neo4j/client.go`, vector provider

### 0.2 Wire SearchMemories
- **File**: `internal/memory/service.go` — `SearchMemories()`
- Currently returns `nil, nil`
- **Wire**: Query → `SpreadingActivation.Search()` or `SearchService` (from `search/search.go`) → apply `decay.ApplyDecay()` → apply `temporal.ApplyTemporalScoring()` → apply reranker → return results
- **Dependencies**: `compression/retrieval/proprietary.go`, `decay/scorer.go`, `temporal/reasoning.go`, `reranker/`

### 0.3 Wire Context Assembly
- **File**: `internal/memory/service.go` — `GetContext()`, `AddToContext()`
- Currently returns `nil`
- **Wire**: Session buffer → search relevant memories → assemble context with token budget → return
- **Dependencies**: `buffer.go`, `session/session.go`

### 0.4 Wire Graph Operations
- **File**: `internal/memory/service.go` — `Traverse()`, `QueryGraph()`
- Currently returns `nil`
- **Wire**: Delegate to `neo4j/client.go` traversal + Cypher query
- **Dependencies**: `neo4j/client.go`

### 0.5 Wire Memory History & Versioning
- **File**: `internal/memory/service.go` — `GetMemoryHistory()`, `UpdateMemory()`
- **Wire**: On update, increment `Version`, set `PreviousVersionID`, write history record
- **Dependencies**: `types.go` (Memory struct already has version fields)

### 0.6 Wire Consolidation & Sleep
- **File**: `internal/memory/service.go` — connect `consolidation/service.go` to `sleep/compute.go`
- **Wire**: Background scheduler submits consolidation tasks to the sleep worker pool
- **Dependencies**: `consolidation/service.go`, `sleep/compute.go`

### 0.7 Wire Multi-Agent Sync
- **File**: `internal/memory/service.go` — `ShareMemoryToGroup()`, agent group methods
- **Wire**: Delegate to `sync/redis.go` pool
- **Dependencies**: `sync/redis.go`

### 0.8 Wire Compression Pipeline
- **File**: `internal/memory/service.go` — connect async compression
- **Wire**: After memory creation, submit to `pipeline/async.go` worker pool
- Measure actual compression stats instead of hardcoded `0.97, 0.8`
- **Dependencies**: `compression/pipeline/async.go`

### 0.9 Fix Spreading Activation Edge Weights
- **File**: `internal/compression/retrieval/proprietary.go`
- Edge-type weights (`SIMILAR_TO=0.9`, `RELATES_TO=0.8`, `CONTRADICTS=0.3`) are defined but never applied
- **Wire**: `getNeighborMemories()` should filter by edge type and apply weight multiplier

### 0.10 Wire Access Count Increment
- **File**: `internal/memory/service.go` — `GetMemory()`
- `AccessCount` field exists, decay scorer uses it, but retrieval never increments it
- **Wire**: Increment `AccessCount` on every read

---

## Phase 1: Core Differentiators (Features Mem0 Can't Replicate)

### 1.1 Memory Worth (MW) Scoring
- **What**: Two atomic counters per memory: `SuccessCount` and `FailureCount`. Updated via feedback. Converges to conditional success probability.
- **Files**: 
  - `internal/memory/types/types.go` — add `SuccessCount int64`, `FailureCount int64`, `WorthScore float64`
  - `internal/memory/feedback/feedback.go` — update counters on positive/negative feedback
  - `internal/memory/decay/scorer.go` — incorporate MW into composite score
- **Why**: Mem0 has no outcome-linked importance. A memory about "always use sudo" that leads to failures gets demoted. Critical for agent safety.
- **Paper**: arXiv:2604.12007

### 1.2 Temporal Phase Rotation (RoMem-style)
- **What**: Instead of deleting outdated facts, rotate them in complex vector space. Volatile relations ("works at") rotate fast; stable relations ("born in") rotate slowly. Outdated facts naturally rank lower without deletion.
- **Files**:
  - `internal/memory/types/types.go` — add `VolatilityScore float64`, `PhaseAngle float64`
  - `internal/memory/temporal/rotation.go` — NEW: phase rotation math (complex multiply on embeddings)
  - `internal/memory/temporal/volatility.go` — NEW: semantic speed gate classifier
  - `internal/vector/` — modify search to apply phase rotation before scoring
- **Why**: Mem0 uses simple recency filtering. This is mathematically principled and preserves historical context.
- **Paper**: arXiv:2604.11544

### 1.3 Four-Signal Composite Importance Score
- **What**: Replace single-signal scoring with composite: (1) semantic relevance, (2) temporal validity (Ebbinghaus decay), (3) confidence (MW counters), (4) graph centrality (Neo4j degree).
- **Files**:
  - `internal/memory/scoring/composite.go` — NEW: composite scorer combining 4 signals
  - `internal/memory/scoring/graph_signal.go` — NEW: Neo4j centrality query
  - Wire into `SearchMemories` pipeline
- **Why**: Each signal is orthogonal. Together they outperform any single heuristic.
- **Paper**: arXiv:2604.20598

### 1.4 Memory Conflict Validity Framework
- **What**: Memories get `ValidityStatus`: `current`, `superseded`, `historically_valid`, `unknown`. Superseded memories aren't deleted — they're kept with provenance for historical queries.
- **Files**:
  - `internal/memory/types/types.go` — add `ValidityStatus string`
  - `internal/memory/templates.go` — update conflict resolution template to emit status
  - `internal/memory/service.go` — apply status in CreateMemory conflict check
  - Search filters by validity status based on query type
- **Why**: Mem0 silently overwrites. "What was the user's previous employer?" becomes answerable.

---

## Phase 2: Advanced Features

### 2.1 Auto-Dreamer Sleep Consolidation
- **What**: Background job reads memory regions, cross-session consolidation produces compact replacements. 12x memory bank reduction.
- **Files**:
  - `internal/memory/sleep/dreamer.go` — NEW: consolidation algorithm
  - `internal/memory/sleep/compute.go` — wire dreamer as task type
  - `internal/memory/sleep/scheduler.go` — NEW: cron-based trigger
  - `internal/memory/templates.go` — consolidation prompts
- **Why**: Mem0 accumulates infinitely. This is self-maintaining memory.
- **Paper**: arXiv:2605.20616

### 2.2 Adaptive Retrieval Routing
- **What**: Classify queries as simple/parallel/iterative and route to optimal retrieval strategy.
- **Files**:
  - `internal/memory/search/router.go` — NEW: query complexity classifier
  - `internal/memory/search/strategies.go` — NEW: three strategy implementations
  - Wire into `SearchMemories`
- **Why**: Simple queries don't need multi-hop. Complex queries need decomposition.
- **Paper**: arXiv:2604.04853

### 2.3 Provenance DAG + Credit Assignment
- **What**: Track which memories were used to create new memories. TD(λ) eligibility traces flow credit back through the chain on success/failure.
- **Files**:
  - `internal/memory/types/types.go` — add `ProvenanceEdges []string`, `QValue float64`
  - `internal/memory/provenance/dag.go` — NEW: DAG tracking
  - `internal/memory/provenance/credit.go` — NEW: TD(λ) credit assignment
  - Neo4j schema: `DERIVED_FROM` edge type
- **Why**: The learning system that compounds over time. This is the long-term moat.
- **Paper**: arXiv:2605.08374

### 2.4 DeferMem Two-Stage Distillation
- **What**: Stage 1 = high-recall candidate fetch. Stage 2 = query-conditioned rewriting/distillation into faithful evidence.
- **Files**:
  - `internal/memory/search/distiller.go` — NEW: post-retrieval distillation
  - `internal/memory/templates.go` — distillation prompt
  - Wire as optional stage in `SearchMemories`
- **Why**: Returns synthesized evidence, not raw memories. Far more useful for LLMs.
- **Paper**: arXiv:2605.22411

### 2.5 Exploitation/Exploration Dual Pool
- **What**: Two memory pools per agent: exploitation (proven patterns) and exploration (LLM-generated candidates). Dynamic reweighting via LLM-as-judge.
- **Files**:
  - `internal/memory/pool/dual.go` — NEW: pool management
  - `internal/memory/pool/judge.go` — NEW: LLM quality assessment
  - Tag memories in vector store with pool affiliation
- **Why**: +23.8% accuracy, prevents pattern ossification.
- **Paper**: arXiv:2605.22721

---

## Phase 3: Self-Improvement & Polish

### 3.1 Self-Evolving Retrieval (EvolveMem)
- **What**: Log per-query failure signals. Background diagnostic agent identifies root causes and proposes config adjustments.
- **Files**:
  - `internal/memory/self_improve/improver.go` — already exists, fill in
  - `internal/memory/self_improve/diagnostics.go` — NEW: failure analysis
  - `internal/memory/self_improve/tuner.go` — NEW: config patch mechanism
- **Paper**: arXiv:2605.13941

### 3.2 SimUtil-UCB Retrieval Bandit
- **What**: UCB score combining similarity, utility (MW), and exploration bonus. Prevents popular memories from dominating.
- **Files**:
  - `internal/memory/search/ucb.go` — NEW: UCB scoring function (~30 lines)
  - Wire as optional scorer in search pipeline
- **Paper**: arXiv:2603.08561

### 3.3 Causal-Semantic Graph (ActMem)
- **What**: Extract causal edges (`CAUSED_BY`, `LED_TO`, `PREVENTED`) from dialogue. Enable "why did X happen" queries.
- **Files**:
  - `internal/memory/templates.go` — causal extraction prompt
  - Neo4j schema: causal edge types
  - `internal/memory/neo4j/client.go` — causal traversal queries
- **Paper**: ActMem

### 3.4 Intra-Session Retrieval (CALMem)
- **What**: Sliding-window embeddings of current session turns, searchable within session. Token-budget-aware injection.
- **Files**:
  - `internal/memory/session/retriever.go` — NEW: per-session ring buffer
  - Redis-backed embedded chunks
- **Paper**: arXiv:2605.20724

---

## Implementation Order

The phases are sequential but within each phase, tasks can be parallelized:

```
Phase 0 (Foundation — MUST be first):
  ├── 0.1-0.3: Core service wiring (CreateMemory, Search, Context) [parallel]
  ├── 0.4-0.5: Graph + versioning [parallel]
  ├── 0.6-0.7: Consolidation + sync [parallel]
  └── 0.8-0.10: Compression + edge weights + access count [parallel]

Phase 1 (Differentiators — after Phase 0):
  ├── 1.1: MW Scoring [independent]
  ├── 1.2: Temporal Phase Rotation [independent]
  ├── 1.3: Four-Signal Scoring [depends on 1.1]
  └── 1.4: Conflict Validity [independent]

Phase 2 (Advanced — after Phase 1):
  ├── 2.1: Sleep Consolidation [depends on 0.6]
  ├── 2.2: Adaptive Retrieval [depends on 0.2]
  ├── 2.3: Provenance DAG [depends on 0.4]
  ├── 2.4: Two-Stage Distillation [depends on 0.2]
  └── 2.5: Dual Pool [independent]

Phase 3 (Self-Improvement — after Phase 2):
  ├── 3.1: EvolveMem [depends on 2.2]
  ├── 3.2: UCB Bandit [depends on 1.1]
  ├── 3.3: Causal Graph [depends on 0.4]
  └── 3.4: Intra-Session [depends on 0.3]
```

## Critical Files

| File | Role |
|---|---|
| `internal/memory/service.go` | Main service — needs ~60 methods wired |
| `internal/memory/types/types.go` | Memory type — needs MW, validity, provenance, volatility fields |
| `internal/memory/templates.go` | LLM prompts — needs causal, distillation, consolidation templates |
| `internal/compression/retrieval/proprietary.go` | Spreading activation — needs edge weight fix |
| `internal/memory/neo4j/client.go` | Graph ops — needs causal edges, centrality queries |
| `internal/memory/search/search.go` | Search service — needs router, distiller, UCB |
| `internal/memory/sleep/compute.go` | Sleep worker — needs dreamer algorithm |
| `internal/memory/decay/scorer.go` | Decay — needs MW integration |
| `internal/memory/temporal/reasoning.go` | Temporal — needs phase rotation |
| `internal/memory/feedback/feedback.go` | Feedback — needs MW counter updates |

## New Files to Create

| File | Purpose |
|---|---|
| `internal/memory/temporal/rotation.go` | Phase rotation math |
| `internal/memory/temporal/volatility.go` | Semantic speed gate |
| `internal/memory/scoring/composite.go` | Four-signal composite scorer |
| `internal/memory/scoring/graph_signal.go` | Neo4j centrality signal |
| `internal/memory/sleep/dreamer.go` | Auto-Dreamer consolidation |
| `internal/memory/sleep/scheduler.go` | Cron-based sleep trigger |
| `internal/memory/search/router.go` | Adaptive retrieval routing |
| `internal/memory/search/strategies.go` | Strategy implementations |
| `internal/memory/search/distiller.go` | Post-retrieval distillation |
| `internal/memory/search/ucb.go` | UCB scoring |
| `internal/memory/provenance/dag.go` | Provenance DAG |
| `internal/memory/provenance/credit.go` | TD(λ) credit assignment |
| `internal/memory/pool/dual.go` | Dual pool management |
| `internal/memory/pool/judge.go` | LLM quality judge |
| `internal/memory/self_improve/diagnostics.go` | Failure analysis |
| `internal/memory/self_improve/tuner.go` | Config auto-tuning |

## Verification

```bash
# Build
go build ./...
go vet ./...

# Tests (add tests for each new feature)
go test ./internal/memory/... -v
go test ./internal/compression/... -v

# Integration test: end-to-end memory lifecycle
# 1. Create memory → verify processor runs → entities extracted → compressed
# 2. Search → verify spreading activation + temporal + decay applied
# 3. Feedback → verify MW counters update
# 4. Sleep consolidation → verify memory count reduced
# 5. Conflict → verify validity status set correctly
```

## Competitive Benchmarks to Run

| Benchmark | Mem0 Score | Target |
|-----------|-----------|--------|
| LoCoMo | 91.6 | 93+ |
| LongMemEval | 94.8 | 96+ |
| BEAM (1M tokens) | 64.1 | 75+ (spreading activation advantage) |
| Multi-hop reasoning | baseline | +23% (spreading activation) |
| Token reduction | ~80% | 80-85% (ProMem) |
| p95 latency | 1.44s | <500ms (Go advantage) |
