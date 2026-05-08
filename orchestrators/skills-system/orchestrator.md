# Skills System Orchestrator

## Overview

The Skills System Orchestrator manages the discovery, execution, and lifecycle of skills within the Hystersis system. It coordinates both file-based and Neo4j-backed skills, handling skill chains and human review workflows.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                  SKILLS SYSTEM ORCHESTRATOR                    │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌──────────────────┐  ┌──────────────────┐  ┌─────────────────┐ │
│  │ Skill Discovery │  │ Skill Executor  │  │ Chain Coordinator│ │
│  │ - Extract skills│  │ - Execute skills│  │ - Manage chains │ │
│  │ - Suggest skills│  │ - Context aware  │  │ - Multi-step    │ │
│  │ - Verify skills │  │ - Performance    │  │ - Workflows     │ │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘ │
│         │                     │                     │          │
│         ▼                     ▼                     ▼          │
│  ┌──────────────────┐  ┌──────────────────┐  ┌─────────────────┐ │
│  │ File-Based     │  │ Neo4j-Backed    │  │ Human Review    │ │
│  │ Skills Registry│  │ Skills Registry  │  │ Workflow        │ │
│  │ (YAML frontmatter)│ │ (Graph Store)    │  │ (Approval)      │ │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘ │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

## Sub-Agents

### 1. Skill Discovery

```go
package skills

// SkillDiscovery handles skill extraction and suggestion

// ExtractSkills extracts skills from content using LLM
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

// SuggestSkills suggests relevant skills based on context
func (s *SkillDiscovery) SuggestSkills(ctx context.Context, trigger string, context string, limit int) ([]*Skill, error) {
    // 1. LLM Processing: Suggest skills based on context
    suggestions, err := s.llmProvider.SuggestSkills(ctx, trigger, context, limit)
    if err != nil {
        return nil, fmt.Errorf("skill suggestion failed: %w", err)
    }
    
    // 2. Validation: Verify suggested skills
    verified, err := s.verifySkills(suggestions)
    if err != nil {
        return nil, fmt.Errorf("skill verification failed: %w", err)
    }
    
    return verified, nil
}
```

### 2. Skill Executor

```go
package skills

// SkillExecutor handles skill execution

// ExecuteSkill executes a skill with given context
func (s *SkillExecutor) ExecuteSkill(ctx context.Context, skillID string, context map[string]interface{}) (map[string]interface{}, error) {
    // 1. Get skill from registry
    skill, err := s.skillRegistry.GetSkill(ctx, skillID)
    if err != nil {
        return nil, fmt.Errorf("skill not found: %w", err)
    }
    
    // 2. Validate skill
    if err := s.validateSkill(skill); err != nil {
        return nil, fmt.Errorf("invalid skill: %w", err)
    }
    
    // 3. Execute skill action
    return s.executeSkillAction(ctx, skill, context)
}

// executeSkillAction executes the skill's action with context
func (s *SkillExecutor) executeSkillAction(
    ctx context.Context,
    skill *Skill,
    context map[string]interface{},
) (map[string]interface{}, error) {
    // Implementation of skill execution
    // ...
}

// validateSkill validates a skill before execution
func (s *SkillExecutor) validateSkill(skill *Skill) error {
    // Implementation of skill validation
    // ...
}
```

### 3. Chain Coordinator

```go
package skills

// ChainCoordinator manages skill chain execution

// ExecuteChain executes a skill chain with given context
func (c *ChainCoordinator) ExecuteChain(ctx context.Context, chainID string, context map[string]interface{}, timeout time.Duration) (map[string]interface{}, error) {
    // 1. Get chain from registry
    chain, err := c.chainRegistry.GetChain(ctx, chainID)
    if err != nil {
        return nil, fmt.Errorf("chain not found: %w", err)
    }
    
    // 2. Validate chain
    if err := c.validateChain(chain); err != nil {
        return nil, fmt.Errorf("invalid chain: %w", err)
    }
    
    // 3. Execute chain steps
    return c.executeChainSteps(ctx, chain, context, timeout)
}

// executeChainSteps executes the steps of a skill chain
func (c *ChainCoordinator) executeChainSteps(
    ctx context.Context,
    chain *SkillChain,
    context map[string]interface{},
    timeout time.Duration,
) (map[string]interface{}, error) {
    // Implementation of chain execution
    // ...
}

// validateChain validates a skill chain before execution
func (c *ChainCoordinator) validateChain(chain *SkillChain) error {
    // Implementation of chain validation
    // ...
}
```

