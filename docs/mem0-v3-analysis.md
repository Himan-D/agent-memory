# Mem0 v3 Memory Algorithm - Technical Analysis for Hystersis

## Overview

Mem0 v3 (April 2026) represents a ground-up redesign achieving:
- **+20 points on LoCoMo** (71.4 → 91.6)
- **+26 points on LongMemEval** (67.8 → 93.4)
- **~50% extraction latency reduction**
- **Under 7K tokens per retrieval** (vs 25K+ for full-context)

The four core innovations:
1. Single-pass ADD-only extraction
2. Multi-signal hybrid retrieval
3. Agent-generated facts as first-class
4. Entity linking without external graph stores

---

## 1. Single-Pass ADD-Only Extraction

### Before (v2): Two LLM Calls

```
Input → [LLM 1: Extract facts] → [LLM 2: Decide ADD/UPDATE/DELETE] → Vector store
```

- Call 1: Extract candidate facts from conversation
- Call 2: Diff against existing memories, decide action per fact

### After (v3): Single LLM Call

```
Input → Retrieve top-10 related memories (context only) → [LLM: Extract facts] → Hash dedup → Store
```

**Key insight**: The model spends capacity on *understanding* input, not diffing against existing state.

### Implementation Details

```
Flow:
1. Retrieve top-10 existing memories (semantic similarity) as context ONLY
2. Single LLM call extracts ALL distinct facts from input
3. Hash-based deduplication (MD5) against existing memories
4. Batch embed extracted memories
5. Batch insert to vector store
6. Extract entities for linking
```

**Prompt design considerations**:
- Instructions focus on extracting discrete, atomic facts
- No UPDATE/DELETE decision logic needed in prompt
- Agent-generated facts explicitly captured (see section 3)

**Why it works better**:
- Single LLM call = ~50% latency reduction
- No UPDATE/DELETE means no information loss
- Retrieval ranking handles conflicts naturally (most recent/relevant surfaces)
- Model capacity used for understanding, not diffing

### Hystersis Implementation

```go
// internal/memory/extraction/v3.go

type ExtractionV3 struct {
    llmProvider llm.Provider
    vectorStore VectorStore
    dedupStore  *DeduplicationStore // hash → memory_id mapping
}

type ExtractionResult struct {
    Facts      []Fact
    Hashes     []string // MD5 of each fact for dedup
}

func (e *ExtractionV3) Extract(ctx context.Context, userID string, input string) (*ExtractionResult, error) {
    // 1. Get context memories (top-10, NOT used for diffing)
    context, err := e.vectorStore.Search(ctx, userID, input, topK=10)
    if err != nil {
        return nil, err
    }
    
    // 2. Single LLM call - extract all facts
    prompt := buildExtractionPrompt(input, context)
    response, err := e.llmProvider.Complete(ctx, prompt)
    if err != nil {
        return nil, err
    }
    
    facts := parseFactsFromResponse(response)
    
    // 3. Hash-based dedup
    var newFacts []Fact
    for _, fact := range facts {
        hash := md5Hash(fact.Content)
        if !e.dedupStore.Exists(hash) {
            newFacts = append(newFacts, fact)
            e.dedupStore.Put(hash, fact.ID)
        }
    }
    
    // 4. Batch embed + store
    if len(newFacts) > 0 {
        embeddings, err := e.llmProvider.Embed(ctx, factsToTexts(newFacts))
        if err != nil {
            return nil, err
        }
        e.vectorStore.BatchUpsert(ctx, userID, newFacts, embeddings)
    }
    
    // 5. Entity extraction for linking (see section 4)
    entities := e.extractEntities(ctx, newFacts)
    e.vectorStore.StoreEntities(ctx, userID, entities)
    
    return &ExtractionResult{Facts: newFacts, Hashes: hashes}, nil
}

func buildExtractionPrompt(input string, context []Memory) string {
    var b strings.Builder
    b.WriteString("Extract all discrete facts from the following input.\n")
    b.WriteString("Each fact should be atomic, self-contained, and capture a single piece of information.\n\n")
    
    if len(context) > 0 {
        b.WriteString("Related existing memories (for context only - do not diff against):\n")
        for _, m := range context {
            b.WriteString(fmt.Sprintf("- %s\n", m.Content))
        }
        b.WriteString("\n")
    }
    
    b.WriteString("Input:\n")
    b.WriteString(input)
    b.WriteString("\n\nExtract facts as JSON array:")
    return b.String()
}
```

