package scoring

import "math"

// CompositeScorer combines four orthogonal signals into a single importance score.
// Signals: (1) semantic relevance, (2) temporal validity, (3) confidence/MW, (4) graph centrality.
// Each signal is independently normalized to [0, 1] and combined via weighted sum.
type CompositeScorer struct {
	SemanticWeight   float64 // default 0.4
	TemporalWeight   float64 // default 0.2
	ConfidenceWeight float64 // default 0.2
	CentralityWeight float64 // default 0.2
}

// ScoreInput holds the four orthogonal signals for composite scoring.
type ScoreInput struct {
	SemanticScore   float64 // from vector similarity [0, 1]
	TemporalScore   float64 // from temporal scorer [0, 1]
	ConfidenceScore float64 // from MW counters [0, 1]
	CentralityScore float64 // from graph degree [0, 1]
}

// ScoreOutput holds the composite score and per-signal breakdown.
type ScoreOutput struct {
	CompositeScore float64
	Breakdown      map[string]float64 // individual signal contributions
}

// NewCompositeScorer creates a CompositeScorer with default weights (0.4, 0.2, 0.2, 0.2).
func NewCompositeScorer() *CompositeScorer {
	return &CompositeScorer{
		SemanticWeight:   0.4,
		TemporalWeight:   0.2,
		ConfidenceWeight: 0.2,
		CentralityWeight: 0.2,
	}
}

// Score computes a weighted composite from the four signals.
func (cs *CompositeScorer) Score(input ScoreInput) ScoreOutput {
	semanticContrib := cs.SemanticWeight * input.SemanticScore
	temporalContrib := cs.TemporalWeight * input.TemporalScore
	confidenceContrib := cs.ConfidenceWeight * input.ConfidenceScore
	centralityContrib := cs.CentralityWeight * input.CentralityScore

	composite := semanticContrib + temporalContrib + confidenceContrib + centralityContrib

	return ScoreOutput{
		CompositeScore: composite,
		Breakdown: map[string]float64{
			"semantic":   semanticContrib,
			"temporal":   temporalContrib,
			"confidence": confidenceContrib,
			"centrality": centralityContrib,
		},
	}
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
		return 0.5 // no observations → neutral
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
