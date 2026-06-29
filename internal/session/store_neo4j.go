package session

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// graphExec is the minimal graph-store surface used by Neo4jStore. It exists
// so tests can substitute an in-memory fake without spinning up Neo4j.
type graphExec interface {
	QueryGraph(cypher string, params map[string]interface{}) ([]map[string]interface{}, error)
}

// Neo4jStore is a Neo4j-backed implementation of the session Store interface.
// It is the cold durable tier: every write is persisted to the graph and
// survives process restarts. Use TieredStore to combine with RedisStore for
// the hot tier.
//
// Node labels:
//   (:Session {id, user_id, created_at, updated_at, tenant_id})
//   (:QATurn  {id, user_id, session_id, question, answer, ...})
//   (:DistilledLesson {id, user_id, session_id, statement, ...})
//
// Relationships:
//   (:QATurn)-[:PART_OF]->(:Session)
//   (:DistilledLesson)-[:DERIVED_FROM]->(:Session)
//   (:DistilledLesson)-[:DERIVED_FROM]->(:QATurn)
type Neo4jStore struct {
	graph graphExec
	now   func() time.Time
}

// NewNeo4jStore wraps a GraphStore-compatible executor. The graphExec
// interface is satisfied by *memory.neo4j.Client (via its QueryGraph method)
// and by the test fake in store_neo4j_test.go.
func NewNeo4jStore(g graphExec) *Neo4jStore {
	return &Neo4jStore{graph: g, now: func() time.Time { return time.Now().UTC() }}
}

// Ping verifies the graph connection is healthy.
func (s *Neo4jStore) Ping(_ context.Context) error {
	if s.graph == nil {
		return errors.New("neo4j store: graph executor is nil")
	}
	_, err := s.graph.QueryGraph("RETURN 1", nil)
	return err
}

// --- Store interface ---

func (s *Neo4jStore) AppendTurn(userID, sessionID string, turn QATurn) error {
	if s.graph == nil {
		return errors.New("neo4j store: graph executor is nil")
	}
	if turn.ID == "" {
		turn.ID = newUUID()
	}
	if turn.CreatedAt.IsZero() {
		turn.CreatedAt = s.now()
	}

	// Upsert Session node + create QATurn node + link with PART_OF. We use
	// MERGE for Session so multiple turns in the same session reuse the
	// same Session node.
	cypher := `
MERGE (sess:Session {id: $session_id})
  ON CREATE SET sess.user_id = $user_id,
                sess.created_at = datetime($created_at),
                sess.updated_at = datetime($created_at)
  ON MATCH SET sess.updated_at = datetime($created_at)
MERGE (turn:QATurn {id: $turn_id})
  ON CREATE SET turn.user_id = $user_id,
                turn.session_id = $session_id,
                turn.question = $question,
                turn.answer = $answer,
                turn.context = $context,
                turn.created_at = datetime($created_at),
                turn.confidence = $confidence,
                turn.harmful_count = $harmful_count
MERGE (turn)-[:PART_OF]->(sess)
`
	params := map[string]interface{}{
		"session_id":   sessionID,
		"user_id":      userID,
		"turn_id":      turn.ID,
		"question":     turn.Question,
		"answer":       turn.Answer,
		"context":      turn.Context,
		"created_at":   turn.CreatedAt.Format(time.RFC3339Nano),
		"confidence":   turn.Confidence,
		"harmful_count": turn.HarmfulCount,
	}
	if _, err := s.graph.QueryGraph(cypher, params); err != nil {
		return fmt.Errorf("neo4j store: append turn: %w", err)
	}
	return nil
}

func (s *Neo4jStore) ListTurns(userID, sessionID string, lastN int) ([]QATurn, error) {
	if s.graph == nil {
		return nil, errors.New("neo4j store: graph executor is nil")
	}
	cypher := `
MATCH (t:QATurn {user_id: $user_id, session_id: $session_id})
RETURN t.id AS id, t.user_id AS user_id, t.session_id AS session_id,
       t.question AS question, t.answer AS answer, t.context AS context,
       t.created_at AS created_at, t.confidence AS confidence,
       t.harmful_count AS harmful_count
ORDER BY t.created_at ASC
`
	rows, err := s.graph.QueryGraph(cypher, map[string]interface{}{
		"user_id":    userID,
		"session_id": sessionID,
	})
	if err != nil {
		return nil, fmt.Errorf("neo4j store: list turns: %w", err)
	}
	turns := make([]QATurn, 0, len(rows))
	for _, r := range rows {
		turns = append(turns, turnFromRow(r))
	}
	if lastN > 0 && len(turns) > lastN {
		turns = turns[len(turns)-lastN:]
	}
	return turns, nil
}

func (s *Neo4jStore) UpsertContext(userID string, entries []ContextEntry) error {
	if s.graph == nil {
		return errors.New("neo4j store: graph executor is nil")
	}
	// ContextEntry is stored as a JSON blob on a single Context node keyed
	// by (user_id). This avoids schema churn for a transient distiller
	// input.
	data, err := json.Marshal(entries)
	if err != nil {
		return fmt.Errorf("neo4j store: marshal context: %w", err)
	}
	cypher := `
MERGE (c:DistillerContext {user_id: $user_id})
  ON CREATE SET c.entries = $entries, c.updated_at = datetime()
  ON MATCH  SET c.entries = $entries, c.updated_at = datetime()
`
	_, err = s.graph.QueryGraph(cypher, map[string]interface{}{
		"user_id": userID,
		"entries": string(data),
	})
	if err != nil {
		return fmt.Errorf("neo4j store: upsert context: %w", err)
	}
	return nil
}

