# Memory Space Orchestrator

## Overview

The Memory Space Orchestrator coordinates all memory operations across the Hystersis system. It manages memory creation, retrieval, compression, and tiering, while coordinating with graph and vector databases.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                MEMORY SPACE ORCHESTRATOR                     │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌──────────────────┐  ┌──────────────────┐  ┌─────────────────┐ │
│  │ Memory Processor │  │ Search Coordinator│  │ Tier Manager     │ │
│  │ - Extract facts │  │ - Vector search  │  │ - Memory routing │ │
│  │ - Compress      │  │ - Graph traversal│  │ - Tier policy   │ │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘ │
│         │                     │                     │          │
│         ▼                     ▼                     ▼          │
│  ┌──────────────────┐  ┌──────────────────┐  ┌─────────────────┐ │
│  │ Graph Store     │  │ Vector Store     │  │ Compression    │ │
│  │ (Neo4j)        │  │ (Qdrant)         │  │ Engine          │ │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘ │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

## Sub-Agents

### 1. Memory Processor

```go
package memory

// MemoryProcessor handles memory extraction and compression

// ExtractFacts extracts facts, entities, and relationships from memory content
func (p *MemoryProcessor) ExtractFacts(ctx context.Context, content string) (*MemoryFacts, error) {
    // 1. LLM Processing: Extract facts using LLM
    facts, err := p.llmProvider.Extract(ctx, content)
    if err != nil {
        return nil, fmt.Errorf("fact extraction failed: %w", err)
    }
    
    // 2. Entity Extraction: Identify and extract entities
    entities, err := p.extractEntities(facts)
    if err != nil {
        return nil, fmt.Errorf("entity extraction failed: %w", err)
    }
    
    // 3. Relationship Analysis: Determine relationships between entities
    relationships, err := p.analyzeRelationships(entities)
    if err != nil {
        return nil, fmt.Errorf("relationship analysis failed: %w", err)
    }
    
    // 4. Conflict Resolution: Check for and resolve conflicts
    if err := p.resolveConflicts(relationships); err != nil {
        return nil, fmt.Errorf("conflict resolution failed: %w", err)
    }
    
    // 5. Compression: Apply ProMem compression
    compressed, err := p.compressionEngine.Compress(ctx, facts)
    if err != nil {
        return nil, fmt.Errorf("compression failed: %w", err)
    }
    
    return &MemoryFacts{
        Facts:        facts,
        Entities:     entities,
        Relationships: relationships,
        Compressed:   compressed,
    }, nil
}

// analyzeRelationships determines relationships between entities
func (p *MemoryProcessor) analyzeRelationships(entities []Entity) ([]Relationship, error) {
    // Implementation of relationship analysis
    // ...
}

// resolveConflicts checks for and resolves conflicts in relationships
func (p *MemoryProcessor) resolveConflicts(relationships []Relationship) error {
    // Implementation of conflict resolution
    // ...
}
```

### 2. Search Coordinator

```go
package memory

// SearchCoordinator handles memory search operations

// Search performs hybrid vector + graph search
func (s *SearchCoordinator) Search(ctx context.Context, query string, options SearchOptions) ([]*Memory, error) {
    // 1. Vector Search: Initial search using vector similarity
    vectorResults, err := s.vectorStore.Search(ctx, query, options.VectorOptions)
    if err != nil {
        return nil, fmt.Errorf("vector search failed: %w", err)
    }
    
    // 2. Graph Traversal: Spread activation through graph
    graphResults, err := s.graphStore.Traverse(ctx, query, vectorResults, options.GraphOptions)
    if err != nil {
        return nil, fmt.Errorf("graph traversal failed: %w", err)
    }
    
    // 3. Hybrid Ranking: Combine and rank results
    return s.rankResults(vectorResults, graphResults, options.RankingOptions), nil
}

// rankResults combines and ranks vector and graph search results
func (s *SearchCoordinator) rankResults(
    vectorResults []*Memory,
    graphResults []*Memory,
    options RankingOptions,
) []*Memory {
    // Implementation of hybrid ranking
    // ...
}
```

### 3. Tier Manager

```go
package memory

// TierManager handles memory tiering and routing

// RouteMemory routes memory to appropriate tier based on policy
func (t *TierManager) RouteMemory(ctx context.Context, memory *Memory) (MemoryTier, error) {
    // 1. Check memory against tier policies
    if t.isWorkingMemory(memory) {
        return TierWorking, nil
    }
    
    if t.isHotMemory(memory) {
        return TierHot, nil
    }
    
    if t.isColdMemory(memory) {
        return TierCold, nil
    }
    
    return TierArchive, nil
}

// isWorkingMemory checks if memory belongs in working tier
func (t *TierManager) isWorkingMemory(memory *Memory) bool {
    // Implementation of working memory check
    // ...
}

// isHotMemory checks if memory belongs in hot tier
func (t *TierManager) isHotMemory(memory *Memory) bool {
    // Implementation of hot memory check
    // ...
}

// isColdMemory checks if memory belongs in cold tier
func (t *TierManager) isColdMemory(memory *Memory) bool {
    // Implementation of cold memory check
    // ...
}
```

## Integration with Other Orchestrators

### Memory ↔ Skills

```go
// SkillDiscovery extracts skills from memory content
func (s *SkillDiscovery) ExtractSkills(ctx context.Context, content string) ([]*Skill, error) {
    // 1. LLM Processing: Extract potential skills
    skills, err := s.llmProvider.ExtractSkills(ctx, content)
    if err != nil {
        return nil, fmt.Errorf("skill extraction failed: %w", err)
    }
    
    // 2. Validation: Verify extracted skills
    verified, err := s.verifySkills(skills)
    if err != nil {
        return nil, fmt.Errorf("skill verification failed: %w", err)
    }
    
    return verified, nil
}

// verifySkills validates extracted skills
func (s *SkillDiscovery) verifySkills(skills []*Skill) ([]*Skill, error) {
    // Implementation of skill verification
    // ...
}
```

