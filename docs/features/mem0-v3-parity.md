# Feature: Mem0 v3 Parity

**Priority**: P1 — Mem0 v3 (April 2026) introduced 4 innovations that improve accuracy by +20-26 points on benchmarks.
**Status**: Analyzed in `docs/mem0-v3-analysis.md`. None implemented yet.
**Estimated effort**: 1 week

> Full technical analysis with code samples: `docs/mem0-v3-analysis.md`

---

## The 4 Innovations to Implement

| # | Innovation | Benchmark Impact | Effort |
|---|-----------|-----------------|--------|
| 1 | Single-pass ADD-only extraction | ~50% latency reduction | 2 days |
| 2 | BM25 keyword search signal | Precision improvement | 1 day |
| 3 | Agent-generated facts as first-class | Recall 46%→100% | 0.5 days |
| 4 | Entity store in vector DB (no Neo4j for entities) | Reduced Neo4j coupling | 1 day |

---

## Innovation 1: Single-Pass ADD-Only Extraction

### What to change

**Current flow** (two LLM calls):
```
Input → [LLM 1: extract facts] → [LLM 2: decide ADD/UPDATE/DELETE vs existing] → store
```

**New flow** (one LLM call):
```
Input → retrieve top-10 existing (context only) → [LLM: extract facts] → hash dedup → store
```

### Create: `internal/memory/extraction/v3.go`

```go
package extraction

type ExtractionV3 struct {
    llmProvider llm.Provider
    vectorStore vector.Store
    dedupStore  *DedupStore
}

type DedupStore struct {
    mu    sync.RWMutex
    hashes map[string]string  // md5(content) → memoryID
    // Optionally backed by Redis for persistence across restarts
}

func (e *ExtractionV3) Extract(ctx context.Context, userID string, input string) (*ExtractionResult, error) {
    // 1. Get top-10 existing memories as context ONLY (not for diffing)
    existing, err := e.vectorStore.Search(ctx, userID, input, 10)
    if err != nil {
        return nil, fmt.Errorf("extraction v3: context fetch: %w", err)
    }
    
    // 2. Single LLM call — extract ALL distinct facts
    prompt := buildV3ExtractionPrompt(input, existing)
    response, err := e.llmProvider.Complete(ctx, prompt)
    if err != nil {
        return nil, fmt.Errorf("extraction v3: llm: %w", err)
    }
    
    facts := parseFactsFromJSON(response)
    
    // 3. Hash-based deduplication — skip facts already stored
    var newFacts []Fact
    for _, fact := range facts {
        hash := md5Hash(normalizeContent(fact.Content))
        if !e.dedupStore.Exists(hash) {
            newFacts = append(newFacts, fact)
        }
    }
    
    // 4. Batch embed + store new facts
    if len(newFacts) > 0 {
        embeddings, err := e.llmProvider.Embed(ctx, factsToTexts(newFacts))
        if err != nil {
            return nil, fmt.Errorf("extraction v3: embed: %w", err)
        }
        if err := e.vectorStore.BatchUpsert(ctx, userID, newFacts, embeddings); err != nil {
            return nil, fmt.Errorf("extraction v3: store: %w", err)
        }
        // Register hashes
        for _, fact := range newFacts {
            e.dedupStore.Put(md5Hash(normalizeContent(fact.Content)), fact.ID)
        }
    }
    
    return &ExtractionResult{
        NewFacts:    newFacts,
        SkippedDups: len(facts) - len(newFacts),
    }, nil
}

func buildV3ExtractionPrompt(input string, context []SearchResult) string {
    var b strings.Builder
    b.WriteString(`Extract all distinct facts from the following input.

Rules:
- Each fact must be atomic and self-contained (one piece of information per fact)
- Extract facts from BOTH user messages AND assistant/agent responses
- Agent statements like "I've booked X" or "I scheduled Y" are facts
- Do NOT decide ADD/UPDATE/DELETE — just extract what's factually stated
- Output JSON array: [{"content": "fact text", "category": "optional"}]

`)
    
    if len(context) > 0 {
        b.WriteString("Existing memories (for context reference only — do NOT diff against):\n")
        for _, m := range context {
            b.WriteString(fmt.Sprintf("- %s\n", m.Content))
        }
        b.WriteString("\n")
    }
    
    b.WriteString("Input:\n")
    b.WriteString(input)
    b.WriteString("\n\nFacts (JSON array):")
    return b.String()
}
```

### Wire into `internal/memory/service.go`

Add extraction mode selector:

```go
type ExtractionMode string

const (
    ExtractionModeV2 ExtractionMode = "v2"  // current two-pass
    ExtractionModeV3 ExtractionMode = "v3"  // new single-pass ADD-only
)

func (s *Service) CreateMemory(ctx context.Context, req CreateMemoryRequest) (*types.Memory, error) {
    if s.config.Memory.ExtractionMode == ExtractionModeV3 {
        return s.createMemoryV3(ctx, req)
    }
    return s.createMemoryV2(ctx, req)
}
```

