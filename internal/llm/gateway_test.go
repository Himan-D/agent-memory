package llm

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// fakeProvider is a deterministic Provider for tests.
type fakeProvider struct {
	provider ProviderType
	resp     string
	err      error
	calls    int
}

func (f *fakeProvider) Name() ProviderType { return f.provider }
func (f *fakeProvider) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	return &CompletionResponse{Content: f.resp, Provider: f.provider, Model: "fake"}, nil
}
func (f *fakeProvider) Embed(ctx context.Context, req *EmbeddingRequest) (*EmbeddingResponse, error) {
	return nil, errors.New("not implemented")
}
func (f *fakeProvider) Rerank(ctx context.Context, req *RerankRequest) (*RerankResponse, error) {
	return nil, errors.New("not implemented")
}

func TestExtractJSON_Object(t *testing.T) {
	in := "Here is your JSON:\n{\"a\": 1, \"b\": [1,2,3]}\nDone."
	got := extractJSON(in)
	want := `{"a": 1, "b": [1,2,3]}`
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestExtractJSON_CodeFence(t *testing.T) {
	in := "```json\n{\"x\": true}\n```"
	if got := extractJSON(in); got != `{"x": true}` {
		t.Fatalf("got %q", got)
	}
}

func TestExtractJSON_Array(t *testing.T) {
	in := "prefix [1, 2, 3] suffix"
	if got := extractJSON(in); got != "[1, 2, 3]" {
		t.Fatalf("got %q", got)
	}
}

func TestExtractJSON_NestedStrings(t *testing.T) {
	in := `noise {"k": "}"} more`
	if got := extractJSON(in); got != `{"k": "}"}` {
		t.Fatalf("got %q", got)
	}
}

func TestExtractJSON_NoJSON(t *testing.T) {
	if got := extractJSON("no json here"); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
}

func TestGateway_CreateStructuredOutput_Success(t *testing.T) {
	fp := &fakeProvider{
		provider: ProviderOpenAI,
		resp:     `{"name":"alice","score":0.9}`,
	}
	g := NewGateway(fp)

	var out struct {
		Name  string  `json:"name"`
		Score float64 `json:"score"`
	}
	raw, err := g.CreateStructuredOutput(context.Background(), StructuredRequest{
		SystemPrompt: "Extract entities.",
		UserInput:    "Alice is here.",
		ResponseModel: &out,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(raw, "alice") {
		t.Fatalf("unexpected raw output: %q", raw)
	}
	if out.Name != "alice" || out.Score != 0.9 {
		t.Fatalf("unexpected parse: %+v", out)
	}
	if fp.calls != 1 {
		t.Fatalf("expected 1 call, got %d", fp.calls)
	}
}

func TestGateway_CreateStructuredOutput_MemoryContext(t *testing.T) {
	fp := &fakeProvider{provider: ProviderOpenAI, resp: `{"ok":true}`}
	g := NewGateway(fp)
	g.WithMemoryContext("user likes dark mode")

	var out map[string]any
	if _, err := g.CreateStructuredOutput(context.Background(), StructuredRequest{
		UserInput:     "what is preferred?",
		ResponseModel: &out,
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fp.calls != 1 {
		t.Fatalf("expected 1 call, got %d", fp.calls)
	}
}

func TestGateway_CreateStructuredOutput_BadJSON(t *testing.T) {
	fp := &fakeProvider{provider: ProviderOpenAI, resp: "no JSON at all"}
	g := NewGateway(fp)
	var out map[string]any
	if _, err := g.CreateStructuredOutput(context.Background(), StructuredRequest{ResponseModel: &out}); err == nil {
		t.Fatal("expected error for missing JSON")
	}
}

func TestGateway_UsageRecorder(t *testing.T) {
	fp := &fakeProvider{provider: ProviderOpenAI, resp: `{"ok":true}`}
	g := NewGateway(fp)
	var gotProvider ProviderType
	var gotLatency time.Duration
	g.UsageRecorder = UsageRecorderFunc(func(p ProviderType, _, _ int, d time.Duration) {
		gotProvider = p
		gotLatency = d
	})
	var out map[string]any
	if _, err := g.CreateStructuredOutput(context.Background(), StructuredRequest{ResponseModel: &out}); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if gotProvider != ProviderOpenAI {
		t.Fatalf("got provider %v", gotProvider)
	}
	if gotLatency < 0 {
		t.Fatalf("latency should be non-negative, got %v", gotLatency)
	}
}

// UsageRecorderFunc adapts a function to the UsageRecorder interface.
type UsageRecorderFunc func(provider ProviderType, promptTokens, completionTokens int, latency time.Duration)

func (f UsageRecorderFunc) RecordLLMUsage(p ProviderType, pt, ct int, d time.Duration) {
	f(p, pt, ct, d)
}
