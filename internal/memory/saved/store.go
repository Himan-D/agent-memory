package saved

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// SavedMemory represents a single explicit, user-visible, toggleable fact.
// This implements ChatGPT's "Saved Memories" pattern where each entry is a
// discrete piece of knowledge the system has extracted from conversation.
type SavedMemory struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Fact      string    `json:"fact"`
	Category  string    `json:"category"` // preference, fact, decision, goal, skill, constraint
	Source    string    `json:"source"`    // conversation/session that created it
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// LLMClassifier determines whether a message contains a fact worth saving.
type LLMClassifier interface {
	ClassifyForSaving(ctx context.Context, message string, existingFacts []SavedMemory) (*SavedMemory, bool, error)
}

// Store implements the ChatGPT "Saved Memories" dual-layer pattern.
// It maintains an explicit list of user facts that can be toggled on/off,
// searched, and injected into prompts.
type Store struct {
	mu       sync.RWMutex
	memories map[string][]SavedMemory // userID -> facts
	llm      LLMClassifier
}

// NewStore creates a new saved memory store with an optional LLM classifier.
// If classifier is nil, ProcessMessage will always return nil (no auto-extraction).
func NewStore(classifier LLMClassifier) *Store {
	return &Store{
		memories: make(map[string][]SavedMemory),
		llm:      classifier,
	}
}

// ProcessMessage runs the LLM classifier to decide if a message contains a saveable fact.
// Returns the saved memory if one was extracted, or nil if the message was not noteworthy.
func (s *Store) ProcessMessage(ctx context.Context, userID, message, sessionID string) (*SavedMemory, error) {
	if s.llm == nil {
		return nil, nil
	}

	s.mu.RLock()
	existing := make([]SavedMemory, len(s.memories[userID]))
	copy(existing, s.memories[userID])
	s.mu.RUnlock()

	mem, shouldSave, err := s.llm.ClassifyForSaving(ctx, message, existing)
	if err != nil {
		return nil, fmt.Errorf("saved: process message: classify: %w", err)
	}
	if !shouldSave || mem == nil {
		return nil, nil
	}

	mem.UserID = userID
	mem.Source = sessionID
	if mem.ID == "" {
		mem.ID = uuid.New().String()
	}
	mem.Active = true
	mem.CreatedAt = time.Now()
	mem.UpdatedAt = time.Now()

	if err := s.Save(userID, *mem); err != nil {
		return nil, fmt.Errorf("saved: process message: save: %w", err)
	}

	return mem, nil
}

// Save stores a fact with deduplication check. If a fact with identical content
// already exists for the user, the existing fact is updated instead of creating a duplicate.
func (s *Store) Save(userID string, mem SavedMemory) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Dedup check: skip if an identical active fact already exists
	for i, existing := range s.memories[userID] {
		if existing.Active && strings.EqualFold(strings.TrimSpace(existing.Fact), strings.TrimSpace(mem.Fact)) {
			// Update existing instead of duplicating
			s.memories[userID][i].UpdatedAt = time.Now()
			if mem.Category != "" {
				s.memories[userID][i].Category = mem.Category
			}
			return nil
		}
	}

	if mem.ID == "" {
		mem.ID = uuid.New().String()
	}
	if mem.CreatedAt.IsZero() {
		mem.CreatedAt = time.Now()
	}
	mem.UpdatedAt = time.Now()
	mem.UserID = userID

	s.memories[userID] = append(s.memories[userID], mem)
	return nil
}

// GetAll returns all active saved memories for a user.
func (s *Store) GetAll(userID string) []SavedMemory {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var active []SavedMemory
	for _, m := range s.memories[userID] {
		if m.Active {
			active = append(active, m)
		}
	}
	if active == nil {
		return []SavedMemory{}
	}
	return active
}

// Search finds relevant facts by keyword/pattern matching against fact content.
func (s *Store) Search(userID, query string) []SavedMemory {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if query == "" {
		return s.GetAll(userID)
	}

	queryLower := strings.ToLower(query)
	queryTerms := strings.Fields(queryLower)

	var matches []SavedMemory
	for _, m := range s.memories[userID] {
		if !m.Active {
			continue
		}
		factLower := strings.ToLower(m.Fact)
		categoryLower := strings.ToLower(m.Category)

		matched := false
		for _, term := range queryTerms {
			if strings.Contains(factLower, term) || strings.Contains(categoryLower, term) {
				matched = true
				break
			}
		}
		if matched {
			matches = append(matches, m)
		}
	}

	if matches == nil {
		return []SavedMemory{}
	}
	return matches
}

// Delete removes a saved memory by ID.
func (s *Store) Delete(userID, memoryID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	facts := s.memories[userID]
	for i, m := range facts {
		if m.ID == memoryID {
			s.memories[userID] = append(facts[:i], facts[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("saved: delete: memory %s not found for user %s", memoryID, userID)
}

// Toggle enables or disables a saved memory.
func (s *Store) Toggle(userID, memoryID string, active bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, m := range s.memories[userID] {
		if m.ID == memoryID {
			s.memories[userID][i].Active = active
			s.memories[userID][i].UpdatedAt = time.Now()
			return nil
		}
	}
	return fmt.Errorf("saved: toggle: memory %s not found for user %s", memoryID, userID)
}

// InjectIntoPrompt formats all active facts for system prompt injection.
// Returns an empty string if the user has no active saved memories.
func (s *Store) InjectIntoPrompt(userID string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var active []SavedMemory
	for _, m := range s.memories[userID] {
		if m.Active {
			active = append(active, m)
		}
	}

	if len(active) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("<saved_memories>\n")

	// Group by category for readability
	byCategory := make(map[string][]SavedMemory)
	for _, m := range active {
		cat := m.Category
		if cat == "" {
			cat = "general"
		}
		byCategory[cat] = append(byCategory[cat], m)
	}

	for cat, facts := range byCategory {
		sb.WriteString(fmt.Sprintf("[%s]\n", cat))
		for _, f := range facts {
			sb.WriteString(fmt.Sprintf("- %s\n", f.Fact))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("</saved_memories>")
	return sb.String()
}

// Count returns the total number of memories (active + inactive) for a user.
func (s *Store) Count(userID string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.memories[userID])
}
