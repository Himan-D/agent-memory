package search

import "math"

// UCBScorer prevents popular memories from dominating retrieval.
// It combines: similarity score + utility (MW worth) + exploration bonus for rarely-retrieved memories.
// Uses the UCB1 (Upper Confidence Bound) formula from multi-armed bandit theory.
type UCBScorer struct {
	// ExplorationConstant controls the exploration-exploitation tradeoff.
	// Higher values favor exploring rarely-retrieved memories. Default: sqrt(2) ~ 1.41.
	ExplorationConstant float64
}

// NewUCBScorer creates a UCBScorer with the default exploration constant of sqrt(2).
func NewUCBScorer() *UCBScorer {
	return &UCBScorer{
		ExplorationConstant: math.Sqrt(2),
	}
}

// Score computes the UCB score for a memory candidate.
// The formula is: exploitation + exploration_constant * sqrt(ln(total) / count)
//
// totalRetrievals: total retrieval operations across all memories (global counter).
// memoryRetrievals: how many times this specific memory was retrieved.
// similarityScore: vector similarity [0, 1].
// worthScore: MW worth score [0, 1].
//
// The exploitation term averages similarity and worth. The exploration term boosts
// memories with low retrieval counts relative to the total.
func (u *UCBScorer) Score(totalRetrievals, memoryRetrievals int64, similarityScore, worthScore float64) float64 {
	// Exploitation: weighted average of similarity and worth
	exploitation := 0.7*similarityScore + 0.3*worthScore

	// Exploration bonus: UCB1 formula
	var exploration float64
	if totalRetrievals > 0 && memoryRetrievals > 0 {
		exploration = u.ExplorationConstant * math.Sqrt(math.Log(float64(totalRetrievals))/float64(memoryRetrievals))
	} else if totalRetrievals > 0 && memoryRetrievals == 0 {
		// Never-retrieved memory gets maximum exploration bonus
		exploration = u.ExplorationConstant * math.Sqrt(math.Log(float64(totalRetrievals)+1))
	}
	// If totalRetrievals == 0, no exploration bonus (system just started)

	return exploitation + exploration
}
