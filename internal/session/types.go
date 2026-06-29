// Package session provides a session-aware layer above the memory service,
// modeled after Cognee's SessionManager. It records ephemeral QA turns with
// feedback and graph-element references, gates low-quality entries, batches
// them chronologically for distillation, and exposes a curator/writer
// pipeline that produces durable entity-anchored lessons.
//
// This package is purely additive: it does not modify the existing memory
// service or any call sites. When no session backend is configured (no
// Redis, no Neo4j), the package degrades to an in-memory implementation that
// is safe for tests and single-process CLI use.
package session

import (
	"context"
	"sync"
	"time"
)

// Default constants mirror Cognee's session-distillation defaults. They are
// var not const so callers can override per-process if needed.
var (
	// MinGateConfidence is the minimum fact/confidence score required to
	// pass the gate filter.
	MinGateConfidence = 0.75
	// BatchCharBudget is the inclusive byte budget for a single curator
	// batch.
	BatchCharBudget = 16_000
	// CuratorBlocksPerBatch controls how many timeline blocks are packed
	// into a single batch.
	CuratorBlocksPerBatch = 6
	// MaxQAQuestionChars truncates question text before batching.
	MaxQAQuestionChars = 1_200
	// MaxQAAnswerChars truncates answer text before batching.
	MaxQAAnswerChars = 1_200
	// MaxCandidateChars truncates candidate memory text before batching.
	MaxCandidateChars = 280
	// CuratorConcurrency bounds parallel curator LLM calls.
	CuratorConcurrency = 5
	// WriterConcurrency bounds parallel writer LLM calls.
	WriterConcurrency = 5
	// SessionHistoryLastN is the default cap when reading recent turns.
	SessionHistoryLastN = 10
)

// QATurn is one question/answer exchange in a session.
type QATurn struct {
	ID                   string    `json:"id"`
	UserID               string    `json:"user_id"`
	SessionID            string    `json:"session_id"`
	Question             string    `json:"question"`
	Answer               string    `json:"answer"`
	Context              string    `json:"context,omitempty"`
	CreatedAt            time.Time `json:"created_at"`
	FeedbackScore        *float64  `json:"feedback_score,omitempty"`
	UsedGraphElementIDs  []string  `json:"used_graph_element_ids,omitempty"`
	HarmfulCount         int       `json:"harmful_count"`
	Confidence           float64   `json:"confidence"`
}

// Session groups QA turns under a logical conversation.
type Session struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Turns     []QATurn  `json:"turns,omitempty"`
}

// ContextEntry is a candidate memory the curator may consume during
// distillation. It mirrors the "candidate" rows in Cognee's distillation.
//
// Confidence and HarmfulCount are populated by LoadScores (typically from
// the source memory's safety classification + composite scoring). Zero
// values are treated as "unset" and the gate falls back to package
// defaults so existing callers are unaffected.
type ContextEntry struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	Section      string    `json:"section"`
	Content      string    `json:"content"`
	CreatedAt    time.Time `json:"created_at"`
	Confidence   float64   `json:"confidence,omitempty"`
	HarmfulCount int       `json:"harmful_count,omitempty"`
}

// ProposedLesson is the curator's candidate output. A proposal is later
// judged for novelty and entity grounding by the writer/rejecter.
type ProposedLesson struct {
	WorkingStatement string   `json:"working_statement"`
	MemberEntryIDs   []string `json:"member_entry_ids"`
}

// CuratorBatchOutput is the curator LLM's structured response for one batch.
type CuratorBatchOutput struct {
	Lessons []ProposedLesson `json:"lessons"`
}

// WrittenLesson is the writer/rejecter's decision.
type WrittenLesson struct {
	Accept     bool   `json:"accept"`
	Reason     string `json:"reason,omitempty"` // "already_known" | "not_durable" | "unsupported"
	Statement  string `json:"statement"`
	Entities   []string `json:"entities,omitempty"`
	WhyLearned string `json:"why_learned,omitempty"`
}

// DistilledLesson is the durable artifact produced by the distiller and
// re-ingested into the memory graph.
type DistilledLesson struct {
	ID            string    `json:"id"`
	UserID        string    `json:"user_id"`
	SessionID     string    `json:"session_id"`
	Statement     string    `json:"statement"`
	Entities      []string  `json:"entities,omitempty"`
	WhyLearned    string    `json:"why_learned,omitempty"`
	MemberEntryIDs []string `json:"member_entry_ids,omitempty"`
	DistilledOn   time.Time `json:"distilled_on"`
}

// TurnPreparation is the result of analyzing a turn before answering. It is
// produced by Manager.PrepareTurn and consumed by the answer-generation path.
type TurnPreparation struct {
	ShouldAnswer  bool   `json:"should_answer"`
	EffectiveQ    string `json:"effective_query"`
	ResponseToUser string `json:"response_to_user,omitempty"`
}

// InMemoryStore is a goroutine-safe store used when no Redis/Neo4j backend
// is available. It is the default backend for tests and CLI-only runs.
type InMemoryStore struct {
	mu       sync.RWMutex
	turns    map[string]map[string][]QATurn // userID -> sessionID -> turns
	sessions map[string]map[string]*Session // userID -> sessionID -> session
	lessons  map[string][]DistilledLesson    // userID -> lessons
	contexts map[string][]ContextEntry       // userID -> entries
}

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		turns:    make(map[string]map[string][]QATurn),
		sessions: make(map[string]map[string]*Session),
		lessons:  make(map[string][]DistilledLesson),
		contexts: make(map[string][]ContextEntry),
	}
}

func (s *InMemoryStore) AppendTurn(userID, sessionID string, turn QATurn) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.turns[userID]; !ok {
		s.turns[userID] = make(map[string][]QATurn)
	}
	if _, ok := s.sessions[userID]; !ok {
		s.sessions[userID] = make(map[string]*Session)
	}
	s.turns[userID][sessionID] = append(s.turns[userID][sessionID], turn)
	sess, ok := s.sessions[userID][sessionID]
	if !ok {
		sess = &Session{
			ID:        sessionID,
			UserID:    userID,
			CreatedAt: turn.CreatedAt,
		}
		s.sessions[userID][sessionID] = sess
	}
	sess.UpdatedAt = turn.CreatedAt
	sess.Turns = s.turns[userID][sessionID]
	return nil
}

func (s *InMemoryStore) ListTurns(userID, sessionID string, lastN int) ([]QATurn, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	turns := s.turns[userID][sessionID]
	if lastN > 0 && len(turns) > lastN {
		turns = turns[len(turns)-lastN:]
	}
	out := make([]QATurn, len(turns))
	copy(out, turns)
	return out, nil
}

func (s *InMemoryStore) UpsertContext(userID string, entries []ContextEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.contexts[userID] = entries
	return nil
}

func (s *InMemoryStore) ListContext(userID string) ([]ContextEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]ContextEntry, len(s.contexts[userID]))
	copy(out, s.contexts[userID])
	return out, nil
}

func (s *InMemoryStore) SaveLesson(userID string, lesson DistilledLesson) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lessons[userID] = append(s.lessons[userID], lesson)
	return nil
}

func (s *InMemoryStore) ListLessons(userID string) ([]DistilledLesson, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]DistilledLesson, len(s.lessons[userID]))
	copy(out, s.lessons[userID])
	return out, nil
}

func (s *InMemoryStore) Ping(ctx context.Context) error {
	return nil
}
