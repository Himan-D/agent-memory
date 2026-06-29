package session

import (
	"strings"
	"testing"
	"time"
)

// fakeGraph records the cypher queries and parameters sent to QueryGraph and
// returns canned rows based on a simple substring match. It is sufficient
// to exercise Neo4jStore logic without a real Neo4j instance.
type fakeGraph struct {
	cyphers []string
	params  []map[string]interface{}
	rows    []map[string]interface{}
	err     error
}

func (f *fakeGraph) QueryGraph(cypher string, params map[string]interface{}) ([]map[string]interface{}, error) {
	f.cyphers = append(f.cyphers, cypher)
	f.params = append(f.params, params)
	if f.err != nil {
		return nil, f.err
	}
	return f.rows, nil
}

func TestNeo4jStore_AppendTurnIssuesCypher(t *testing.T) {
	f := &fakeGraph{}
	s := NewNeo4jStore(f)
	err := s.AppendTurn("u1", "s1", QATurn{ID: "t1", Question: "Q", Answer: "A"})
	if err != nil {
		t.Fatal(err)
	}
	if len(f.cyphers) != 1 {
		t.Fatalf("expected 1 cypher call, got %d", len(f.cyphers))
	}
	c := f.cyphers[0]
	if !strings.Contains(c, "MERGE (sess:Session") {
		t.Fatalf("cypher missing Session MERGE: %s", c)
	}
	if !strings.Contains(c, "MERGE (turn:QATurn") {
		t.Fatalf("cypher missing QATurn MERGE: %s", c)
	}
	if f.params[0]["turn_id"] != "t1" {
		t.Fatalf("expected turn_id=t1, got %v", f.params[0]["turn_id"])
	}
}

func TestNeo4jStore_AppendTurnGeneratesIDWhenEmpty(t *testing.T) {
	f := &fakeGraph{}
	s := NewNeo4jStore(f)
	if err := s.AppendTurn("u1", "s1", QATurn{Question: "Q", Answer: "A"}); err != nil {
		t.Fatal(err)
	}
	id, ok := f.params[0]["turn_id"].(string)
	if !ok || id == "" {
		t.Fatalf("expected generated turn_id, got %v", f.params[0]["turn_id"])
	}
	if !strings.Contains(id, "-") {
		t.Fatalf("expected UUID-like id, got %q", id)
	}
}

func TestNeo4jStore_AppendTurnTimestampDefaults(t *testing.T) {
	f := &fakeGraph{}
	s := NewNeo4jStore(f)
	before := time.Now().UTC().Add(-time.Second)
	if err := s.AppendTurn("u1", "s1", QATurn{ID: "t1"}); err != nil {
		t.Fatal(err)
	}
	after := time.Now().UTC().Add(time.Second)
	ts, ok := f.params[0]["created_at"].(string)
	if !ok {
		t.Fatalf("expected created_at string, got %T", f.params[0]["created_at"])
	}
	parsed, err := time.Parse(time.RFC3339Nano, ts)
	if err != nil {
		t.Fatalf("parse created_at: %v", err)
	}
	if parsed.Before(before) || parsed.After(after) {
		t.Fatalf("created_at out of expected range: %v", parsed)
	}
}

func TestNeo4jStore_AppendTurnError(t *testing.T) {
	f := &fakeGraph{err: errFake("boom")}
	s := NewNeo4jStore(f)
	err := s.AppendTurn("u1", "s1", QATurn{ID: "t1"})
	if err == nil {
		t.Fatal("expected error from AppendTurn")
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Fatalf("error should wrap original: %v", err)
	}
}

func TestNeo4jStore_ListTurnsRowsToQATurns(t *testing.T) {
	f := &fakeGraph{
		rows: []map[string]interface{}{
			{
				"id":            "t1",
				"user_id":       "u1",
				"session_id":    "s1",
				"question":      "Q1",
				"answer":        "A1",
				"context":       "",
				"created_at":    time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
				"confidence":    0.9,
				"harmful_count": int64(0),
			},
			{
				"id":            "t2",
				"user_id":       "u1",
				"session_id":    "s1",
				"question":      "Q2",
				"answer":        "A2",
				"context":       "",
				"created_at":    time.Date(2026, 1, 1, 0, 0, 1, 0, time.UTC),
				"confidence":    0.8,
				"harmful_count": int64(1),
			},
		},
	}
	s := NewNeo4jStore(f)
	turns, err := s.ListTurns("u1", "s1", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(turns) != 2 {
		t.Fatalf("expected 2 turns, got %d", len(turns))
	}
	if turns[0].ID != "t1" || turns[0].Question != "Q1" {
		t.Fatalf("turn[0] mismatch: %+v", turns[0])
	}
	if turns[1].HarmfulCount != 1 {
		t.Fatalf("turn[1] harmful_count mismatch: %d", turns[1].HarmfulCount)
	}
}

func TestNeo4jStore_ListTurnsLastN(t *testing.T) {
	f := &fakeGraph{rows: []map[string]interface{}{
		{"id": "a", "user_id": "u1", "session_id": "s1", "question": "qa", "answer": "aa", "created_at": time.Now(), "confidence": 0.5, "harmful_count": int64(0)},
		{"id": "b", "user_id": "u1", "session_id": "s1", "question": "qb", "answer": "ab", "created_at": time.Now(), "confidence": 0.5, "harmful_count": int64(0)},
		{"id": "c", "user_id": "u1", "session_id": "s1", "question": "qc", "answer": "ac", "created_at": time.Now(), "confidence": 0.5, "harmful_count": int64(0)},
	}}
	s := NewNeo4jStore(f)
	got, err := s.ListTurns("u1", "s1", 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 turn, got %d", len(got))
	}
	if got[0].ID != "c" {
		t.Fatalf("expected last turn (c), got %s", got[0].ID)
	}
}

func TestNeo4jStore_UpsertContextStoresJSON(t *testing.T) {
	f := &fakeGraph{}
	s := NewNeo4jStore(f)
	entries := []ContextEntry{{ID: "a"}, {ID: "b"}}
	if err := s.UpsertContext("u1", entries); err != nil {
		t.Fatal(err)
	}
	raw, ok := f.params[0]["entries"].(string)
	if !ok || raw == "" {
		t.Fatalf("expected entries JSON string, got %v", f.params[0]["entries"])
	}
	if !strings.Contains(raw, `"a"`) || !strings.Contains(raw, `"b"`) {
		t.Fatalf("entries JSON missing IDs: %s", raw)
	}
}

func TestNeo4jStore_ListContextRoundTrip(t *testing.T) {
	want := `[{"id":"a","section":"facts","content":"x"},{"id":"b","section":"facts","content":"y"}]`
	f := &fakeGraph{rows: []map[string]interface{}{{"entries": want}}}
	s := NewNeo4jStore(f)
	got, err := s.ListContext("u1")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].ID != "a" || got[1].ID != "b" {
		t.Fatalf("unexpected context: %+v", got)
	}
}

