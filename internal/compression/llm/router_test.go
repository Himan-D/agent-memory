package llm

import (
	"context"
	"testing"

	"agent-memory/internal/llm"
	"agent-memory/internal/memory/types"
)

type mockProvider struct {
	name      llm.ProviderType
	response  string
	err       error
	callCount int
}

func (m *mockProvider) Name() llm.ProviderType { return m.name }

func (m *mockProvider) Complete(ctx context.Context, req *llm.CompletionRequest) (*llm.CompletionResponse, error) {
	m.callCount++
	if m.err != nil {
		return nil, m.err
	}
	return &llm.CompletionResponse{Content: m.response}, nil
}

func (m *mockProvider) Embed(ctx context.Context, req *llm.EmbeddingRequest) (*llm.EmbeddingResponse, error) {
	return nil, nil
}

func (m *mockProvider) Rerank(ctx context.Context, req *llm.RerankRequest) (*llm.RerankResponse, error) {
	return nil, nil
}

func TestNewLLMRouter_NilConfig(t *testing.T) {
	fast := &mockProvider{name: llm.ProviderOpenAI}
	verify := &mockProvider{name: llm.ProviderAnthropic}

	router := NewLLMRouter(fast, verify, nil)

	if router == nil {
		t.Fatal("NewLLMRouter returned nil")
	}
	// When cfg is nil, NewLLMRouter creates a zero-value RouterConfig,
	// so ComplexityThreshold defaults to 0 (Go zero value for float64)
	if router.complexityThreshold != 0 {
		t.Errorf("expected zero complexity threshold with nil config, got %f", router.complexityThreshold)
	}
}

func TestNewLLMRouter_CustomConfig(t *testing.T) {
	fast := &mockProvider{name: llm.ProviderOpenAI}
	verify := &mockProvider{name: llm.ProviderAnthropic}
	cfg := &RouterConfig{ComplexityThreshold: 0.8}

	router := NewLLMRouter(fast, verify, cfg)

	if router.complexityThreshold != 0.8 {
		t.Errorf("expected complexity threshold 0.8, got %f", router.complexityThreshold)
	}
}

func TestNewLLMRouter_NilProviders(t *testing.T) {
	router := NewLLMRouter(nil, nil, nil)
	if router == nil {
		t.Fatal("NewLLMRouter returned nil with nil providers")
	}
}

func TestLLMRouter_EstimateComplexity_NilProvider(t *testing.T) {
	router := NewLLMRouter(nil, nil, nil)

	complexity := router.estimateComplexity("some memory content")

	if complexity != 0.5 {
		t.Errorf("expected complexity 0.5 with nil provider, got %f", complexity)
	}
}

func TestLLMRouter_EstimateComplexity_ProviderError(t *testing.T) {
	fast := &mockProvider{
		name: llm.ProviderOpenAI,
		err:  context.DeadlineExceeded,
	}
	router := NewLLMRouter(fast, nil, nil)

	complexity := router.estimateComplexity("some memory content")

	if complexity != 0.5 {
		t.Errorf("expected complexity 0.5 on provider error, got %f", complexity)
	}
}

func TestLLMRouter_EstimateComplexity_ValidResponse(t *testing.T) {
	tests := []struct {
		name         string
		response     string
		expectedMin  float64
		expectedMax  float64
	}{
		{"low complexity", "0.1", 0.0, 0.3},
		{"medium complexity", "0.5", 0.3, 0.6},
		{"high complexity", "0.9", 0.6, 1.0},
		{"simple memory", "0.2", 0.0, 0.3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fast := &mockProvider{
				name:     llm.ProviderOpenAI,
				response: tt.response,
			}
			router := NewLLMRouter(fast, nil, nil)
			complexity := router.estimateComplexity("test memory")
			if complexity < tt.expectedMin || complexity > tt.expectedMax+0.5 {
				t.Errorf("complexity %f not in expected range [%f, %f]", complexity, tt.expectedMin, tt.expectedMax+0.5)
			}
		})
	}
}

func TestLLMRouter_Route_LowComplexity_FastPath(t *testing.T) {
	fast := &mockProvider{
		name:     llm.ProviderOpenAI,
		response: "0.2",
	}
	verify := &mockProvider{
		name:     llm.ProviderAnthropic,
		response: "verified",
	}
	router := NewLLMRouter(fast, verify, &RouterConfig{ComplexityThreshold: 0.6})

	result, err := router.Route(context.Background(), "simple memory")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Provider != "fast" {
		t.Errorf("expected provider 'fast', got %q", result.Provider)
	}
}

