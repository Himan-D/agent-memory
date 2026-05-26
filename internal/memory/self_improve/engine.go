package self_improve

import (
	"context"
	"sync"
	"time"
)

// SelfImprovementEngine learns from feedback and improves memory system
type SelfImprovementEngine struct {
	mu               sync.RWMutex
	synonyms         map[string][]string        // memory_id -> list of learned synonyms
	importanceScores map[string]float64         // memory_id -> importance adjustment
	learningHistory  map[string]*LearningRecord // memory_id -> learning history
	markForReview    map[string]*ReviewRecord   // memory_id -> review record
}

// LearningRecord tracks feedback history
type LearningRecord struct {
	PositiveFeedbackCount int
	NegativeFeedbackCount int
	LastUpdated           time.Time
}

// ReviewRecord marks memory for correction
type ReviewRecord struct {
	MemoryID string
	Reason   string
	MarkedAt time.Time
	Priority int // 1=low, 5=critical
}

// NewSelfImprovementEngine creates a new engine
func NewSelfImprovementEngine() *SelfImprovementEngine {
	return &SelfImprovementEngine{
		synonyms:         make(map[string][]string),
		importanceScores: make(map[string]float64),
		learningHistory:  make(map[string]*LearningRecord),
		markForReview:    make(map[string]*ReviewRecord),
	}
}

// LearnFromPositiveFeedback improves based on correct retrieval
func (sie *SelfImprovementEngine) LearnFromPositiveFeedback(ctx context.Context, memoryID string, userQuery string) error {
	sie.mu.Lock()
	defer sie.mu.Unlock()

	// Increment positive feedback count
	history := sie.getOrCreateHistory(memoryID)
	history.PositiveFeedbackCount++
	history.LastUpdated = time.Now()

	// Boost importance score
	sie.importanceScores[memoryID] += 0.15
	if sie.importanceScores[memoryID] > 1.0 {
		sie.importanceScores[memoryID] = 1.0
	}

	// Extract and store query terms as synonyms for future matching
	if userQuery != "" {
		sie.addSynonym(memoryID, userQuery)
	}

	return nil
}

// LearnFromNegativeFeedback learns from incorrect results
func (sie *SelfImprovementEngine) LearnFromNegativeFeedback(ctx context.Context, memoryID string, feedback string, isCritical bool) error {
	sie.mu.Lock()
	defer sie.mu.Unlock()

	// Increment negative feedback count
	history := sie.getOrCreateHistory(memoryID)
	history.NegativeFeedbackCount++
	history.LastUpdated = time.Now()

	// Penalize importance score
	penalty := 0.1
	if isCritical {
		penalty = 0.25
	}
	sie.importanceScores[memoryID] -= penalty
	if sie.importanceScores[memoryID] < -1.0 {
		sie.importanceScores[memoryID] = -1.0
	}

	// Mark for review if critical
	if isCritical || history.NegativeFeedbackCount >= 3 {
		priority := 3
		if isCritical {
			priority = 5
		}
		sie.markForReview[memoryID] = &ReviewRecord{
			MemoryID: memoryID,
			Reason:   feedback,
			MarkedAt: time.Now(),
			Priority: priority,
		}
	}

	return nil
}

// ExpandQuery uses learned synonyms for smarter search
func (sie *SelfImprovementEngine) ExpandQuery(ctx context.Context, query string) []string {
	sie.mu.RLock()
	defer sie.mu.RUnlock()

	expanded := []string{query}

	// Collect all synonyms across memories
	for _, syns := range sie.synonyms {
		for _, syn := range syns {
			if syn != query {
				expanded = append(expanded, syn)
			}
		}
	}

	return expanded
}

// GetImportanceAdjustment returns the learned importance adjustment
func (sie *SelfImprovementEngine) GetImportanceAdjustment(memoryID string) float64 {
	sie.mu.RLock()
	defer sie.mu.RUnlock()

	score, ok := sie.importanceScores[memoryID]
	if !ok {
		return 0.0
	}
	return score
}

// GetLearningHistory returns feedback history for a memory
func (sie *SelfImprovementEngine) GetLearningHistory(memoryID string) *LearningRecord {
	sie.mu.RLock()
	defer sie.mu.RUnlock()

	history, ok := sie.learningHistory[memoryID]
	if !ok {
		return nil
	}

	// Return a copy
	return &LearningRecord{
		PositiveFeedbackCount: history.PositiveFeedbackCount,
		NegativeFeedbackCount: history.NegativeFeedbackCount,
		LastUpdated:           history.LastUpdated,
	}
}

// GetMemoriesForReview returns memories marked for correction
func (sie *SelfImprovementEngine) GetMemoriesForReview(limit int) []*ReviewRecord {
	sie.mu.RLock()
	defer sie.mu.RUnlock()

	var records []*ReviewRecord
	for _, record := range sie.markForReview {
		records = append(records, record)
	}

	// Sort by priority (descending)
	for i := 0; i < len(records)-1; i++ {
		for j := i + 1; j < len(records); j++ {
			if records[j].Priority > records[i].Priority {
				records[i], records[j] = records[j], records[i]
			}
		}
	}

	if len(records) > limit && limit > 0 {
		records = records[:limit]
	}

	return records
}

// ClearReviewMarker clears review flag after correction
func (sie *SelfImprovementEngine) ClearReviewMarker(memoryID string) {
	sie.mu.Lock()
	defer sie.mu.Unlock()
	delete(sie.markForReview, memoryID)
}

// Helper functions

func (sie *SelfImprovementEngine) getOrCreateHistory(memoryID string) *LearningRecord {
	history, ok := sie.learningHistory[memoryID]
	if !ok {
		history = &LearningRecord{LastUpdated: time.Now()}
		sie.learningHistory[memoryID] = history
	}
	return history
}

func (sie *SelfImprovementEngine) addSynonym(memoryID, query string) {
	if _, ok := sie.synonyms[memoryID]; !ok {
		sie.synonyms[memoryID] = []string{}
	}

	// Avoid duplicates
	for _, syn := range sie.synonyms[memoryID] {
		if syn == query {
			return
		}
	}

	sie.synonyms[memoryID] = append(sie.synonyms[memoryID], query)
}

// GetStats returns learning statistics
func (sie *SelfImprovementEngine) GetStats() map[string]interface{} {
	sie.mu.RLock()
	defer sie.mu.RUnlock()

	totalPositive := 0
	totalNegative := 0
	totalSynonyms := 0

	for _, history := range sie.learningHistory {
		totalPositive += history.PositiveFeedbackCount
		totalNegative += history.NegativeFeedbackCount
	}

	for _, syns := range sie.synonyms {
		totalSynonyms += len(syns)
	}

	return map[string]interface{}{
		"total_positive_feedback": totalPositive,
		"total_negative_feedback": totalNegative,
		"total_synonyms_learned":  totalSynonyms,
		"memories_marked_review":  len(sie.markForReview),
		"unique_memories_learned": len(sie.learningHistory),
	}
}
