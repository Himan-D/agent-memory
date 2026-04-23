# Hystersis - Developer Guide

## Build & Test Commands

```bash
go build ./...     # Build all packages
go test ./...      # Run all tests
go run ./cmd/server # Run API server
go run ./cmd/agent  # Run CLI agent
```

## ⚠️ Required Testing Workflow

**ALWAYS test your code BEFORE committing to git:**

1. **Build the code** - Run `go build ./...` to check for compilation errors
2. **Start required services** - Ensure Neo4j and Qdrant are running:
   ```bash
   # Docker compose for dependencies
   docker-compose up -d neo4j qdrant
   ```
3. **Run the server** - Test locally with `go run ./cmd/server`
4. **Test the specific feature** - Verify the endpoint/functionality works:
   - API endpoints: `curl http://localhost:8080/your-endpoint`
   - Frontend: Visit http://localhost:8080 (or localhost:5173 for dev)
5. **Only commit after successful testing**

### Git Push Workflow

```bash
# 1. Build first
go build ./...

# 2. Test your changes
# (start server, test endpoint manually)

# 3. Commit with descriptive message
git add -A
git commit -m "Description of what you fixed/added"

# 4. Push to remote
git push origin main
```

**NEVER push code that doesn't compile or hasn't been tested.**

## Project Structure

```
cmd/
  server/api.go     # REST API server (~2600 lines)
  agent/            # CLI agent harness
  cli/              # CLI commands
internal/
  memory/service.go  # Core memory service (~2500 lines)
  memory/neo4j/      # Neo4j graph implementation (~3500 lines)
  memory/types/      # Type definitions
  memory/compression.go  # Basic compression (legacy - OPEN SOURCE)
  compression/           # PROPRIETARY - Core differentiator vs Mem0
    extractor/           # ProMem-style extraction
    retrieval/           # Spreading activation retrieval
    pipeline/            # Async compression pipeline
    llm/                 # Hybrid LLM routing
  skills/            # Skills registry system
  sso/               # SSO providers (OIDC, SAML, LDAP)
  reranker/          # Reranking (Cohere + LLM)
  audit/             # Audit logging middleware
  analytics/         # Dashboard analytics
  webhook/           # Webhook system
  config/            # Configuration
```

---

# ⚠️ CRITICAL: Proprietary Compression Engine

This section defines the **primary competitive differentiator** vs Mem0. ALL coding agents must implement this exactly as specified. Failure to follow these guidelines will result in feature duplication without competitive advantage.

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                   COMPRESSION ENGINE (PROPRIETARY)             │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐     │
│  │  LLM Router  │───▶│  EXTRACTOR   │───▶│   COMPRESS    │     │
│  │  (Hybrid)    │    │  (ProMem)    │    │   PIPELINE    │     │
│  └──────────────┘    └──────────────┘    └──────────────┘     │
│         │                    │                    │              │
│         ▼                    ▼                    ▼              │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐     │
│  │ Fast: GPT-4o │    │ Self-Question│    │  Async Queue  │     │
│  │ Verify:Claude│    │ Verification │    │  Worker Pool  │     │
│  └──────────────┘    └──────────────┘    └──────────────┘     │
│                                                                 │
│  ┌──────────────────────────────────────────────────────┐     │
│  │           SPREADING ACTIVATION (Proprietary)          │     │
│  │  Query → Vector Search → Graph Propagation → Ranking │     │
│  └──────────────────────────────────────────────────────┘     │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

## Open Source vs Proprietary Boundary