func (s *Neo4jStore) ListContext(userID string) ([]ContextEntry, error) {
	if s.graph == nil {
		return nil, errors.New("neo4j store: graph executor is nil")
	}
	rows, err := s.graph.QueryGraph(`
MATCH (c:DistillerContext {user_id: $user_id})
RETURN c.entries AS entries
`, map[string]interface{}{"user_id": userID})
	if err != nil {
		return nil, fmt.Errorf("neo4j store: list context: %w", err)
	}
	if len(rows) == 0 {
		return []ContextEntry{}, nil
	}
	raw, _ := rows[0]["entries"].(string)
	if raw == "" {
		return []ContextEntry{}, nil
	}
	var entries []ContextEntry
	if err := json.Unmarshal([]byte(raw), &entries); err != nil {
		return nil, fmt.Errorf("neo4j store: unmarshal context: %w", err)
	}
	return entries, nil
}

func (s *Neo4jStore) SaveLesson(userID string, lesson DistilledLesson) error {
	if s.graph == nil {
		return errors.New("neo4j store: graph executor is nil")
	}
	if lesson.ID == "" {
		lesson.ID = newUUID()
	}
	if lesson.UserID == "" {
		lesson.UserID = userID
	}
	if lesson.DistilledOn.IsZero() {
		lesson.DistilledOn = s.now()
	}
	cypher := `
MERGE (sess:Session {id: $session_id})
  ON CREATE SET sess.user_id = $user_id,
                sess.created_at = datetime($distilled_on),
                sess.updated_at = datetime($distilled_on)
  ON MATCH  SET sess.updated_at = datetime($distilled_on)
MERGE (l:DistilledLesson {id: $lesson_id})
  ON CREATE SET l.user_id = $user_id,
                l.session_id = $session_id,
                l.statement = $statement,
                l.entities = $entities,
                l.why_learned = $why_learned,
                l.member_entry_ids = $member_entry_ids,
                l.distilled_on = datetime($distilled_on)
MERGE (l)-[:DERIVED_FROM]->(sess)
`
	_, err := s.graph.QueryGraph(cypher, map[string]interface{}{
		"session_id":       lesson.SessionID,
		"user_id":          userID,
		"lesson_id":        lesson.ID,
		"statement":        lesson.Statement,
		"entities":         lesson.Entities,
		"why_learned":      lesson.WhyLearned,
		"member_entry_ids": lesson.MemberEntryIDs,
		"distilled_on":     lesson.DistilledOn.Format(time.RFC3339Nano),
	})
	if err != nil {
		return fmt.Errorf("neo4j store: save lesson: %w", err)
	}
	return nil
}

func (s *Neo4jStore) ListLessons(userID string) ([]DistilledLesson, error) {
	if s.graph == nil {
		return nil, errors.New("neo4j store: graph executor is nil")
	}
	rows, err := s.graph.QueryGraph(`
MATCH (l:DistilledLesson {user_id: $user_id})
RETURN l.id AS id, l.user_id AS user_id, l.session_id AS session_id,
       l.statement AS statement, l.entities AS entities,
       l.why_learned AS why_learned, l.member_entry_ids AS member_entry_ids,
       l.distilled_on AS distilled_on
ORDER BY l.distilled_on DESC
`, map[string]interface{}{"user_id": userID})
	if err != nil {
		return nil, fmt.Errorf("neo4j store: list lessons: %w", err)
	}
	out := make([]DistilledLesson, 0, len(rows))
	for _, r := range rows {
		out = append(out, lessonFromRow(r))
	}
	return out, nil
}

// --- helpers ---

// turnFromRow converts a Cypher result row into a QATurn. The Neo4j driver
// returns datetimes as time.Time; string fields as string; numeric fields
// as int64/float64.
func turnFromRow(r map[string]interface{}) QATurn {
	t := QATurn{
		ID:        asString(r["id"]),
		UserID:    asString(r["user_id"]),
		SessionID: asString(r["session_id"]),
		Question:  asString(r["question"]),
		Answer:    asString(r["answer"]),
		Context:   asString(r["context"]),
	}
	if ts, ok := r["created_at"].(time.Time); ok {
		t.CreatedAt = ts
	}
	if v, ok := r["confidence"].(float64); ok {
		t.Confidence = v
	}
	if v, ok := r["harmful_count"].(int64); ok {
		t.HarmfulCount = int(v)
	}
	return t
}

func lessonFromRow(r map[string]interface{}) DistilledLesson {
	l := DistilledLesson{
		ID:            asString(r["id"]),
		UserID:        asString(r["user_id"]),
		SessionID:     asString(r["session_id"]),
		Statement:     asString(r["statement"]),
		WhyLearned:    asString(r["why_learned"]),
	}
	if entities, ok := r["entities"].([]interface{}); ok {
		l.Entities = make([]string, 0, len(entities))
		for _, e := range entities {
			if s, ok := e.(string); ok {
				l.Entities = append(l.Entities, s)
			}
		}
	}
	if members, ok := r["member_entry_ids"].([]interface{}); ok {
		l.MemberEntryIDs = make([]string, 0, len(members))
		for _, m := range members {
			if s, ok := m.(string); ok {
				l.MemberEntryIDs = append(l.MemberEntryIDs, s)
			}
		}
	}
	if ts, ok := r["distilled_on"].(time.Time); ok {
		l.DistilledOn = ts
	}
	return l
}

func asString(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// newUUID returns a fresh UUIDv4 string.
func newUUID() string { return uuid.NewString() }
