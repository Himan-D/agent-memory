package history

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// Message represents a single conversation turn for indexing purposes.
type Message struct {
	Role    string
	Content string
}

// ConversationReference holds a summarized record of a past conversation session.
// This implements ChatGPT's "Reference Chat History" — the implicit pattern layer
// that indexes past conversations and retrieves relevant history for new queries.
type ConversationReference struct {
	SessionID string    `json:"session_id"`
	UserID    string    `json:"user_id"`
	Summary   string    `json:"summary"`
	Topics    []string  `json:"topics"`
	CreatedAt time.Time `json:"created_at"`
}

// ReferenceStore indexes past conversations and retrieves relevant history for
// new queries. Unlike saved memories (explicit facts), reference history captures
// the implicit patterns and topics from complete conversation sessions.
type ReferenceStore struct {
	mu         sync.RWMutex
	references map[string][]ConversationReference // userID -> references
}

// NewReferenceStore creates a new empty reference store.
func NewReferenceStore() *ReferenceStore {
	return &ReferenceStore{
		references: make(map[string][]ConversationReference),
	}
}

// IndexConversation extracts key patterns from a completed conversation session
// and stores them as a reference. It builds a summary from the message content
// and extracts topic keywords using a simple heuristic (words > 5 chars).
func (rs *ReferenceStore) IndexConversation(userID, sessionID string, messages []Message) error {
	if userID == "" || sessionID == "" {
		return fmt.Errorf("history: index: userID and sessionID are required")
	}
	if len(messages) == 0 {
		return fmt.Errorf("history: index: no messages to index")
	}

	// Build a summary from the conversation
	var summaryParts []string
	var allContent strings.Builder
	for _, msg := range messages {
		if msg.Role == "user" && len(msg.Content) > 0 {
			// Take first sentence or first 100 chars as summary material
			snippet := msg.Content
			if len(snippet) > 100 {
				snippet = snippet[:100]
			}
			summaryParts = append(summaryParts, snippet)
		}
		allContent.WriteString(msg.Content)
		allContent.WriteString(" ")
	}

	summary := "Conversation"
	if len(summaryParts) > 0 {
		summary = strings.Join(summaryParts, "; ")
		if len(summary) > 300 {
			summary = summary[:300] + "..."
		}
	}

	// Extract topics using a simple heuristic: words > 5 chars, deduplicated
	topics := extractTopics(allContent.String())

	ref := ConversationReference{
		SessionID: sessionID,
		UserID:    userID,
		Summary:   summary,
		Topics:    topics,
		CreatedAt: time.Now(),
	}

	rs.mu.Lock()
	defer rs.mu.Unlock()

	// Dedup: replace if same session already indexed
	refs := rs.references[userID]
	for i, existing := range refs {
		if existing.SessionID == sessionID {
			refs[i] = ref
			return nil
		}
	}
	rs.references[userID] = append(refs, ref)
	return nil
}

// FindRelevant retrieves past conversation references relevant to the current query.
// Scoring is based on keyword overlap between the query and the reference topics/summary.
func (rs *ReferenceStore) FindRelevant(userID, query string, limit int) []ConversationReference {
	rs.mu.RLock()
	defer rs.mu.RUnlock()

	if limit <= 0 {
		limit = 5
	}

	refs := rs.references[userID]
	if len(refs) == 0 {
		return []ConversationReference{}
	}

	queryTerms := strings.Fields(strings.ToLower(query))
	if len(queryTerms) == 0 {
		return []ConversationReference{}
	}

	type scored struct {
		ref   ConversationReference
		score int
	}

	var scoredRefs []scored
	for _, ref := range refs {
		score := 0
		summaryLower := strings.ToLower(ref.Summary)
		for _, term := range queryTerms {
			if strings.Contains(summaryLower, term) {
				score += 2
			}
			for _, topic := range ref.Topics {
				if strings.Contains(strings.ToLower(topic), term) {
					score += 3 // topic match is stronger signal
				}
			}
		}
		if score > 0 {
			scoredRefs = append(scoredRefs, scored{ref: ref, score: score})
		}
	}

	// Sort by score descending
	for i := 0; i < len(scoredRefs); i++ {
		for j := i + 1; j < len(scoredRefs); j++ {
			if scoredRefs[j].score > scoredRefs[i].score {
				scoredRefs[i], scoredRefs[j] = scoredRefs[j], scoredRefs[i]
			}
		}
	}

	if len(scoredRefs) > limit {
		scoredRefs = scoredRefs[:limit]
	}

	results := make([]ConversationReference, len(scoredRefs))
	for i, s := range scoredRefs {
		results[i] = s.ref
	}
	return results
}

// FormatForPrompt renders relevant references as a prompt-ready string.
func (rs *ReferenceStore) FormatForPrompt(refs []ConversationReference) string {
	if len(refs) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("<past_conversations>\n")
	for _, ref := range refs {
		sb.WriteString(fmt.Sprintf("- [%s] %s", ref.SessionID, ref.Summary))
		if len(ref.Topics) > 0 {
			sb.WriteString(fmt.Sprintf(" (topics: %s)", strings.Join(ref.Topics, ", ")))
		}
		sb.WriteString("\n")
	}
	sb.WriteString("</past_conversations>")
	return sb.String()
}

// Count returns the total number of indexed conversation references for a user.
func (rs *ReferenceStore) Count(userID string) int {
	rs.mu.RLock()
	defer rs.mu.RUnlock()
	return len(rs.references[userID])
}

// extractTopics extracts topic keywords from text using a simple heuristic:
// words longer than 5 characters, deduplicated, limited to 10.
func extractTopics(text string) []string {
	words := strings.Fields(strings.ToLower(text))
	seen := make(map[string]bool)
	var topics []string

	// Common stop words to exclude
	stopWords := map[string]bool{
		"about": true, "would": true, "could": true, "should": true,
		"there": true, "their": true, "these": true, "those": true,
		"which": true, "where": true, "while": true, "through": true,
		"between": true, "before": true, "after": true, "during": true,
		"because": true, "really": true, "actually": true, "something": true,
	}

	for _, w := range words {
		// Clean punctuation
		w = strings.Trim(w, ".,;:!?\"'()[]{}/-")
		if len(w) <= 5 || seen[w] || stopWords[w] {
			continue
		}
		seen[w] = true
		topics = append(topics, w)
		if len(topics) >= 10 {
			break
		}
	}
	return topics
}
