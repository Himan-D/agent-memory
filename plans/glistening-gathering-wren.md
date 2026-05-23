# Plan: Implement Cutting-Edge Memory Research (2025-2026 Papers)

## Context

After 4 implementation passes, Hystersis has a solid algorithmic core. This plan adds techniques from 18 papers published in the last 6 months — specifically the ones with the highest benchmark impact and lowest implementation cost. We also fix remaining code-level gaps found in the audit.

**Selection criteria**: Papers that (1) show measurable benchmark improvement, (2) are implementable in Go without training infrastructure, and (3) fill gaps Hystersis doesn't already cover.

---

## Phase 1: Immediate Wins (Low effort, highest benchmark lift)

### 1.1 Source Authority Signal (MEMTIER — +33pp on LongMemEval-S)
- **Paper**: arXiv:2605.03675
- **What**: Add a 5th signal to the composite scorer — `source_authority` — weighting memories by their provenance (observed > told > inferred > external)
- **Files**:
  - `internal/memory/types/types.go` — add `SourceType string` field (values: `observed`, `told`, `inferred`, `external`) and `SourceAuthority float64`
  - `internal/memory/scoring/composite.go` — add 5th signal weight (default 0.15, redistribute from others)
  - `internal/memory/service.go` — populate `SourceType` in CreateMemory from request metadata
- **Effort**: Low

### 1.2 Prospection-Guided Retrieval (PGR — 3x recall on hard queries)
- **Paper**: arXiv:2605.14177
- **What**: Before retrieval, expand query into 3-5 plausible next interaction steps via LLM. Use these as additional search probes, union results.
- **Files**:
  - `internal/memory/search/prospection.go` — NEW: `Prospector` that generates future-step queries via LLM
  - `internal/memory/templates.go` — add prospection prompt template
  - `internal/memory/service.go` — wire into SearchMemories before vector search
- **Effort**: Low-Medium

### 1.3 Dimensional Memory Fields (DimMem — 24% token reduction)
- **Paper**: arXiv:2605.15759
- **What**: Store each memory with typed fields: `time_ref`, `location`, `reason`, `purpose`, `keywords`. Enable dimension-aware filtering before vector search.
- **Files**:
  - `internal/memory/types/types.go` — add `Dimensions` struct with 5 fields
  - `internal/memory/templates.go` — add dimension extraction prompt
  - `internal/memory/service.go` — extract dimensions in CreateMemory pipeline, filter on them in SearchMemories
- **Effort**: Low

### 1.4 Safety-Triggered Forgetting (FSFM)
- **Paper**: arXiv:2604.20300
- **What**: Classify memory content on write for safety. Malicious/sensitive injection → immediate purge. Four forgetting classes: passive decay (have), active deletion (have), safety-triggered (MISSING), adaptive RL-based (defer).
- **Files**:
  - `internal/memory/safety/classifier.go` — NEW: content safety classifier (keyword + pattern matching, optional LLM call)
  - `internal/memory/service.go` — in CreateMemory, run safety check before graph write. Reject or quarantine unsafe content.
- **Effort**: Low-Medium

### 1.5 FAMA Evaluation Metric
- **Paper**: arXiv:2604.20006
- **What**: Penalize stale memory reuse in evaluation. Standard accuracy rewards any correct answer — FAMA discounts answers from outdated memories.
- **Files**:
  - `internal/evaluation/fama.go` — NEW: FAMA scoring function
  - `internal/evaluation/benchmark.go` — integrate FAMA into benchmark runs
- **Effort**: Low

### 1.6 Wire Existing Synonym Expansion
- **Audit finding**: `TuningStore.GetSynonyms` and `AddLearnedSynonym` exist but are never called during search
- **Files**:
  - `internal/memory/service.go` — in SearchMemories, before embedding, check for learned synonyms and expand the query string
  - `internal/memory/self_improve/improver.go` — confirm GetSynonyms is accessible
