package temporal

import "math"

// ShiftDetector identifies semantic shifts in conversation.
// Based on GAM paper (arXiv:2604.12285).
// When a shift is detected, memories move from event graph to topic graph.
type ShiftDetector struct {
	Threshold        float64 // cosine distance threshold for shift detection (default: 0.4)
	WindowSize       int     // number of recent embeddings to compare (default: 5)
	recentEmbeddings [][]float32
}

// NewShiftDetector creates a ShiftDetector with default parameters.
func NewShiftDetector() *ShiftDetector {
	return &ShiftDetector{
		Threshold:        0.4,
		WindowSize:       5,
		recentEmbeddings: make([][]float32, 0),
	}
}

// DetectShift returns true if the new embedding represents a topic shift.
// It computes the cosine distance between the new embedding and the centroid
// of the recent window. A distance above the threshold indicates a shift.
// Returns false if the window is empty (not enough context yet).
func (sd *ShiftDetector) DetectShift(newEmbedding []float32) bool {
	if len(sd.recentEmbeddings) == 0 || len(newEmbedding) == 0 {
		return false
	}

	centroid := sd.computeCentroid()
	if centroid == nil {
		return false
	}

	distance := cosineDistance(newEmbedding, centroid)
	return distance > sd.Threshold
}

// AddEmbedding adds an embedding to the sliding window.
// When the window is full, the oldest embedding is removed.
func (sd *ShiftDetector) AddEmbedding(embedding []float32) {
	if len(embedding) == 0 {
		return
	}

	sd.recentEmbeddings = append(sd.recentEmbeddings, embedding)
	if len(sd.recentEmbeddings) > sd.WindowSize {
		sd.recentEmbeddings = sd.recentEmbeddings[1:]
	}
}

// computeCentroid computes the element-wise mean of all embeddings in the window.
func (sd *ShiftDetector) computeCentroid() []float32 {
	if len(sd.recentEmbeddings) == 0 {
		return nil
	}

	dim := len(sd.recentEmbeddings[0])
	centroid := make([]float32, dim)
	count := float32(len(sd.recentEmbeddings))

	for _, emb := range sd.recentEmbeddings {
		if len(emb) != dim {
			continue
		}
		for i := range emb {
			centroid[i] += emb[i]
		}
	}

	for i := range centroid {
		centroid[i] /= count
	}

	return centroid
}

// cosineDistance computes 1 - cosineSimilarity between two vectors.
// Returns 1.0 (maximum distance) if either vector has zero magnitude.
func cosineDistance(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 1.0
	}

	var dot, normA, normB float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}

	magA := math.Sqrt(normA)
	magB := math.Sqrt(normB)
	if magA == 0 || magB == 0 {
		return 1.0
	}

	similarity := dot / (magA * magB)
	return 1.0 - similarity
}
