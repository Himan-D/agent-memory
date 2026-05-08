# Memory Space Orchestrator

> Actionable build spec for memory operations, search, tiering, chunking, and self-improvement.
> Status as of 2026-05-09: Core CRUD and search functional. Chunking and self-improvement empty.

---

## What This System Does

The Memory Space coordinates everything related to storing, retrieving, and improving memories. It sits between the API layer and the storage backends (Neo4j + Qdrant + Redis).

**Responsibilities:**
1. Create, update, delete memories with LLM processing
2. Run semantic + hybrid + spreading-activation search
3. Route memories to correct tier (Working → Hot → Cold → Archive)
4. Chunk large memories for better retrieval precision
5. Self-improve via user feedback (importance adjustment, synonym learning)
6. Maintain memory relationships and versioning

---

## Current State

| Component | File | Status |
|-----------|------|--------|
| Memory CRUD | `internal/memory/service.go` | ✅ Full |
| Neo4j Graph | `internal/memory/neo4j/` | ✅ Full |
| Tier Routing | `internal/memory/tier/router.go` | ✅ Working→Hot→Cold |
| Semantic Search | `internal/vector/` | ✅ Full |
| Hybrid Search | `internal/retrieval/` | ✅ Full |
| Spreading Activation | `internal/compression/retrieval/` | ✅ Full |
| Memory Chunking | `internal/memory/chunking/` | ❌ Empty directory |
| Self-Improvement | `internal/memory/self_improve/` | ❌ Empty directory |
| Session Package | `internal/memory/session/` | ❌ Empty directory |
| Search Package | `internal/memory/search/` | ❌ Empty directory |
| Archive Backend | `internal/memory/tier/archive.go` | ❌ Missing |

---

## Task 1: Memory Chunking

**Why**: Large memory content (>512 tokens) retrieves poorly — the vector embedding averages out too much. Chunking gives each piece its own vector, improving recall for specific facts within long content.

**Create**: `internal/memory/chunking/splitter.go`

```go
package chunking

type ChunkConfig struct {
    MaxTokens      int    // Default: 512
    OverlapTokens  int    // Default: 50 — overlap preserves context at boundaries
    Strategy       string // "sentence" | "paragraph" | "fixed"
    MinChunkTokens int    // Default: 100 — ignore tiny trailing chunks
}

type Chunk struct {
    Index         int
    Content       string
    TokenCount    int
    StartOffset   int    // byte offset in original content
    EndOffset     int
    ParentMemoryID string
}

type Splitter struct {
    config ChunkConfig
}

func (s *Splitter) Split(content string, parentMemoryID string) []Chunk {
    // Strategy "sentence": split on period/question mark/exclamation
    //   → accumulate sentences until token count > MaxTokens
    //   → start new chunk with overlap from end of previous chunk
    // Strategy "paragraph": split on double newlines
    // Strategy "fixed": split every MaxTokens tokens with OverlapTokens overlap
}

// TokenCount estimates tokens using len(content)/4 approximation
// For production accuracy, use tiktoken-go if available
func TokenCount(content string) int {
    return len(content) / 4
}
```

**Create**: `internal/memory/chunking/merger.go`

```go
// Merger combines top-scoring chunks from the same parent memory
// Called during search result post-processing
type Merger struct{}

type ChunkResult struct {
    ParentMemoryID string
    Chunks         []ScoredChunk
    BestScore      float64
}

// MergeChunkResults groups search results by parent memory ID and picks best chunks
func (m *Merger) MergeChunkResults(results []SearchResult, topChunksPerMemory int) []SearchResult {
    // Group by ParentMemoryID
    // For each parent: keep top N chunks by score
    // Return one result per parent with combined score = max(chunk scores)
    // Content = concatenation of top chunks (deduped by offset overlap)
}
```

**Wire into `internal/memory/service.go`**:

In `CreateMemory()`:
```go
func (s *Service) CreateMemory(ctx context.Context, req CreateMemoryRequest) (*types.Memory, error) {
    tokenCount := chunking.TokenCount(req.Content)
    
    if tokenCount > s.config.Chunking.MaxTokens {
        // Split into chunks
        chunks := s.chunker.Split(req.Content, "") // parentID set after first insert
        
        // Create parent memory record (content = summary or first 200 chars)
        parent := s.createMemoryRecord(req, generateSummary(chunks))
        parentID := parent.ID
        
        // Create chunk memories linked to parent
        for i, chunk := range chunks {
            chunkReq := req
            chunkReq.Content = chunk.Content
            chunkReq.ParentMemoryID = parentID
            chunkReq.Metadata["chunk_index"] = i
            s.createMemoryRecord(chunkReq, chunk.Content)
        }
        return parent, nil
    }
    
    // Normal single-chunk path
    return s.createMemoryRecord(req, req.Content)
}
```

In `Search()`:
```go
// After getting search results:
if s.config.Chunking.Enabled {
    results = s.merger.MergeChunkResults(results, 3)
}
```

