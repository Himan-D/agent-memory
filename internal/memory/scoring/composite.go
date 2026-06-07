package scoring

import (
	"math"

	"agent-memory/internal/memory/types"
)

// CompositeScorer combines five orthogonal signals into a single importance score,
// with an additional negation penalty.
// Signals: (1) semantic relevance, (2) temporal validity, (3) confidence/MW,
// (4) graph centrality, (5) source authority.
// Each signal is independently normalized to [0, 1] and combined via weighted sum.
// A negation penalty (up to 0.10) is subtracted when conflicting/negated memories are detected.
type CompositeScorer struct {
	SemanticWeight   float64 // default 0.30
	TemporalWeight   float64 // default 0.20
	ConfidenceWeight float64 // default 0.15
	CentralityWeight float64 // default 0.10
	AuthorityWeight  float64 // default 0.15
	// Remaining 0.10 is reserved for negation penalty headroom.
}

// ScoreInput holds the five orthogonal signals plus negation penalty for composite scoring.
type ScoreInput struct {
	SemanticScore   float64 // from vector similarity [0, 1]
	TemporalScore   float64 // from temporal scorer [0, 1]
	ConfidenceScore float64 // from MW counters [0, 1]
	CentralityScore float64 // from graph degree [0, 1]
	AuthorityScore  float64 // from source type [0, 1]
	NegationPenalty float64 // 0.0 = no negation conflict, 1.0 = direct contradiction
}

// ScoreOutput holds the composite score and per-signal breakdown.
type ScoreOutput struct {
	CompositeScore float64
	Breakdown      map[string]float64 // individual signal contributions
}

// NewCompositeScorer creates a CompositeScorer with default weights
// (0.30, 0.20, 0.15, 0.10, 0.15) plus 0.10 negation penalty headroom.
func NewCompositeScorer() *CompositeScorer {
	return &CompositeScorer{
		SemanticWeight:   0.30,
		TemporalWeight:   0.20,
		ConfidenceWeight: 0.15,
		CentralityWeight: 0.10,
		AuthorityWeight:  0.15,
	}
}

// Score computes a weighted composite from the five signals, minus negation penalty.
func (cs *CompositeScorer) Score(input ScoreInput) ScoreOutput {
	semanticContrib := cs.SemanticWeight * input.SemanticScore
	temporalContrib := cs.TemporalWeight * input.TemporalScore
	confidenceContrib := cs.ConfidenceWeight * input.ConfidenceScore
	centralityContrib := cs.CentralityWeight * input.CentralityScore
	authorityContrib := cs.AuthorityWeight * input.AuthorityScore

	// Negation penalty: up to 0.10 deducted for contradicted memories.
	negationPenalty := 0.10 * clamp01(input.NegationPenalty)

	composite := semanticContrib + temporalContrib + confidenceContrib +
		centralityContrib + authorityContrib - negationPenalty

	// Clamp final score to [0, 1].
	if composite < 0 {
		composite = 0
	}
	if composite > 1 {
		composite = 1
	}

	return ScoreOutput{
		CompositeScore: composite,
		Breakdown: map[string]float64{
			"semantic":         semanticContrib,
			"temporal":         temporalContrib,
			"confidence":       confidenceContrib,
			"centrality":       centralityContrib,
			"authority":        authorityContrib,
			"negation_penalty": negationPenalty,
		},
	}
}

// ComputeAuthorityScore returns the authority score for a given source type.
// Falls back to 0.5 (neutral) for unknown source types.
func ComputeAuthorityScore(sourceType string) float64 {
	if score, ok := types.SourceAuthorityScores[sourceType]; ok {
		return score
	}
	return 0.5
}

// ComputeConfidenceFromMW computes a confidence score from Memory Worth counters.
// Uses Wilson score interval lower bound for robustness with small sample sizes.
// Returns a value in [0, 1].
//
// The Wilson score interval lower bound is:
//
//	(p + z^2/(2n) - z*sqrt((p*(1-p) + z^2/(4n))/n)) / (1 + z^2/n)
//
// where p = success/(success+failure), n = success+failure, z = 1.96 (95% CI).
func ComputeConfidenceFromMW(successCount, failureCount int64) float64 {
	n := float64(successCount + failureCount)
	if n == 0 {
		return 0.5 // no observations -> neutral
	}

	p := float64(successCount) / n
	z := 1.96 // 95% confidence interval

	z2 := z * z
	denominator := 1.0 + z2/n
	center := p + z2/(2.0*n)
	spread := z * math.Sqrt((p*(1.0-p)+z2/(4.0*n))/n)

	lowerBound := (center - spread) / denominator

	// Clamp to [0, 1]
	if lowerBound < 0 {
		lowerBound = 0
	}
	if lowerBound > 1 {
		lowerBound = 1
	}

	return lowerBound
}

// ComputeTemporalScore computes temporal validity using Ebbinghaus exponential decay.
// The forgetting curve: R = e^(-t/S) where S (stability) scales with access count.
// halfLifeDays is the base half-life; each retrieval extends stability logarithmically.
// Returns a value in [0, 1].
func ComputeTemporalScore(ageDays float64, accessCount int64, halfLifeDays float64) float64 {
	if halfLifeDays <= 0 {
		halfLifeDays = 7.0
	}

	// Each access increases the stability (half-life) logarithmically.
	// S = halfLifeDays * (1 + ln(1 + accessCount))
	stability := halfLifeDays * (1.0 + math.Log(1.0+float64(accessCount)))

	// Ebbinghaus: R = e^(-t/S)
	retention := math.Exp(-ageDays / stability)

	// Clamp to [0, 1]
	if retention < 0 {
		retention = 0
	}
	if retention > 1 {
		retention = 1
	}

	return retention
}

// clamp01 clamps v to [0, 1].
func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
