package self_improve

import (
	"context"
	"fmt"
	"testing"
	"time"

	"agent-memory/internal/memory/types"
)

// ─── mock helpers ─────────────────────────────────────────────────────────────

type mockFeedbackCollector struct {
	feedback []*types.Feedback
}

func (m *mockFeedbackCollector) GetAllFeedback(_ context.Context, _ string) ([]*types.Feedback, error) {
	return m.feedback, nil
}
func (m *mockFeedbackCollector) GetPositiveFeedback(_ context.Context, _ string) ([]*types.Feedback, error) {
	var out []*types.Feedback
	for _, fb := range m.feedback {
		if fb.Type == types.FeedbackPositive {
			out = append(out, fb)
		}
	}
	return out, nil
}
func (m *mockFeedbackCollector) GetNegativeFeedback(_ context.Context, _ string) ([]*types.Feedback, error) {
	var out []*types.Feedback
	for _, fb := range m.feedback {
		if fb.Type == types.FeedbackNegative || fb.Type == types.FeedbackVeryNegative {
			out = append(out, fb)
		}
	}
	return out, nil
}

type mockTuningStore struct {
	importanceUpdates map[string]types.ImportanceLevel
	synonyms          map[string]map[string][]string
	events            []TuningEvent
}

func newMockTuningStore() *mockTuningStore {
	return &mockTuningStore{
		importanceUpdates: make(map[string]types.ImportanceLevel),
		synonyms:          make(map[string]map[string][]string),
	}
}
func (m *mockTuningStore) UpdateMemoryImportance(_ context.Context, id string, imp types.ImportanceLevel) error {
	m.importanceUpdates[id] = imp
	return nil
}
func (m *mockTuningStore) UpdateMemoryEmbedding(_ context.Context, _ string, _ string) error { return nil }
func (m *mockTuningStore) AddSynonym(_ context.Context, memoryID, word, synonym string) error {
	if m.synonyms[memoryID] == nil {
		m.synonyms[memoryID] = make(map[string][]string)
	}
	m.synonyms[memoryID][word] = append(m.synonyms[memoryID][word], synonym)
	return nil
}
func (m *mockTuningStore) GetSynonyms(_ context.Context, memoryID, word string) ([]string, error) {
	if wm, ok := m.synonyms[memoryID]; ok {
		return wm[word], nil
	}
	return nil, nil
}
func (m *mockTuningStore) RecordTuningEvent(_ context.Context, event *TuningEvent) error {
	m.events = append(m.events, *event)
	return nil
}
func (m *mockTuningStore) GetTuningHistory(_ context.Context, memoryID string) ([]TuningEvent, error) {
	var out []TuningEvent
	for _, ev := range m.events {
		if ev.MemoryID == memoryID {
			out = append(out, ev)
		}
	}
	return out, nil
}

// ─── SelfImprovementEngine tests ─────────────────────────────────────────────