**Config additions** in `internal/config/config.go`:
```go
type ChunkingConfig struct {
    Enabled        bool   `env:"CHUNKING_ENABLED" default:"true"`
    MaxTokens      int    `env:"CHUNKING_MAX_TOKENS" default:"512"`
    OverlapTokens  int    `env:"CHUNKING_OVERLAP_TOKENS" default:"50"`
    Strategy       string `env:"CHUNKING_STRATEGY" default:"sentence"`
}
```

**Tests**: Create `internal/memory/chunking/splitter_test.go`:
- Test that content >512 tokens is split
- Test overlap: last 50 tokens of chunk N appear at start of chunk N+1
- Test merger: two chunks from same parent → one result with max score
- Test edge case: content exactly at boundary, very short content

---

## Task 2: Self-Improvement Engine

**Why**: User feedback is currently stored but has zero effect on future behavior. The self-improvement engine closes this loop.

**Create**: `internal/memory/self_improve/engine.go`

```go
package selfimprove

type SelfImprovementEngine struct {
    memoryStore  MemoryStore
    synonymStore *SynonymStore
    llmProvider  llm.Provider
}

// ProcessFeedback is the main entry point, called from the feedback API handler
func (e *SelfImprovementEngine) ProcessFeedback(
    ctx context.Context,
    memoryID string,
    feedbackType string,  // "positive" | "negative" | "very_negative"
    userID string,
    comment string,
) error {
    memory, err := e.memoryStore.GetMemory(ctx, memoryID)
    if err != nil {
        return fmt.Errorf("self-improve: get memory: %w", err)
    }

    switch feedbackType {
    case "positive":
        // Bump importance score
        newScore := min(1.0, memory.ImportanceScore + 0.1)
        return e.memoryStore.UpdateImportanceScore(ctx, memoryID, newScore)

    case "negative":
        // Reduce importance
        newScore := max(0.0, memory.ImportanceScore - 0.2)
        if err := e.memoryStore.UpdateImportanceScore(ctx, memoryID, newScore); err != nil {
            return err
        }
        // Queue for LLM correction
        return e.queueForCorrection(ctx, memory, comment)

    case "very_negative":
        // Reduce importance to 0, flag for human review
        if err := e.memoryStore.UpdateImportanceScore(ctx, memoryID, 0.0); err != nil {
            return err
        }
        return e.memoryStore.FlagForReview(ctx, memoryID, comment)
    }
    return nil
}

// queueForCorrection calls LLM to suggest corrected content
func (e *SelfImprovementEngine) queueForCorrection(
    ctx context.Context,
    memory *types.Memory,
    comment string,
) error {
    prompt := fmt.Sprintf(`
Memory: %s

User feedback: %s

Provide a corrected version of this memory that addresses the feedback.
Return only the corrected content, no explanation.
`, memory.Content, comment)

    corrected, err := e.llmProvider.Complete(ctx, prompt)
    if err != nil {
        return fmt.Errorf("self-improve: correction llm: %w", err)
    }
    
    return e.memoryStore.SuggestCorrection(ctx, memory.ID, corrected)
}
```

**Create**: `internal/memory/self_improve/synonym_store.go`

```go
// SynonymStore learns term equivalences from feedback patterns.
// If a user consistently finds memory X when searching for term Y,
// and X doesn't contain Y, we store (Y → terms in X) as a synonym pair.

type SynonymPair struct {
    QueryTerm    string
    MemoryTerms  []string
    Confidence   float64
    TenantID     string
    CreatedAt    time.Time
}

type SynonymStore struct {
    graphStore GraphStore
}

// Record is called when a user adds positive feedback to a search result
func (s *SynonymStore) Record(ctx context.Context, queryTerm, memoryID, tenantID string) error {
    // Extract key terms from memory content
    // If queryTerm not in memory content → candidate synonym pair
    // After 3+ occurrences of (queryTerm → memory terms) → persist as synonym
}

// GetSynonyms returns learned synonyms for query expansion
func (s *SynonymStore) GetSynonyms(ctx context.Context, term, tenantID string) ([]string, error) {
    // Look up synonym pairs in Neo4j where QueryTerm matches
    // Return expanded terms to add to search
}
```

**Wire into feedback handler** in `cmd/server/api.go`:

```go
func (a *API) handleAddFeedback(w http.ResponseWriter, r *http.Request) {
    // ... existing validation ...
    
    // After storing feedback in DB:
    if a.selfImprove != nil {
        go func() {
            if err := a.selfImprove.ProcessFeedback(
                context.Background(),
                req.MemoryID,
                req.Type,
                tenantID,
                req.Comment,
            ); err != nil {
                a.logger.Error("self-improve feedback", "error", err)
            }
        }()
    }
    
    jsonOK(w, map[string]string{"status": "accepted"})
}
```

**Wire synonym expansion into search** in `internal/memory/service.go`:

```go
// In Search(), before calling vector store:
if a.synonymStore != nil {
    expanded, _ := a.synonymStore.GetSynonyms(ctx, query, tenantID)
    if len(expanded) > 0 {
        query = query + " " + strings.Join(expanded, " ")
    }
}
```