---

## 2. Multi-Signal Hybrid Retrieval

### Architecture

```
Query
    → Preprocess (lemmatize, extract entities)
    → Parallel scoring (3 signals):
        1. Semantic (vector similarity)
        2. BM25 (keyword matching)
        3. Entity (entity graph boost)
    → Score fusion → Top-K
```

### Signal Details

| Signal | Purpose | Implementation |
|--------|---------|----------------|
| **Semantic** | Core relevance via embedding similarity | Standard vector search (cosine) |
| **BM25** | Keyword/phrase matching | Sparse vectors or full-text index |
| **Entity** | Entity-based boosting | Entity store lookup (see section 4) |

**Key constraint**: BM25 and entity are *boost signals*, not recall expanders. Only semantic search provides candidates.

### Score Fusion

```go
// internal/memory/retrieval/hybrid.go

type HybridRetriever struct {
    vectorStore VectorStore
    semanticWeight    float64 // default: 0.6
    bm25Weight        float64 // default: 0.25
    entityWeight      float64 // default: 0.15
}

type RetrievalResult struct {
    MemoryID  string
    Content   string
    Score     float64 // Combined score [0, 1]
    SemanticScore float64
    BM25Score float64
    EntityScore float64
}

func (h *HybridRetriever) Search(ctx context.Context, userID, query string, topK int) ([]RetrievalResult, error) {
    // 1. Preprocess query
    entities := h.extractEntities(query)
    lemmatized := h.lemmatize(query)
    
    // 2. Semantic search (primary recall)
    semanticResults, err := h.vectorStore.SemanticSearch(ctx, userID, query, topK*2)
    if err != nil {
        return nil, err
    }
    
    // 3. BM25 keyword search
    bm25Results, err := h.vectorStore.KeywordSearch(ctx, userID, lemmatized, topK*2)
    if err != nil {
        bm25Results = make(map[string]float64)
    }
    
    // 4. Entity matching
    entityScores, err := h.vectorStore.EntitySearch(ctx, userID, entities, topK*2)
    if err != nil {
        entityScores = make(map[string]float64)
    }
    
    // 5. Fuse scores
    return h.fuseScores(semanticResults, bm25Results, entityScores, topK)
}

func (h *HybridRetriever) fuseScores(
    semantic map[string]float64,
    bm25 map[string]float64,
    entity map[string]float64,
    topK int,
) ([]RetrievalResult, error) {
    
    // Normalize each signal to [0, 1]
    semanticNorm := normalize(semantic)
    bm25Norm := normalize(bm25)
    entityNorm := normalize(entity)
    
    // Get all memory IDs
    allIDs := unionKeys(semantic, bm25, entity)
    
    var results []RetrievalResult
    for _, id := range allIDs {
        s := semanticNorm[id]
        b := bm25Norm[id]
        e := entityNorm[id]
        
        // Weighted fusion
        combined := (h.semanticWeight * s) + 
                    (h.bm25Weight * b) + 
                    (h.entityWeight * e)
        
        results = append(results, RetrievalResult{
            MemoryID:       id,
            Score:          combined,
            SemanticScore:  s,
            BM25Score:      b,
            EntityScore:    e,
        })
    }
    
    // Sort by combined score, take top-K
    sort.Slice(results, func(i, j int) bool {
        return results[i].Score > results[j].Score
    })
    
    if len(results) > topK {
        results = results[:topK]
    }
    
    return results, nil
}

func normalize(m map[string]float64) map[string]float64 {
    if len(m) == 0 {
        return m
    }
    
    var maxVal float64
    for _, v := range m {
        if v > maxVal {
            maxVal = v
        }
    }
    
    if maxVal == 0 {
        return m
    }
    
    result := make(map[string]float64, len(m))
    for k, v := range m {
        result[k] = v / maxVal
    }
    return result
}
```

