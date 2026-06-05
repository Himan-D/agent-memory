package extractor

import (
	"context"
	"strings"
	"testing"

	"agent-memory/internal/llm"
)

// mockProvider implements llm.Provider for offline tests
type mockProvider struct{}

func (m *mockProvider) Name() llm.ProviderType { return llm.ProviderLocal }

func (m *mockProvider) Complete(ctx context.Context, req *llm.CompletionRequest) (*llm.CompletionResponse, error) {
	content := ""
	if len(req.Messages) == 0 {
		return &llm.CompletionResponse{Content: content, Model: req.Model, Provider: llm.ProviderLocal}, nil
	}

	msg := strings.ToLower(req.Messages[len(req.Messages)-1].Content)
	switch {
	case strings.Contains(msg, "output format") || strings.Contains(msg, "compression target") || strings.Contains(msg, "compress this memory"):
		// Return a simple TOON triplet
		content = "{subject: user, action: likes, context: ml}\n{subject: user, action: uses, context: transformers}"
	case strings.Contains(msg, "generate self-questions"):
		content = `{"questions": ["What preferences are expressed?","What entities are mentioned?"]}`
	case strings.Contains(msg, "answer based on the memory"):
		content = "User likes neural networks and transformers"
	case strings.Contains(msg, "verify these answers") || strings.Contains(msg, "verify these facts"):
		// Return JSON verifying facts
		content = `[{"fact":"{subject: user, action: likes, context: ml}","verified":true,"confidence":0.95},{"fact":"{subject: user, action: uses, context: transformers}","verified":true,"confidence":0.9}]`
	case strings.Contains(msg, "identify missing information gaps"):
		content = `{"gaps": []}`
	default:
		content = `{"fact":"fallback summary","confidence":0.7}`
	}

	return &llm.CompletionResponse{Content: content, Model: req.Model, Provider: llm.ProviderLocal}, nil
}

func (m *mockProvider) Embed(ctx context.Context, req *llm.EmbeddingRequest) (*llm.EmbeddingResponse, error) {
	// Return a dummy embedding
	vec := make([]float32, 8)
	for i := range vec {
		vec[i] = 0.1
	}
	return &llm.EmbeddingResponse{Embedding: vec, Model: req.Model, Provider: llm.ProviderLocal}, nil
}

func (m *mockProvider) Rerank(ctx context.Context, req *llm.RerankRequest) (*llm.RerankResponse, error) {
	// Return trivial rerank (preserve order)
	res := llm.RerankResponse{Model: req.Model, Provider: llm.ProviderLocal}
	for i, doc := range req.Documents {
		res.Results = append(res.Results, llm.RerankResult{Index: i, Document: doc, Score: 1.0 - float64(i)*0.1})
	}
	return &res, nil
}

func loadSample(t *testing.T) string {
	// Use a representative long memory inline to avoid file IO
	return `On 2026-05-26 the team discussed migration to Neo4j and Qdrant for vector and graph storage. The proposed architecture included a ProMem compression pipeline, async workers, and a tiered memory router. Key decisions: use gRPC for internal comms, ensure privacy filtering, and add an observability /compression/stats endpoint.`
}

func BenchmarkMemoryExtractor_Extract(b *testing.B) {
	provider := &mockProvider{}
	extractor := NewMemoryExtractor(provider)
	ctx := context.Background()
	memory := loadSample(nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := extractor.Extract(ctx, memory)
		if err != nil {
			b.Fatalf("extract failed: %v", err)
		}
	}
}

func TestMockProviderQuick(t *testing.T) {
	provider := &mockProvider{}
	ctx := context.Background()
	req := &llm.CompletionRequest{Model: "test", Messages: []llm.Message{{Role: "user", Content: "Generate self-questions"}}}
	res, err := provider.Complete(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res.Content, "questions") {
		t.Fatalf("unexpected mock response: %s", res.Content)
	}
}
