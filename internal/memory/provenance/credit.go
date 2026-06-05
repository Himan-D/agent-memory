package provenance

import "math"

// CreditAssigner implements TD(lambda) eligibility traces for memory credit assignment.
// When memory B (derived from A) leads to success, credit flows to A with decay.
// This enables outcome-linked importance: memories that contribute to successful
// downstream outcomes receive higher Q-values over time.
type CreditAssigner struct {
	// Gamma is the discount factor for future rewards (default: 0.95).
	// Higher values propagate credit further through the provenance chain.
	Gamma float64

	// Lambda is the eligibility trace decay (default: 0.8).
	// Controls how quickly credit decays as it flows to more distant ancestors.
	Lambda float64
}

// NewCreditAssigner creates a CreditAssigner with default parameters.
func NewCreditAssigner() *CreditAssigner {
	return &CreditAssigner{
		Gamma:  0.95,
		Lambda: 0.8,
	}
}

// AssignCredit propagates reward through the provenance DAG.
//
// memoryID: the memory that received direct feedback.
// reward: +1 for positive outcome, -1 for negative outcome.
// ancestors: ordered list of ancestor memory IDs (nearest first, as returned by DAG.GetAncestors).
//
// Returns a map of memoryID to credit delta. The credit for each ancestor decays
// exponentially with depth: credit[i] = reward * (gamma * lambda)^depth.
//
// The originating memory always receives the full reward.
func (ca *CreditAssigner) AssignCredit(memoryID string, reward float64, ancestors []string) map[string]float64 {
	credits := make(map[string]float64, len(ancestors)+1)

	// The memory that received direct feedback gets full reward
	credits[memoryID] = reward

	// Propagate to ancestors with exponential decay
	decayProduct := ca.Gamma * ca.Lambda
	for i, ancestorID := range ancestors {
		depth := float64(i + 1)
		credit := reward * math.Pow(decayProduct, depth)

		// Stop propagating if credit is negligibly small
		if math.Abs(credit) < 1e-6 {
			break
		}

		credits[ancestorID] = credit
	}

	return credits
}
