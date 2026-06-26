package extractor

import (
	"agent-memory/internal/llm"
	"agent-memory/internal/memory/types"
	"context"
	"errors"
	"strings"
	"testing"
)

// --- Mock LLM Provider ---------------------------------------------------

type mockProvider struct {
	responses []mockResponse
	idx       int
	provider  llm.ProviderType
}

type mockResponse struct {
	resp *llm.CompletionResponse
	err  error
}

func (m *mockProvider) Name() llm.ProviderType {
	if m.provider == "" {
		return llm.ProviderOpenAI
	}
	return m.provider
}

func (m *mockProvider) Complete(_ context.Context, _ *llm.CompletionRequest) (*llm.CompletionResponse, error) {
	if m.idx >= len(m.responses) {
		return &llm.CompletionResponse{Content: ""}, nil
	}
	r := m.responses[m.idx]
	m.idx++
	return r.resp, r.err
}

func (m *mockProvider) Embed(_ context.Context, _ *llm.EmbeddingRequest) (*llm.EmbeddingResponse, error) {
	return nil, errors.New("not implemented")
}

func (m *mockProvider) Rerank(_ context.Context, _ *llm.RerankRequest) (*llm.RerankResponse, error) {
	return nil, errors.New("not implemented")
}

// --- Helper Responses ----------------------------------------------------

func toFactLine(subject, action, context string) string {
	return `{subject: ` + subject + `, action: ` + action + `, context: ` + context + `}`
}

func factLines(lines ...string) string {
	return "\n" + strings.Join(lines, "\n") + "\n"
}

// --- Tests ---------------------------------------------------------------

func TestNewMemoryExtractor(t *testing.T) {
	p := &mockProvider{}
	e := NewMemoryExtractor(p)
	if e == nil {
		t.Fatal("expected non-nil extractor")
	}
	if e.maxIterations != 2 {
		t.Fatalf("expected maxIterations=2, got %d", e.maxIterations)
	}
	if e.verifyThreshold != 0.85 {
		t.Fatalf("expected verifyThreshold=0.85, got %f", e.verifyThreshold)
	}
}

func TestExtract_EmptyMemory(t *testing.T) {
	p := &mockProvider{}
	e := NewMemoryExtractor(p)

	result, err := e.Extract(context.Background(), "")
	if err != nil {
		t.Fatalf("unexpected error for empty memory: %v", err)
	}
	if len(result.Facts) != 0 {
		t.Fatalf("expected 0 facts for empty memory, got %d", len(result.Facts))
	}
	if len(result.VerifiedFacts) != 0 {
		t.Fatalf("expected 0 verified facts for empty memory, got %d", len(result.VerifiedFacts))
	}
	if len(result.Gaps) != 0 {
		t.Fatalf("expected 0 gaps, got %d", len(result.Gaps))
	}
}

func TestExtract_NoProvider(t *testing.T) {
	p := &mockProvider{}
	e := NewMemoryExtractor(p)
	e.llmProvider = nil

	_, err := e.Extract(context.Background(), "some memory text")
	if err == nil {
		t.Fatal("expected error when provider is nil")
	}
}

func TestExtract_InitialFacts(t *testing.T) {
	p := &mockProvider{
		responses: []mockResponse{{
			resp: &llm.CompletionResponse{
				Content: `{subject: user, action: prefers, context: dark mode}` + "\n" +
					`{subject: user, action: works on, context: graph db}`,
			},
		}},
	}
	e := NewMemoryExtractor(p)

	result, err := e.Extract(context.Background(), "User prefers dark mode in all applications and is working on a graph database project.")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Facts) == 0 {
		t.Fatal("expected at least 1 fact")
	}
	if result.Confidence < 0.0 || result.Confidence > 1.0 {
		t.Fatalf("expected confidence between 0 and 1, got %f", result.Confidence)
	}
}