### Vector Store Requirements

Mem0 requires two new capabilities from vector stores:

| Capability | Purpose | Fallback |
|------------|---------|----------|
| `keyword_search()` | BM25/full-text matching | Semantic-only |
| `search_batch()` | Batch entity matching | Sequential search |

**Qdrant**: Uses sparse vectors via `fastembed` for BM25
**Others**: Native full-text search capabilities

### Hystersis Implementation

```go
// internal/vector/store.go - Interface additions

type VectorStore interface {
    // Existing
    Search(ctx context.Context, userID, query string, topK int) ([]SearchResult, error)
    Upsert(ctx context.Context, userID string, memory Memory, embedding []float32) error
    BatchUpsert(ctx context.Context, userID string, memories []Memory, embeddings [][]float32) error
    
    // New for v3
    KeywordSearch(ctx context.Context, userID, query string, topK int) (map[string]float64, error)
    BatchSearch(ctx context.Context, userID string, queries []string, topK int) (map[string][]SearchResult, error)
}

// internal/memory/retrieval/v3.go

type RetrieverV3 struct {
    vectorStore  VectorStore
    entityStore  *EntityStore
    hybridConfig HybridConfig
}

type HybridConfig struct {
    SemanticWeight float64
    BM25Weight     float64
    EntityWeight   float64
    DefaultTopK    int
}

func NewRetrieverV3(vs VectorStore, es *EntityStore) *RetrieverV3 {
    return &RetrieverV3{
        vectorStore: vs,
        entityStore: es,
        hybridConfig: HybridConfig{
            SemanticWeight: 0.6,
            BM25Weight:     0.25,
            EntityWeight:   0.15,
            DefaultTopK:    20,
        },
    }
}
```

---

## 3. Agent-Generated Facts as First-Class

### The Problem

Previous algorithms focused on extracting facts from *user* input. Agent-generated information was often ignored or deprioritized.

**Example**: 
- User: "Book me a flight to NYC"
- Agent: "I've booked your flight for March 3rd"
- Old system: Ignores agent's confirmation
- New system: Stores "Flight booked for March 3rd" with equal weight

### Implementation

The key is including *both* user messages and agent responses in the extraction input:

```go
// internal/memory/extraction/v3.go

type Conversation struct {
    Messages []Message
}

type Message struct {
    Role    string // "user" or "assistant"
    Content string
}

// Agent-generated facts come from:
// 1. Assistant/agent responses that contain factual assertions
// 2. Tool use confirmations
// 3. Summary statements from the agent

func (e *ExtractionV3) ExtractFromConversation(ctx context.Context, userID string, conv Conversation) (*ExtractionResult, error) {
    // Combine all messages - user AND agent
    var allText strings.Builder
    for _, msg := range conv.Messages {
        rolePrefix := map[string]string{"user": "User", "assistant": "Assistant", "system": "System"}[msg.Role]
        allText.WriteString(fmt.Sprintf("%s: %s\n", rolePrefix, msg.Content))
    }
    
    return e.Extract(ctx, userID, allText.String())
}
```

### Prompt Design

```python
# Prompt should explicitly handle both user and agent content

prompt = """
Extract all discrete facts from this conversation.

Guidelines:
- Extract facts from BOTH user messages AND assistant/agent responses
- Agent statements like "I've booked X" or "I've scheduled Y" are facts
- Tool confirmations are facts
- Focus on factual information, not opinions or questions

Conversation:
{conversation_text}

Output facts as JSON array:
"""
```

### Why It Matters

**Metric improvement**: Assistant memory recall went from 46% → 100%