**Tests**: `internal/memory/self_improve/engine_test.go`:
- Positive feedback → importance increases by 0.1 (capped at 1.0)
- Negative feedback → importance decreases by 0.2 (floored at 0.0)
- Negative feedback with comment → correction suggested via LLM
- Very negative → importance=0, flagged for review

---

## Task 3: Memory Search Package

**Why**: `internal/memory/search/` is empty but there's duplicate search logic spread across `internal/retrieval/` and `internal/vector/`. This package unifies them.

**Create**: `internal/memory/search/coordinator.go`

```go
package search

// SearchCoordinator is the single entry point for all memory search operations.
// It selects the right search strategy based on request parameters.
type SearchCoordinator struct {
    vectorStore    vector.Store
    graphStore     neo4j.GraphStore
    reranker       reranker.Reranker
    multiSignal    *retrieval.MultiSignalRetriever
    spreading      *compression_retrieval.SpreadingActivation
    synonymStore   *selfimprove.SynonymStore
    config         SearchConfig
}

type SearchRequest struct {
    Query      string
    UserID     string
    TenantID   string
    Mode       string    // "semantic" | "hybrid" | "spreading" | "graph"
    Limit      int
    Threshold  float64
    Filters    SearchFilters
    Rerank     bool
}

type SearchConfig struct {
    DefaultMode     string  // env: SEARCH_DEFAULT_MODE, default "hybrid"
    SpreadingEnabled bool   // env: SPREADING_ACTIVATION_ENABLED, default true
    SynonymExpansion bool   // env: SYNONYM_EXPANSION_ENABLED, default true
}

func (c *SearchCoordinator) Search(ctx context.Context, req SearchRequest) ([]SearchResult, error) {
    // 1. Optional: expand query with learned synonyms
    query := c.maybeExpandQuery(ctx, req.Query, req.TenantID)
    
    // 2. Route to appropriate search strategy
    switch req.Mode {
    case "spreading":
        return c.spreadingSearch(ctx, query, req)
    case "graph":
        return c.graphSearch(ctx, query, req)
    case "hybrid":
        return c.hybridSearch(ctx, query, req)
    default:
        return c.semanticSearch(ctx, query, req)
    }
}
```

**Create**: `internal/memory/search/filters.go`

```go
// SearchFilters provides AND/OR/NOT filter logic
// Already partially implemented in internal/retrieval/ — move/consolidate here
type SearchFilters struct {
    Logic  string         // "AND" | "OR"
    Rules  []FilterRule
    Nested []SearchFilters
}

type FilterRule struct {
    Field    string  // "category" | "importance" | "tags" | "user_id" | "agent_id"
    Operator string  // "eq" | "in" | "contains" | "gt" | "lt"
    Value    interface{}
}
```

---

## Task 4: Memory Session Package

**Why**: Session handling is spread across `cmd/server/api.go` and `internal/memory/service.go`. Sessions need a dedicated abstraction for MCP server and multi-agent scenarios.

**Create**: `internal/memory/session/manager.go`

```go
package session

type SessionManager struct {
    store  SessionStore
    config SessionConfig
}

type SessionConfig struct {
    MaxMessages    int           // env: SESSION_MAX_MESSAGES, default 1000
    TTL            time.Duration // env: SESSION_TTL, default 24h
    AutoCompress   bool          // env: SESSION_AUTO_COMPRESS, default true
    CompressAfter  int           // Compress session when > N messages
}

type Session struct {
    ID        string
    AgentID   string
    TenantID  string
    Messages  []Message
    Metadata  map[string]interface{}
    CreatedAt time.Time
    UpdatedAt time.Time
    ExpiresAt *time.Time
}

// GetContext returns the last N messages formatted for LLM injection
// Respects context window limits via token counting
func (m *SessionManager) GetContext(ctx context.Context, sessionID string, limit int) (string, error) {
    session, err := m.store.GetSession(ctx, sessionID)
    if err != nil {
        return "", err
    }
    
    messages := session.Messages
    if len(messages) > limit {
        messages = messages[len(messages)-limit:]
    }
    
    var buf strings.Builder
    for _, msg := range messages {
        buf.WriteString(fmt.Sprintf("%s: %s\n", msg.Role, msg.Content))
    }
    return buf.String(), nil
}

// Compact summarizes old messages to free up context window
func (m *SessionManager) Compact(ctx context.Context, sessionID string, llm llm.Provider) error {
    // Summarize messages older than most recent CompressAfter messages
    // Store summary as a system message
    // Delete original messages
}
```

---

## Testing Checklist

```bash
# After implementing chunking:
go test ./internal/memory/chunking/...

# After implementing self-improvement:
go test ./internal/memory/self_improve/...

# After implementing search coordinator:
go test ./internal/memory/search/...

# Full build
go build ./...

# Integration: test chunking end-to-end
curl -X POST http://localhost:8080/memories \
  -H "X-API-Key: test" \
  -d '{"content": "very long content > 512 tokens...", "user_id": "u1"}'
# → Should create parent + chunk memories
# → Search should return parent with merged chunk score
```
