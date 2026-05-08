# Compression Engine Orchestrator

## Overview

The Compression Engine Orchestrator coordinates the proprietary compression operations within the Hystersis system. It manages ProMem extraction, spreading activation retrieval, and the async compression pipeline, while optimizing performance and maintaining data integrity.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                  COMPRESSION ENGINE ORCHESTRATOR                │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌──────────────────┐  ┌──────────────────┐  ┌─────────────────┐ │
│  │ ProMem Extractor│  │ Spreading Activation│  │ Async Pipeline  │ │
│  │ - Fact extraction│  │ - Graph propagation│  │ - Non-blocking  │ │
│  │ - Self-questioning│ │ - Multi-hop reasoning│  │ - Worker pool   │ │
│  │ - Verification   │  │ - Hybrid ranking    │  │ - Async queue   │ │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘ │
│         │                     │                     │          │
│         ▼                     ▼                     ▼          │
│  ┌──────────────────┐  ┌──────────────────┐  ┌─────────────────┐ │
│  │ LLM Router     │  │ Performance      │  │ Hyperparameter  │ │
│  │ - Hybrid routing│  │ Optimizer        │  │ Tuner           │ │
│  │ - Complexity    │  │ - Resource        │  │ - Decay factor  │ │
│  │ assessment     │  │ management        │  │ - Threshold     │ │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘ │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

## Sub-Agents

### 1. ProMem Extractor

```go
package compression

// MemoryExtractor implements ProMem-style fact extraction

// Extract performs ProMem-style extraction with self-verification
func (e *MemoryExtractor) Extract(ctx context.Context, memory string) (*ExtractionResult, error) {
    // 1. Self-Question Generation: Ask "what does this memory mean?"
    questions := e.generateQuestions(memory)
    
    // 2. Answer in context
    answers := e.answerQuestions(ctx, questions, memory)
    
    // 3. Verification (using verifyProvider - Claude)
    verified := e.verifyWithProvider(ctx, answers, memory)
    
    // 4. Gap Detection
    gaps := e.detectGaps(verified, memory)
    supplements := e.extractGaps(ctx, gaps)
    
    return merge(verified, supplements), nil
}

// generateQuestions generates self-questioning prompts
func (e *MemoryExtractor) generateQuestions(memory string) []string {
    // Implementation of question generation
    // ...
}

// answerQuestions answers self-questioning prompts
func (e *MemoryExtractor) answerQuestions(ctx context.Context, questions []string, memory string) []string {
    // Implementation of question answering
    // ...
}

// verifyWithProvider verifies answers with LLM provider
func (e *MemoryExtractor) verifyWithProvider(ctx context.Context, answers []string, memory string) []string {
    // Implementation of verification
    // ...
}

// detectGaps detects gaps in verified facts
func (e *MemoryExtractor) detectGaps(verified []string, memory string) []string {
    // Implementation of gap detection
    // ...
}

// extractGaps extracts information to fill gaps
func (e *MemoryExtractor) extractGaps(ctx context.Context, gaps []string) []string {
    // Implementation of gap extraction
    // ...
}

// merge merges verified facts with supplements
func merge(verified []string, supplements []string) []string {
    // Implementation of merging
    // ...
}
```

### 2. Spreading Activation

```go
package compression

// SpreadingActivation implements graph-based retrieval beyond vector similarity

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

// initializeActivation initializes activation map with initial nodes
func (s *SpreadingActivation) initializeActivation(initialNodes []*types.Memory) map[string]float64 {
    // Implementation of activation initialization
    // ...
}

// propagate propagates activation through graph
func (s *SpreadingActivation) propagate(
    ctx context.Context,
    activationMap map[string]float64,
    decayFactor float64,
) map[string]float64 {
    // Implementation of activation propagation
    // ...
}

// rankByActivation ranks nodes by activation level
func (s *SpreadingActivation) rankByActivation(ctx context.Context, activationMap map[string]float64) []*types.Memory {
    // Implementation of ranking by activation
    // ...
}
```

### 3. Async Pipeline

