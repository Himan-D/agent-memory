package session

import (
	"context"
	"errors"
	"testing"
)

// stubStore records calls for inspection in tests.
type stubStore struct {
	turns       map[string][]QATurn
	ctx         map[string][]ContextEntry
	lessons     map[string][]DistilledLesson
	pingErr     error
	appendErr   error
	listErr     error
	upsertErr   error
	saveErr     error
	listCtxErr  error
	listLesErr  error
}

func newStubStore() *stubStore {
	return &stubStore{
		turns:   map[string][]QATurn{},
		ctx:     map[string][]ContextEntry{},
		lessons: map[string][]DistilledLesson{},
	}
}

func (s *stubStore) AppendTurn(userID, sessionID string, turn QATurn) error {
	if s.appendErr != nil {
		return s.appendErr
	}
	k := userID + "/" + sessionID
	s.turns[k] = append(s.turns[k], turn)
	return nil
}
func (s *stubStore) ListTurns(userID, sessionID string, lastN int) ([]QATurn, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	k := userID + "/" + sessionID
	out := append([]QATurn(nil), s.turns[k]...)
	if lastN > 0 && len(out) > lastN {
		out = out[len(out)-lastN:]
	}
	return out, nil
}
func (s *stubStore) UpsertContext(userID string, entries []ContextEntry) error {
	if s.upsertErr != nil {
		return s.upsertErr
	}
	s.ctx[userID] = entries
	return nil
}
func (s *stubStore) ListContext(userID string) ([]ContextEntry, error) {
	if s.listCtxErr != nil {
		return nil, s.listCtxErr
	}
	return append([]ContextEntry(nil), s.ctx[userID]...), nil
}
func (s *stubStore) SaveLesson(userID string, lesson DistilledLesson) error {
	if s.saveErr != nil {
		return s.saveErr
	}
	s.lessons[userID] = append(s.lessons[userID], lesson)
	return nil
}
func (s *stubStore) ListLessons(userID string) ([]DistilledLesson, error) {
	if s.listLesErr != nil {
		return nil, s.listLesErr
	}
	return append([]DistilledLesson(nil), s.lessons[userID]...), nil
}
func (s *stubStore) Ping(ctx context.Context) error { return s.pingErr }

func TestTieredStore_WritesBothTiers(t *testing.T) {
	hot, cold := newStubStore(), newStubStore()
	ts := NewTieredStore(TieredStoreConfig{Hot: hot, Cold: cold})
	if err := ts.AppendTurn("u1", "s1", QATurn{ID: "t1"}); err != nil {
		t.Fatal(err)
	}
	if got := len(hot.turns["u1/s1"]); got != 1 {
		t.Fatalf("hot: expected 1 turn, got %d", got)
	}
	if got := len(cold.turns["u1/s1"]); got != 1 {
		t.Fatalf("cold: expected 1 turn, got %d", got)
	}
}

func TestTieredStore_HotFailureStillWritesCold(t *testing.T) {
	hot, cold := newStubStore(), newStubStore()
	hot.appendErr = errors.New("redis down")
	ts := NewTieredStore(TieredStoreConfig{Hot: hot, Cold: cold})
	if err := ts.AppendTurn("u1", "s1", QATurn{ID: "t1"}); err != nil {
		t.Fatalf("expected nil error when only hot fails, got %v", err)
	}
	if len(cold.turns["u1/s1"]) != 1 {
		t.Fatalf("cold tier should still have received the write")
	}
	if ts.HotHealth() {
		t.Fatal("hot tier should be marked unhealthy")
	}
}

func TestTieredStore_ColdFailureIsFatal(t *testing.T) {
	hot, cold := newStubStore(), newStubStore()
	cold.appendErr = errors.New("neo4j down")
	ts := NewTieredStore(TieredStoreConfig{Hot: hot, Cold: cold})
	err := ts.AppendTurn("u1", "s1", QATurn{ID: "t1"})
	if err == nil {
		t.Fatal("expected error from cold-tier failure")
	}
	if !errors.Is(err, errors.Unwrap(err)) && !contains(err.Error(), "neo4j") {
		t.Fatalf("expected wrapped neo4j error, got %v", err)
	}
}

