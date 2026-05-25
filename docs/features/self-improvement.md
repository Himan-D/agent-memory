# Feature: Self-Improvement System

**Priority**: P1 — Feedback is stored but has zero effect on behavior.
**Status**: `internal/memory/self_improve/` directory exists and is empty.
**Estimated effort**: 2-3 days

---

## What to Build

Three components in `internal/memory/self_improve/`:

| File | Responsibility |
|------|---------------|
| `engine.go` | Processes feedback, adjusts importance, queues corrections |
| `synonym_store.go` | Learns query→term mappings from repeated positive feedback patterns |
| `correction_queue.go` | Async LLM-powered content correction for negatively-rated memories |

---

## Component 1: `engine.go`

```go
package selfimprove

type Engine struct {
    memoryStore  MemoryStore      // reads/updates memories
    synonymStore *SynonymStore    // learns term mappings
    corrQueue    *CorrectionQueue // queues LLM corrections
    config       EngineConfig
}

type EngineConfig struct {
    PositiveDelta    float64 // Default: 0.1 — how much positive feedback increases score
    NegativeDelta    float64 // Default: 0.2 — how much negative feedback decreases score
    MaxImportance    float64 // Default: 1.0
    MinImportance    float64 // Default: 0.0
    AutoCorrectEnabled bool  // env: SELF_IMPROVE_AUTO_CORRECT, default true
    SynonymEnabled    bool   // env: SELF_IMPROVE_SYNONYM_LEARNING, default true
}

// MemoryStore interface required by the engine
type MemoryStore interface {
    GetMemory(ctx context.Context, id string) (*types.Memory, error)
    UpdateImportanceScore(ctx context.Context, id string, score float64) error
    FlagForReview(ctx context.Context, id string, reason string) error
    SuggestCorrection(ctx context.Context, id string, correctedContent string) error
}

// ProcessFeedback is called by cmd/server/api.go handleAddFeedback
func (e *Engine) ProcessFeedback(
    ctx context.Context,
    memoryID string,
    feedbackType string,
    comment string,
    queryTerm string,   // the search query that returned this memory (for synonym learning)
    tenantID string,
) error {
    memory, err := e.memoryStore.GetMemory(ctx, memoryID)
    if err != nil {
        return fmt.Errorf("self-improve: get memory %s: %w", memoryID, err)
    }
    
    switch feedbackType {
    case "positive":
        newScore := clamp(memory.ImportanceScore+e.config.PositiveDelta, e.config.MinImportance, e.config.MaxImportance)
        if err := e.memoryStore.UpdateImportanceScore(ctx, memoryID, newScore); err != nil {
            return fmt.Errorf("self-improve: update score: %w", err)
        }
        // Synonym learning: if query term doesn't appear in memory content → candidate synonym
        if e.config.SynonymEnabled && queryTerm != "" {
            e.synonymStore.RecordHit(ctx, queryTerm, memory.Content, tenantID)
        }
        
    case "negative":
        newScore := clamp(memory.ImportanceScore-e.config.NegativeDelta, e.config.MinImportance, e.config.MaxImportance)
        if err := e.memoryStore.UpdateImportanceScore(ctx, memoryID, newScore); err != nil {
            return fmt.Errorf("self-improve: update score: %w", err)
        }
        if e.config.AutoCorrectEnabled && comment != "" {
            e.corrQueue.Enqueue(CorrectionJob{
                MemoryID:      memoryID,
                OriginalContent: memory.Content,
                UserComment:   comment,
                TenantID:      tenantID,
            })
        }
        
    case "very_negative":
        if err := e.memoryStore.UpdateImportanceScore(ctx, memoryID, e.config.MinImportance); err != nil {
            return fmt.Errorf("self-improve: zero score: %w", err)
        }
        if err := e.memoryStore.FlagForReview(ctx, memoryID, comment); err != nil {
            return fmt.Errorf("self-improve: flag review: %w", err)
        }
    }
    
    return nil
}

func clamp(val, min, max float64) float64 {
    if val < min { return min }
    if val > max { return max }
    return val
}
```