```go
package compression

// CompressionPipeline handles non-blocking compression jobs

// CompressAsync processes compression non-blocking
func (p *CompressionPipeline) CompressAsync(job CompressionJob) {
    p.jobQueue <- job  // Returns immediately, processes in background
}

// worker processes compression jobs from the queue
func (p *CompressionPipeline) worker(ctx context.Context, workerID int) {
    for {
        select {
        case job := <-p.jobQueue:
            // Process job
            result, err := p.processJob(ctx, job)
            
            // Send result back to caller
            if job.Done != nil {
                job.Done <- result
            }
        case <-ctx.Done():
            return
        }
    }
}

// processJob processes a single compression job
func (p *CompressionPipeline) processJob(ctx context.Context, job CompressionJob) (Result, error) {
    // 1. Extract facts
    facts, err := p.extractor.Extract(ctx, job.MemoryContent)
    if err != nil {
        return Result{}, fmt.Errorf("fact extraction failed: %w", err)
    }
    
    // 2. Compress facts
    compressed, err := p.compressor.Compress(ctx, facts)
    if err != nil {
        return Result{}, fmt.Errorf("compression failed: %w", err)
    }
    
    // 3. Validate compression
    if err := p.validator.Validate(ctx, compressed); err != nil {
        return Result{}, fmt.Errorf("compression validation failed: %w", err)
    }
    
    // 4. Calculate token reduction
    tokenReduction := p.calculateTokenReduction(job.MemoryContent, compressed)
    
    return Result{
        Compressed:     compressed,
        TokenReduction: tokenReduction,
    }, nil
}

// calculateTokenReduction calculates token reduction percentage
func (p *CompressionPipeline) calculateTokenReduction(original, compressed string) float64 {
    // Implementation of token reduction calculation
    // ...
}
```

## Integration with Other Orchestrators

### Compression ↔ Memory

```go
// MemoryCompressionCoordinator coordinates compression of memory content

// CompressMemory compresses memory content
func (m *MemoryCompressionCoordinator) CompressMemory(ctx context.Context, memory *Memory) (*CompressedMemory, error) {
    // 1. Check complexity threshold
    if !m.isComplex(memory) {
        return m.fastCompression(ctx, memory)
    }
    
    // 2. Complex compression with verification
    return m.complexCompression(ctx, memory)
}

// isComplex checks if memory exceeds complexity threshold
func (m *MemoryCompressionCoordinator) isComplex(memory *Memory) bool {
    // Implementation of complexity check
    // ...
}

// fastCompression performs fast path compression
func (m *MemoryCompressionCoordinator) fastCompression(ctx context.Context, memory *Memory) (*CompressedMemory, error) {
    // 1. Use fast LLM provider for extraction
    extractor := &MemoryExtractor{llmProvider: m.fastProvider}
    facts, err := extractor.Extract(ctx, memory.Content)
    if err != nil {
        return nil, fmt.Errorf("fast extraction failed: %w", err)
    }
    
    // 2. Compress facts
    compressed, err := m.compressionEngine.Compress(ctx, facts)
    if err != nil {
        return nil, fmt.Errorf("fast compression failed: %w", err)
    }
    
    return &CompressedMemory{
        MemoryID:      memory.ID,
        Content:      compressed,
        TokenReduction: m.calculateTokenReduction(memory.Content, compressed),
    }, nil
}

// complexCompression performs complex compression with verification
func (m *MemoryCompressionCoordinator) complexCompression(ctx context.Context, memory *Memory) (*CompressedMemory, error) {
    // 1. Use fast LLM provider for initial extraction
    extractor := &MemoryExtractor{llmProvider: m.fastProvider}
    facts, err := extractor.Extract(ctx, memory.Content)
    if err != nil {
        return nil, fmt.Errorf("initial extraction failed: %w", err)
    }
    
    // 2. Use verify LLM provider for verification
    verifier := &MemoryExtractor{llmProvider: m.verifyProvider}
    verified, err := verifier.Extract(ctx, memory.Content)
    if err != nil {
        return nil, fmt.Errorf("verification failed: %w", err)
    }
    
    // 3. Merge verified facts with initial extraction
    merged := merge(facts, verified)
    
    // 4. Compress merged facts
    compressed, err := m.compressionEngine.Compress(ctx, merged)
    if err != nil {
        return nil, fmt.Errorf("complex compression failed: %w", err)
    }
    
    return &CompressedMemory{
        MemoryID:      memory.ID,
        Content:      compressed,
        TokenReduction: m.calculateTokenReduction(memory.Content, compressed),
    }, nil
}
```

### Compression ↔ Skills