This enables agents to maintain memory of their own actions:
- "I told the user X"
- "I booked Y for the user"
- "I reminded the user about Z"

---

## 4. Entity Linking Without External Graph Stores

### The Innovation

Instead of Neo4j/Memgraph, entities are stored in the *same vector store* as the memories:

- Main collection: `{user_id}_memories`
- Entity collection: `{user_id}_entities`

### Extraction Flow

```
Memory added
    → Extract entities (spaCy NER + pattern matching)
        - Proper nouns (person, place, organization)
        - Quoted text
        - Compound noun phrases
    → Store in parallel entity collection
    → Same embeddings, same vector store
```

### Query Flow

```
Query "What did I tell Alice?"
    → Extract query entities: [Alice]
    → Search entity collection: which memories reference "Alice"?
    → Boost scores of those memories
    → Combine with semantic + BM25 scores
```

### Implementation

```go
// internal/memory/entity/extractor.go

type EntityExtractor struct {
    nlp *spacy.Language // spaCy model
}

type Entity struct {
    Text      string
    Type      string // "PERSON", "ORG", "GPE", "QUOTED", "COMPOUND"
    MemoryID  string
    UserID    string
}

func (e *EntityExtractor) ExtractFromMemory(ctx context.Context, memory Memory) []Entity {
    var entities []Entity
    
    doc := e.nlp(memory.Content)
    
    // Named entities (spaCy)
    for _, ent := range doc.Ents {
        entities = append(entities, Entity{
            Text:     ent.Text,
            Type:     ent.Label_,
            MemoryID: memory.ID,
            UserID:   memory.UserID,
        })
    }
    
    // Quoted text
    quoted := extractQuoted(memory.Content)
    for _, q := range quoted {
        entities = append(entities, Entity{
            Text:     q,
            Type:     "QUOTED",
            MemoryID: memory.ID,
            UserID:   memory.UserID,
        })
    }
    
    // Compound noun phrases
    compounds := extractCompounds(doc)
    entities = append(entities, compounds...)
    
    return entities
}

func extractQuoted(text string) []string {
    re := regexp.MustCompile(`"([^"]+)"`)
    matches := re.FindAllStringSubmatch(text, -1)
    var results []string
    for _, m := range matches {
        if len(m) > 1 {
            results = append(results, m[1])
        }
    }
    return results
}

func extractCompounds(doc *spacy.Doc) []Entity {
    var entities []Entity
    for i := 0; i < len(doc)-1; i++ {
        if doc[i].Pos_ == "NOUN" && doc[i+1].Pos_ == "NOUN" {
            entities = append(entities, Entity{
                Text:     doc[i].Text + " " + doc[i+1].Text,
                Type:     "COMPOUND",
                MemoryID: "",
                UserID:   "",
            })
        }
    }
    return entities
}
```

### Entity Store

```go
// internal/memory/entity/store.go

type EntityStore struct {
    vectorStore VectorStore
}

func (es *EntityStore) Store(ctx context.Context, userID string, entities []Entity) error {
    // Store in {userID}_entities collection
    collection := entityCollectionName(userID)
    
    var memories []Memory
    var embeddings [][]float32
    
    for _, entity := range entities {
        mem := Memory{
            ID:      uuid.New().String(),
            UserID:  userID,
            Content: entity.Text,
            Type:    "entity:" + entity.Type,
        }
        memories = append(memories, mem)
    }
    
    if len(memories) > 0 {
        emb, err := es.embed(ctx, entityTexts(entities))
        if err != nil {
            return err
        }
        embeddings = emb
        es.vectorStore.BatchUpsert(ctx, collection, memories, embeddings)
    }
    
    return nil
}

func (es *EntityStore) Search(ctx context.Context, userID string, queryEntities []Entity, topK int) (map[string]float64, error) {
    collection := entityCollectionName(userID)
    
    // Search entity collection for each query entity
    entityTexts := make([]string, len(queryEntities))
    for i, e := range queryEntities {
        entityTexts[i] = e.Text
    }
    
    results, err := es.vectorStore.SearchBatch(ctx, collection, entityTexts, topK)
    if err != nil {
        return nil, err
    }
    
    // Map memory_id → entity boost score
    scores := make(map[string]float64)
    for _, result := range results {
        scores[result.MemoryID] = result.Score
    }
    
    return scores, nil
}

