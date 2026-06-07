package search

import "strings"

// GranularityRouter selects the optimal memory tier for a query.
// Based on TriMem paper (arXiv:2605.19952).
type GranularityRouter struct{}

// Granularity represents the memory granularity tier.
type Granularity string

const (
	GranularityRaw     Granularity = "raw"     // full dialogue segments
	GranularityFact    Granularity = "fact"     // atomic extracted facts
	GranularityProfile Granularity = "profile"  // synthesized user profiles
)

// NewGranularityRouter creates a new GranularityRouter.
func NewGranularityRouter() *GranularityRouter {
	return &GranularityRouter{}
}

// profileSignals are query terms that suggest the user is asking about
// preferences, personality, or synthesized profile information.
var profileSignals = []string{
	"prefer", "personality", "like", "dislike", "favorite", "style",
	"habit", "tendency", "always", "never", "usually", "typically",
	"kind of person", "type of", "describe me", "who am i",
	"what do i like", "what do i prefer",
}

// factSignals are query terms that suggest a factoid question.
var factSignals = []string{
	"who is", "what is", "when did", "where is", "how many",
	"what was", "who was", "where did", "when was",
	"name of", "date of", "number of", "age of",
}

// rawSignals are query terms that suggest the user wants conversation
// context or full dialogue history.
var rawSignals = []string{
	"conversation", "said", "discussed", "talked about", "mentioned",
	"last time", "previous session", "chat history", "our discussion",
	"what did i say", "what did you say", "context",
}

// Route determines which memory granularity to query based on query characteristics.
// Heuristics:
//   - Profile questions (preferences, personality) -> profile
//   - Factoid questions (who, what, when, where) -> fact
//   - Context/history questions -> raw
//   - Default -> fact
func (gr *GranularityRouter) Route(query string) Granularity {
	lower := strings.ToLower(query)

	// Check profile signals first -- these are the most specific.
	for _, signal := range profileSignals {
		if strings.Contains(lower, signal) {
			return GranularityProfile
		}
	}

	// Check raw/context signals.
	for _, signal := range rawSignals {
		if strings.Contains(lower, signal) {
			return GranularityRaw
		}
	}

	// Check factoid signals.
	for _, signal := range factSignals {
		if strings.Contains(lower, signal) {
			return GranularityFact
		}
	}

	// Default to fact granularity.
	return GranularityFact
}
