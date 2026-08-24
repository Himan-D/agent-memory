package memory

import (
	"context"
	"strings"
	"testing"

	"agent-memory/internal/llm"
	"agent-memory/internal/memory/types"
)

type captureLLM struct {
	last       *llm.CompletionRequest
	extractSys string
	calls      int
}

func (c *captureLLM) Complete(ctx context.Context, req *llm.CompletionRequest) (*llm.CompletionResponse, error) {
	c.last = req
	c.calls++
	sys := ""
	if req != nil && len(req.Messages) > 0 {
		sys = req.Messages[0].Content
	}
	switch {
	case strings.Contains(sys, "importance classifier"):
		return &llm.CompletionResponse{Content: `{"store":true,"importance":8,"reason":"test","categories":["fact"]}`}, nil
	case strings.Contains(sys, "memory extraction system"):
		c.extractSys = sys
		return &llm.CompletionResponse{Content: `[{"fact":"user likes dark mode","category":"preference","importance":"high"}]`}, nil
	case strings.Contains(sys, "categor"):
		return &llm.CompletionResponse{Content: `["preference","work"]`}, nil
	default:
		return &llm.CompletionResponse{Content: `["preference"]`}, nil
	}
}

func TestProcessContentWithInstructionsInjectsIntoSystemPrompt(t *testing.T) {
	cap := &captureLLM{}
	p := NewMemoryProcessor(cap)
	p.SetConfig(&Config{
		Enabled:             true,
		AutoExtractFacts:    true,
		AutoExtractEntities: false,
		DefaultImportance:   ImportanceMedium,
	})

	custom := "Always tag facts with project=alpha and prefer short imperative phrasing."
	_, err := p.ProcessContentWithInstructions(context.Background(), "I prefer dark mode in the IDE", "user-1", MemoryTypeUser, custom)
	if err != nil {
		t.Fatalf("ProcessContentWithInstructions: %v", err)
	}
	if cap.extractSys == "" {
		t.Fatal("expected extractFacts LLM call")
	}
	if !strings.Contains(cap.extractSys, custom) {
		t.Fatalf("system prompt missing custom instructions:\n%s", cap.extractSys)
	}
	if !strings.Contains(cap.extractSys, "Additional instructions from the caller") {
		t.Fatalf("system prompt missing custom-instruction framing:\n%s", cap.extractSys)
	}
}

func TestProcessContentWithoutInstructionsKeepsBasePrompt(t *testing.T) {
	cap := &captureLLM{}
	p := NewMemoryProcessor(cap)
	p.SetConfig(&Config{
		Enabled:             true,
		AutoExtractFacts:    true,
		AutoExtractEntities: false,
		DefaultImportance:   ImportanceMedium,
	})

	_, err := p.ProcessContent(context.Background(), "I like tea", "user-1", MemoryTypeUser)
	if err != nil {
		t.Fatalf("ProcessContent: %v", err)
	}
	if cap.extractSys == "" {
		t.Fatal("expected extractFacts LLM call")
	}
	if strings.Contains(cap.extractSys, "Additional instructions from the caller") {
		t.Fatalf("unexpected custom-instruction framing without custom text:\n%s", cap.extractSys)
	}
}

func TestResolveCustomInstructionsFromFieldAndMetadata(t *testing.T) {
	if got := resolveCustomInstructions(nil); got != "" {
		t.Fatalf("nil: got %q", got)
	}
	fromField := &types.Memory{CustomInstructions: " from-field "}
	if got := resolveCustomInstructions(fromField); got != "from-field" {
		t.Fatalf("field: got %q", got)
	}
	fromMeta := &types.Memory{Metadata: map[string]interface{}{"custom_instructions": "from-meta"}}
	if got := resolveCustomInstructions(fromMeta); got != "from-meta" {
		t.Fatalf("meta: got %q", got)
	}
	// Field wins over metadata
	both := &types.Memory{
		CustomInstructions: "field-wins",
		Metadata:           map[string]interface{}{"custom_instructions": "meta"},
	}
	if got := resolveCustomInstructions(both); got != "field-wins" {
		t.Fatalf("precedence: got %q", got)
	}
}

func TestMemorySkipProcessing(t *testing.T) {
	if memorySkipProcessing(nil) {
		t.Fatal("nil should not skip")
	}
	if memorySkipProcessing(&types.Memory{}) {
		t.Fatal("empty should not skip")
	}
	if !memorySkipProcessing(&types.Memory{Metadata: map[string]interface{}{"skip_processing": true}}) {
		t.Fatal("bool true should skip")
	}
	if !memorySkipProcessing(&types.Memory{Metadata: map[string]interface{}{"skip_processing": "true"}}) {
		t.Fatal("string true should skip")
	}
}