```go
// SkillCompressionExtractor extracts skills from compressed content

// ExtractSkillsFromCompressed extracts skills from compressed content
func (s *SkillCompressionExtractor) ExtractSkillsFromCompressed(
    ctx context.Context,
    compressed *CompressedMemory,
) ([]*Skill, error) {
    // 1. Decompress content
    content, err := s.compressionEngine.Decompress(ctx, compressed.Content)
    if err != nil {
        return nil, fmt.Errorf("decompression failed: %w", err)
    }
    
    // 2. Extract skills from decompressed content
    skills, err := s.skillDiscovery.ExtractSkills(ctx, content)
    if err != nil {
        return nil, fmt.Errorf("skill extraction from compressed content failed: %w", err)
    }
    
    // 3. Associate skills with compressed memory
    for _, skill := range skills {
        skill.SourceMemory = compressed.MemoryID
        skill.Compressed = true
    }
    
    return skills, nil
}

// CompressedSkillExecutor executes skills with compressed content context
func (s *CompressedSkillExecutor) ExecuteSkillsWithCompressed(
    ctx context.Context,
    skillIDs []string,
    compressed *CompressedMemory,
) (map[string]interface{}, error) {
    // 1. Decompress content
    content, err := s.compressionEngine.Decompress(ctx, compressed.Content)
    if err != nil {
        return nil, fmt.Errorf("decompression failed: %w", err)
    }
    
    // 2. Create context with compressed information
    context := map[string]interface{}{
        "memory":        compressed,
        "memory_content": content,
        "compressed":    true,
    }
    
    // 3. Execute each skill
    for _, skillID := range skillIDs {
        result, err := s.skillExecutor.ExecuteSkill(ctx, skillID, context)
        if err != nil {
            return nil, fmt.Errorf("skill execution failed: %w", err)
        }
    }
    
    return context, nil
}
```

## Performance Optimization

```go
// PerformanceOptimizer optimizes compression performance

// OptimizeHyperparameters optimizes compression hyperparameters
func (p *PerformanceOptimizer) OptimizeHyperparameters(ctx context.Context) error {
    // 1. Analyze current performance
    currentMetrics, err := p.getCurrentMetrics(ctx)
    if err != nil {
        return fmt.Errorf("failed to get current metrics: %w", err)
    }
    
    // 2. Adjust hyperparameters based on analysis
    if err := p.adjustHyperparameters(ctx, currentMetrics); err != nil {
        return fmt.Errorf("failed to adjust hyperparameters: %w", err)
    }
    
    return nil
}

// getCurrentMetrics gets current performance metrics
func (p *PerformanceOptimizer) getCurrentMetrics(ctx context.Context) (*PerformanceMetrics, error) {
    // Implementation of metric collection
    // ...
}

// adjustHyperparameters adjusts compression hyperparameters
func (p *PerformanceOptimizer) adjustHyperparameters(ctx context.Context, metrics *PerformanceMetrics) error {
    // Implementation of hyperparameter adjustment
    // ...
}

// ResourceManager manages system resources for compression

// MonitorResources monitors system resources
func (r *ResourceManager) MonitorResources(ctx context.Context) error {
    // 1. Get current resource usage
    usage, err := r.getResourceUsage(ctx)
    if err != nil {
        return fmt.Errorf("failed to get resource usage: %w", err)
    }
    
    // 2. Adjust resource allocation based on usage
    if err := r.adjustResourceAllocation(ctx, usage); err != nil {
        return fmt.Errorf("failed to adjust resource allocation: %w", err)
    }
    
    return nil
}

// getResourceUsage gets current resource usage
func (r *ResourceManager) getResourceUsage(ctx context.Context) (*ResourceUsage, error) {
    // Implementation of resource usage collection
    // ...
}

// adjustResourceAllocation adjusts resource allocation
func (r *ResourceManager) adjustResourceAllocation(ctx context.Context, usage *ResourceUsage) error {
    // Implementation of resource allocation adjustment
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

// CompressionError represents a compression-specific error
type CompressionError struct {
    Error   string `json:"error"`
    Details string `json:"details"`
}

// IsCompressionError checks if an error is a compression error
func IsCompressionError(err error) bool {
    _, ok := err.(*CompressionError)
    return ok
}

// NewCompressionError creates a new compression error
func NewCompressionError(format string, args ...interface{}) error {
    return &CompressionError{
        Error:   fmt.Sprintf(format, args...),
        Details: "Compression operation failed",
    }
}
```

## Testing

