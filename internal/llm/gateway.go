package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync/atomic"
	"time"
)

// Gateway centralizes structured-output generation across LLM providers. It
// mirrors Cognee's LLMGateway.acreate_structured_output and is intentionally
// additive: existing callers of Provider.Complete are unaffected.
//
// Responsibilities:
//   - Prepend an optional "agent memory" context block to the user message.
//   - Issue a CompletionRequest and parse the response as JSON into the
//     caller's response model.
//   - Record token usage via a pluggable UsageRecorder (typically
//     internal/metrics).
//
// To plug in a different structured-output framework (e.g. BAML or JSON-mode
// per provider), set Framework to a non-empty value before calling
// CreateStructuredOutput. The default "native" framework simply appends
// "Respond with JSON only" to the system prompt and parses the response.
type Gateway struct {
	provider Provider
	// MemoryContext is an optional agent memory fragment prepended to user
	// messages. It is set via WithMemoryContext, typically from the session
	// manager.
	MemoryContext atomic.Value // string
	// Framework selects the structured-output strategy. Empty string == "native".
	Framework string
	// UsageRecorder is invoked after each successful call. May be nil.
	UsageRecorder UsageRecorder
}

// UsageRecorder is the subset of internal/metrics needed by the gateway. It is
// defined here to avoid an import cycle on the metrics package.
type UsageRecorder interface {
	RecordLLMUsage(provider ProviderType, promptTokens, completionTokens int, latency time.Duration)
}

// NewGateway wraps a Provider with the structured-output gateway. The provider
// must not be nil.
func NewGateway(p Provider) *Gateway {
	g := &Gateway{provider: p, Framework: "native"}
	g.MemoryContext.Store("")
	return g
}

// Provider returns the underlying provider. Useful for callers that need to
// fall back to raw completion.
func (g *Gateway) Provider() Provider { return g.provider }

// WithMemoryContext sets the agent memory context that will be prepended to
// every user message. Passing an empty string clears the context.
func (g *Gateway) WithMemoryContext(ctx string) {
	g.MemoryContext.Store(ctx)
}

// StructuredRequest is the input to CreateStructuredOutput.
type StructuredRequest struct {
	SystemPrompt string
	UserInput    string
	// ResponseModel is the zero-valued target struct into which the JSON
	// response will be unmarshaled.
	ResponseModel any
	// Temperature and MaxTokens override provider defaults when non-zero.
	Temperature float64
	MaxTokens   int
}

// CreateStructuredOutput runs a completion and unmarshals the result into
// req.ResponseModel. It returns the raw completion text and any error.
//
// The response model pointer must be non-nil. The JSON content is extracted
// from the completion text by stripping any leading prose and locating the
// outermost JSON object or array.
func (g *Gateway) CreateStructuredOutput(ctx context.Context, req StructuredRequest) (string, error) {
	if g.provider == nil {
		return "", fmt.Errorf("llm gateway: provider is nil")
	}
	if req.ResponseModel == nil {
		return "", fmt.Errorf("llm gateway: response model is nil")
	}

	system := req.SystemPrompt
	if g.Framework == "" {
		g.Framework = "native"
	}
	if g.Framework == "native" && !strings.Contains(strings.ToLower(system), "json") {
		system = strings.TrimRight(system, "\n") + "\n\nRespond with valid JSON only. No prose, no markdown fences."
	}

	user := req.UserInput
	if mc, _ := g.MemoryContext.Load().(string); mc != "" {
		user = "[Agent memory context]\n" + mc + "\n\n" + user
	}

	compl := &CompletionRequest{
		Messages: []Message{
			{Role: "system", Content: system},
			{Role: "user", Content: user},
		},
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
	}
	if compl.Temperature == 0 {
		compl.Temperature = 0.2 // lower temp by default for structured output
	}
	if compl.MaxTokens == 0 {
		compl.MaxTokens = 1024
	}

	start := time.Now()
	resp, err := g.provider.Complete(ctx, compl)
	if err != nil {
		return "", fmt.Errorf("llm gateway: complete: %w", err)
	}

	if g.UsageRecorder != nil {
		g.UsageRecorder.RecordLLMUsage(
			g.provider.Name(),
			estimateTokens(user+system),
			estimateTokens(resp.Content),
			time.Since(start),
		)
	}

	jsonText := extractJSON(resp.Content)
	if jsonText == "" {
		return resp.Content, fmt.Errorf("llm gateway: no JSON found in response")
	}
	if err := json.Unmarshal([]byte(jsonText), req.ResponseModel); err != nil {
		return resp.Content, fmt.Errorf("llm gateway: parse response: %w", err)
	}
	return resp.Content, nil
}

// extractJSON returns the first balanced top-level JSON object or array in s.
// It returns "" if no balanced JSON is found.
func extractJSON(s string) string {
	s = strings.TrimSpace(s)
	// Strip common markdown code fences.
	if strings.HasPrefix(s, "```") {
		if idx := strings.Index(s, "\n"); idx >= 0 {
			s = s[idx+1:]
		}
		if idx := strings.LastIndex(s, "```"); idx >= 0 {
			s = s[:idx]
		}
		s = strings.TrimSpace(s)
	}

	openIdx := -1
	openCh := byte(0)
	closeCh := byte(0)
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '{':
			openIdx = i
			openCh, closeCh = '{', '}'
			goto scan
		case '[':
			openIdx = i
			openCh, closeCh = '[', ']'
			goto scan
		}
	}
	return ""
scan:
	depth := 0
	inStr := false
	escape := false
	for i := openIdx; i < len(s); i++ {
		c := s[i]
		if escape {
			escape = false
			continue
		}
		if c == '\\' && inStr {
			escape = true
			continue
		}
		if c == '"' {
			inStr = !inStr
			continue
		}
		if inStr {
			continue
		}
		switch c {
		case openCh:
			depth++
		case closeCh:
			depth--
			if depth == 0 {
				return s[openIdx : i+1]
			}
		}
	}
	return ""
}

// estimateTokens is a rough approximation: ~4 characters per token. It's good
// enough for usage tracking; callers needing exact counts should use the
// provider's response.
func estimateTokens(s string) int {
	if s == "" {
		return 0
	}
	return (len(s) + 3) / 4
}
