package pool

// DualPool manages two memory pools per agent.
// Exploitation pool: consolidated past trajectories that reliably work.
// Exploration pool: LLM-generated candidate solutions for novel contexts.
// The split follows multi-armed bandit theory to balance proven knowledge with discovery.
type DualPool struct {
	// ExplorationRatio is the fraction of retrieval results drawn from the exploration pool.
	// Default: 0.2 (80% exploitation, 20% exploration).
	ExplorationRatio float64
}

// NewDualPool creates a DualPool with the default exploration ratio of 0.2.
func NewDualPool() *DualPool {
	return &DualPool{
		ExplorationRatio: 0.2,
	}
}

// Route decides which pool a new memory should go to.
// Novel content with low worth goes to exploration; reinforced content goes to exploitation.
// Returns "exploitation" or "exploration".
func (dp *DualPool) Route(content string, worthScore float64, isNovel bool) string {
	// Novel content defaults to exploration
	if isNovel {
		return PoolExploration
	}

	// High worth score → exploitation (proven useful)
	if worthScore >= 0.6 {
		return PoolExploitation
	}

	// Low worth but not novel → still exploitation (existing knowledge)
	if worthScore >= 0.3 {
		return PoolExploitation
	}

	// Very low worth → exploration (needs more testing)
	return PoolExploration
}

// MergeResults interleaves results from both pools based on the exploration ratio.
// It ensures the final list contains at most `limit` items, with approximately
// ExplorationRatio fraction from exploration and the rest from exploitation.
func (dp *DualPool) MergeResults(exploitation, exploration []interface{}, limit int) []interface{} {
	if limit <= 0 {
		return nil
	}

	explorationSlots := int(float64(limit) * dp.ExplorationRatio)
	if explorationSlots < 1 && len(exploration) > 0 {
		explorationSlots = 1 // Always include at least one exploration result if available
	}
	exploitationSlots := limit - explorationSlots

	// Cap to available results
	if explorationSlots > len(exploration) {
		explorationSlots = len(exploration)
		exploitationSlots = limit - explorationSlots
	}
	if exploitationSlots > len(exploitation) {
		exploitationSlots = len(exploitation)
		// Give remaining slots back to exploration
		remaining := limit - exploitationSlots
		if remaining > len(exploration) {
			remaining = len(exploration)
		}
		explorationSlots = remaining
	}

	result := make([]interface{}, 0, exploitationSlots+explorationSlots)

	// Interleave: mostly exploitation with exploration mixed in
	exploitIdx := 0
	exploreIdx := 0

	for len(result) < limit {
		// Add exploitation results
		if exploitIdx < exploitationSlots {
			result = append(result, exploitation[exploitIdx])
			exploitIdx++
		}

		// Periodically insert exploration results
		if exploreIdx < explorationSlots && (exploitIdx%4 == 0 || exploitIdx >= exploitationSlots) {
			result = append(result, exploration[exploreIdx])
			exploreIdx++
		}

		// Break if we've exhausted both pools
		if exploitIdx >= exploitationSlots && exploreIdx >= explorationSlots {
			break
		}
	}

	if len(result) > limit {
		result = result[:limit]
	}

	return result
}

// ShouldPromote determines if a memory should be promoted from exploration to exploitation.
// A memory is promoted when it has proven useful: high worth score and sufficient retrievals.
func (dp *DualPool) ShouldPromote(worthScore float64, retrievalCount int64) bool {
	// Require both a minimum worth threshold and minimum retrieval count
	// to avoid promoting memories that got lucky on a single retrieval.
	return worthScore >= 0.7 && retrievalCount >= 3
}

// Pool type constants.
const (
	PoolExploitation = "exploitation"
	PoolExploration  = "exploration"
)