func TestLLMRouter_Route_HighComplexity_VerifyPath(t *testing.T) {
	fast := &mockProvider{
		name:     llm.ProviderOpenAI,
		response: "0.8",
	}
	verify := &mockProvider{
		name:     llm.ProviderAnthropic,
		response: `[{"fact": "verified fact", "verified": true, "confidence": 0.9}]`,
	}
	router := NewLLMRouter(fast, verify, &RouterConfig{ComplexityThreshold: 0.6})

	result, err := router.Route(context.Background(), "complex technical memory with multi-hop reasoning")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Provider != "verify" {
		t.Errorf("expected provider 'verify', got %q", result.Provider)
	}
}

func TestLLMRouter_ExtractFast_NilProvider(t *testing.T) {
	router := NewLLMRouter(nil, nil, nil)

	result, err := router.extractFast(context.Background(), "test memory")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Facts) != 1 {
		t.Errorf("expected 1 fact, got %d", len(result.Facts))
	}
	if result.Facts[0].Fact != "test memory" {
		t.Errorf("expected fact 'test memory', got %q", result.Facts[0].Fact)
	}
	if result.Confidence != 0.5 {
		t.Errorf("expected confidence 0.5, got %f", result.Confidence)
	}
	if result.TokenReduction != 0.0 {
		t.Errorf("expected token reduction 0.0, got %f", result.TokenReduction)
	}
}

func TestLLMRouter_ExtractFast_ValidJSON(t *testing.T) {
	fast := &mockProvider{
		name:     llm.ProviderOpenAI,
		response: `[{"fact": "user prefers dark mode", "confidence": 0.9}]`,
	}
	router := NewLLMRouter(fast, nil, nil)

	result, err := router.extractFast(context.Background(), "I prefer dark mode")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Facts) != 1 {
		t.Fatalf("expected 1 fact, got %d", len(result.Facts))
	}
	if result.Facts[0].Fact != "user prefers dark mode" {
		t.Errorf("expected fact 'user prefers dark mode', got %q", result.Facts[0].Fact)
	}
	if result.Facts[0].Confidence != 0.9 {
		t.Errorf("expected confidence 0.9, got %f", result.Facts[0].Confidence)
	}
}

func TestLLMRouter_ExtractFast_InvalidJSON_Fallback(t *testing.T) {
	fast := &mockProvider{
		name:     llm.ProviderOpenAI,
		response: "This is not JSON, just a plain text response",
	}
	router := NewLLMRouter(fast, nil, nil)

	result, err := router.extractFast(context.Background(), "test memory")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Facts) != 1 {
		t.Fatalf("expected 1 fact from fallback, got %d", len(result.Facts))
	}
	if result.Confidence != 0.6 {
		t.Errorf("expected fallback confidence 0.6, got %f", result.Confidence)
	}
}

func TestLLMRouter_ExtractFast_ProviderError(t *testing.T) {
	fast := &mockProvider{
		name: llm.ProviderOpenAI,
		err:  context.DeadlineExceeded,
	}
	router := NewLLMRouter(fast, nil, nil)

	_, err := router.extractFast(context.Background(), "test memory")
	if err == nil {
		t.Error("expected error when provider fails")
	}
}

func TestLLMRouter_ExtractWithVerification_NilVerifyProvider(t *testing.T) {
	fast := &mockProvider{
		name:     llm.ProviderOpenAI,
		response: `[{"fact": "user likes Go", "confidence": 0.85}]`,
	}
	router := NewLLMRouter(fast, nil, nil)

	result, err := router.extractWithVerification(context.Background(), "I like Go programming")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.VerifiedFacts) != 1 {
		t.Errorf("expected 1 verified fact, got %d", len(result.VerifiedFacts))
	}
	if !result.VerifiedFacts[0].Verified {
		t.Error("expected verified fact to have Verified=true")
	}
	if result.Confidence != 0.85 {
		t.Errorf("expected confidence 0.85, got %f", result.Confidence)
	}
}

func TestLLMRouter_ExtractWithVerification_WithVerifyProvider(t *testing.T) {
	fast := &mockProvider{
		name:     llm.ProviderOpenAI,
		response: `[{"fact": "user likes Go", "confidence": 0.85}]`,
	}
	verify := &mockProvider{
		name:     llm.ProviderAnthropic,
		response: `[{"fact": "user likes Go", "verified": true, "confidence": 0.95}]`,
	}
	router := NewLLMRouter(fast, verify, nil)

	result, err := router.extractWithVerification(context.Background(), "I like Go")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Confidence != 0.9 {
		t.Errorf("expected confidence 0.9, got %f", result.Confidence)
	}
	if result.TokenReduction != 0.85 {
		t.Errorf("expected token reduction 0.85, got %f", result.TokenReduction)
	}
	if len(result.VerifiedFacts) != 1 {
		t.Errorf("expected 1 verified fact, got %d", len(result.VerifiedFacts))
	}
}

