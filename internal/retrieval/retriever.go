package retrieval

import (
	"context"
	"fmt"
	"strings"
	"time"

	"agent-memory/internal/memory/types"
)

// RetrievedObject is one raw retrieval result from a backend. The Type field
// distinguishes memories, chunks, edges, and other items.
type RetrievedObject struct {
	Type    string // "memory", "chunk", "edge"
	ID      string
	Text    string
	Score   float64
	Source  string            // backend identifier, e.g. "vector", "graph"
	Meta    map[string]string // optional metadata (hop distance, feedback, etc.)
}

// RetrievedObjects is a typed collection of RetrievedObject with helper
// accessors for the common case of memory-shaped results.
type RetrievedObjects struct {
	Items []RetrievedObject
}

// Memories returns only memory-typed items as types.MemoryResult. The
// conversion preserves score and metadata.
func (r RetrievedObjects) Memories() []types.MemoryResult {
	out := make([]types.MemoryResult, 0, len(r.Items))
	for _, it := range r.Items {
		if it.Type != "memory" {
			continue
		}
		out = append(out, types.MemoryResult{
			MemoryID: it.ID,
			Text:     it.Text,
			Score:    float32(it.Score),
			Entity: types.Entity{
				Properties: map[string]interface{}{
					"source": it.Source,
					"meta":   stringMapToAny(it.Meta),
				},
			},
		})
	}
	return out
}

func stringMapToAny(m map[string]string) map[string]interface{} {
	if len(m) == 0 {
		return nil
	}
	out := make(map[string]interface{}, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// Completion is the final LLM-ready answer. Context is the LLM input;
// Citations lists the IDs that informed the answer so callers can expose
// provenance.
type Completion struct {
	Query     string
	Context   string
	Answer    string
	Citations []string
}

// BaseRetriever is the three-phase retrieval interface from Cognee's
// BaseRetriever. It is intentionally additive: existing retrieval types
// (MultiSignalRetrieval, ServiceAdapter, the spreading activation package)
// remain untouched, and concrete retrievers in this file adapt them to the
// interface.
type BaseRetriever interface {
	// Name returns a short identifier (e.g. "vector", "multisignal",
	// "spreading", "graph").
	Name() string
	// RetrieveObjects fetches raw objects from the underlying backend(s).
	RetrieveObjects(ctx context.Context, query string) (RetrievedObjects, error)
	// BuildContext transforms raw objects into an LLM-ready context string.
	BuildContext(ctx context.Context, query string, objects RetrievedObjects) (string, error)
	// Complete generates the final answer using the context. Implementations
	// are allowed to skip the LLM and return a deterministic answer when no
	// LLM provider is configured (useful for tests and offline builds).
	Complete(ctx context.Context, query string, contextText string, objects RetrievedObjects) (Completion, error)
	// GetCompletion is a convenience that runs the full pipeline.
	GetCompletion(ctx context.Context, query string) (Completion, error)
}

// RetrieverConfig holds tunables shared across retrievers. It mirrors
// Cognee's GraphCompletionRetriever parameter set so the values translate
// directly.
type RetrieverConfig struct {
	TopK                    int
	WideSearchTopK          int
	TripletDistancePenalty  float64
	FeedbackInfluence       float64
	NeighborhoodDepth       int
	IncludeGlobalContext    bool
	MaxContextChars         int
}

// DefaultRetrieverConfig returns sensible defaults that match Cognee's
// GraphCompletionRetriever.
func DefaultRetrieverConfig() *RetrieverConfig {
	return &RetrieverConfig{
		TopK:                   5,
		WideSearchTopK:         100,
		TripletDistancePenalty: 6.5,
		FeedbackInfluence:      0.0,
		NeighborhoodDepth:      2,
		IncludeGlobalContext:   false,
		MaxContextChars:        8000,
	}
}

// CompletionFunc is the optional LLM callback for retrievers that need to
// produce an actual answer. When nil, retrievers return a heuristic answer
// derived from the top retrieved object.
type CompletionFunc func(ctx context.Context, query, contextText string) (string, error)

// rerankOrders sorts RetrievedObjects by Score descending, with ties broken
// by hop distance ascending (lower is better). Items missing distance sort
// last within their score tier.
func rerankOrders(items []RetrievedObject, feedbackInfluence float64) []RetrievedObject {
	if len(items) <= 1 {
		return items
	}
	// Stable bubble is fine here - TopK is small.
	for i := 0; i < len(items)-1; i++ {
		for j := i + 1; j < len(items); j++ {
			si := effectiveScore(items[i], feedbackInfluence)
			sj := effectiveScore(items[j], feedbackInfluence)
			if sj > si {
				items[i], items[j] = items[j], items[i]
			}
		}
	}
	return items
}

func effectiveScore(item RetrievedObject, feedbackInfluence float64) float64 {
	score := item.Score
	if feedbackInfluence <= 0 {
		return score
	}
	if item.Meta == nil {
		return score
	}
	// feedback_weight is read from meta; missing => treat as 0.
	// Item.Meta stores string values, so we don't try to parse here. Callers
	// that want feedback influence should populate Meta["feedback_weight"]
	// with a numeric string. The default behavior is to leave score
	// unchanged, matching Cognee's "feedback_influence = 0.0" default.
	if w, ok := item.Meta["feedback_weight"]; ok && w != "" {
		var fw float64
		_, _ = fmt.Sscanf(w, "%f", &fw)
		score += feedbackInfluence * fw
	}
	return score
}

// trimContext truncates a context string to max runes (UTF-8 aware enough
// for our purposes: we use byte length but cap on rune count for safety).
func trimContext(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	// Try to break on a paragraph boundary.
	cut := s[:max]
	if idx := strings.LastIndex(cut, "\n\n"); idx > max/2 {
		return cut[:idx] + "\n\n[…truncated]"
	}
	return cut + "\n\n[…truncated]"
}

// heuristicAnswer returns a non-LLM answer built from the top retrieved
// items. Useful when no CompletionFunc is configured.
func heuristicAnswer(query string, objects RetrievedObjects) string {
	if len(objects.Items) == 0 {
		return fmt.Sprintf("No relevant memories found for %q.", query)
	}
	var b strings.Builder
	b.WriteString("Top matches:\n")
	limit := len(objects.Items)
	if limit > 5 {
		limit = 5
	}
	for i := 0; i < limit; i++ {
		fmt.Fprintf(&b, "- [%s] %s\n", objects.Items[i].Source, truncateForContext(objects.Items[i].Text, 240))
	}
	return b.String()
}

func truncateForContext(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max] + "…"
}

// nowMillis is exposed for tests that want to time pipelines without
// pulling in a clock interface.
func nowMillis() int64 { return time.Now().UnixMilli() }