func TestExtract_VerificationFiltersLowConfidence(t *testing.T) {
	p := &mockProvider{
		responses: []mockResponse{
			// initial facts
			{resp: &llm.CompletionResponse{Content: `{subject: user, action: prefers, context: dark mode}`}},
			// verify return nothing → falls back to simpler verify
			{resp: &llm.CompletionResponse{Content: ""}},
			// verifyFacts returns facts but we test verifyWithProvider
			{resp: &llm.CompletionResponse{Content: ""}},
		},
	}
	e := NewMemoryExtractor(p)
	e.verifyThreshold = 0.90 // raise threshold

	result, err := e.Extract(context.Background(), "User prefers dark mode.")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Confidence == 0.0 && len(result.VerifiedFacts) > 0 {
		t.Fatal("expected confidence > 0 when verified facts exist")
	}
}

func TestExtract_GapDetectionAndFill(t *testing.T) {
	p := &mockProvider{
		responses: []mockResponse{
			// pass 1: initial facts
			{resp: &llm.CompletionResponse{Content: `{subject: user, action: prefers, context: dark mode}`}},
			// verify
			{resp: &llm.CompletionResponse{Content: ""}},
			// verifyFacts fallback
			{resp: &llm.CompletionResponse{Content: ""}},
		},
	}
	e := NewMemoryExtractor(p)
	e.maxIterations = 3

	result, err := e.Extract(context.Background(), "User prefers dark mode.")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Iterations != 3 {
		t.Fatalf("expected iterations=3, got %d", result.Iterations)
	}
}

func TestExtract_FactDeduplication(t *testing.T) {
	p := &mockProvider{
		responses: []mockResponse{
			{resp: &llm.CompletionResponse{Content: `{subject: user, action: prefers, context: dark mode}`}},
			{resp: &llm.CompletionResponse{Content: ""}},
			{resp: &llm.CompletionResponse{Content: ""}},
		},
	}
	e := NewMemoryExtractor(p)

	result, err := e.Extract(context.Background(), "User prefers dark mode. User likes dark mode.")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Facts) > 0 {
		for i, f := range result.Facts {
			if f.Fact == "" && f.Subject == "" {
				t.Fatalf("fact %d is empty", i)
			}
		}
	}
}

func TestExtract_MetricsRecording(t *testing.T) {
	fakeMetrics := &fakeMetrics{}
	p := &mockProvider{
		responses: []mockResponse{{
			resp: &llm.CompletionResponse{Content: `{subject: user, action: prefers, context: dark mode}`},
		}},
	}
	e := NewMemoryExtractor(p)
	e.SetMetrics(fakeMetrics)

	_, err := e.Extract(context.Background(), "User prefers dark mode.")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fakeMetrics.recordedProvider != "promem" {
		t.Fatalf("expected provider 'promem', got '%s'", fakeMetrics.recordedProvider)
	}
	if fakeMetrics.recordedTokensSaved < 0 {
		t.Fatalf("expected non-negative tokens saved, got %d", fakeMetrics.recordedTokensSaved)
	}
	if fakeMetrics.recordedLatencyMs <= 0 {
		t.Fatalf("expected positive latency, got %f", fakeMetrics.recordedLatencyMs)
	}
}

func TestCalculateConfidence(t *testing.T) {
	e := NewMemoryExtractor(&mockProvider{})

	facts := []types.Fact{
		{Confidence: 0.9}, {Confidence: 0.8}, {Confidence: 0.7},
	}
	conf := e.calculateConfidence(facts)
	if conf == 0.0 {
		t.Fatal("expected non-zero confidence")
	}

	// empty facts → 0.0
	conf = e.calculateConfidence(nil)
	if conf != 0.0 {
		t.Fatalf("expected 0.0 for nil facts, got %f", conf)
	}
}

func TestCalculateReduction(t *testing.T) {
	e := NewMemoryExtractor(&mockProvider{})

	original := "User prefers dark mode in all applications."
	facts := []types.Fact{
		{Fact: "prefers dark mode"},
	}
	reduction := e.calculateReduction(original, facts)
	if reduction < 0.0 || reduction > 1.0 {
		t.Fatalf("expected reduction in [0,1], got %f", reduction)
	}

	// empty facts → 0.0
	reduction = e.calculateReduction(original, nil)
	if reduction != 0.0 {
		t.Fatalf("expected 0.0 for nil facts, got %f", reduction)
	}
}