func TestTieredStore_ReadsPreferHot(t *testing.T) {
	hot, cold := newStubStore(), newStubStore()
	hot.turns["u1/s1"] = []QATurn{{ID: "from-hot"}}
	cold.turns["u1/s1"] = []QATurn{{ID: "from-cold"}}
	ts := NewTieredStore(TieredStoreConfig{Hot: hot, Cold: cold})
	got, err := ts.ListTurns("u1", "s1", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != "from-hot" {
		t.Fatalf("expected to read from hot tier, got %+v", got)
	}
}

func TestTieredStore_HotReadFailureFallsBackToCold(t *testing.T) {
	hot, cold := newStubStore(), newStubStore()
	hot.listErr = errors.New("redis down")
	hot.turns["u1/s1"] = []QATurn{{ID: "from-hot"}}
	cold.turns["u1/s1"] = []QATurn{{ID: "from-cold"}}
	ts := NewTieredStore(TieredStoreConfig{Hot: hot, Cold: cold})
	got, err := ts.ListTurns("u1", "s1", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != "from-cold" {
		t.Fatalf("expected cold fallback, got %+v", got)
	}
	if ts.HotHealth() {
		t.Fatal("hot tier should be marked unhealthy after read failure")
	}
}

func TestTieredStore_HotRecovery(t *testing.T) {
	hot, cold := newStubStore(), newStubStore()
	ts := NewTieredStore(TieredStoreConfig{Hot: hot, Cold: cold})

	// First write fails on hot; marks unhealthy.
	hot.appendErr = errors.New("down")
	_ = ts.AppendTurn("u1", "s1", QATurn{ID: "t1"})
	if ts.HotHealth() {
		t.Fatal("expected unhealthy")
	}

	// Hot recovers.
	hot.appendErr = nil
	_ = ts.AppendTurn("u1", "s1", QATurn{ID: "t2"})
	if !ts.HotHealth() {
		t.Fatal("expected hot to recover after successful write")
	}
	if got := len(hot.turns["u1/s1"]); got != 1 {
		t.Fatalf("hot tier should have t2 only, got %d", got)
	}
}

func TestTieredStore_NilHot(t *testing.T) {
	cold := newStubStore()
	ts := NewTieredStore(TieredStoreConfig{Cold: cold})
	if err := ts.AppendTurn("u1", "s1", QATurn{ID: "t1"}); err != nil {
		t.Fatal(err)
	}
	if len(cold.turns["u1/s1"]) != 1 {
		t.Fatal("cold tier should have received the write")
	}
}

func TestTieredStore_NilCold(t *testing.T) {
	hot := newStubStore()
	ts := NewTieredStore(TieredStoreConfig{Hot: hot})
	if err := ts.AppendTurn("u1", "s1", QATurn{ID: "t1"}); err != nil {
		t.Fatal(err)
	}
	if !ts.HotHealth() {
		t.Fatal("hot tier should be healthy")
	}
}

func TestTieredStore_ContextAndLessonsBothTiers(t *testing.T) {
	hot, cold := newStubStore(), newStubStore()
	ts := NewTieredStore(TieredStoreConfig{Hot: hot, Cold: cold})
	if err := ts.UpsertContext("u1", []ContextEntry{{ID: "a"}}); err != nil {
		t.Fatal(err)
	}
	if len(hot.ctx["u1"]) != 1 || len(cold.ctx["u1"]) != 1 {
		t.Fatalf("expected both tiers to have context")
	}
	if err := ts.SaveLesson("u1", DistilledLesson{Statement: "X"}); err != nil {
		t.Fatal(err)
	}
	if len(hot.lessons["u1"]) != 1 || len(cold.lessons["u1"]) != 1 {
		t.Fatalf("expected both tiers to have lesson")
	}
}

func TestTieredStore_PingAggregatesErrors(t *testing.T) {
	hot, cold := newStubStore(), newStubStore()
	hot.pingErr = errors.New("h")
	cold.pingErr = errors.New("c")
	ts := NewTieredStore(TieredStoreConfig{Hot: hot, Cold: cold})
	err := ts.Ping(context.Background())
	if err == nil {
		t.Fatal("expected aggregated error")
	}
	if !contains(err.Error(), "h") || !contains(err.Error(), "c") {
		t.Fatalf("expected both errors present, got %v", err)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
