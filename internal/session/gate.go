package session

import (
	"agent-memory/internal/memory/safety"
)

// Gate implements Cognee's gate filter for distillation candidates. Entries
// pass when they have no harmful flags and meet the minimum confidence
// threshold.
//
// Scoring is now real: each ContextEntry carries a Confidence and a
// HarmfulCount populated by LoadScores (typically from the source memory's
// safety classification + composite scoring). The gate reads those fields
// and falls back to package defaults when scores are unset (zero values).
type Gate struct {
	MinConfidence float64
}

// SafetyClassifier is the subset of safety.Classifier used by LoadScores.
// Defined as an interface to avoid making the session package depend on
// the safety package's full surface.
type SafetyClassifier interface {
	Classify(content string) *safety.ClassificationResult
}

// ConfidenceScorer is the subset of scoring.CompositeScorer used by
// LoadScores. Returns a score in [0, 1].
type ConfidenceScorer interface {
	Score(content string) float64
}

// NewGate returns a Gate using the package default MinGateConfidence.
func NewGate() *Gate {
	return &Gate{MinConfidence: MinGateConfidence}
}

// Allow returns true when entry is safe to feed into the curator.
func (g *Gate) Allow(entry ContextEntry) bool {
	if entry.HarmfulCount > 0 {
		return false
	}
	if entry.Confidence < g.minConfidence() {
		return false
	}
	return true
}

// AllowAll returns the entries that pass the gate, preserving order.
func (g *Gate) AllowAll(entries []ContextEntry) []ContextEntry {
	out := make([]ContextEntry, 0, len(entries))
	for _, e := range entries {
		if g.Allow(e) {
			out = append(out, e)
		}
	}
	return out
}

func (g *Gate) minConfidence() float64 {
	if g.MinConfidence <= 0 {
		return MinGateConfidence
	}
	return g.MinConfidence
}

// LoadScores populates the entry's Confidence and HarmfulCount fields using
// the supplied safety classifier and confidence scorer. The entry is
// modified in place; the returned value is the same pointer for chaining.
//
// Behavior:
//   - HarmfulCount is set to 0 when content is safe, or 1 when any
//     harmful pattern is detected.
//   - Confidence is set to the scorer's value when provided. If the scorer
//     returns 0, the entry's existing confidence (if any) is preserved.
//   - When either dependency is nil, the corresponding field is left
//     untouched, so the gate's default behavior still applies.
func LoadScores(entry *ContextEntry, classifier SafetyClassifier, scorer ConfidenceScorer) *ContextEntry {
	if entry == nil {
		return nil
	}
	if classifier != nil {
		if r := classifier.Classify(entry.Content); r != nil && !r.Safe {
			entry.HarmfulCount = 1
		}
	}
	if scorer != nil {
		if s := scorer.Score(entry.Content); s > 0 {
			entry.Confidence = s
		}
	}
	return entry
}

// LoadScoresAll applies LoadScores to every entry and returns the slice for
// chaining.
func LoadScoresAll(entries []ContextEntry, classifier SafetyClassifier, scorer ConfidenceScorer) []ContextEntry {
	for i := range entries {
		LoadScores(&entries[i], classifier, scorer)
	}
	return entries
}