## Integration with Other Orchestrators

### Skills ↔ Memory

```go
// MemorySkillExtractor extracts skills from memory content
func (m *MemorySkillExtractor) ExtractSkillsFromMemory(ctx context.Context, memory *Memory) ([]*Skill, error) {
    // 1. Get memory content
    content := memory.Content
    
    // 2. Extract skills from content
    skills, err := m.skillDiscovery.ExtractSkills(ctx, content)
    if err != nil {
        return nil, fmt.Errorf("skill extraction from memory failed: %w", err)
    }
    
    // 3. Associate skills with memory
    for _, skill := range skills {
        skill.SourceMemory = memory.ID
    }
    
    return skills, nil
}

// MemorySkillExecutor executes skills with memory context
func (m *MemorySkillExecutor) ExecuteSkillsWithMemory(
    ctx context.Context,
    skillIDs []string,
    memory *Memory,
) (map[string]interface{}, error) {
    // 1. Create context with memory information
    context := map[string]interface{}{
        "memory": memory,
        "memory_content": memory.Content,
    }
    
    // 2. Execute each skill
    for _, skillID := range skillIDs {
        result, err := m.skillExecutor.ExecuteSkill(ctx, skillID, context)
        if err != nil {
            return nil, fmt.Errorf("skill execution failed: %w", err)
        }
        
        // 3. Update memory with skill execution results
        if err := m.updateMemoryWithSkillResult(ctx, memory, skillID, result); err != nil {
            return nil, fmt.Errorf("failed to update memory with skill result: %w", err)
        }
    }
    
    return context, nil
}

// updateMemoryWithSkillResult updates memory with skill execution results
func (m *MemorySkillExecutor) updateMemoryWithSkillResult(
    ctx context.Context,
    memory *Memory,
    skillID string,
    result map[string]interface{},
) error {
    // Implementation of memory update
    // ...
}
```

### Skills ↔ Compression

```go
// SkillCompressionExtractor extracts skills from compressed memory
func (s *SkillCompressionExtractor) ExtractSkillsFromCompressed(
    ctx context.Context,
    compressed *CompressedMemory,
) ([]*Skill, error) {
    // 1. Decompress memory
    memory, err := s.compressionEngine.Decompress(ctx, compressed)
    if err != nil {
        return nil, fmt.Errorf("decompression failed: %w", err)
    }
    
    // 2. Extract skills from decompressed content
    skills, err := s.skillDiscovery.ExtractSkills(ctx, memory.Content)
    if err != nil {
        return nil, fmt.Errorf("skill extraction from compressed memory failed: %w", err)
    }
    
    // 3. Associate skills with compressed memory
    for _, skill := range skills {
        skill.SourceMemory = memory.ID
        skill.Compressed = true
    }
    
    return skills, nil
}

// CompressedSkillExecutor executes skills with compressed memory context
func (s *CompressedSkillExecutor) ExecuteSkillsWithCompressed(
    ctx context.Context,
    skillIDs []string,
    compressed *CompressedMemory,
) (map[string]interface{}, error) {
    // 1. Decompress memory
    memory, err := s.compressionEngine.Decompress(ctx, compressed)
    if err != nil {
        return nil, fmt.Errorf("decompression failed: %w", err)
    }
    
    // 2. Create context with compressed memory information
    context := map[string]interface{}{
        "memory":        memory,
        "memory_content": memory.Content,
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

## Human Review Workflow

```go
package skills

// ReviewWorkflow handles human review of skills

// SubmitForReview submits a skill for human review
func (r *ReviewWorkflow) SubmitForReview(ctx context.Context, skillID string, notes string) (string, error) {
    // 1. Get skill from registry
    skill, err := r.skillRegistry.GetSkill(ctx, skillID)
    if err != nil {
        return "", fmt.Errorf("skill not found: %w", err)
    }
    
    // 2. Create review record
    review := &Review{
        SkillID: skill.ID,
        Status:  ReviewStatusPending,
        Notes:   notes,
        CreatedAt: time.Now(),
    }
    
    // 3. Save review record
    reviewID, err := r.reviewStore.CreateReview(ctx, review)
    if err != nil {
        return "", fmt.Errorf("failed to create review: %w", err)
    }
    
    // 4. Update skill status
    skill.ReviewStatus = ReviewStatusPending
    if err := r.skillRegistry.UpdateSkill(ctx, skill); err != nil {
        return "", fmt.Errorf("failed to update skill status: %w", err)
    }
    
    return reviewID, nil
}