---

## Component 2: `synonym_store.go`

```go
package selfimprove

// SynonymStore records query→memory-term mappings.
// When user gives positive feedback to a memory that doesn't contain their query term,
// that's evidence the query term is a synonym for the memory's terms.
// After SynonymThreshold hits, store the pair permanently.

type SynonymStore struct {
    graphStore   GraphStore
    hitCounts    map[synonymKey]int   // in-memory counter before threshold
    mu           sync.RWMutex
    config       SynonymConfig
}

type synonymKey struct {
    QueryTerm string
    TenantID  string
}

type SynonymConfig struct {
    Threshold     int     // Default: 3 — hits before persisting synonym
    MaxSynonyms   int     // Default: 5 — max terms to associate
    MinTermLength int     // Default: 3 — ignore short words
}

// RecordHit records that queryTerm found memory with memoryContent useful
func (s *SynonymStore) RecordHit(ctx context.Context, queryTerm, memoryContent, tenantID string) {
    // Skip if queryTerm appears in memoryContent (not a synonym, it's the same)
    if strings.Contains(strings.ToLower(memoryContent), strings.ToLower(queryTerm)) {
        return
    }
    
    key := synonymKey{QueryTerm: strings.ToLower(queryTerm), TenantID: tenantID}
    
    s.mu.Lock()
    s.hitCounts[key]++
    count := s.hitCounts[key]
    s.mu.Unlock()
    
    if count >= s.config.Threshold {
        // Extract key terms from memory content
        terms := extractKeyTerms(memoryContent, s.config.MaxSynonyms, s.config.MinTermLength)
        s.persistSynonym(ctx, queryTerm, terms, tenantID)
        
        s.mu.Lock()
        delete(s.hitCounts, key)  // reset counter
        s.mu.Unlock()
    }
}

// GetExpansions returns synonym terms for a query, used to expand search
func (s *SynonymStore) GetExpansions(ctx context.Context, query string, tenantID string) []string {
    // Look up in Neo4j: MATCH (syn:Synonym {query: query, tenantID: tenantID}) RETURN syn.terms
    // Return terms to append to search query
}

// extractKeyTerms returns the most significant non-stop words from content
func extractKeyTerms(content string, maxTerms, minLength int) []string {
    words := tokenize(strings.ToLower(content))
    var terms []string
    for _, w := range words {
        if len(w) >= minLength && !isStopWord(w) {
            terms = append(terms, w)
            if len(terms) >= maxTerms {
                break
            }
        }
    }
    return terms
}
```

---

## Component 3: `correction_queue.go`

```go
package selfimprove

// CorrectionQueue processes negatively-rated memories via LLM
// to suggest improved content. Non-blocking — runs in background.

type CorrectionQueue struct {
    jobs        chan CorrectionJob
    llmProvider llm.Provider
    memoryStore MemoryStore
    workerCount int   // Default: 2
}

type CorrectionJob struct {
    MemoryID        string
    OriginalContent string
    UserComment     string
    TenantID        string
    CreatedAt       time.Time
}

func (q *CorrectionQueue) Start(ctx context.Context) {
    for i := 0; i < q.workerCount; i++ {
        go q.worker(ctx)
    }
}

func (q *CorrectionQueue) Enqueue(job CorrectionJob) {
    job.CreatedAt = time.Now()
    select {
    case q.jobs <- job:
    default:
        // Queue full — drop job rather than block (non-critical path)
    }
}

func (q *CorrectionQueue) worker(ctx context.Context) {
    for {
        select {
        case job := <-q.jobs:
            q.processJob(ctx, job)
        case <-ctx.Done():
            return
        }
    }
}

func (q *CorrectionQueue) processJob(ctx context.Context, job CorrectionJob) {
    prompt := fmt.Sprintf(`
