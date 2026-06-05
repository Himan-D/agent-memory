package temporal

import "math"

// PhaseRotator applies complex-number rotation to embedding vectors based on memory age and volatility.
// Volatile relations (e.g., "works at") rotate fast; stable relations (e.g., "born in") rotate slowly.
// Based on RoMem-style temporal reasoning where time is encoded as rotation in embedding space.
type PhaseRotator struct {
	// BaseRotationRate is the rotation speed in radians per day for neutral volatility (default: 0.01).
	BaseRotationRate float64
}

// NewPhaseRotator creates a PhaseRotator with the default base rotation rate of 0.01 radians/day.
func NewPhaseRotator() *PhaseRotator {
	return &PhaseRotator{
		BaseRotationRate: 0.01,
	}
}

// ComputePhaseAngle computes the current phase angle for a memory based on its age and volatility.
// volatility: 0.0 (stable, e.g. birthdate) to 1.0 (volatile, e.g. current job).
// ageDays: days since memory creation.
// Returns angle in radians. Volatile memories accumulate phase faster, causing their
// temporal relevance to decay more rapidly.
func (pr *PhaseRotator) ComputePhaseAngle(volatility float64, ageDays float64) float64 {
	// Clamp volatility to [0, 1]
	if volatility < 0 {
		volatility = 0
	}
	if volatility > 1 {
		volatility = 1
	}

	// Scale rotation rate by volatility: stable facts barely rotate, volatile facts spin fast.
	// The multiplier ranges from 0.1 (very stable) to 10.0 (very volatile).
	scaleFactor := 0.1 + 9.9*volatility
	effectiveRate := pr.BaseRotationRate * scaleFactor

	return effectiveRate * ageDays
}

// ApplyPhaseRotation adjusts a similarity score based on phase difference between query time and memory time.
// Uses cos(phaseAngle) as the decay factor — fresh memories have angle~0 (cos=1.0),
// old volatile memories drift toward cos=0.
// The result is clamped to [0, similarityScore] so rotation never boosts beyond the raw similarity.
func (pr *PhaseRotator) ApplyPhaseRotation(similarityScore float64, phaseAngle float64) float64 {
	decay := math.Cos(phaseAngle)

	// Clamp decay to [0, 1] — negative cosine means the memory has rotated past relevance.
	if decay < 0 {
		decay = 0
	}
	if decay > 1 {
		decay = 1
	}

	return similarityScore * decay
}

// RotateEmbedding applies complex rotation to an embedding vector.
// For each pair of dimensions (2i, 2i+1), applies the 2D rotation matrix:
//
//	[[cos theta, -sin theta],
//	 [sin theta,  cos theta]]
//
// If the embedding has an odd number of dimensions, the last dimension is left unchanged.
func (pr *PhaseRotator) RotateEmbedding(embedding []float32, phaseAngle float64) []float32 {
	if len(embedding) == 0 {
		return embedding
	}

	cosTheta := float32(math.Cos(phaseAngle))
	sinTheta := float32(math.Sin(phaseAngle))

	result := make([]float32, len(embedding))

	// Process pairs of dimensions
	pairs := len(embedding) / 2
	for i := 0; i < pairs; i++ {
		idx := i * 2
		x := embedding[idx]
		y := embedding[idx+1]

		result[idx] = x*cosTheta - y*sinTheta
		result[idx+1] = x*sinTheta + y*cosTheta
	}

	// If odd number of dimensions, copy the last one unchanged.
	if len(embedding)%2 != 0 {
		result[len(embedding)-1] = embedding[len(embedding)-1]
	}

	return result
}