// ProcessReview processes a human review decision
func (r *ReviewWorkflow) ProcessReview(ctx context.Context, reviewID string, approved bool, notes string) error {
    // 1. Get review record
    review, err := r.reviewStore.GetReview(ctx, reviewID)
    if err != nil {
        return fmt.Errorf("review not found: %w", err)
    }
    
    // 2. Get skill
    skill, err := r.skillRegistry.GetSkill(ctx, review.SkillID)
    if err != nil {
        return fmt.Errorf("skill not found: %w", err)
    }
    
    // 3. Process review decision
    if approved {
        skill.Verified = true
        skill.ReviewNotes = notes
    } else {
        skill.Verified = false
        skill.ReviewNotes = notes
        skill.Status = SkillStatusRejected
    }
    
    // 4. Update skill
    if err := r.skillRegistry.UpdateSkill(ctx, skill); err != nil {
        return fmt.Errorf("failed to update skill: %w", err)
    }
    
    // 5. Update review status
    review.Status = ReviewStatusCompleted
    review.Decision = approved
    review.DecisionNotes = notes
    review.UpdatedAt = time.Now()
    
    if err := r.reviewStore.UpdateReview(ctx, review); err != nil {
        return fmt.Errorf("failed to update review: %w", err)
    }
    
    return nil
}

// ListPendingReviews lists pending reviews
func (r *ReviewWorkflow) ListPendingReviews(ctx context.Context) ([]*Review, error) {
    // 1. Get pending reviews from store
    reviews, err := r.reviewStore.GetPendingReviews(ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to get pending reviews: %w", err)
    }
    
    return reviews, nil
}
```

## Testing

```go
func TestSkillDiscovery_ExtractSkills(t *testing.T) {
    // Setup test data
    testContent := "This content contains skills that should be extracted."
    expectedSkills := []*Skill{{
        Name:        "test-skill",
        Description: "Test skill description",
        Trigger:     "test-trigger",
    }}
    
    // Create test discovery with mock LLM provider
    mockLLM := &MockLLMProvider{}
    mockLLM.On("ExtractSkills", testContent).Return(expectedSkills, nil)
    
    discovery := &SkillDiscovery{llmProvider: mockLLM}
    
    // Execute test
    skills, err := discovery.ExtractSkills(context.Background(), testContent)
    
    // Verify results
    if err != nil {
        t.Errorf("ExtractSkills returned error: %v", err)
    }
    
    if len(skills) != 1 {
        t.Errorf("Expected 1 skill, got %d", len(skills))
    }
    
    // Verify mock calls
    mockLLM.AssertExpectations(t)
}

func TestSkillExecutor_ExecuteSkill(t *testing.T) {
    // Setup test data
    testSkill := &Skill{
        ID:          "skill-1",
        Name:        "test-skill",
        Description: "Test skill description",
        Trigger:     "test-trigger",
        Action:      "test-action",
    }
    testContext := map[string]interface{}{
        "key": "value",
    }
    expectedResult := map[string]interface{}{
        "result": "success",
    }
    
    // Create test executor with mock skill registry and mock executor
    mockRegistry := &MockSkillRegistry{}
    mockRegistry.On("GetSkill", testSkill.ID).Return(testSkill, nil)
    
    mockExecutor := &MockSkillExecutor{}
    mockExecutor.On("Execute", testSkill, testContext).Return(expectedResult, nil)
    
    executor := &SkillExecutor{
        skillRegistry: mockRegistry,
        executor:      mockExecutor,
    }
    
    // Execute test
    result, err := executor.ExecuteSkill(context.Background(), testSkill.ID, testContext)
    
    // Verify results
    if err != nil {
        t.Errorf("ExecuteSkill returned error: %v", err)
    }
    
    if result == nil {
        t.Error("Expected result, got nil")
    }
    
    // Verify mock calls
    mockRegistry.AssertExpectations(t)
    mockExecutor.AssertExpectations(t)
}