func TestLearnFromPositiveFeedback(t *testing.T) {
	engine := NewSelfImprovementEngine()
	ctx := context.Background()

	if err := engine.LearnFromPositiveFeedback(ctx, "mem-1", "test query"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	adj := engine.GetImportanceAdjustment("mem-1")
	if adj <= 0 {
		t.Errorf("expected positive importance adjustment, got %v", adj)
	}
}

func TestLearnFromNegativeFeedback(t *testing.T) {
	engine := NewSelfImprovementEngine()
	ctx := context.Background()

	if err := engine.LearnFromNegativeFeedback(ctx, "mem-2", "wrong answer", false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	adj := engine.GetImportanceAdjustment("mem-2")
	if adj >= 0 {
		t.Errorf("expected negative importance adjustment, got %v", adj)
	}
}

func TestLearnFromNegativeFeedback_Critical(t *testing.T) {
	engine := NewSelfImprovementEngine()
	ctx := context.Background()

	if err := engine.LearnFromNegativeFeedback(ctx, "mem-3", "critical failure", true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	reviews := engine.GetMemoriesForReview(10)
	found := false
	for _, rv := range reviews {
		if rv.MemoryID == "mem-3" {
			found = true
			break
		}
	}
	if !found {
		t.Error("critical negative feedback should flag memory for review")
	}
}

func TestImportanceAdjustmentCapped(t *testing.T) {
	engine := NewSelfImprovementEngine()
	ctx := context.Background()

	for i := 0; i < 20; i++ {
		_ = engine.LearnFromPositiveFeedback(ctx, "mem-cap", "query")
	}

	adj := engine.GetImportanceAdjustment("mem-cap")
	if adj > 1.0 {
		t.Errorf("importance adjustment exceeded 1.0: got %v", adj)
	}
}

func TestExpandQuery(t *testing.T) {
	engine := NewSelfImprovementEngine()
	ctx := context.Background()

	_ = engine.LearnFromPositiveFeedback(ctx, "mem-syn", "golang programming")
	expanded := engine.ExpandQuery(ctx, "golang programming")
	if len(expanded) == 0 {
		t.Error("ExpandQuery returned empty result")
	}
}

func TestGetLearningHistory(t *testing.T) {
	engine := NewSelfImprovementEngine()
	ctx := context.Background()

	_ = engine.LearnFromPositiveFeedback(ctx, "mem-hist", "q")
	_ = engine.LearnFromNegativeFeedback(ctx, "mem-hist", "bad", false)

	history := engine.GetLearningHistory("mem-hist")
	if history == nil {
		t.Fatal("expected learning history, got nil")
	}
	if history.PositiveFeedbackCount < 1 {
		t.Errorf("expected positive count >= 1, got %d", history.PositiveFeedbackCount)
	}
	if history.NegativeFeedbackCount < 1 {
		t.Errorf("expected negative count >= 1, got %d", history.NegativeFeedbackCount)
	}
}

// ─── SelfImprover tests ───────────────────────────────────────────────────────

func TestProcessTuningCycle_InsufficientFeedback(t *testing.T) {
	fc := &mockFeedbackCollector{feedback: []*types.Feedback{
		{ID: "f1", MemoryID: "mem-a", Type: types.FeedbackPositive, CreatedAt: time.Now()},
	}}
	ts := newMockTuningStore()
	imp := NewSelfImprover(fc, ts, DefaultConfig())

	result, err := imp.ProcessTuningCycle(context.Background(), "mem-a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.ChangesMade) != 0 {
		t.Errorf("expected no changes with insufficient feedback, got %v", result.ChangesMade)
	}
	if len(result.Recommendations) == 0 {
		t.Error("expected waiting recommendation")
	}
}

func TestProcessTuningCycle_PositiveMajority(t *testing.T) {
	feedback := make([]*types.Feedback, 5)
	for i := range feedback {
		feedback[i] = &types.Feedback{
			ID:        fmt.Sprintf("f%d", i),
			MemoryID:  "mem-b",
			Type:      types.FeedbackPositive,
			CreatedAt: time.Now(),
		}
	}
	fc := &mockFeedbackCollector{feedback: feedback}
	ts := newMockTuningStore()
	imp := NewSelfImprover(fc, ts, DefaultConfig())

	result, err := imp.ProcessTuningCycle(context.Background(), "mem-b")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.ChangesMade) == 0 {
		t.Error("expected at least one change with positive majority feedback")
	}
	if _, ok := ts.importanceUpdates["mem-b"]; !ok {
		t.Error("expected importance update for mem-b")
	}
}

func TestProcessTuningCycle_NegativeMajority(t *testing.T) {
	feedback := make([]*types.Feedback, 5)
	for i := range feedback {
		feedback[i] = &types.Feedback{
			ID:        fmt.Sprintf("fn%d", i),
			MemoryID:  "mem-c",
			Type:      types.FeedbackNegative,
			Comment:   "this is wrong content to improve",
			CreatedAt: time.Now(),
		}
	}
	fc := &mockFeedbackCollector{feedback: feedback}
	ts := newMockTuningStore()
	imp := NewSelfImprover(fc, ts, DefaultConfig())

	_, err := imp.ProcessTuningCycle(context.Background(), "mem-c")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if imp := ts.importanceUpdates["mem-c"]; imp != types.ImportanceLow {
		t.Errorf("expected importance to be low after repeated negatives, got %v", imp)
	}
}

func TestAddLearnedSynonym(t *testing.T) {
	fc := &mockFeedbackCollector{}
	ts := newMockTuningStore()
	imp := NewSelfImprover(fc, ts, DefaultConfig())

	if err := imp.AddLearnedSynonym(context.Background(), "mem-syn", "go", "golang"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	syns, err := ts.GetSynonyms(context.Background(), "mem-syn", "go")
	if err != nil {
		t.Fatalf("get synonyms error: %v", err)
	}
	if len(syns) == 0 || syns[0] != "golang" {
		t.Errorf("expected synonym 'golang', got %v", syns)
	}
}

func TestGetTuningInsights(t *testing.T) {
	feedback := []*types.Feedback{
		{ID: "f1", MemoryID: "mem-i", Type: types.FeedbackPositive, CreatedAt: time.Now()},
		{ID: "f2", MemoryID: "mem-i", Type: types.FeedbackPositive, CreatedAt: time.Now()},
		{ID: "f3", MemoryID: "mem-i", Type: types.FeedbackPositive, CreatedAt: time.Now()},
	}
	fc := &mockFeedbackCollector{feedback: feedback}
	ts := newMockTuningStore()
	imp := NewSelfImprover(fc, ts, DefaultConfig())

	_, _ = imp.ProcessTuningCycle(context.Background(), "mem-i")

	insights, err := imp.GetTuningInsights(context.Background(), "mem-i")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if insights.TotalEvents == 0 {
		t.Error("expected at least one tuning event in insights")
	}
}

// ─── InMemoryTuningStore tests ─────────────────────────────────────────────────

func TestInMemoryTuningStore_Synonyms(t *testing.T) {
	ts := NewInMemoryTuningStore(nil)
	ctx := context.Background()

	if err := ts.AddSynonym(ctx, "m1", "db", "database"); err != nil {
		t.Fatalf("AddSynonym error: %v", err)
	}
	syns, err := ts.GetSynonyms(ctx, "m1", "db")
	if err != nil {
		t.Fatalf("GetSynonyms error: %v", err)
	}
	if len(syns) != 1 || syns[0] != "database" {
		t.Errorf("expected [database], got %v", syns)
	}
}

func TestInMemoryTuningStore_History(t *testing.T) {
	ts := NewInMemoryTuningStore(nil)
	ctx := context.Background()

	ev := &TuningEvent{
		MemoryID:  "m2",
		EventType: TuningEventImportanceIncrease,
		NewValue:  "high",
		Trigger:   "test",
		CreatedAt: time.Now(),
	}
	if err := ts.RecordTuningEvent(ctx, ev); err != nil {
		t.Fatalf("RecordTuningEvent error: %v", err)
	}

	history, err := ts.GetTuningHistory(ctx, "m2")
	if err != nil {
		t.Fatalf("GetTuningHistory error: %v", err)
	}
	if len(history) != 1 {
		t.Errorf("expected 1 event, got %d", len(history))
	}
	if history[0].EventType != TuningEventImportanceIncrease {
		t.Errorf("wrong event type: %v", history[0].EventType)
	}
}

func TestInMemoryTuningStore_UpdateImportanceNilGraph(t *testing.T) {
	ts := NewInMemoryTuningStore(nil)
	// With nil graph, UpdateMemoryImportance should return gracefully (not panic)
	err := ts.UpdateMemoryImportance(context.Background(), "mem-x", types.ImportanceHigh)
	if err == nil {
		// nil graph means no error propagation — this is acceptable
		return
	}
	// If it does return an error, it should be descriptive
	if err.Error() == "" {
		t.Error("expected non-empty error string")
	}
}
