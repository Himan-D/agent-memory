package extractor

import (
	"context"
	"strings"
	"testing"

	"agent-memory/internal/llm"
	"agent-memory/internal/memory/types"
)

type mockExtractorProvider struct {
	responses map[string]string
	callCount int
}

func (m *mockExtractorProvider) Name() llm.ProviderType { return llm.ProviderLocal }

func (m *mockExtractorProvider) Complete(ctx context.Context, req *llm.CompletionRequest) (*llm.CompletionResponse, error) {
	m.callCount++
	msg := strings.ToLower(req.Messages[len(req.Messages)-1].Content)
	for key, resp := range m.responses {
		if strings.Contains(msg, key) {
			return &llm.CompletionResponse{Content: resp, Model: req.Model, Provider: llm.ProviderLocal}, nil
		}
	}
	return &llm.CompletionResponse{Content: `{"fact":"fallback","confidence":0.5}`, Model: req.Model, Provider: llm.ProviderLocal}, nil
}

func (m *mockExtractorProvider) Embed(ctx context.Context, req *llm.EmbeddingRequest) (*llm.EmbeddingResponse, error) {
	vec := make([]float32, 8)
	for i := range vec {
		vec[i] = 0.1
	}
	return &llm.EmbeddingResponse{Embedding: vec, Model: req.Model, Provider: llm.ProviderLocal}, nil
}

func (m *mockExtractorProvider) Rerank(ctx context.Context, req *llm.RerankRequest) (*llm.RerankResponse, error) {
	res := llm.RerankResponse{Model: req.Model, Provider: llm.ProviderLocal}
	for i, doc := range req.Documents {
		res.Results = append(res.Results, llm.RerankResult{Index: i, Document: doc, Score: 1.0 - float64(i)*0.1})
	}
	return &res, nil
}

func TestNewMemoryExtractor_Defaults(t *testing.T) {
	provider := &mockExtractorProvider{responses: map[string]string{}}
	ext := NewMemoryExtractor(provider)

	if ext.maxIterations != 2 {
		t.Errorf("expected maxIterations=2, got %d", ext.maxIterations)
	}
	if ext.verifyThreshold != 0.85 {
		t.Errorf("expected verifyThreshold=0.85, got %f", ext.verifyThreshold)
	}
	if ext.fastModel != "gpt-4o-mini" {
		t.Errorf("expected fastModel=gpt-4o-mini, got %s", ext.fastModel)
	}
	if ext.verifyModel != "claude-3-5-sonnet" {
		t.Errorf("expected verifyModel=claude-3-5-sonnet, got %s", ext.verifyModel)
	}
}

func TestExtract_IterationLoop(t *testing.T) {
	provider := &mockExtractorProvider{
		responses: map[string]string{
			"compress this memory": "{subject: user, action: likes, context: go}\n{subject: user, action: uses, context: docker}",
			"generate self-questions": `{"questions": ["What tech does user like?"]}`,
			"answer based on the memory": "User likes Go and uses Docker",
			"verify these answers": `[{"fact":"{subject: user, action: likes, context: go}","confidence":0.95,"verified":true}]`,
			"identify missing information gaps": `{"gaps": [{"question": "What version of Go?"}]}`,
			"extract the missing information": "User uses Go 1.22",
		},
	}

	ext := NewMemoryExtractor(provider)
	ext.maxIterations = 3

	result, err := ext.Extract(context.Background(), "The user develops in Go and deploys with Docker containers")
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result.Facts) == 0 {
		t.Error("expected facts to be extracted")
	}
	if len(result.Gaps) == 0 {
		t.Error("expected gaps to be detected with iterations > 1")
	}
}

func TestExtract_NoGaps_BreaksEarly(t *testing.T) {
	provider := &mockExtractorProvider{
		responses: map[string]string{
			"compress this memory": "{subject: user, action: likes, context: python}",
			"generate self-questions": `{"questions": ["What does the user like?"]}`,
			"answer based on the memory": "The user likes Python",
			"verify these answers": `[{"fact":"{subject: user, action: likes, context: python}","confidence":0.9,"verified":true}]`,
			"identify missing information gaps": `{"gaps": []}`,
		},
	}

	ext := NewMemoryExtractor(provider)
	ext.maxIterations = 5

	result, err := ext.Extract(context.Background(), "User likes Python")
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result.Gaps) != 0 {
		t.Errorf("expected no gaps when none detected, got %d", len(result.Gaps))
	}
}

func TestExtract_AnswersAccumulate(t *testing.T) {
	callCount := 0
	provider := &mockExtractorProvider{
		responses: map[string]string{
			"compress this memory": "{subject: user, action: knows, context: rust}",
			"generate self-questions": `{"questions": ["What language?", "What framework?"]}`,
			"answer based on the memory": func() string {
				callCount++
				return "The user knows Rust and Actix"
			}(),
			"verify these answers": `[{"fact":"{subject: user, action: knows, context: rust}","confidence":0.9,"verified":true},{"fact":"{subject: user, action: uses, context: actix}","confidence":0.85,"verified":true}]`,
			"identify missing information gaps": `{"gaps": []}`,
		},
	}

	ext := NewMemoryExtractor(provider)
	result, err := ext.Extract(context.Background(), "User programs in Rust using the Actix framework")
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	_ = callCount
	_ = result

	if len(result.VerifiedFacts) == 0 {
		t.Error("expected verified facts to be produced")
	}
}

func TestDeduplicateFacts(t *testing.T) {
	facts := []types.Fact{
		{Fact: "User likes Go", Confidence: 0.9},
		{Fact: "user likes go", Confidence: 0.85},
		{Fact: "User uses Docker", Confidence: 0.8},
		{Fact: "  User uses Docker  ", Confidence: 0.75},
		{Fact: "User prefers Vim", Confidence: 0.7},
	}

	deduped := deduplicateFacts(facts)

	if len(deduped) != 3 {
		t.Errorf("expected 3 unique facts, got %d", len(deduped))
	}

	seen := make(map[string]bool)
	for _, f := range deduped {
		key := strings.ToLower(strings.TrimSpace(f.Fact))
		if seen[key] {
			t.Errorf("duplicate fact found: %s", f.Fact)
		}
		seen[key] = true
	}
}

func TestDeduplicateFacts_Empty(t *testing.T) {
	result := deduplicateFacts(nil)
	if len(result) != 0 {
		t.Errorf("expected 0 facts for nil input, got %d", len(result))
	}

	result = deduplicateFacts([]types.Fact{})
	if len(result) != 0 {
		t.Errorf("expected 0 facts for empty input, got %d", len(result))
	}
}