**Config**: Add `MEMORY_EXTRACTION_MODE=v3` to `internal/config/config.go`.

---

## Innovation 2: BM25 Keyword Search Signal

### What to change

BM25 adds keyword matching as a scoring signal alongside vector similarity. Qdrant supports sparse vectors for BM25 via its native `fastembed` integration.

### Update: `internal/vector/provider.go`

Add `KeywordSearch` to the `VectorStore` interface:

```go
type VectorStore interface {
    // Existing:
    Search(ctx context.Context, userID, query string, topK int) ([]SearchResult, error)
    Upsert(ctx context.Context, userID string, memory Memory, embedding []float32) error
    BatchUpsert(ctx context.Context, userID string, memories []Memory, embeddings [][]float32) error
    
    // New for BM25:
    KeywordSearch(ctx context.Context, userID, query string, topK int) (map[string]float64, error)
    // Returns: memoryID → BM25 score map
    
    // New for batch entity search:
    SearchBatch(ctx context.Context, collection string, queries []string, topK int) (map[string][]SearchResult, error)
}
```

### Update: `internal/vector/providers.go` (Qdrant implementation)

```go
// KeywordSearch uses Qdrant sparse vectors for BM25-style keyword matching.
// Requires the collection to have sparse vector support enabled.
func (q *QdrantProvider) KeywordSearch(ctx context.Context, userID, query string, topK int) (map[string]float64, error) {
    // Option 1: Use Qdrant's built-in BM25 via sparse vectors
    // Requires: collection configured with sparse vector field
    // Qdrant API: POST /collections/{name}/points/search with sparse vector
    
    // Option 2 (fallback): Simple TF-IDF approximation using existing payload filtering
    // Filter points where payload.content contains any word from query
    // Score by count of matching words / total words in query
    
    // Start with Option 2 as a working fallback, then upgrade to sparse vectors
    words := tokenize(query)
    scores := make(map[string]float64)
    
    for _, word := range words {
        results, err := q.searchByPayloadField(ctx, userID, "content", word, topK)
        if err != nil {
            continue
        }
        for _, r := range results {
            scores[r.ID] += 1.0 / float64(len(words))
        }
    }
    
    return scores, nil
}
```

### Create: `internal/retrieval/hybrid_v3.go`

Three-signal fusion (semantic 60% + BM25 25% + entity 15%):

```go
type HybridRetrieverV3 struct {
    vectorStore  vector.Store
    entityStore  *EntityStore
    weights      HybridWeights
}

type HybridWeights struct {
    Semantic float64  // default 0.60
    BM25     float64  // default 0.25
    Entity   float64  // default 0.15
}

func (h *HybridRetrieverV3) Search(ctx context.Context, userID, query string, topK int) ([]SearchResult, error) {
    // 1. Parallel search: semantic + BM25 + entity
    var (
        semanticResults map[string]float64
        bm25Results     map[string]float64
        entityScores    map[string]float64
        wg              sync.WaitGroup
        mu              sync.Mutex
    )
    
    wg.Add(3)
    
    go func() {
        defer wg.Done()
        results, err := h.vectorStore.Search(ctx, userID, query, topK*2)
        if err == nil {
            mu.Lock()
            semanticResults = resultsToScoreMap(results)
            mu.Unlock()
        }
    }()
    
    go func() {
        defer wg.Done()
        scores, err := h.vectorStore.KeywordSearch(ctx, userID, query, topK*2)
        if err == nil {
            mu.Lock()
            bm25Results = scores
            mu.Unlock()
        }
    }()
    
    go func() {
        defer wg.Done()
        entities := extractQueryEntities(query)
        if len(entities) > 0 {
            scores, err := h.entityStore.Search(ctx, userID, entities, topK)
            if err == nil {
                mu.Lock()
                entityScores = scores
                mu.Unlock()
            }
        }
    }()
    
    wg.Wait()
    
    // 2. Normalize + fuse scores
    return h.fuseAndRank(semanticResults, bm25Results, entityScores, topK), nil
}
```

---

## Innovation 3: Agent-Generated Facts as First-Class

### What to change

The extraction prompt must explicitly instruct the LLM to capture both user and assistant content.

This is already partially handled by `buildV3ExtractionPrompt` above (see the "Rules" section that says "Extract facts from BOTH user messages AND assistant/agent responses").

**Also update**: The conversation format passed to extraction should include role labels:

```go
func (e *ExtractionV3) ExtractFromConversation(ctx context.Context, userID string, messages []Message) (*ExtractionResult, error) {
    var buf strings.Builder
    for _, msg := range messages {
        roleLabel := map[string]string{
            "user":      "User",
            "assistant": "Assistant",
            "system":    "System",
            "tool":      "Tool",
        }[msg.Role]
        buf.WriteString(fmt.Sprintf("%s: %s\n", roleLabel, msg.Content))
    }
    return e.Extract(ctx, userID, buf.String())
}
```

