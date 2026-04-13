package feedback

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

type FeedbackType string

const (
	FeedbackTypePositive FeedbackType = "positive"
	FeedbackTypeNegative FeedbackType = "negative"
	FeedbackTypeNeutral  FeedbackType = "neutral"
)

type Feedback struct {
	ID        string                 `json:"id"`
	QueryID   string                 `json:"query_id"`
	Type      FeedbackType           `json:"type"`
	Score     float64                `json:"score"`
	Comment   string                 `json:"comment,omitempty"`
	ResultID  string                 `json:"result_id,omitempty"`
	SessionID string                 `json:"session_id,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
}

type Store interface {
	Save(f *Feedback) error
	GetByQuery(queryID string) ([]*Feedback, error)
	GetBySession(sessionID string) ([]*Feedback, error)
	Delete(id string) error
}

type MemoryStore struct {
	mu        sync.RWMutex
	feedbacks map[string]*Feedback
	byQuery   map[string][]string
	bySession map[string][]string
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		feedbacks: make(map[string]*Feedback),
		byQuery:   make(map[string][]string),
		bySession: make(map[string][]string),
	}
}

func (s *MemoryStore) Save(f *Feedback) error {
	if f.ID == "" {
		return fmt.Errorf("feedback ID is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	f.CreatedAt = time.Now()
	s.feedbacks[f.ID] = f

	s.byQuery[f.QueryID] = append(s.byQuery[f.QueryID], f.ID)

	if f.SessionID != "" {
		s.bySession[f.SessionID] = append(s.bySession[f.SessionID], f.ID)
	}

	return nil
}

func (s *MemoryStore) GetByQuery(queryID string) ([]*Feedback, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ids := s.byQuery[queryID]
	results := make([]*Feedback, 0, len(ids))

	for _, id := range ids {
		if f, ok := s.feedbacks[id]; ok {
			results = append(results, f)
		}
	}

	return results, nil
}

func (s *MemoryStore) GetBySession(sessionID string) ([]*Feedback, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ids := s.bySession[sessionID]
	results := make([]*Feedback, 0, len(ids))

	for _, id := range ids {
		if f, ok := s.feedbacks[id]; ok {
			results = append(results, f)
		}
	}

	return results, nil
}

func (s *MemoryStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	f, ok := s.feedbacks[id]
	if !ok {
		return fmt.Errorf("feedback not found: %s", id)
	}

	delete(s.feedbacks, id)

	if ids, ok := s.byQuery[f.QueryID]; ok {
		newIDs := make([]string, 0, len(ids))
		for _, existingID := range ids {
			if existingID != id {
				newIDs = append(newIDs, existingID)
			}
		}
		s.byQuery[f.QueryID] = newIDs
	}

	if f.SessionID != "" {
		if ids, ok := s.bySession[f.SessionID]; ok {
			newIDs := make([]string, 0, len(ids))
			for _, existingID := range ids {
				if existingID != id {
					newIDs = append(newIDs, existingID)
				}
			}
			s.bySession[f.SessionID] = newIDs
		}
	}

	return nil
}

type FeedbackManager struct {
	store Store
}

func NewManager(store Store) *FeedbackManager {
	if store == nil {
		store = NewMemoryStore()
	}
	return &FeedbackManager{store: store}
}

func (m *FeedbackManager) Add(queryID string, fbType FeedbackType, score float64, resultID, sessionID string) (*Feedback, error) {
	f := &Feedback{
		ID:        generateID(),
		QueryID:   queryID,
		Type:      fbType,
		Score:     score,
		ResultID:  resultID,
		SessionID: sessionID,
		Metadata:  make(map[string]interface{}),
	}

	if err := m.store.Save(f); err != nil {
		return nil, err
	}

	return f, nil
}

func (m *FeedbackManager) AddComment(feedbackID, comment string) error {
	feedbacks, err := m.store.GetByQuery("")
	if err != nil {
		return err
	}

	for _, f := range feedbacks {
		if f.ID == feedbackID {
			f.Comment = comment
			return nil
		}
	}

	return fmt.Errorf("feedback not found: %s", feedbackID)
}

func (m *FeedbackManager) GetByQuery(queryID string) ([]*Feedback, error) {
	return m.store.GetByQuery(queryID)
}

func (m *FeedbackManager) GetBySession(sessionID string) ([]*Feedback, error) {
	return m.store.GetBySession(sessionID)
}

func (m *FeedbackManager) Delete(id string) error {
	return m.store.Delete(id)
}

func (m *FeedbackManager) GetAverageScore(queryID string) (float64, error) {
	feedbacks, err := m.GetByQuery(queryID)
	if err != nil {
		return 0, err
	}

	if len(feedbacks) == 0 {
		return 0, nil
	}

	var sum float64
	for _, f := range feedbacks {
		sum += f.Score
	}

	return sum / float64(len(feedbacks)), nil
}

func (m *FeedbackManager) GetImprovements(queryID string) ([]string, error) {
	feedbacks, err := m.GetByQuery(queryID)
	if err != nil {
		return nil, err
	}

	var improvements []string
	for _, f := range feedbacks {
		if f.Type == FeedbackTypeNegative && f.Comment != "" {
			improvements = append(improvements, f.Comment)
		}
	}

	return improvements, nil
}

func generateID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}