You are correcting a stored memory based on user feedback.

Original memory:
%s

User feedback: %s

Provide a corrected version that addresses the feedback.
Keep the same format and level of detail.
Output only the corrected memory content, nothing else.
`, job.OriginalContent, job.UserComment)

    corrected, err := q.llmProvider.Complete(ctx, prompt)
    if err != nil {
        // Log but don't fail — correction is best-effort
        return
    }
    
    corrected = strings.TrimSpace(corrected)
    if corrected == "" || corrected == job.OriginalContent {
        return  // LLM didn't suggest meaningful change
    }
    
    // Store as a suggestion (human review required before applying)
    q.memoryStore.SuggestCorrection(ctx, job.MemoryID, corrected)
}
```

---

## Integration: Wire into `cmd/server/api.go`

**In `handleAddFeedback`**, after storing the feedback record in the DB, kick off the engine asynchronously:

```go
// Wire self-improvement (non-blocking)
if a.selfImprove != nil {
    queryTerm := r.URL.Query().Get("query_term")  // caller can pass the search query that found this memory
    go func() {
        if err := a.selfImprove.ProcessFeedback(
            context.Background(),
            req.MemoryID,
            req.Type,
            req.Comment,
            queryTerm,
            tenantID,
        ); err != nil {
            a.logger.Warn("self-improve feedback processing failed", "error", err)
        }
    }()
}
```

**In `handleSearch`**, before performing vector search, expand query with learned synonyms:

```go
query := req.Query
if a.selfImprove != nil {
    expansions, _ := a.selfImprove.SynonymStore.GetExpansions(r.Context(), query, tenantID)
    if len(expansions) > 0 {
        query = query + " " + strings.Join(expansions, " ")
    }
}
```

**New API endpoint** — get correction suggestions for a memory:

```go
// GET /memories/{id}/corrections
// Returns pending correction suggestions for human review
r.With(a.requireAuth, a.requireScope("write")).Get("/memories/{id}/corrections", a.handleGetCorrections)
r.With(a.requireAuth, a.requireScope("write")).Post("/memories/{id}/corrections/{corrID}/apply", a.handleApplyCorrection)
```

---

## Config Additions

```bash
SELF_IMPROVE_ENABLED=true
SELF_IMPROVE_AUTO_CORRECT=true          # Queue LLM corrections for negative feedback
SELF_IMPROVE_SYNONYM_LEARNING=true      # Learn query→term synonyms from positive feedback
SELF_IMPROVE_SYNONYM_THRESHOLD=3        # Hits before persisting synonym
SELF_IMPROVE_POSITIVE_DELTA=0.1         # Importance increase per positive feedback
SELF_IMPROVE_NEGATIVE_DELTA=0.2         # Importance decrease per negative feedback
SELF_IMPROVE_CORRECTION_WORKERS=2       # Background correction worker count
```

---

## Tests

`internal/memory/self_improve/engine_test.go`:
```go
func TestPositiveFeedback_IncreasesImportance(t *testing.T) {
    // Memory with score 0.5 → after positive feedback → score 0.6
}

func TestNegativeFeedback_DecreasesImportance(t *testing.T) {
    // Memory with score 0.1 → after negative feedback → score 0.0 (clamped, not -0.1)
}

func TestVeryNegativeFeedback_ZeroesAndFlags(t *testing.T) {
    // Memory with score 0.8 → after very_negative → score 0.0, flagged=true
}

func TestSynonymLearning_AfterThresholdHits(t *testing.T) {
    // 3 hits of ("ML" finding memory "machine learning") → persist synonym
}

func TestSynonymLearning_NoHitIfTermInContent(t *testing.T) {
    // "machine learning" finding memory with "machine learning" → no synonym (it's not a synonym)
}

func TestCorrectionQueue_LLMCalledForNegativeFeedback(t *testing.T) {
    // Negative feedback with comment → LLM called → correction suggested
}
```