```go
func TestMemoryExtractor_Extract(t *testing.T) {
    // Setup test data
    testMemory := "This is a test memory content with some facts and entities."
    expectedFacts := []Fact{{
        Text: "This is a test memory content",
        Type: "statement",
    }}
    
    // Create test extractor with mock LLM provider
    mockLLM := &MockLLMProvider{}
    mockLLM.On("Extract", testMemory).Return(expectedFacts, nil)
    
    extractor := &MemoryExtractor{llmProvider: mockLLM}
    
    // Execute test
    result, err := extractor.Extract(context.Background(), testMemory)
    
    // Verify results
    if err != nil {
        t.Errorf("Extract returned error: %v", err)
    }
    
    if len(result.Facts) != 1 {
        t.Errorf("Expected 1 fact, got %d", len(result.Facts))
    }
    
    // Verify mock calls
    mockLLM.AssertExpectations(t)
}

func TestSpreadingActivation_Retrieve(t *testing.T) {
    // Setup test data
    testQuery := "test query"
    expectedResults := []*types.Memory{{
        ID: "memory-1",
        Content: "Test memory content",
    }}
    
    // Create test spreading activation with mock stores
    mockVector := &MockVectorStore{}
    mockVector.On("Search", testQuery, 50).Return(expectedResults, nil)
    
    mockGraph := &MockGraphStore{}
    mockGraph.On("InitializeActivation", expectedResults).Return(map[string]float64{"memory-1": 1.0})
    mockGraph.On("Propagate", mock.Anything, map[string]float64{"memory-1": 1.0}, 0.85).Return(map[string]float64{"memory-1": 0.85})
    mockGraph.On("RankByActivation", mock.Anything, map[string]float64{"memory-1": 0.85}).Return(expectedResults)
    
    spreading := &SpreadingActivation{
        vectorStore:  mockVector,
        graphStore:   mockGraph,
        maxHops:      1,
        decayFactor:  0.85,
    }
    
    // Execute test
    results, err := spreading.Retrieve(context.Background(), testQuery)
    
    // Verify results
    if err != nil {
        t.Errorf("Retrieve returned error: %v", err)
    }
    
    if len(results) != 1 {
        t.Errorf("Expected 1 result, got %d", len(results))
    }
    
    // Verify mock calls
    mockVector.AssertExpectations(t)
    mockGraph.AssertExpectations(t)
}

func TestCompressionPipeline_CompressAsync(t *testing.T) {
    // Setup test data
    testJob := CompressionJob{
        MemoryID:     "memory-1",
        MemoryContent: "Test memory content",
        Priority:     0,
        Done:         make(chan Result),
    }
    expectedResult := Result{
        Compressed:     "compressed content",
        TokenReduction: 0.5,
    }
    
    // Create test pipeline with mock components
    mockExtractor := &MockMemoryExtractor{}
    mockExtractor.On("Extract", testJob.MemoryContent).Return(&ExtractionResult{Facts: []Fact{{
        Text: "Test memory content",
        Type: "statement",
    }}}, nil)
    
    mockCompressor := &MockCompressor{}
    mockCompressor.On("Compress", mock.Anything, mock.Anything).Return("compressed content", nil)
    
    mockValidator := &MockValidator{}
    mockValidator.On("Validate", mock.Anything, "compressed content").Return(nil)
    
    pipeline := &CompressionPipeline{
        jobQueue:     make(chan CompressionJob),
        extractor:    mockExtractor,
        compressor:  mockCompressor,
        validator:    mockValidator,
    }
    
    // Start worker in background
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    go pipeline.worker(ctx, 1)
    
    // Execute test
    pipeline.CompressAsync(testJob)
    
    // Get result
    result := <-testJob.Done
    
    // Verify results
    if result.Compressed != expectedResult.Compressed {
        t.Errorf("Expected compressed content %q, got %q", expectedResult.Compressed, result.Compressed)
    }
    
    if result.TokenReduction != expectedResult.TokenReduction {
        t.Errorf("Expected token reduction %.2f, got %.2f", expectedResult.TokenReduction, result.TokenReduction)
    }
    
    // Verify mock calls
    mockExtractor.AssertExpectations(t)
    mockCompressor.AssertExpectations(t)
    mockValidator.AssertExpectations(t)
}

func TestMemoryCompressionCoordinator_CompressMemory(t *testing.T) {
    // Setup test data
    testMemory := &Memory{
        ID:       "memory-1",
        Content:  "Test memory content",
    }
    expectedComplexResult := &CompressedMemory{
        MemoryID:      testMemory.ID,
        Content:      "compressed content",
        TokenReduction: 0.5,
    }
    
    // Create test coordinator with mock components
    mockFastProvider := &MockLLMProvider{}
    mockFastProvider.On("Extract", testMemory.Content).Return(&ExtractionResult{Facts: []Fact{{
        Text: "Test memory content",
        Type: "statement",
    }}}, nil)
    
    mockVerifyProvider := &MockLLMProvider{}
    mockVerifyProvider.On("Extract", testMemory.Content).Return(&ExtractionResult{Facts: []Fact{{
        Text: "Test memory content",
        Type: "statement",
    }}}, nil)
    
    mockCompressionEngine := &MockCompressionEngine{}
    mockCompressionEngine.On("Compress", mock.Anything, mock.Anything).Return("compressed content", nil)
    
    coordinator := &MemoryCompressionCoordinator{
        fastProvider:        mockFastProvider,
        verifyProvider:     mockVerifyProvider,
        compressionEngine: mockCompressionEngine,
        complexityThreshold: 0.5,
    }
    
    // Execute test
    result, err := coordinator.CompressMemory(context.Background(), testMemory)
    
    // Verify results
    if err != nil {
        t.Errorf("CompressMemory returned error: %v", err)
    }
    
    if result == nil {
        t.Error("Expected result, got nil")
    }
    
    if result.Content != expectedComplexResult.Content {
        t.Errorf("Expected compressed content %q, got %q", expectedComplexResult.Content, result.Content)
    }
    
    // Verify mock calls
    mockFastProvider.AssertExpectations(t)
    mockVerifyProvider.AssertExpectations(t)
    mockCompressionEngine.AssertExpectations(t)
}

func TestSkillCompressionExtractor_ExtractSkillsFromCompressed(t *testing.T) {
    // Setup test data
    testCompressed := &CompressedMemory{
        MemoryID:      "memory-1",
        Content:      "compressed content",
        TokenReduction: 0.5,
    }
    expectedSkills := []*Skill{{
        Name:        "test-skill",
        Description: "Test skill description",
        Trigger:     "test-trigger",
    }}
    
    // Create test extractor with mock components
    mockCompressionEngine := &MockCompressionEngine{}
    mockCompressionEngine.On("Decompress", testCompressed.Content).Return("decompressed content", nil)
    
    mockSkillDiscovery := &MockSkillDiscovery{}
    mockSkillDiscovery.On("ExtractSkills", "decompressed content").Return(expectedSkills, nil)
    
    extractor := &SkillCompressionExtractor{
        compressionEngine: mockCompressionEngine,
        skillDiscovery:   mockSkillDiscovery,
    }
    
    // Execute test
    skills, err := extractor.ExtractSkillsFromCompressed(context.Background(), testCompressed)
    
    // Verify results
    if err != nil {
        t.Errorf("ExtractSkillsFromCompressed returned error: %v", err)
    }
    
    if len(skills) != 1 {
        t.Errorf("Expected 1 skill, got %d", len(skills))
    }
    
    // Verify mock calls
    mockCompressionEngine.AssertExpectations(t)
    mockSkillDiscovery.AssertExpectations(t)
}

func TestPerformanceOptimizer_OptimizeHyperparameters(t *testing.T) {
    // Setup test data
    testMetrics := &PerformanceMetrics{
        Accuracy:      0.95,
        TokenReduction: 0.8,
        Latency:      100,
    }
    
    // Create test optimizer with mock metrics collector
    mockMetricsCollector := &MockMetricsCollector{}
    mockMetricsCollector.On("GetCurrentMetrics", mock.Anything).Return(testMetrics, nil)
    
    optimizer := &PerformanceOptimizer{
        metricsCollector: mockMetricsCollector,
    }
    
    // Execute test
    err := optimizer.OptimizeHyperparameters(context.Background())
    
    // Verify results
    if err != nil {
        t.Errorf("OptimizeHyperparameters returned error: %v", err)
    }
    
    // Verify mock calls
    mockMetricsCollector.AssertExpectations(t)
}

func TestResourceManager_MonitorResources(t *testing.T) {
    // Setup test data
    testUsage := &ResourceUsage{
        CPU:      50,
        Memory:   50,
        Disk:     50,
    }
    
    // Create test manager with mock resource collector
    mockResourceCollector := &MockResourceCollector{}
    mockResourceCollector.On("GetResourceUsage", mock.Anything).Return(testUsage, nil)
    
    manager := &ResourceManager{
        resourceCollector: mockResourceCollector,
    }
    
    // Execute test
    err := manager.MonitorResources(context.Background())
    
    // Verify results
    if err != nil {
        t.Errorf("MonitorResources returned error: %v", err)
    }
    
    // Verify mock calls
    mockResourceCollector.AssertExpectations(t)
}
```

---

*This documentation provides a comprehensive overview of the Compression Engine Orchestrator and its sub-agents, including their integration with other system components.*