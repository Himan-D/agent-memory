package self_improve

import (
	"context"
	"fmt"
	"time"

	"agent-memory/internal/memory/types"
)

// ============================================================
// ServiceFeedbackCollector adapts the memory.GraphStore to
// satisfy the FeedbackCollector interface required by SelfImprover.
// ============================================================

// GraphFeedbackStore is the subset of neo4j.Client methods used here.
// Keeping it narrow avoids an import cycle between memory and self_improve.
type GraphFeedbackStore interface {
	GetFeedbackByMemory(ctx context.Context, memoryID string) ([]*types.Feedback, error)
}

// ServiceFeedbackCollector implements FeedbackCollector via the graph store.
type ServiceFeedbackCollector struct {
	store GraphFeedbackStore
}

// NewServiceFeedbackCollector creates a FeedbackCollector backed by the graph store.
func NewServiceFeedbackCollector(store GraphFeedbackStore) *ServiceFeedbackCollector {
	return &ServiceFeedbackCollector{store: store}
}

func (c *ServiceFeedbackCollector) GetAllFeedback(ctx context.Context, memoryID string) ([]*types.Feedback, error) {
	return c.store.GetFeedbackByMemory(ctx, memoryID)
}

func (c *ServiceFeedbackCollector) GetPositiveFeedback(ctx context.Context, memoryID string) ([]*types.Feedback, error) {
	all, err := c.store.GetFeedbackByMemory(ctx, memoryID)
	if err != nil {
		return nil, err
	}
	var out []*types.Feedback
	for _, fb := range all {
		if fb.Type == types.FeedbackPositive {
			out = append(out, fb)
		}
	}
	return out, nil
}

func (c *ServiceFeedbackCollector) GetNegativeFeedback(ctx context.Context, memoryID string) ([]*types.Feedback, error) {
	all, err := c.store.GetFeedbackByMemory(ctx, memoryID)
	if err != nil {
		return nil, err
	}
	var out []*types.Feedback
	for _, fb := range all {
		if fb.Type == types.FeedbackNegative || fb.Type == types.FeedbackVeryNegative {
			out = append(out, fb)
		}
	}
	return out, nil
}

// ============================================================
// InMemoryTuningStore is a lightweight in-process TuningStore
// that persists tuning events, synonyms, and delegates importance
// / embedding updates to the graph store.
// ============================================================

// GraphTuningOps is the subset of graph-store operations needed by the tuning store.
type GraphTuningOps interface {
	GetMemory(id string) (*types.Memory, error)
	UpdateMemory(mem *types.Memory) error
}

// InMemoryTuningStore satisfies TuningStore. Synonyms and tuning history
// are kept in memory for fast access; importance updates write through to
// the real graph store.
type InMemoryTuningStore struct {
	graph    GraphTuningOps
	synonyms map[string]map[string][]string // memoryID → word → []synonyms
	history  map[string][]TuningEvent       // memoryID → events
}

// NewInMemoryTuningStore creates a tuning store that writes importance updates
// through to the provided graph ops implementation (normally *neo4j.Client).
func NewInMemoryTuningStore(graph GraphTuningOps) *InMemoryTuningStore {
	return &InMemoryTuningStore{
		graph:    graph,
		synonyms: make(map[string]map[string][]string),
		history:  make(map[string][]TuningEvent),
	}
}

func (s *InMemoryTuningStore) UpdateMemoryImportance(ctx context.Context, memoryID string, importance types.ImportanceLevel) error {
	if s.graph == nil {
		return nil
	}
	mem, err := s.graph.GetMemory(memoryID)
	if err != nil {
		return fmt.Errorf("tuning: get memory %s: %w", memoryID, err)
	}
	if mem == nil {
		return fmt.Errorf("tuning: memory %s not found", memoryID)
	}
	mem.Importance = importance
	mem.UpdatedAt = time.Now()
	return s.graph.UpdateMemory(mem)
}

func (s *InMemoryTuningStore) UpdateMemoryEmbedding(_ context.Context, _ string, _ string) error {
	// Re-embedding requires an external embedder. The caller (tuneNegativeMemory)
	// uses comment text as an additional synonym hint rather than a full re-embed.
	// Full re-embedding is handled asynchronously by the compression pipeline.
	// This is a no-op here; the content update is recorded as a TuningEvent.
	return nil
}

func (s *InMemoryTuningStore) AddSynonym(_ context.Context, memoryID, word, synonym string) error {
	if s.synonyms[memoryID] == nil {
		s.synonyms[memoryID] = make(map[string][]string)
	}
	s.synonyms[memoryID][word] = append(s.synonyms[memoryID][word], synonym)
	return nil
}

func (s *InMemoryTuningStore) GetSynonyms(_ context.Context, memoryID, word string) ([]string, error) {
	if m, ok := s.synonyms[memoryID]; ok {
		return m[word], nil
	}
	return nil, nil
}

func (s *InMemoryTuningStore) RecordTuningEvent(_ context.Context, event *TuningEvent) error {
	if event.ID == "" {
		event.ID = fmt.Sprintf("%s-%d", event.MemoryID, time.Now().UnixNano())
	}
	s.history[event.MemoryID] = append(s.history[event.MemoryID], *event)
	return nil
}

func (s *InMemoryTuningStore) GetTuningHistory(_ context.Context, memoryID string) ([]TuningEvent, error) {
	return s.history[memoryID], nil
}