func entityCollectionName(userID string) string {
    return fmt.Sprintf("%s_entities", userID)
}
```

### Graceful Degradation

| Missing | Impact |
|---------|--------|
| spaCy | No entity extraction, no BM25 lemmatization |
| fastembed (Qdrant) | No BM25 keyword search |
| Entity store | No entity boosting |

Search always works with semantic-only; hybrid features layer on top.

---

## 5. Comparison: Mem0 v3 vs Hystersis Current

| Feature | Mem0 v3 | Hystersis (Current) | Priority |
|---------|---------|---------------------|----------|
| Extraction | Single-pass ADD-only | Two-pass (extract + merge) | High |
| UPDATE/DELETE | Removed | Still present | High |
| Retrieval | Hybrid (semantic + BM25 + entity) | Semantic-only | High |
| Entity linking | Built-in, same vector store | Neo4j graph | Medium |
| Agent facts | First-class | Not explicitly handled | High |
| BM25 | Sparse vectors / full-text | None | High |

---

## 6. Implementation Roadmap for Hystersis

### Phase 1: Single-Pass Extraction (Week 1-2)

1. Replace two-pass extraction with single LLM call
2. Remove UPDATE/DELETE logic from extraction
3. Implement hash-based deduplication
4. Add memory accumulation model

### Phase 2: Multi-Signal Retrieval (Week 2-3)

1. Add BM25/keyword search capability to vector store interface
2. Implement entity extraction with spaCy
3. Add parallel scoring with fusion
4. Add entity collection to vector store

### Phase 3: Agent Facts + Polish (Week 3-4)

1. Handle both user and agent messages in extraction
2. Tune weights for score fusion
3. Benchmark against LoCoMo/LongMemEval
4. Update API endpoints

---

## 7. Key Design Decisions

### Why ADD-only works better

1. **No information loss**: UPDATE/DELETE can lose context
2. **Simpler logic**: One LLM call instead of two
3. **Better latency**: ~50% reduction
4. **Retrieval handles conflicts**: Most relevant surfaces

### Why embed entities in vector store

1. **No external dependency**: Remove Neo4j requirement
2. **Same infrastructure**: Leverage existing vector DB
3. **Unified scoring**: Entity boost is just another signal
4. **Simpler operations**: One store to manage

### Weight tuning recommendations

Based on Mem0 defaults:
- Semantic: 60%
- BM25: 25%
- Entity: 15%

Tune based on your use case:
- More keyword-heavy → increase BM25
- More entity-centric → increase entity
- General purpose → keep defaults

---

## 8. Dependencies

### Required for full feature set

```bash
# Python (if using spaCy)
pip install spacy
python -m spacy download en_core_web_sm

# For BM25 (Qdrant users)
pip install fastembed
```

### Go equivalents

```go
// Entity extraction - options:
// 1. Use spaCy via exec (call python subprocess)
// 2. Use pure Go NER (prose, go-nlp)
// 3. Use LLM for entity extraction

// BM25 - options:
// 1. Use embedding model's sparse capabilities
// 2. Use pure Go BM25 (github.com/blevesearch/bleve)
// 3. Delegate to vector store's full-text search

// For Qdrant: sparse vectors via fastembed
```

---

## Summary

Mem0 v3's innovations are:

1. **Single-pass ADD-only**: One LLM call, memories accumulate, retrieval handles conflicts
2. **Hybrid retrieval**: Three signals (semantic, BM25, entity) fused into one score
3. **Agent facts**: Both user and agent content processed equally
4. **Entity linking**: No external graph store—entities in same vector DB

These are practical to implement and deliver substantial accuracy improvements (+20-26 points on benchmarks).