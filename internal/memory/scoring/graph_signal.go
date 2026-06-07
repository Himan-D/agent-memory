package scoring

import "math"

// GraphSignalProvider computes graph-based importance signals for memories.
// These signals measure how central and connected a memory is within the knowledge graph.
type GraphSignalProvider struct{}

// ComputeCentrality returns a normalized degree centrality score [0, 1] for a memory node.
// Higher centrality = more connected = more important in the knowledge graph.
// relationCount is the number of edges connected to this node.
// maxRelations is the maximum edge count across all nodes (for normalization).
// Returns 0 if maxRelations is 0.
func ComputeCentrality(relationCount int, maxRelations int) float64 {
	if maxRelations <= 0 {
		return 0
	}

	centrality := float64(relationCount) / float64(maxRelations)

	// Clamp to [0, 1]
	if centrality > 1 {
		centrality = 1
	}

	return centrality
}

// ComputePageRank approximates PageRank for a node using its neighbor scores.
// This is a local approximation — full PageRank would require the full graph.
// The formula is: PR(v) = (1 - d) + d * sum(neighborScores) / len(neighborScores)
// dampingFactor is typically 0.85.
// Returns the approximated PageRank score.
func ComputePageRank(nodeScore float64, neighborScores []float64, dampingFactor float64) float64 {
	if dampingFactor < 0 {
		dampingFactor = 0
	}
	if dampingFactor > 1 {
		dampingFactor = 1
	}

	if len(neighborScores) == 0 {
		return (1.0 - dampingFactor) + dampingFactor*nodeScore
	}

	// Sum contributions from neighbors
	var neighborSum float64
	for _, ns := range neighborScores {
		neighborSum += ns
	}
	avgNeighbor := neighborSum / float64(len(neighborScores))

	// Local PageRank approximation
	pr := (1.0 - dampingFactor) + dampingFactor*avgNeighbor

	// Apply logarithmic scaling for large neighbor counts to avoid over-weighting hubs
	if len(neighborScores) > 1 {
		hubBonus := math.Log2(float64(len(neighborScores))) * 0.01
		pr += hubBonus
	}

	return pr
}
