package reranker

import (
	"context"
	"testing"

	"agent-memory/internal/config"
	"agent-memory/internal/llm"
	"agent-memory/internal/memory/types"
)

func TestDisabledReranker(t *testing.T) {
	d := &disabledReranker{}
	if d.Name() != "disabled" {
		t.Errorf("expected name disabled, got %s", d.Name())
	}

	results := []types.MemoryResult{
		{Text: "a"}, {Text: "b"}, {Text: "c"},
	}

	got, err := d.Rerank(context.Background(), "test", results, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 3 {
		t.Errorf("expected 3 results, got %d", len(got))
	}

	got, err = d.Rerank(context.Background(), "test", results, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("expected 2 results, got %d", len(got))
	}

	if err := d.Close(); err != nil {
		t.Errorf("unexpected close error: %v", err)
	}
}

func TestDisabledReranker_Empty(t *testing.T) {
	d := &disabledReranker{}
	got, err := d.Rerank(context.Background(), "test", nil, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}

func TestNewProvider_Disabled(t *testing.T) {
	cfg := config.RerankerConfig{Provider: "disabled"}
	p, err := NewProvider(cfg, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Name() != "disabled" {
		t.Errorf("expected disabled, got %s", p.Name())
	}
}

func TestNewProvider_Empty(t *testing.T) {
	cfg := config.RerankerConfig{Provider: ""}
	p, err := NewProvider(cfg, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Name() != "disabled" {
		t.Errorf("expected disabled, got %s", p.Name())
	}
}

func TestNewProvider_Unknown(t *testing.T) {
	cfg := config.RerankerConfig{Provider: "unknown"}
	_, err := NewProvider(cfg, nil)
	if err == nil {
		t.Fatal("expected error for unknown provider")
	}
}

func TestNewProvider_CohereMissingKey(t *testing.T) {
	cfg := config.RerankerConfig{Provider: "cohere"}
	_, err := NewProvider(cfg, nil)
	if err == nil {
		t.Fatal("expected error for missing cohere API key")
	}
}

func TestNewProvider_CohereEmptyKey(t *testing.T) {
	cfg := config.RerankerConfig{Provider: "cohere", APIKey: ""}
	_, err := NewProvider(cfg, nil)
	if err == nil {
		t.Fatal("expected error for empty cohere API key")
	}
}

func TestNewProvider_Cohere(t *testing.T) {
	cfg := config.RerankerConfig{
		Provider: "cohere",
		APIKey:   "test-key-123",
		BaseURL:  "https://test.cohere.ai",
		Model:    "test-model",
	}
	p, err := NewProvider(cfg, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Name() != "cohere" {
		t.Errorf("expected cohere, got %s", p.Name())
	}

	cohere, ok := p.(*CohereReranker)
	if !ok {
		t.Fatal("expected *CohereReranker type")
	}
	if cohere.apiKey != "test-key-123" {
		t.Errorf("expected apiKey test-key-123, got %s", cohere.apiKey)
	}
	if cohere.baseURL != "https://test.cohere.ai" {
		t.Errorf("expected baseURL https://test.cohere.ai, got %s", cohere.baseURL)
	}
	if cohere.model != "test-model" {
		t.Errorf("expected model test-model, got %s", cohere.model)
	}
}

func TestNewProvider_CohereDefaults(t *testing.T) {
	cfg := config.RerankerConfig{
		Provider: "cohere",
		APIKey:   "test-key",
	}
	p, err := NewProvider(cfg, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cohere := p.(*CohereReranker)
	if cohere.baseURL != "https://api.cohere.ai" {
		t.Errorf("expected default baseURL, got %s", cohere.baseURL)
	}
	if cohere.model != "cohere/rerank-english-v2.0" {
		t.Errorf("expected default model, got %s", cohere.model)
	}
}

func TestNewProvider_LLMNoClient(t *testing.T) {
	cfg := config.RerankerConfig{Provider: "llm"}
	_, err := NewProvider(cfg, nil)
	if err == nil {
		t.Fatal("expected error for LLM provider without client")
	}
}

func TestNewProvider_LLM(t *testing.T) {
	cfg := config.RerankerConfig{Provider: "llm"}
	mock := &mockLLMProvider{}
	p, err := NewProvider(cfg, mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Name() != "llm" {
		t.Errorf("expected llm, got %s", p.Name())
	}
}

type mockLLMProvider struct{}

func (m *mockLLMProvider) Name() llm.ProviderType { return "mock" }
func (m *mockLLMProvider) Complete(ctx context.Context, req *llm.CompletionRequest) (*llm.CompletionResponse, error) {
	return &llm.CompletionResponse{Content: "mock"}, nil
}
func (m *mockLLMProvider) Embed(ctx context.Context, req *llm.EmbeddingRequest) (*llm.EmbeddingResponse, error) {
	return &llm.EmbeddingResponse{Embedding: []float32{0.1}}, nil
}
func (m *mockLLMProvider) Rerank(ctx context.Context, req *llm.RerankRequest) (*llm.RerankResponse, error) {
	return &llm.RerankResponse{}, nil
}

func TestRerankerConfig_ToProviderConfig(t *testing.T) {
	tests := []struct {
		name     string
		cfg      config.RerankerConfig
		wantName string
		wantErr  bool
	}{
		{"disabled", config.RerankerConfig{Provider: "disabled"}, "disabled", false},
		{"empty", config.RerankerConfig{Provider: ""}, "disabled", false},
		{"cohere valid", config.RerankerConfig{Provider: "cohere", APIKey: "key"}, "cohere", false},
		{"cohere no key", config.RerankerConfig{Provider: "cohere"}, "", true},
		{"unknown", config.RerankerConfig{Provider: "nope"}, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := NewProvider(tt.cfg, nil)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if p.Name() != tt.wantName {
				t.Errorf("expected %s, got %s", tt.wantName, p.Name())
			}
		})
	}
}