### Memory ↔ Compression

```go
// CompressionCoordinator manages memory compression

// CompressMemory compresses memory content
func (c *CompressionCoordinator) CompressMemory(ctx context.Context, memory *Memory) (*CompressedMemory, error) {
    // 1. Check complexity threshold
    if !c.isComplex(memory) {
        return c.fastCompression(ctx, memory)
    }
    
    // 2. Complex compression with verification
    return c.complexCompression(ctx, memory)
}

// isComplex checks if memory exceeds complexity threshold
func (c *CompressionCoordinator) isComplex(memory *Memory) bool {
    // Implementation of complexity check
    // ...
}

// fastCompression performs fast path compression
func (c *CompressionCoordinator) fastCompression(ctx context.Context, memory *Memory) (*CompressedMemory, error) {
    // Implementation of fast compression
    // ...
}

// complexCompression performs complex compression with verification
func (c *CompressionCoordinator) complexCompression(ctx context.Context, memory *Memory) (*CompressedMemory, error) {
    // Implementation of complex compression
    // ...
}
```

## Error Handling

```go
// safeHTTPError wraps errors for API responses
func safeHTTPError(w http.ResponseWriter, status int, err error) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    
    var apiError *APIError
    if errors.As(err, &apiError) {
        json.NewEncoder(w).Encode(apiError)
        return
    }
    
    json.NewEncoder(w).Encode(&APIError{
        Error:   err.Error(),
        Message: http.StatusText(status),
    })
}

// APIError represents a standardized API error response
type APIError struct {
    Error   string `json:"error"`
    Message string `json:"message"`
}
```

## Performance Optimization

```go
// WithCache creates a new MemoryProcessor with caching enabled
func WithCache(processor *MemoryProcessor, cache *Cache) *MemoryProcessor {
    processor.cache = cache
    return processor
}

// WithRateLimiter creates a new MemoryProcessor with rate limiting
func WithRateLimiter(processor *MemoryProcessor, limiter *RateLimiter) *MemoryProcessor {
    processor.rateLimiter = limiter
    return processor
}

// WithTimeout creates a new MemoryProcessor with timeout settings
func WithTimeout(processor *MemoryProcessor, timeout time.Duration) *MemoryProcessor {
    processor.timeout = timeout
    return processor
}
```

## Testing

```go
func TestMemoryProcessor_ExtractFacts(t *testing.T) {
    // Setup test data
    testContent := "This is a test memory content with some facts and entities."
    expectedFacts := []Fact{{
        Text: "This is a test memory content",
        Type: "statement",
    }}
    
    // Create test processor with mock LLM provider
    mockLLM := &MockLLMProvider{}
    mockLLM.On("Extract", testContent).Return(expectedFacts, nil)
    
    processor := &MemoryProcessor{llmProvider: mockLLM}
    
    // Execute test
    facts, err := processor.ExtractFacts(context.Background(), testContent)
    
    // Verify results
    if err != nil {
        t.Errorf("ExtractFacts returned error: %v", err)
    }
    
    if len(facts.Facts) != 1 {
        t.Errorf("Expected 1 fact, got %d", len(facts.Facts))
    }
    
    // Verify mock calls
    mockLLM.AssertExpectations(t)
}

func TestSearchCoordinator_Search(t *testing.T) {
    // Setup test data
    testQuery := "test query"
    expectedResults := []*Memory{{
        ID: "memory-1",
        Content: "Test memory content",
    }}
    
    // Create test coordinator with mock stores
    mockVector := &MockVectorStore{}
    mockVector.On("Search", testQuery).Return(expectedResults, nil)
    
    mockGraph := &MockGraphStore{}
    mockGraph.On("Traverse", testQuery, expectedResults).Return(expectedResults, nil)
    
    coordinator := &SearchCoordinator{
        vectorStore: mockVector,
        graphStore:  mockGraph,
    }
    
    // Execute test
    results, err := coordinator.Search(context.Background(), testQuery, SearchOptions{})
    
    // Verify results
    if err != nil {
        t.Errorf("Search returned error: %v", err)
    }
    
    if len(results) != 1 {
        t.Errorf("Expected 1 result, got %d", len(results))
    }
    
    // Verify mock calls
    mockVector.AssertExpectations(t)
    mockGraph.AssertExpectations(t)
}

func TestTierManager_RouteMemory(t *testing.T) {
    // Setup test data
    testMemory := &Memory{
        ID: "memory-1",
        Content: "Test memory content",
        AccessCount: 100,
    }
    
    // Create test manager with mock policies
    mockTierPolicy := &MockTierPolicy{}
    mockTierPolicy.On("IsWorkingMemory", testMemory).Return(false)
    mockTierPolicy.On("IsHotMemory", testMemory).Return(true)
    
    manager := &TierManager{
        tierPolicy: mockTierPolicy,
    }
    
    // Execute test
    tier, err := manager.RouteMemory(context.Background(), testMemory)
    
    // Verify results
    if err != nil {
        t.Errorf("RouteMemory returned error: %v", err)
    }
    
    if tier != TierHot {
        t.Errorf("Expected TierHot, got %v", tier)
    }
    
    // Verify mock calls
    mockTierPolicy.AssertExpectations(t)
}
```

---

*This documentation provides a comprehensive overview of the Memory Space Orchestrator and its sub-agents, including their integration with other system components.*