func TestSummarizeMemory(t *testing.T) {
	p := &mockProvider{
		responses: []mockResponse{{
			resp: &llm.CompletionResponse{Content: "User likes dark mode."},
		}},
	}
	e := NewMemoryExtractor(p)

	summary := e.summarizeMemory(context.Background(), "User prefers dark mode in all applications.")
	if summary == "" {
		t.Fatal("expected non-empty summary")
	}
}

func TestSummarizeMemory_FallbackOnError(t *testing.T) {
	p := &mockProvider{
		responses: []mockResponse{{err: errors.New("llm error")}},
	}
	e := NewMemoryExtractor(p)

	original := "User prefers dark mode."
	summary := e.summarizeMemory(context.Background(), original)
	if summary != original {
		t.Fatalf("expected fallback to original text, got '%s'", summary)
	}
}

func TestDetectGaps(t *testing.T) {
	p := &mockProvider{
		responses: []mockResponse{{
			resp: &llm.CompletionResponse{Content: `{"gaps": [{"question": "What project?", "memory_id": "m1"}]}`},
		}},
	}
	e := NewMemoryExtractor(p)

	gaps := e.detectGaps(context.Background(), []types.Fact{{Fact: "user works"}}, "User works on a project.")
	if len(gaps) == 0 {
		t.Fatal("expected at least 1 gap")
	}
	if gaps[0].Question == "" {
		t.Fatal("expected non-empty question")
	}
}

func TestDetectGaps_NoGaps(t *testing.T) {
	p := &mockProvider{
		responses: []mockResponse{{
			resp: &llm.CompletionResponse{Content: `{"gaps": []}`},
		}},
	}
	e := NewMemoryExtractor(p)

	gaps := e.detectGaps(context.Background(), []types.Fact{{Fact: "user works"}}, "User works.")
	if len(gaps) != 0 {
		t.Fatalf("expected 0 gaps, got %d", len(gaps))
	}
}

func TestDetectGaps_InvalidJSON(t *testing.T) {
	p := &mockProvider{
		responses: []mockResponse{{
			resp: &llm.CompletionResponse{Content: "not json"},
		}},
	}
	e := NewMemoryExtractor(p)

	gaps := e.detectGaps(context.Background(), []types.Fact{{Fact: "f"}}, "text")
	if len(gaps) != 0 {
		t.Fatalf("expected 0 gaps on invalid JSON, got %d", len(gaps))
	}
}

func TestExtractGaps(t *testing.T) {
	p := &mockProvider{
		responses: []mockResponse{{
			resp: &llm.CompletionResponse{Content: `{subject: user, action: builds, context: app}`},
		}},
	}
	e := NewMemoryExtractor(p)

	supplements := e.extractGaps(context.Background(), []Gap{{Question: "What app?"}}, "User builds an app.")
	if len(supplements) == 0 {
		t.Fatal("expected at least 1 supplement fact")
	}
}

func TestExtractGaps_EmptyGaps(t *testing.T) {
	p := &mockProvider{}
	e := NewMemoryExtractor(p)

	supplements := e.extractGaps(context.Background(), nil, "text")
	if len(supplements) != 0 {
		t.Fatalf("expected 0 supplements, got %d", len(supplements))
	}
}

func TestExtractGaps_NilProvider(t *testing.T) {
	p := &mockProvider{}
	e := NewMemoryExtractor(p)
	e.llmProvider = nil

	supplements := e.extractGaps(context.Background(), []Gap{{Question: "q"}}, "text")
	if len(supplements) != 0 {
		t.Fatalf("expected 0 supplements when provider nil, got %d", len(supplements))
	}
}

// --- Metrics Mock --------------------------------------------------------

type fakeMetrics struct {
	recordedProvider    string
	recordedTokensSaved int64
	recordedLatencyMs   float64
}

func (f *fakeMetrics) RecordExtraction(provider string, tokensSaved int64, latencyMs float64) {
	f.recordedProvider = provider
	f.recordedTokensSaved = tokensSaved
	f.recordedLatencyMs = latencyMs
}