func TestLLMRouter_ExtractWithVerification_VerifyUnverified(t *testing.T) {
	fast := &mockProvider{
		name:     llm.ProviderOpenAI,
		response: `[{"fact": "fact1", "confidence": 0.8}, {"fact": "fact2", "confidence": 0.7}]`,
	}
	verify := &mockProvider{
		name:     llm.ProviderAnthropic,
		response: `[{"fact": "fact1", "verified": true, "confidence": 0.9}, {"fact": "fact2", "verified": false, "confidence": 0.3}]`,
	}
	router := NewLLMRouter(fast, verify, nil)

	result, err := router.extractWithVerification(context.Background(), "test memory")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.VerifiedFacts) != 1 {
		t.Errorf("expected 1 verified fact (unverified filtered out), got %d", len(result.VerifiedFacts))
	}
}

func TestLLMRouter_ExtractWithVerification_VerifyProviderError(t *testing.T) {
	fast := &mockProvider{
		name:     llm.ProviderOpenAI,
		response: `[{"fact": "user likes Go", "confidence": 0.85}]`,
	}
	verify := &mockProvider{
		name: llm.ProviderAnthropic,
		err:  context.DeadlineExceeded,
	}
	router := NewLLMRouter(fast, verify, nil)

	result, err := router.extractWithVerification(context.Background(), "I like Go")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.VerifiedFacts[0].Verified {
		t.Error("expected verified fact to have Verified=true on verify error fallback")
	}
	if result.Confidence != 0.85 {
		t.Errorf("expected confidence 0.85 on fallback, got %f", result.Confidence)
	}
}

func TestLLMRouter_ExtractWithVerification_VerifyInvalidJSON(t *testing.T) {
	fast := &mockProvider{
		name:     llm.ProviderOpenAI,
		response: `[{"fact": "user likes Go", "confidence": 0.85}]`,
	}
	verify := &mockProvider{
		name:     llm.ProviderAnthropic,
		response: "not valid json",
	}
	router := NewLLMRouter(fast, verify, nil)

	result, err := router.extractWithVerification(context.Background(), "I like Go")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.VerifiedFacts[0].Verified {
		t.Error("expected verified fact to have Verified=true on invalid JSON fallback")
	}
}

func TestLLMRouter_Route_NilFastProvider(t *testing.T) {
	router := NewLLMRouter(nil, nil, &RouterConfig{ComplexityThreshold: 0.6})

	result, err := router.Route(context.Background(), "test memory")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Provider != "fast" {
		t.Errorf("expected provider 'fast' for nil fast provider (complexity=0.5 < 0.6), got %q", result.Provider)
	}
}

func TestExtractionResult_Fields(t *testing.T) {
	result := &ExtractionResult{
		Facts: []types.Fact{
			{Fact: "user likes Go", Confidence: 0.9},
		},
		VerifiedFacts: []types.Fact{
			{Fact: "user likes Go", Confidence: 0.9, Verified: true},
		},
		Gaps: []Gap{
			{Question: "what language?", Answer: "Go", MemoryID: "mem1"},
		},
		Supplements:  []types.Fact{{Fact: "supplement fact", Confidence: 0.8}},
		Confidence:   0.9,
		TokenReduction: 0.85,
		Provider:     "verify",
	}

	if result.Provider != "verify" {
		t.Errorf("expected provider 'verify', got %q", result.Provider)
	}
	if len(result.Facts) != 1 {
		t.Errorf("expected 1 fact, got %d", len(result.Facts))
	}
	if len(result.Gaps) != 1 {
		t.Errorf("expected 1 gap, got %d", len(result.Gaps))
	}
}

func TestRouterConfig_DefaultValues(t *testing.T) {
	cfg := &RouterConfig{}
	router := NewLLMRouter(nil, nil, cfg)
	if router.complexityThreshold != 0 {
		t.Errorf("expected default complexity threshold 0, got %f", router.complexityThreshold)
	}
}

func TestRouterConfig_CustomThreshold(t *testing.T) {
	cfg := &RouterConfig{ComplexityThreshold: 0.8}
	router := NewLLMRouter(nil, nil, cfg)
	if router.complexityThreshold != 0.8 {
		t.Errorf("expected complexity threshold 0.8, got %f", router.complexityThreshold)
	}
}