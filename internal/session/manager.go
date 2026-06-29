package session

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Store is the persistence interface required by Manager. Implementations
// include InMemoryStore (default for tests/CLI) and any future Neo4j/Redis
// backed store.
type Store interface {
	AppendTurn(userID, sessionID string, turn QATurn) error
	ListTurns(userID, sessionID string, lastN int) ([]QATurn, error)
	UpsertContext(userID string, entries []ContextEntry) error
	ListContext(userID string) ([]ContextEntry, error)
	SaveLesson(userID string, lesson DistilledLesson) error
	ListLessons(userID string) ([]DistilledLesson, error)
	Ping(ctx context.Context) error
}

// Manager is the high-level entry point for the session layer. It is safe
// for concurrent use.
type Manager struct {
	store Store
	gate  *Gate
}

// NewManager wraps a Store with default gate filtering. The store must not be
// nil.
func NewManager(store Store) *Manager {
	return &Manager{store: store, gate: NewGate()}
}

// WithGate replaces the gate (useful for tests that want different
// thresholds).
func (m *Manager) WithGate(g *Gate) *Manager {
	if g != nil {
		m.gate = g
	}
	return m
}

// Store returns the underlying Store. Useful for advanced callers.
func (m *Manager) Store() Store { return m.store }

// AddQATurn records a turn for the given user/session. The turn ID and
// timestamp are populated if empty. HarmfulCount and Confidence are
// preserved; callers that want them gated should call PrepareTurn first.
func (m *Manager) AddQATurn(_ context.Context, userID, sessionID, question, answer, ctxStr string, feedbackScore *float64, usedIDs []string) (QATurn, error) {
	now := time.Now().UTC()
	turn := QATurn{
		ID:                  uuid.NewString(),
		UserID:              userID,
		SessionID:           sessionID,
		Question:            question,
		Answer:              answer,
		Context:             ctxStr,
		CreatedAt:           now,
		FeedbackScore:       feedbackScore,
		UsedGraphElementIDs: append([]string(nil), usedIDs...),
		Confidence:          MinGateConfidence,
	}
	if err := m.store.AppendTurn(userID, sessionID, turn); err != nil {
		return QATurn{}, err
	}
	return turn, nil
}

// GetSession returns the most recent lastN turns for a session. lastN <= 0
// returns all turns.
func (m *Manager) GetSession(_ context.Context, userID, sessionID string, lastN int) ([]QATurn, error) {
	return m.store.ListTurns(userID, sessionID, lastN)
}

// UpsertContext replaces the candidate entries for a user. Used by callers
// that want the distiller to consider a curated set of memories.
func (m *Manager) UpsertContext(_ context.Context, userID string, entries []ContextEntry) error {
	return m.store.UpsertContext(userID, entries)
}

// FilteredContext returns context entries that pass the gate. Order is
// preserved.
func (m *Manager) FilteredContext(userID string) ([]ContextEntry, error) {
	entries, err := m.store.ListContext(userID)
	if err != nil {
		return nil, err
	}
	return m.gate.AllowAll(entries), nil
}

// SaveLesson persists a distilled lesson.
func (m *Manager) SaveLesson(_ context.Context, userID string, lesson DistilledLesson) error {
	if lesson.ID == "" {
		lesson.ID = uuid.NewString()
	}
	if lesson.DistilledOn.IsZero() {
		lesson.DistilledOn = time.Now().UTC()
	}
	if lesson.UserID == "" {
		lesson.UserID = userID
	}
	return m.store.SaveLesson(userID, lesson)
}

// ListLessons returns all distilled lessons for a user.
func (m *Manager) ListLessons(_ context.Context, userID string) ([]DistilledLesson, error) {
	return m.store.ListLessons(userID)
}

// PrepareTurn performs a lightweight analysis of an incoming query and
// returns a TurnPreparation. The current implementation is a no-op pass
// (always ShouldAnswer=true with the original query). Hooks for safety and
// routing live here so call sites don't have to reimplement them.
func (m *Manager) PrepareTurn(_ context.Context, _ string, query string) TurnPreparation {
	return TurnPreparation{
		ShouldAnswer: true,
		EffectiveQ:   query,
	}
}