- **Effort**: Very Low

### 1.7 Fix ExecuteChain + Wire ExtractSkills
- **Audit finding**: `ExecuteChain` is a no-op, `ExtractSkills` never calls the processor
- **Files**:
  - `internal/memory/service.go` — `ExecuteChain`: walk `SkillChain.Steps`, call `ExecuteSkill` sequentially, thread outputs
  - `internal/memory/service.go` — `ExtractSkills`: call `s.processor.ExtractSkills` instead of returning empty
- **Effort**: Low

---

## Phase 2: Strategic Differentiators (Medium effort, architectural value)

### 2.1 Three-Granularity Storage (TriMem)
- **Paper**: arXiv:2605.19952
- **What**: Store three representations per memory: (a) raw dialogue segment (lossless), (b) atomic facts (ProMem output), (c) synthesized user profile (cross-session aggregate). Retrieval selects tier based on query type.
- **Files**:
  - `internal/memory/types/types.go` — add `RawSegment string`, `SynthesizedProfile string` to Memory
  - `internal/memory/service.go` — in CreateMemory, store raw content before ProMem extraction. Periodically synthesize profiles.
  - `internal/memory/search/granularity.go` — NEW: tier selection logic (factoid queries → atomic facts, context queries → raw segments, personality queries → profiles)
- **Effort**: Medium

### 2.2 Causal Memory Intervention (CMI)
- **Paper**: arXiv:2605.17641
- **What**: Post-retrieval causal filter. For each top-k candidate, run a controlled intervention (answer with vs without the memory). Keep only memories that causally improve the answer.
- **Files**:
  - `internal/memory/search/causal.go` — NEW: `CausalFilter` that runs LLM intervention scoring
  - `internal/memory/templates.go` — add causal intervention prompt
  - `internal/memory/service.go` — wire as optional post-retrieval step (expensive, opt-in)
- **Effort**: Medium

### 2.3 Concept Nodes + Edge-Type PPR (GAAMA)
- **Paper**: arXiv:2603.27910
- **What**: Add `concept` nodes as cross-cutting graph hubs. Use edge-type-aware Personalized PageRank instead of uniform spreading activation.
- **Files**:
  - `internal/memory/neo4j/client.go` — add `CreateConcept`, `LinkToConcept` methods. Add PPR query using Neo4j GDS.
  - `internal/memory/types/types.go` — add `Concept` type
  - `internal/compression/retrieval/proprietary.go` — extend spreading activation to use PPR scores when available
- **Effort**: Medium

### 2.4 Negation Memory (PolarMem)
- **Paper**: arXiv:2602.00415
- **What**: Store verified negations as first-class memory nodes. Inhibitory edges suppress hallucination. Logic-dominant retrieval penalizes candidates that violate stored constraints.
- **Files**:
  - `internal/memory/types/types.go` — add `Polarity string` field (`positive`, `negative`, `unknown`)
  - `internal/memory/neo4j/client.go` — add `CONTRADICTS_NEGATION` edge type
  - `internal/memory/scoring/composite.go` — in scoring pipeline, penalize candidates that match stored negations
  - `internal/memory/templates.go` — add negation extraction prompt
- **Effort**: Medium

### 2.5 Event/Topic Dual Graph (GAM)
- **Paper**: arXiv:2604.12285
- **What**: Split graph into Event Progression Graph (working memory, updated every turn) and Topic Associative Network (long-term, updated on semantic shift). Consolidation moves events → topics.
- **Files**:
  - `internal/memory/neo4j/client.go` — add `EventNode`, `TopicNode` label differentiation
  - `internal/memory/service.go` — in CreateMemory, always write to event graph. Background job consolidates events → topics when semantic shift detected.
  - `internal/memory/temporal/shift_detector.go` — NEW: semantic shift detector (embedding distance threshold between recent turns)
- **Effort**: Medium

