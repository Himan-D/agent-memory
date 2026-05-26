package temporal

import "strings"

// VolatilityClassifier determines how quickly a memory's temporal relevance decays.
// Stable facts (birthdate, name) have low volatility. Dynamic facts (job, address) have high volatility.
// Scores range from 0.0 (permanent/immutable) to 1.0 (highly transient).
type VolatilityClassifier struct {
	// predicateScores maps relation predicates to hand-curated volatility scores.
	predicateScores map[string]float64
}

// NewVolatilityClassifier creates a classifier with a curated predicate-to-volatility map.
func NewVolatilityClassifier() *VolatilityClassifier {
	vc := &VolatilityClassifier{
		predicateScores: make(map[string]float64),
	}

	// Very stable (0.05) — essentially permanent facts
	for _, p := range []string{"born_in", "birthdate", "date_of_birth", "birth_year", "name", "full_name", "maiden_name"} {
		vc.predicateScores[p] = 0.05
	}

	// Stable (0.1) — rarely change
	for _, p := range []string{"nationality", "gender", "ethnicity", "blood_type", "species", "native_language"} {
		vc.predicateScores[p] = 0.1
	}

	// Moderately stable (0.2) — change over years
	for _, p := range []string{"alma_mater", "graduated_from", "degree", "education", "religion", "mother", "father", "sibling"} {
		vc.predicateScores[p] = 0.2
	}

	// Moderate (0.4) — change occasionally
	for _, p := range []string{"hobby", "interest", "skill", "speaks", "certified_in", "member_of"} {
		vc.predicateScores[p] = 0.4
	}

	// Moderate-high (0.6) — change with some regularity
	for _, p := range []string{"uses", "prefers", "likes", "dislikes", "favorite", "owns", "drives", "subscribes_to"} {
		vc.predicateScores[p] = 0.6
	}

	// High (0.7) — change within months/year
	for _, p := range []string{"lives_in", "address", "location", "city", "resides_in", "phone_number", "email"} {
		vc.predicateScores[p] = 0.7
	}

	// Very high (0.8) — change within weeks/months
	for _, p := range []string{"works_at", "employed_by", "job_title", "role", "position", "salary", "team", "manager", "reports_to"} {
		vc.predicateScores[p] = 0.8
	}

	// Extremely high (0.85) — frequently change
	for _, p := range []string{"relationship_status", "dating", "partner", "married_to"} {
		vc.predicateScores[p] = 0.85
	}

	// Near-transient (0.9) — change within days/weeks
	for _, p := range []string{"current_project", "working_on", "assigned_to", "mood", "status", "availability", "current_task"} {
		vc.predicateScores[p] = 0.9
	}

	return vc
}

// Classify returns a volatility score [0.0, 1.0] for a given predicate/relation type.
// Falls back to 0.5 (neutral) for unknown predicates.
func (vc *VolatilityClassifier) Classify(predicate string) float64 {
	normalized := strings.ToLower(strings.TrimSpace(predicate))

	if score, ok := vc.predicateScores[normalized]; ok {
		return score
	}

	return 0.5
}

// ClassifyContent uses heuristics on memory content to estimate volatility when no predicate is available.
// It scans for keywords that indicate temporal stability or instability.
func (vc *VolatilityClassifier) ClassifyContent(content string) float64 {
	lower := strings.ToLower(content)

	// High-volatility indicators — transient state
	highVolatility := []string{
		"currently", "right now", "at the moment", "these days",
		"this week", "today", "working on", "just started",
		"recently", "planning to", "about to", "will be",
	}
	// Low-volatility indicators — permanent facts
	lowVolatility := []string{
		"born in", "was born", "birthdate", "maiden name",
		"nationality", "always", "since childhood", "grew up in",
		"native", "permanently", "never changes",
	}
	// Medium-volatility indicators
	mediumVolatility := []string{
		"lives in", "works at", "employed", "married",
		"dating", "relationship", "moved to", "started at",
		"job", "position", "address",
	}

	var highCount, lowCount, medCount int
	for _, kw := range highVolatility {
		if strings.Contains(lower, kw) {
			highCount++
		}
	}
	for _, kw := range lowVolatility {
		if strings.Contains(lower, kw) {
			lowCount++
		}
	}
	for _, kw := range mediumVolatility {
		if strings.Contains(lower, kw) {
			medCount++
		}
	}

	total := highCount + lowCount + medCount
	if total == 0 {
		return 0.5 // neutral default
	}

	// Weighted average: high=0.85, medium=0.6, low=0.1
	weightedSum := float64(highCount)*0.85 + float64(medCount)*0.6 + float64(lowCount)*0.1
	return weightedSum / float64(total)
}