**Wire**: In `handleAddMessage` (adding a session message), after storing the message, if extraction is enabled, call `ExtractFromConversation` with the last 5-10 messages.

---

## Innovation 4: Entity Store in Vector DB

### What to change

Currently entities go to Neo4j only. Adding them to Qdrant enables entity-based score boosting without graph traversal.

### Create: `internal/memory/entity/store.go`

```go
package entity

const EntityCollectionSuffix = "_entities"

type EntityStore struct {
    vectorStore vector.Store
    llmProvider llm.Provider
}

// Store saves entities extracted from a memory into the vector entity collection
func (s *EntityStore) Store(ctx context.Context, userID string, entities []Entity) error {
    if len(entities) == 0 {
        return nil
    }
    
    collection := userID + EntityCollectionSuffix
    
    texts := make([]string, len(entities))
    for i, e := range entities {
        texts[i] = e.Text
    }
    
    embeddings, err := s.llmProvider.Embed(ctx, texts)
    if err != nil {
        return fmt.Errorf("entity store: embed: %w", err)
    }
    
    memories := make([]vector.Memory, len(entities))
    for i, e := range entities {
        memories[i] = vector.Memory{
            ID:       uuid.New().String(),
            UserID:   userID,
            Content:  e.Text,
            Type:     "entity:" + e.Type,
            Metadata: map[string]interface{}{"memory_id": e.MemoryID},
        }
    }
    
    return s.vectorStore.BatchUpsert(ctx, collection, memories, embeddings)
}

// Search finds memory IDs associated with query entities
// Returns: memoryID → entity match score
func (s *EntityStore) Search(ctx context.Context, userID string, queryEntities []Entity, topK int) (map[string]float64, error) {
    collection := userID + EntityCollectionSuffix
    
    queries := make([]string, len(queryEntities))
    for i, e := range queryEntities {
        queries[i] = e.Text
    }
    
    batchResults, err := s.vectorStore.SearchBatch(ctx, collection, queries, topK)
    if err != nil {
        return nil, fmt.Errorf("entity store: batch search: %w", err)
    }
    
    // Map memory_id → max entity score
    scores := make(map[string]float64)
    for _, results := range batchResults {
        for _, r := range results {
            memoryID, _ := r.Metadata["memory_id"].(string)
            if memoryID != "" && r.Score > scores[memoryID] {
                scores[memoryID] = r.Score
            }
        }
    }
    
    return scores, nil
}
```

### Create: `internal/memory/entity/extractor.go`

NER without external dependencies (pure Go):

```go
package entity

// ExtractEntities extracts named entities using simple heuristics
// For production accuracy, replace with LLM-based extraction or prose library
func ExtractEntities(content string, memoryID string) []Entity {
    var entities []Entity
    
    // 1. Quoted text
    for _, match := range quotedRegex.FindAllString(content, -1) {
        entities = append(entities, Entity{Text: match[1:len(match)-1], Type: "QUOTED", MemoryID: memoryID})
    }
    
    // 2. Capitalized word sequences (likely proper nouns)
    words := strings.Fields(content)
    for i, word := range words {
        if isCapitalized(word) && !isCommonWord(word) {
            // Check for multi-word entity (consecutive capitalized words)
            j := i + 1
            for j < len(words) && isCapitalized(words[j]) {
                j++
            }
            entityText := strings.Join(words[i:j], " ")
            entities = append(entities, Entity{Text: entityText, Type: "PROPER_NOUN", MemoryID: memoryID})
        }
    }
    
    return deduplicateEntities(entities)
}
```

---

## Config Additions

Add to `internal/config/config.go`:

```go
type MemoryConfig struct {
    // ... existing fields ...
    ExtractionMode   string `env:"MEMORY_EXTRACTION_MODE" default:"v2"`  // "v2" | "v3"
    EntityStoreEnabled bool `env:"ENTITY_STORE_ENABLED" default:"false"` // enable vector entity store
    BM25Enabled       bool `env:"BM25_ENABLED" default:"false"`          // enable BM25 search signal
}
```

Start with all new features opt-in (default false) to avoid breaking existing deployments.

---

## Tests

Create `internal/memory/extraction/v3_test.go`:
```go
// Deduplication: same content twice → only stored once
// Agent facts: assistant message "I booked a flight" → extracted as fact
// No UPDATE/DELETE: extraction only adds, never removes
// Latency: single-pass should be ~50% faster than two-pass (benchmark test)

func BenchmarkExtractionV2vsV3(b *testing.B) {
    input := "long conversation with many facts..."
    b.Run("v2", func(b *testing.B) { /* two-pass */ })
    b.Run("v3", func(b *testing.B) { /* single-pass */ })
}
```
