package session

import (
	"context"
	"errors"
	"strings"
	"unicode"
)

// Neo4jNoveltyWriterConfig configures a Neo4jNoveltyWriter.
type Neo4jNoveltyWriterConfig struct {
	// SimilarityThreshold is the max normalized similarity above which a
	// proposed lesson is rejected as "already_known". 0.0 disables the
	// check (always passes the novelty gate).
	SimilarityThreshold float64

	// MinEntities is the minimum number of capitalized noun phrases
	// required for the lesson to be considered entity-anchored. Below
	// this, the lesson is rejected as "not_durable".
	MinEntities int

	// MaxStatementChars truncates overly long statements. 0 disables.
	MaxStatementChars int

	// StopWords are entity-candidates to ignore when counting entity
	// anchors. Always includes common sentence starters; callers can
	// extend.
	StopWords []string
}

// DefaultNeo4jNoveltyWriterConfig returns the production defaults.
func DefaultNeo4jNoveltyWriterConfig() Neo4jNoveltyWriterConfig {
	return Neo4jNoveltyWriterConfig{
		SimilarityThreshold: 0.92,
		MinEntities:         1,
		MaxStatementChars:   500,
		StopWords:           defaultEntityStopWords(),
	}
}

// Neo4jNoveltyWriter is a WriterFunc that rejects proposals which duplicate
// existing lessons (high similarity in priorLessons) or lack entity
// anchoring. It accepts otherwise and extracts entity names from the
// statement via a simple heuristic (capitalized noun phrases).
//
// The novelty search itself is performed by a NoveltySearchFunc wired into
// DistillOptions; this writer consumes the result and makes the
// accept/reject decision deterministically.
type Neo4jNoveltyWriter struct {
	cfg Neo4jNoveltyWriterConfig
}

// NewNeo4jNoveltyWriter returns a writer with the given config. Pass
// DefaultNeo4jNoveltyWriterConfig() for production defaults.
func NewNeo4jNoveltyWriter(cfg Neo4jNoveltyWriterConfig) *Neo4jNoveltyWriter {
	if cfg.MinEntities <= 0 {
		cfg.MinEntities = 1
	}
	return &Neo4jNoveltyWriter{cfg: cfg}
}

// Evaluate implements WriterFunc.
func (w *Neo4jNoveltyWriter) Evaluate(ctx context.Context, proposed ProposedLesson, members []string, priorLessons []string, glossary []string) (WrittenLesson, error) {
	if err := ctx.Err(); err != nil {
		return WrittenLesson{}, err
	}
	statement := strings.TrimSpace(proposed.WorkingStatement)
	if statement == "" {
		return WrittenLesson{Accept: false, Reason: "not_durable"}, nil
	}
	if w.cfg.MaxStatementChars > 0 && len(statement) > w.cfg.MaxStatementChars {
		statement = statement[:w.cfg.MaxStatementChars] + "..."
	}

	if w.isDuplicate(statement, priorLessons) {
		return WrittenLesson{Accept: false, Reason: "already_known"}, nil
	}

	entities := w.extractEntities(statement)
	if len(entities) < w.cfg.MinEntities {
		return WrittenLesson{Accept: false, Reason: "not_durable"}, nil
	}

	why := ""
	if len(members) > 0 {
		why = "derived from session members"
	}
	return WrittenLesson{
		Accept:     true,
		Statement:  statement,
		Entities:   entities,
		WhyLearned: why,
	}, nil
}

// EvaluateAsFunc adapts Evaluate to the WriterFunc signature used by
// session.DistillOptions.Writer.
func (w *Neo4jNoveltyWriter) EvaluateAsFunc() WriterFunc {
	return w.Evaluate
}

// isDuplicate performs a coarse similarity check against prior lessons.
// Real implementations use vector cosine similarity; this implementation
// uses normalized substring containment as a conservative fallback so the
// gate stays deterministic when the novelty search returns only literal
// prior statements (no embeddings).
func (w *Neo4jNoveltyWriter) isDuplicate(statement string, priorLessons []string) bool {
	if w.cfg.SimilarityThreshold <= 0 || len(priorLessons) == 0 {
		return false
	}
	norm := normalizeForCompare(statement)
	if norm == "" {
		return false
	}
	for _, prior := range priorLessons {
		pNorm := normalizeForCompare(prior)
		if pNorm == "" {
			continue
		}
		// Containment check: if either is a substring of the other, treat
		// as duplicate. This is conservative -- vector similarity would
		// catch more semantic duplicates but needs embeddings.
		if strings.Contains(norm, pNorm) || strings.Contains(pNorm, norm) {
			return true
		}
	}
	return false
}

// extractEntities pulls capitalized noun phrases from the statement using
// a simple heuristic: runs of consecutive capitalized words. The result
// is de-duplicated and filtered against the stop-word list.
func (w *Neo4jNoveltyWriter) extractEntities(statement string) []string {
	stop := make(map[string]struct{}, len(w.cfg.StopWords))
	for _, s := range w.cfg.StopWords {
		stop[strings.ToLower(s)] = struct{}{}
	}

	seen := make(map[string]struct{})
	var out []string

	words := strings.Fields(statement)
	for i, word := range words {
		// Strip leading/trailing punctuation for the capitalization check.
		trimmed := strings.TrimFunc(word, func(r rune) bool {
			return !unicode.IsLetter(r) && !unicode.IsDigit(r)
		})
		if trimmed == "" {
			continue
		}
		if !unicode.IsUpper([]rune(trimmed)[0]) {
			continue
		}
		// Skip stop words like "The", "I", etc.
		if _, ok := stop[strings.ToLower(trimmed)]; ok {
			continue
		}
		// Collect this word and any immediately-following capitalized words
		// as one phrase.
		phrase := trimmed
		for j := i + 1; j < len(words); j++ {
			next := strings.TrimFunc(words[j], func(r rune) bool {
				return !unicode.IsLetter(r) && !unicode.IsDigit(r)
			})
			if next == "" {
				break
			}
			if !unicode.IsUpper([]rune(next)[0]) {
				break
			}
			if _, ok := stop[strings.ToLower(next)]; ok {
				break
			}
			phrase += " " + next
		}
		// Skip if it's a single stop word that already passed.
		if _, ok := stop[strings.ToLower(phrase)]; ok {
			continue
		}
		if _, dup := seen[phrase]; dup {
			continue
		}
		seen[phrase] = struct{}{}
		out = append(out, phrase)
	}
	return out
}

func normalizeForCompare(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
		case unicode.IsSpace(r):
			b.WriteByte(' ')
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

func defaultEntityStopWords() []string {
	return []string{
		"I", "We", "They", "The", "A", "An", "This", "That", "These", "Those",
		"It", "Its", "He", "She", "His", "Her", "Their", "Our", "Your",
		"When", "Where", "How", "Why", "What", "Which", "Who",
		"Today", "Yesterday", "Tomorrow", "Now", "Then",
	}
}

// Ensure the writer satisfies WriterFunc at compile time.
var _ WriterFunc = (*Neo4jNoveltyWriter)(nil).Evaluate

// ErrWriterUnavailable is returned when the writer is wired without a
// novelty search backend. Callers can use errors.Is to detect this case.
var ErrWriterUnavailable = errors.New("novelty writer unavailable")
