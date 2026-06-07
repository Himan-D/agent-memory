package search

import (
	"strings"
	"unicode"
)

// QueryComplexity indicates the retrieval strategy to use for a query.
type QueryComplexity string

const (
	// ComplexitySimple routes to direct vector lookup.
	ComplexitySimple QueryComplexity = "simple"
	// ComplexityParallel routes to decompose + merge.
	ComplexityParallel QueryComplexity = "parallel"
	// ComplexityIterative routes to chain-of-query narrowing.
	ComplexityIterative QueryComplexity = "iterative"
)

// Router classifies queries and routes to the optimal retrieval strategy.
// Simple queries get direct vector search, complex queries get decomposition.
type Router struct {
	EntityThreshold int      // entity count threshold for parallel routing (default: 2)
	LengthThreshold int      // word count threshold (default: 15)
	HopIndicators   []string // words indicating multi-hop need
}

// NewRouter creates a Router with default thresholds and hop indicators.
func NewRouter() *Router {
	return &Router{
		EntityThreshold: 2,
		LengthThreshold: 15,
		HopIndicators: []string{
			"related to", "connected to", "because", "who also",
			"through", "via", "along with", "together with",
			"as well as", "in addition to", "combined with",
			"that also", "which also", "and also",
			"between", "among", "across",
		},
	}
}

// ClassifyQuery determines the retrieval strategy for a query based on heuristics:
// entity count, word count, and presence of multi-hop indicators.
func (r *Router) ClassifyQuery(query string) QueryComplexity {
	words := strings.Fields(query)
	wordCount := len(words)
	entityCount := countEntities(query)
	hasHopIndicators := r.hasHopIndicators(query)

	// Multi-hop indicators → iterative narrowing
	if hasHopIndicators && entityCount >= r.EntityThreshold {
		return ComplexityIterative
	}

	// Multiple entities or long query → parallel decomposition
	if entityCount >= r.EntityThreshold || wordCount > r.LengthThreshold {
		return ComplexityParallel
	}

	return ComplexitySimple
}

// DecomposeQuery splits a complex query into sub-queries for parallel search.
// It uses sentence boundaries, conjunctions, and entity boundaries as split points.
func (r *Router) DecomposeQuery(query string) []string {
	// Split on common conjunctions and question boundaries
	separators := []string{" and ", " but ", " or ", " while ", " whereas ", " also "}
	parts := []string{query}

	for _, sep := range separators {
		var newParts []string
		for _, p := range parts {
			subParts := strings.Split(p, sep)
			for _, sp := range subParts {
				sp = strings.TrimSpace(sp)
				if sp != "" {
					newParts = append(newParts, sp)
				}
			}
		}
		parts = newParts
	}

	// If no decomposition happened, try splitting on question marks
	if len(parts) <= 1 {
		qParts := strings.Split(query, "?")
		var filtered []string
		for _, qp := range qParts {
			qp = strings.TrimSpace(qp)
			if qp != "" {
				filtered = append(filtered, qp)
			}
		}
		if len(filtered) > 1 {
			parts = filtered
		}
	}

	// Deduplicate
	seen := make(map[string]bool)
	var unique []string
	for _, p := range parts {
		if !seen[p] {
			seen[p] = true
			unique = append(unique, p)
		}
	}

	return unique
}

// hasHopIndicators checks if the query contains any multi-hop indicator phrases.
func (r *Router) hasHopIndicators(query string) bool {
	lower := strings.ToLower(query)
	for _, indicator := range r.HopIndicators {
		if strings.Contains(lower, indicator) {
			return true
		}
	}
	return false
}

// countEntities estimates the number of named entities in a query.
// Uses a heuristic: sequences of capitalized words not at sentence start.
func countEntities(query string) int {
	words := strings.Fields(query)
	if len(words) == 0 {
		return 0
	}

	entityCount := 0
	inEntity := false

	for i, word := range words {
		// Skip first word (always capitalized in a sentence)
		if i == 0 {
			continue
		}

		// Check if word starts with uppercase
		runes := []rune(word)
		if len(runes) > 0 && unicode.IsUpper(runes[0]) {
			if !inEntity {
				entityCount++
				inEntity = true
			}
		} else {
			inEntity = false
		}
	}

	return entityCount
}