### ⛔ PROPRIETARY (DO NOT OPEN SOURCE)
- **ProMem extraction algorithm** - `internal/compression/extractor/`
- **Spreading activation implementation** - `internal/compression/retrieval/`
- **LLM routing logic** - `internal/compression/llm/router.go`
- **Hyperparameter tuning** (decay, threshold, hops)
- **Benchmark results code** (show metrics in marketing, don't share implementation)

### ✅ CAN BE OPEN SOURCE
- Basic compression.go (simple summarization)
- Vector similarity search (standard)
- Neo4j storage (infrastructure)
- API endpoints (table stakes)
- Configuration handling

---

## Feature Implementation Priority

| Priority | Feature | Target | Location | Timeline |
|----------|---------|--------|----------|----------|
| 🔴 P1 | **ProMem Extraction** | 97%+ accuracy, 80-85% compression | `internal/compression/extractor/` | Week 1-3 |
| 🔴 P1 | **Spreading Activation** | +23% multi-hop reasoning | `internal/compression/retrieval/` | Week 3-4 |
| 🟠 P2 | **Async Pipeline** | <5ms write latency impact | `internal/compression/pipeline/` | Week 4-5 |
| 🟠 P2 | **Tiered Memory** | Working→Hot→Cold→Archive | `internal/memory/tier/` | Week 4-5 |
| 🟡 P3 | **Observability** | Real-time stats | `internal/metrics/` | Week 6-8 |

---

## Implementation: Hybrid LLM Router

### File: `internal/compression/llm/router.go`

```go
package llm

// LLMRouter routes compression tasks to appropriate LLM providers
// based on complexity threshold.
// 
// FAST PATH: GPT-4o-mini, Groq - for simple extraction
// VERIFY PATH: Claude, GPT-4o - for complex verification
type LLMRouter struct {
    fastProvider   Provider    // Low-cost, fast model
    verifyProvider Provider    // High-accuracy model for verification
    complexityThreshold float64 // Default: 0.6
}

// Route determines which provider to use based on memory complexity
func (r *LLMRouter) Route(ctx context.Context, memory string) (*ExtractionResult, error) {
    complexity := r.estimateComplexity(memory)
    
    if complexity < r.complexityThreshold {
        // Simple memory - use fast provider
        return r.fastProvider.Extract(ctx, memory)
    }
    
    // Complex memory - run fast extraction + verify
    return r.extractWithVerification(ctx, memory)
}
```

### Environment Variables

```bash
# Compression Engine LLM Configuration
COMPRESSION_LLM_FAST_PROVIDER=openai       # or: groq, deepseek
COMPRESSION_LLM_FAST_MODEL=gpt-4o-mini
COMPRESSION_LLM_VERIFY_PROVIDER=anthropic  # or: openai
COMPRESSION_LLM_VERIFY_MODEL=claude-3-5-sonnet
COMPRESSION_COMPLEXITY_THRESHOLD=0.6
```

---

## Implementation: ProMem-Style Extraction

### File: `internal/compression/extractor/proprietary.go`

```go
package extractor

// MemoryExtractor implements ProMem-style extraction
// Reference: "ProMem: Proactive Memory Extraction for LLM Agents" (arXiv:2601.04463)
//
// ALGORITHM (PROPRIETARY):
// 1. Self-Question Generation: Ask "what does this memory mean?"
// 2. Answer Verification: Validate against original memory  
// 3. Gap Detection: Identify missing critical information
// 4. Active Extraction: Pull key facts, not just summarize
type MemoryExtractor struct {
    llmProvider    llm.Provider
    maxIterations  int         // 2-3 rounds
    verifyThreshold float64   // 0.85 confidence
}

type ExtractionResult struct {
    Facts         []Fact
    VerifiedFacts []Fact      // After self-verification
    Gaps          []Gap       // Missing info
    Supplements   []Fact      // Fill-in for gaps
    Confidence    float64
    TokenReduction float64
}

// Extract performs ProMem-style extraction with self-verification
func (e *MemoryExtractor) Extract(ctx context.Context, memory string) (*ExtractionResult, error) {
    // Phase 1: Self-Question Generation
    questions := e.generateQuestions(memory)
    
    // Phase 2: Answer in context  
    answers := e.answerQuestions(ctx, questions, memory)
    
    // Phase 3: Verification (using verifyProvider - Claude)
    verified := e.verifyWithProvider(ctx, answers, memory)
    
    // Phase 4: Gap Detection
    gaps := e.detectGaps(verified, memory)
    supplements := e.extractGaps(ctx, gaps)
    
    return merge(verified, supplements), nil
}
```

---

## Implementation: Spreading Activation Retrieval

### File: `internal/compression/retrieval/proprietary.go`

```go
package retrieval

// SpreadingActivation implements graph-based retrieval beyond vector similarity
// Reference: "Synapse: Empowering LLM Agents with Episodic-Semantic Memory" (arXiv:2601.02744)
//
// ALGORITHM (PROPRIETARY):
// 1. Convert query to embedding → initial activation
// 2. Propagate through Neo4j graph with decay (0.85 per hop)
// 3. Collect activated nodes above threshold (0.1)
// 4. Rank by activation level + vector similarity
// 
// KEY ADVANTAGE: +23% improvement in multi-hop reasoning vs pure vector
type SpreadingActivation struct {
    graphStore    GraphStore   // Neo4j interface
    vectorStore   VectorStore  // Qdrant interface
    
    // Hyperparameters (PROPRIETARY - tune these values)
    initialBudget float64   // Default: 1.0
    decayFactor   float64   // Default: 0.85 (per-hop decay)
    threshold     float64   // Default: 0.1 (stopping threshold)
    maxHops       int       // Default: 3 (max propagation depth)
}

type ActivationResult struct {
    Nodes        []ActivatedNode
    TotalScore   float64
    HopBreakdown []int        // Nodes reached per hop
}

// Retrieve performs spreading activation search
func (s *SpreadingActivation) Retrieve(ctx context.Context, query string) ([]*types.Memory, error) {
    // 1. Get initial nodes via vector similarity (Qdrant)
    initialNodes := s.vectorStore.Search(ctx, query, topK=50)
    
    // 2. Inject activation into graph nodes
    activationMap := s.initializeActivation(initialNodes)
    
    // 3. Propagate through graph (multi-hop)
    for hop := 0; hop < s.maxHops; hop++ {
        activationMap = s.propagate(ctx, activationMap, s.decayFactor)
    }
    
    // 4. Rank by activation level
    results := s.rankByActivation(ctx, activationMap)
    
    return results, nil
}
```

---

## Implementation: Async Pipeline

### File: `internal/compression/pipeline/async.go`

```go
package pipeline

// CompressionPipeline handles non-blocking compression jobs
// Ensures <5ms write latency impact by processing asynchronously
type CompressionPipeline struct {
    jobQueue      chan CompressionJob
    workerPool    int          // 4-8 workers
    smallModel    string       // Distilled model for fast compression
    
    // Components
    extractor     *extractor.MemoryExtractor
    compressor    *Compressor
    validator     *Validator
}

type CompressionJob struct {
    MemoryID     string
    Priority     int           // 0=critical, 1=high, 2=normal
    Done         chan Result
}

type Result struct {
    Compressed   string
    TokenReduction float64
    Error         error
}

// CompressAsync processes compression non-blocking
func (p *CompressionPipeline) CompressAsync(job CompressionJob) {
    p.jobQueue <- job  // Returns immediately, processes in background
}
```

---

## Implementation: Tiered Memory

### File: `internal/memory/tier/router.go`

```go
package tier

// TierPolicy defines memory retention policy
type TierPolicy string

const (
    TierPolicyAggressive   TierPolicy = "aggressive"   // 1-day hot
    TierPolicyBalanced     TierPolicy = "balanced"     // 7-day hot (default)
    TierPolicyConservative TierPolicy = "conservative" // 30-day hot
)

// MemoryTier represents storage tier
type MemoryTier string

const (
    TierWorking MemoryTier = "working"  // In-memory, <5ms
    TierHot     MemoryTier = "hot"      // Redis, <20ms
    TierCold    MemoryTier = "cold"     // Neo4j+Qdrant, <100ms
    TierArchive MemoryTier = "archive"  // Object storage, >1s
)

// MemoryRouter routes memories to appropriate tiers
type MemoryRouter struct {
    config TierConfig
}

type TierConfig struct {
    WorkingMaxTokens int   // Default: 4096
    HotMaxTokens     int   // Default: 32768
    HotRetentionDays int   // Default: 7
    ArchiveThreshold int   // Default: 100 (access count)
}
```

---

## SDK API Endpoints

### REST API (add to cmd/server/api.go)

```go
// Compression endpoints
router.GET("/compression/stats", handlers.GetCompressionStats)
router.PUT("/compression/mode", handlers.SetCompressionMode)
router.GET("/compression/mode", handlers.GetCompressionMode)

// Tier endpoints  
router.GET("/tier/policy", handlers.GetTierPolicy)
router.PUT("/tier/policy", handlers.SetTierPolicy)

// Enhanced search (spreading activation)
router.GET("/search/enhanced", handlers.SearchEnhanced)
```

### Response Formats

```json
// GET /compression/stats
{
    "accuracy_retention": 0.973,
    "token_reduction": 0.84,
    "total_tokens_saved": 1500000,
    "extractions_performed": 450,
    "spreading_activations": 230,
    "avg_latency_ms": 187,
    "p95_latency_ms": 245
}

// PUT /compression/mode
{ "mode": "extract" }  // or: "balanced", "aggressive"

// PUT /tier/policy
{ "policy": "balanced" }  // or: "aggressive", "conservative"

// GET /search/enhanced?mode=spreading&query=...
{
    "results": [...],
    "mode": "spreading",
    "activation_hops": 3
}
```

---

## Benchmark Targets

| Metric | Target | Current (Mem0) | Advantage |
|--------|--------|----------------|-----------|
| Accuracy Retention | ≥97% | 91% | +6% |
| Token Reduction | 80-85% | 80% | +5% |
| Multi-hop Reasoning | +23% vs vector | baseline | +23% |
| P95 Latency | <200ms | ~400ms | 2x faster |
| Write Impact | <5ms | N/A | Non-blocking |

---

## Implementation Roadmap

### Week 1-2: Foundation
- [ ] Design LLM Router with configurable providers
- [ ] Build fast-path extraction (GPT-4o-mini/Groq)
- [ ] Create self-questioning prompt templates

### Week 2-3: ProMem + Verification
- [ ] Implement self-questioning loop (2-3 iterations)
- [ ] Add verification loop with Claude
- [ ] Gap detection & supplementation
- [ ] Baseline benchmark ≥97% accuracy

### Week 3-4: Spreading Activation
- [ ] Graph propagation foundation
- [ ] Multi-hop propagation (2-3 hops)
- [ ] Ranking by activation + hybrid mode
- [ ] Benchmark: vs pure vector similarity (+23%)

### Week 4-5: Tiered Memory + Async
- [ ] Redis hot tier layer
- [ ] Tier routing logic
- [ ] Async job queue with worker pool

### Week 6-7: Observability + SDK
- [ ] Metrics dashboard
- [ ] Python SDK: compression stats, tier policy
- [ ] Benchmark suite on provided datasets

### Week 8: Production Hardening
- [ ] Load testing (1000+ concurrent)
- [ ] Error handling
- [ ] Release prep

---

## Architecture

### Memory Flow
1. Content → `MemoryProcessor.ExtractFacts()` via LLM
2. Facts → importance score
3. Entities → extract and link in graph
4. Conflicts → `ResolveConflict` template

### Key Interfaces
- `GraphStore` (`internal/memory/store.go`): Neo4j operations
- `VectorStore` (`internal/memory/store.go`): Qdrant operations
- `Provider` (`internal/llm/provider.go`): LLM completions

---

## Critical Conventions

### Error Handling
- Use `safeHTTPError()` not `http.Error(w, err.Error(), ...)` in API handlers
- Error wrapping: `fmt.Errorf("feature: operation: %w", err)`

### Rate Limiter
- Has cleanup goroutine - call `Stop()` on shutdown
- Located at `cmd/server/api.go`

### Neo4j Timeouts
- Configurable via `NEO4J_QUERY_TIMEOUT` env var (default 60s)
- Use `c.queryTimeout()` helper method

### Batch Operations
- Use `GetMemoriesByIDs()` instead of loops
- Use `BatchCreateMemories()` / `BatchDeleteMemories()` for bulk

---

## SSO Providers

| Provider | Status | File |
|---------|--------|------|
| OIDC | ✅ Full | `internal/sso/oidc.go` |
| SAML | ✅ Full | `internal/sso/saml.go` |
| LDAP | ✅ Full | `internal/sso/ldap.go` |

---

## Features vs Mem0

Agent-memory exceeds Mem0 Pro/Enterprise:

| Feature | Mem0 | Agent-Memory |
|---------|------|--------------|
| Graph Memory | Pro | ✅ |
| Analytics | Pro | ✅ Advanced |
| SSO | Enterprise | ✅ OIDC+SAML+LDAP |
| Audit Logs | Enterprise | ✅ Full |
| Memory Versioning | ❌ | ✅ |
| Skill Chains | ❌ | ✅ |
| Compression | 80% | **85% → 90%+** |
| LDAP SSO | ❌ | ✅ |
| **ProMem Extraction** | ❌ | ✅ **PROPRIETARY** |
| **Spreading Activation** | ❌ | ✅ **PROPRIETARY** |
| **Tiered Memory** | ❌ | ✅ |

---

## Adding New Features

### New LLM Template
Add template in `internal/memory/templates.go`

### New Memory Processor Method
Add method in `internal/memory/processor.go`

### New Vector Provider
1. Add `ProviderType` in `internal/vector/provider.go`
2. Add implementation in `internal/vector/providers.go`

### New API Endpoint
Add handler in `cmd/server/api.go` with `safeHTTPError()`

---

## Environment Variables

```bash
# Neo4j
NEO4J_URI=bolt://localhost:7687
NEO4J_USER=neo4j
NEO4J_PASSWORD=secret
NEO4J_QUERY_TIMEOUT=60

# Qdrant
QDRANT_URL=http://localhost:6333

# LLM
LLM_PROVIDER=openai
LLM_API_KEY=sk-...

# Redis (for tiered memory)
REDIS_URL=redis://localhost:6379

# COMPRESSION ENGINE (NEW)
COMPRESSION_ENABLED=true
COMPRESSION_LLM_FAST_PROVIDER=openai
COMPRESSION_LLM_FAST_MODEL=gpt-4o-mini
COMPRESSION_LLM_VERIFY_PROVIDER=anthropic
COMPRESSION_LLM_VERIFY_MODEL=claude-3-5-sonnet
COMPRESSION_COMPLEXITY_THRESHOLD=0.6
COMPRESSION_MODE=extract
TIER_POLICY=balanced
```

---

## CLI Agent Commands

Run with `go run ./cmd/agent`:

```
/help           - Show help
/agents        - List agents
/init [prompt] - Initialize memory
/remember [txt] - Save memory
/memory        - View context
/search [query]- Search memories
/compact       - Compact context
/skills        - List skills
/exit          - Quit
```

---

## Stubs to Complete

- [ ] ProMem Extraction Engine (`internal/compression/extractor/`)
- [ ] Spreading Activation Retrieval (`internal/compression/retrieval/`)
- [ ] Async Compression Pipeline (`internal/compression/pipeline/`)
- [ ] Hybrid LLM Router (`internal/compression/llm/`)
- [ ] Tiered Memory System (`internal/memory/tier/`)
- [ ] Compression Observability (`internal/metrics/compression.go`)

---

## Key References

- ProMem Paper: "ProMem: Proactive Memory Extraction for LLM Agents" (arXiv:2601.04463)
- Synapse Paper: "Synapse: Empowering LLM Agents with Episodic-Semantic Memory" (arXiv:2601.02744)
- TurboQuant: Google KV cache compression (6x memory reduction)
- ComprExIT: Context Compression via Explicit Information Transmission (arXiv:2602.03784)