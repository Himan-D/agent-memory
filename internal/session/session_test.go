package session

import (
	"context"
	"testing"
	"time"
)

func TestInMemoryStore_AppendTurn(t *testing.T) {
	s := NewInMemoryStore()
	turn := QATurn{
		ID:        "t1",
		UserID:    "u1",
		SessionID: "s1",
		Question:  "Q?",
		Answer:    "A.",
		CreatedAt: time.Now(),
	}
	if err := s.AppendTurn("u1", "s1", turn); err != nil {
		t.Fatal(err)
	}
	turns, err := s.ListTurns("u1", "s1", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(turns) != 1 || turns[0].ID != "t1" {
		t.Fatalf("unexpected turns: %+v", turns)
	}
}

func TestInMemoryStore_ListTurnsLastN(t *testing.T) {
	s := NewInMemoryStore()
	now := time.Now()
	for i := 0; i < 5; i++ {
		_ = s.AppendTurn("u1", "s1", QATurn{
			ID:        time.Now().String(),
			CreatedAt: now.Add(time.Duration(i) * time.Second),
		})
	}
	got, err := s.ListTurns("u1", "s1", 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 turns, got %d", len(got))
	}
}

func TestGate_AllowAll_PreservesOrder(t *testing.T) {
	g := NewGate()
	entries := []ContextEntry{
		{ID: "a"},
		{ID: "b"},
		{ID: "c"},
	}
	got := g.AllowAll(entries)
	if len(got) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(got))
	}
	for i, e := range got {
		if e.ID != entries[i].ID {
			t.Fatalf("order broken at %d", i)
		}
	}
}

func TestBatcher_BuildsChronologicalBatches(t *testing.T) {
	b := NewBatcher()
	b.BlocksPerBatch = 2
	turns := []QATurn{
		{ID: "t1", Question: "q1", Answer: "a1", CreatedAt: time.Unix(1, 0)},
		{ID: "t2", Question: "q2", Answer: "a2", CreatedAt: time.Unix(2, 0)},
		{ID: "t3", Question: "q3", Answer: "a3", CreatedAt: time.Unix(3, 0)},
		{ID: "t4", Question: "q4", Answer: "a4", CreatedAt: time.Unix(4, 0)},
	}
	context := []ContextEntry{
		{ID: "c1", Section: "facts", Content: "fact one", CreatedAt: time.Unix(5, 0)},
	}
	got := b.Build(turns, context)
	if len(got) == 0 {
		t.Fatal("expected at least one batch")
	}
	// Every batch must have a Text and Members; every member id is non-empty.
	for i, batch := range got {
		if batch.Text == "" {
			t.Fatalf("batch %d has empty text", i)
		}
		for _, m := range batch.Members {
			if m == "" {
				t.Fatalf("batch %d has empty member id", i)
			}
		}
	}
}

func TestBatcher_ChronologicalAcrossSources(t *testing.T) {
	b := NewBatcher()
	b.BlocksPerBatch = 100
	turns := []QATurn{
		{ID: "t1", Question: "q1", Answer: "a1", CreatedAt: time.Unix(2, 0)},
		{ID: "t2", Question: "q2", Answer: "a2", CreatedAt: time.Unix(4, 0)},
	}
	context := []ContextEntry{
		{ID: "c1", Section: "facts", Content: "x", CreatedAt: time.Unix(1, 0)},
		{ID: "c2", Section: "facts", Content: "y", CreatedAt: time.Unix(3, 0)},
	}
	got := b.Build(turns, context)
	if len(got) != 1 {
		t.Fatalf("expected 1 batch, got %d", len(got))
	}
	text := got[0].Text
	// Order by timestamp: c1 (t=1), t1/q1 (t=2), c2/y (t=3), t2/q2 (t=4).
	// The batcher doesn't embed turn IDs in text, so we match on the
	// visible content: "Candidate c1", "q1", "Candidate c2", "q2".
	wantOrder := []string{"Candidate c1", "q1", "Candidate c2", "q2"}
	prev := -1
	for _, marker := range wantOrder {
		idx := indexOf(text, marker)
		if idx < 0 {
			t.Fatalf("missing %q in batch text: %q", marker, text)
		}
		if idx <= prev {
			t.Fatalf("order broken for %q", marker)
		}
		prev = idx
	}
}

func TestBatcher_RespectsCharBudget(t *testing.T) {
	b := NewBatcher()
	b.CharBudget = 100
	b.BlocksPerBatch = 100
	turns := []QATurn{}
	for i := 0; i < 10; i++ {
		turns = append(turns, QATurn{
			ID:        "t" + itoaS(i),
			Question:  "Q?",
			Answer:    repeatS("A", 50), // each block is ~55 chars
			CreatedAt: time.Unix(int64(i), 0),
		})
	}
	got := b.Build(turns, nil)
	if len(got) < 2 {
		t.Fatalf("expected multiple batches under tight budget, got %d", len(got))
	}
}

func TestManager_AddAndList(t *testing.T) {
	m := NewManager(NewInMemoryStore())
	ctx := context.Background()

	score := 0.9
	if _, err := m.AddQATurn(ctx, "u1", "s1", "Hi?", "Hello.", "", &score, []string{"mem-1"}); err != nil {
		t.Fatal(err)
	}
	got, err := m.GetSession(ctx, "u1", "s1", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 turn, got %d", len(got))
	}
	if got[0].FeedbackScore == nil || *got[0].FeedbackScore != 0.9 {
		t.Fatalf("feedback not stored: %+v", got[0].FeedbackScore)
	}
}

func TestManager_FilteredContext(t *testing.T) {
	m := NewManager(NewInMemoryStore())
	ctx := context.Background()
	if err := m.UpsertContext(ctx, "u1", []ContextEntry{{ID: "a"}, {ID: "b"}}); err != nil {
		t.Fatal(err)
	}
	got, err := m.FilteredContext("u1")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(got))
	}
}

func TestManager_SaveAndListLessons(t *testing.T) {
	m := NewManager(NewInMemoryStore())
	ctx := context.Background()
	if err := m.SaveLesson(ctx, "u1", DistilledLesson{Statement: "X"}); err != nil {
		t.Fatal(err)
	}
	if err := m.SaveLesson(ctx, "u1", DistilledLesson{Statement: "Y"}); err != nil {
		t.Fatal(err)
	}
	got, err := m.ListLessons(ctx, "u1")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 lessons, got %d", len(got))
	}
}

func TestManager_PrepareTurn(t *testing.T) {
	m := NewManager(NewInMemoryStore())
	tp := m.PrepareTurn(context.Background(), "u1", "what is X?")
	if !tp.ShouldAnswer || tp.EffectiveQ != "what is X?" {
		t.Fatalf("unexpected prep: %+v", tp)
	}
}

// helpers used only by tests
func itoaS(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	buf := make([]byte, 0, 8)
	for n > 0 {
		buf = append([]byte{byte('0' + n%10)}, buf...)
		n /= 10
	}
	if neg {
		buf = append([]byte{'-'}, buf...)
	}
	return string(buf)
}

func repeatS(s string, n int) string {
	out := make([]byte, 0, len(s)*n)
	for i := 0; i < n; i++ {
		out = append(out, s...)
	}
	return string(out)
}

func indexOf(haystack, needle string) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}