func TestNeo4jStore_ListContextEmptyReturnsEmptySlice(t *testing.T) {
	f := &fakeGraph{rows: nil}
	s := NewNeo4jStore(f)
	got, err := s.ListContext("u1")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty slice, got %d entries", len(got))
	}
}

func TestNeo4jStore_SaveLessonIssuesCypher(t *testing.T) {
	f := &fakeGraph{}
	s := NewNeo4jStore(f)
	if err := s.SaveLesson("u1", DistilledLesson{SessionID: "s1", Statement: "X"}); err != nil {
		t.Fatal(err)
	}
	c := f.cyphers[0]
	if !strings.Contains(c, "MERGE (l:DistilledLesson") {
		t.Fatalf("cypher missing DistilledLesson MERGE: %s", c)
	}
	if f.params[0]["statement"] != "X" {
		t.Fatalf("expected statement=X, got %v", f.params[0]["statement"])
	}
}

func TestNeo4jStore_ListLessonsRowsToLessons(t *testing.T) {
	f := &fakeGraph{rows: []map[string]interface{}{
		{
			"id":              "l1",
			"user_id":         "u1",
			"session_id":      "s1",
			"statement":       "X",
			"entities":        []interface{}{"Alice", "Bob"},
			"why_learned":     "because",
			"member_entry_ids": []interface{}{"t1", "t2"},
			"distilled_on":    time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}}
	s := NewNeo4jStore(f)
	got, err := s.ListLessons("u1")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 lesson, got %d", len(got))
	}
	if got[0].Statement != "X" {
		t.Fatalf("statement mismatch: %+v", got[0])
	}
	if len(got[0].Entities) != 2 || got[0].Entities[0] != "Alice" {
		t.Fatalf("entities mismatch: %+v", got[0].Entities)
	}
	if len(got[0].MemberEntryIDs) != 2 || got[0].MemberEntryIDs[1] != "t2" {
		t.Fatalf("member_entry_ids mismatch: %+v", got[0].MemberEntryIDs)
	}
}

func TestNeo4jStore_NilGraphErrors(t *testing.T) {
	s := NewNeo4jStore(nil)
	if err := s.AppendTurn("u1", "s1", QATurn{ID: "t1"}); err == nil {
		t.Fatal("expected error for nil graph on AppendTurn")
	}
	if _, err := s.ListTurns("u1", "s1", 0); err == nil {
		t.Fatal("expected error for nil graph on ListTurns")
	}
	if err := s.UpsertContext("u1", nil); err == nil {
		t.Fatal("expected error for nil graph on UpsertContext")
	}
	if _, err := s.ListContext("u1"); err == nil {
		t.Fatal("expected error for nil graph on ListContext")
	}
	if err := s.SaveLesson("u1", DistilledLesson{}); err == nil {
		t.Fatal("expected error for nil graph on SaveLesson")
	}
	if _, err := s.ListLessons("u1"); err == nil {
		t.Fatal("expected error for nil graph on ListLessons")
	}
	if err := s.Ping(nil); err == nil {
		t.Fatal("expected error for nil graph on Ping")
	}
}

func TestNeo4jStore_Ping(t *testing.T) {
	f := &fakeGraph{}
	s := NewNeo4jStore(f)
	if err := s.Ping(nil); err != nil {
		t.Fatalf("Ping should succeed against fake, got %v", err)
	}
}

func TestNeo4jStore_PingError(t *testing.T) {
	f := &fakeGraph{err: errFake("down")}
	s := NewNeo4jStore(f)
	if err := s.Ping(nil); err == nil {
		t.Fatal("expected Ping error")
	}
}

// errFake is a tiny error type used by tests.
type errFake string

func (e errFake) Error() string { return string(e) }