func TestChainCoordinator_ExecuteChain(t *testing.T) {
    // Setup test data
    testChain := &SkillChain{
        ID:     "chain-1",
        Name:   "test-chain",
        Trigger: "test-trigger",
        Steps: []ChainStep{{
            SkillID: "skill-1",
            Order:   1,
        }},
    }
    testContext := map[string]interface{}{
        "key": "value",
    }
    expectedResult := map[string]interface{}{
        "result": "success",
    }
    
    // Create test coordinator with mock chain registry and mock executor
    mockRegistry := &MockChainRegistry{}
    mockRegistry.On("GetChain", testChain.ID).Return(testChain, nil)
    
    mockExecutor := &MockSkillExecutor{}
    mockExecutor.On("Execute", testChain.Steps[0].SkillID, testContext).Return(expectedResult, nil)
    
    coordinator := &ChainCoordinator{
        chainRegistry: mockRegistry,
        executor:      mockExecutor,
    }
    
    // Execute test
    result, err := coordinator.ExecuteChain(context.Background(), testChain.ID, testContext, 10*time.Second)
    
    // Verify results
    if err != nil {
        t.Errorf("ExecuteChain returned error: %v", err)
    }
    
    if result == nil {
        t.Error("Expected result, got nil")
    }
    
    // Verify mock calls
    mockRegistry.AssertExpectations(t)
    mockExecutor.AssertExpectations(t)
}

func TestReviewWorkflow_SubmitForReview(t *testing.T) {
    // Setup test data
    testSkill := &Skill{
        ID:          "skill-1",
        Name:        "test-skill",
        Description: "Test skill description",
        Trigger:     "test-trigger",
        Action:      "test-action",
    }
    testNotes := "Test review notes"
    
    // Create test workflow with mock skill registry and mock review store
    mockRegistry := &MockSkillRegistry{}
    mockRegistry.On("GetSkill", testSkill.ID).Return(testSkill, nil)
    
    mockReviewStore := &MockReviewStore{}
    mockReviewStore.On("CreateReview", mock.Anything, mock.Anything).Return("review-1", nil)
    
    workflow := &ReviewWorkflow{
        skillRegistry: mockRegistry,
        reviewStore:   mockReviewStore,
    }
    
    // Execute test
    reviewID, err := workflow.SubmitForReview(context.Background(), testSkill.ID, testNotes)
    
    // Verify results
    if err != nil {
        t.Errorf("SubmitForReview returned error: %v", err)
    }
    
    if reviewID == "" {
        t.Error("Expected review ID, got empty string")
    }
    
    // Verify mock calls
    mockRegistry.AssertExpectations(t)
    mockReviewStore.AssertExpectations(t)
}

func TestReviewWorkflow_ProcessReview(t *testing.T) {
    // Setup test data
    testReview := &Review{
        ID:        "review-1",
        SkillID:   "skill-1",
        Status:    ReviewStatusPending,
        CreatedAt: time.Now(),
    }
    testSkill := &Skill{
        ID:          "skill-1",
        Name:        "test-skill",
        Description: "Test skill description",
        Trigger:     "test-trigger",
        Action:      "test-action",
        Verified:    false,
        ReviewStatus: ReviewStatusPending,
    }
    
    // Create test workflow with mock skill registry and mock review store
    mockRegistry := &MockSkillRegistry{}
    mockRegistry.On("GetSkill", testReview.SkillID).Return(testSkill, nil)
    mockRegistry.On("UpdateSkill", mock.Anything, mock.Anything).Return(nil)
    
    mockReviewStore := &MockReviewStore{}
    mockReviewStore.On("GetReview", testReview.ID).Return(testReview, nil)
    mockReviewStore.On("UpdateReview", mock.Anything, mock.Anything).Return(nil)
    
    workflow := &ReviewWorkflow{
        skillRegistry: mockRegistry,
        reviewStore:   mockReviewStore,
    }
    
    // Execute test
    err := workflow.ProcessReview(context.Background(), testReview.ID, true, "Approved")
    
    // Verify results
    if err != nil {
        t.Errorf("ProcessReview returned error: %v", err)
    }
    
    // Verify mock calls
    mockRegistry.AssertExpectations(t)
    mockReviewStore.AssertExpectations(t)
}
```

---

*This documentation provides a comprehensive overview of the Skills System Orchestrator and its sub-agents, including their integration with other system components.*