### 2.6 Prospective Memory (Reminders)
- **Audit finding**: No "remind me at X" capability
- **Files**:
  - `internal/memory/types/types.go` — add `RemindAt *time.Time`, `RemindCondition string` to Memory
  - `internal/memory/service.go` — background worker polls for due reminders, surfaces them via search boost or notification
- **Effort**: Medium

---

## Phase 3: Long-Term R&D (High effort, highest ceiling)

### 3.1 Hypergraph Memory (HyperMem — SOTA 92.73% LoCoMo)
- **Paper**: arXiv:2604.08256 (ACL 2026)
- **What**: Three-level hypergraph (topics → episodes → facts). Hyperedges capture joint dependencies. Coarse-to-fine retrieval.
- **Implementation**: Reify hyperedges as Neo4j nodes connected to member nodes. Requires retrieval rewrite.
- **Effort**: Hard

### 3.2 Adaptive Memory Crystallization (AMC)
- **Paper**: arXiv:2604.13085
- **What**: SDE-governed stability states (Liquid→Glass→Crystal) with mathematical convergence guarantees.
- **Implementation**: Discrete-time approximation of Itô SDE per memory. Replaces threshold-based tier promotion.
- **Effort**: Hard

### 3.3 Generative Memory (Mem-π — +30%)
- **Paper**: arXiv:2605.21463
- **What**: Instead of retrieving static text, generate context-specific guidance on demand.
- **Implementation**: Small prompted model for generative pass. Confidence-threshold heuristic for when-to-abstain.
- **Effort**: Hard

---

## Implementation Order

```
Phase 1 (Immediate — 1 day):
  ├── 1.1: Source authority signal [types.go + composite.go]
  ├── 1.2: Prospection-guided retrieval [NEW search/prospection.go]
  ├── 1.3: Dimensional memory fields [types.go + templates.go]
  ├── 1.4: Safety-triggered forgetting [NEW safety/classifier.go]
  ├── 1.5: FAMA metric [NEW evaluation/fama.go]
  ├── 1.6: Wire synonym expansion [service.go]
  └── 1.7: Fix ExecuteChain + ExtractSkills [service.go]

Phase 2 (Strategic — 2-3 days):
  ├── 2.1: TriMem three-granularity [types.go + NEW search/granularity.go]
  ├── 2.2: Causal intervention filter [NEW search/causal.go]
  ├── 2.3: Concept nodes + PPR [neo4j/client.go]
  ├── 2.4: Negation memory [types.go + scoring]
  ├── 2.5: Event/topic dual graph [neo4j + NEW temporal/shift_detector.go]
  └── 2.6: Prospective memory/reminders [types.go + service.go]
```

## New Files to Create

| File | Purpose | Paper |
|---|---|---|
| `internal/memory/search/prospection.go` | ToT query expansion for 3x recall | PGR |
| `internal/memory/search/granularity.go` | Tier selection for TriMem | TriMem |
| `internal/memory/search/causal.go` | Causal intervention filter | CMI |
| `internal/memory/safety/classifier.go` | Content safety classifier | FSFM |
| `internal/memory/temporal/shift_detector.go` | Semantic shift detection | GAM |
| `internal/evaluation/fama.go` | Staleness-penalizing metric | FAMA |

## Files to Modify

| File | Changes |
|---|---|
| `internal/memory/types/types.go` | SourceType, Dimensions, Polarity, RemindAt, RawSegment, Concept |
| `internal/memory/scoring/composite.go` | 5th signal (source authority), negation penalty |
| `internal/memory/service.go` | Wire prospection, dimensions, safety, synonyms, chain execution |
| `internal/memory/templates.go` | Prospection, dimension extraction, causal, negation prompts |
| `internal/memory/neo4j/client.go` | Concept nodes, PPR query, event/topic labels |

## Verification

```bash
go build ./... && go vet ./...
go test ./internal/memory/... -v
go test ./internal/compression/... -v
go test ./internal/evaluation/... -v
```
