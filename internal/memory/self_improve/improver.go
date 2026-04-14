package self_improve

import (
	"context"
	"fmt"
	"math"
	"time"

	"agent-memory/internal/memory/types"
)

type FeedbackCollector interface {
	GetNegativeFeedback(ctx context.Context, memoryID string) ([]*types.Feedback, error)
	GetPositiveFeedback(ctx context.Context, memoryID string) ([]*types.Feedback, error)
	GetAllFeedback(ctx context.Context, memoryID string) ([]*types.Feedback, error)
}

type TuningStore interface {
	UpdateMemoryImportance(ctx context.Context, memoryID string, importance types.ImportanceLevel) error
	UpdateMemoryEmbedding(ctx context.Context, memoryID string, newContent string) error
	AddSynonym(ctx context.Context, memoryID, word, synonym string) error
	GetSynonyms(ctx context.Context, memoryID, word string) ([]string, error)
	RecordTuningEvent(ctx context.Context, event *TuningEvent) error
	GetTuningHistory(ctx context.Context, memoryID string) ([]TuningEvent, error)
}

type TuningEvent struct {
	ID         string                 `json:"id"`
	MemoryID   string                 `json:"memory_id"`
	EventType  TuningEventType        `json:"event_type"`
	OldValue   string                 `json:"old_value"`
	NewValue   string                 `json:"new_value"`
	Trigger    string                 `json:"trigger"`
	FeedbackID string                 `json:"feedback_id,omitempty"`
	Confidence float64                `json:"confidence"`
	CreatedAt  time.Time              `json:"created_at"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

type TuningEventType string

const (
	TuningEventImportanceIncrease TuningEventType = "importance_increase"
	TuningEventImportanceDecrease TuningEventType = "importance_decrease"
	TuningEventContentUpdate      TuningEventType = "content_update"
	TuningEventSynonymAdd         TuningEventType = "synonym_add"
	TuningEventConceptLink        TuningEventType = "concept_link"
)

type SelfImprover struct {
	feedbackCollector FeedbackCollector
	tuningStore       TuningStore
	config            *Config
}

type Config struct {
	MinFeedbackForTuning     int     `json:"min_feedback_for_tuning"`
	PositiveThreshold        float64 `json:"positive_threshold"`
	NegativeThreshold        float64 `json:"negative_threshold"`
	ImportanceBoostStep      float64 `json:"importance_boost_step"`
	AutoTuneEnabled          bool    `json:"auto_tune_enabled"`
	LearnFromNegativeEnabled bool    `json:"learn_from_negative_enabled"`
}

func DefaultConfig() *Config {
	return &Config{
		MinFeedbackForTuning:     3,
		PositiveThreshold:        0.7,
		NegativeThreshold:        0.3,
		ImportanceBoostStep:      0.2,
		AutoTuneEnabled:          true,
		LearnFromNegativeEnabled: true,
	}
}

func NewSelfImprover(fc FeedbackCollector, ts TuningStore, cfg *Config) *SelfImprover {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	return &SelfImprover{
		feedbackCollector: fc,
		tuningStore:       ts,
		config:            cfg,
	}
}

type TuningResult struct {
	MemoryID        string                 `json:"memory_id"`
	ChangesMade     []string               `json:"changes_made"`
	Confidence      float64                `json:"confidence"`
	FeedbackCount   int                    `json:"feedback_count"`
	PositiveRatio   float64                `json:"positive_ratio"`
	NextReviewAt    time.Time              `json:"next_review_at"`
	Recommendations []TuningRecommendation `json:"recommendations,omitempty"`
}

type TuningRecommendation struct {
	Type        string `json:"type"`
	Priority    string `json:"priority"`
	Description string `json:"description"`
	Action      string `json:"action"`
}

func (s *SelfImprover) ProcessTuningCycle(ctx context.Context, memoryID string) (*TuningResult, error) {
	feedback, err := s.feedbackCollector.GetAllFeedback(ctx, memoryID)
	if err != nil {
		return nil, fmt.Errorf("get feedback: %w", err)
	}

	result := &TuningResult{
		MemoryID:      memoryID,
		ChangesMade:   []string{},
		FeedbackCount: len(feedback),
	}

	if len(feedback) < s.config.MinFeedbackForTuning {
		result.Recommendations = append(result.Recommendations, TuningRecommendation{
			Type:        "waiting",
			Priority:    "low",
			Description: fmt.Sprintf("Need %d more feedback for tuning", s.config.MinFeedbackForTuning-len(feedback)),
			Action:      "wait_for_more_feedback",
		})
		return result, nil
	}

	positiveCount := 0
	negativeCount := 0

	for _, fb := range feedback {
		switch fb.Type {
		case types.FeedbackPositive:
			positiveCount++
		case types.FeedbackNegative, types.FeedbackVeryNegative:
			negativeCount++
		}
	}

	if len(feedback) > 0 {
		result.PositiveRatio = float64(positiveCount) / float64(len(feedback))
		result.Confidence = math.Min(1.0, float64(len(feedback))/10.0)
	}

	if result.PositiveRatio >= s.config.PositiveThreshold {
		change := s.tunePositiveMemory(ctx, memoryID, positiveCount)
		if change != "" {
			result.ChangesMade = append(result.ChangesMade, change)
		}
	} else if result.PositiveRatio <= s.config.NegativeThreshold {
		changes := s.tuneNegativeMemory(ctx, memoryID, negativeCount, feedback)
		result.ChangesMade = append(result.ChangesMade, changes...)
	}

	result.NextReviewAt = time.Now().Add(24 * time.Hour)

	return result, nil
}

func (s *SelfImprover) tunePositiveMemory(ctx context.Context, memoryID string, positiveCount int) string {
	boost := float64(positiveCount) * s.config.ImportanceBoostStep
	importance := types.ImportanceMedium
	if boost > 0.5 {
		importance = types.ImportanceHigh
	} else if boost > 0.3 {
		importance = types.ImportanceMedium
	} else {
		importance = types.ImportanceLow
	}

	if err := s.tuningStore.UpdateMemoryImportance(ctx, memoryID, importance); err != nil {
		return ""
	}

	event := &TuningEvent{
		MemoryID:   memoryID,
		EventType:  TuningEventImportanceIncrease,
		NewValue:   string(importance),
		Trigger:    "positive_feedback",
		Confidence: float64(positiveCount) / 10.0,
		CreatedAt:  time.Now(),
	}
	s.tuningStore.RecordTuningEvent(ctx, event)

	return fmt.Sprintf("importance_increased_to_%s", importance)
}

func (s *SelfImprover) tuneNegativeMemory(ctx context.Context, memoryID string, negativeCount int, feedback []*types.Feedback) []string {
	var changes []string

	if !s.config.LearnFromNegativeEnabled {
		return changes
	}

	improvements := s.extractImprovements(feedback)
	for _, improvement := range improvements {
		if err := s.tuningStore.UpdateMemoryEmbedding(ctx, memoryID, improvement); err != nil {
			continue
		}

		event := &TuningEvent{
			MemoryID:   memoryID,
			EventType:  TuningEventContentUpdate,
			NewValue:   improvement,
			Trigger:    "negative_feedback",
			Confidence: 0.5,
			CreatedAt:  time.Now(),
		}
		s.tuningStore.RecordTuningEvent(ctx, event)

		changes = append(changes, fmt.Sprintf("content_updated_with_%s", improvement[:min(30, len(improvement))]))
	}

	if negativeCount >= 3 {
		if err := s.tuningStore.UpdateMemoryImportance(ctx, memoryID, types.ImportanceLow); err != nil {
			return changes
		}
		event := &TuningEvent{
			MemoryID:   memoryID,
			EventType:  TuningEventImportanceDecrease,
			NewValue:   string(types.ImportanceLow),
			Trigger:    "repeated_negative",
			Confidence: 0.8,
			CreatedAt:  time.Now(),
		}
		s.tuningStore.RecordTuningEvent(ctx, event)
		changes = append(changes, "importance_decreased_to_low")
	}

	return changes
}

func (s *SelfImprover) extractImprovements(feedback []*types.Feedback) []string {
	var improvements []string
	for _, fb := range feedback {
		if (fb.Type == types.FeedbackNegative || fb.Type == types.FeedbackVeryNegative) && fb.Comment != "" {
			improvements = append(improvements, fb.Comment)
		}
	}
	return improvements
}

func (s *SelfImprover) AddLearnedSynonym(ctx context.Context, memoryID, word, synonym string) error {
	if err := s.tuningStore.AddSynonym(ctx, memoryID, word, synonym); err != nil {
		return err
	}

	event := &TuningEvent{
		MemoryID:   memoryID,
		EventType:  TuningEventSynonymAdd,
		OldValue:   word,
		NewValue:   synonym,
		Trigger:    "learned",
		Confidence: 0.7,
		CreatedAt:  time.Now(),
	}
	return s.tuningStore.RecordTuningEvent(ctx, event)
}

func (s *SelfImprover) GetTuningInsights(ctx context.Context, memoryID string) (*TuningInsights, error) {
	history, err := s.tuningStore.GetTuningHistory(ctx, memoryID)
	if err != nil {
		return nil, err
	}

	insights := &TuningInsights{
		MemoryID:    memoryID,
		TotalEvents: len(history),
		ByType:      make(map[TuningEventType]int),
	}

	for _, event := range history {
		insights.ByType[event.EventType]++
		insights.TotalConfidence += event.Confidence
	}

	if len(history) > 0 {
		insights.AvgConfidence = insights.TotalConfidence / float64(len(history))
	}

	return insights, nil
}

type TuningInsights struct {
	MemoryID        string                  `json:"memory_id"`
	TotalEvents     int                     `json:"total_events"`
	ByType          map[TuningEventType]int `json:"by_type"`
	AvgConfidence   float64                 `json:"avg_confidence"`
	TotalConfidence float64                 `json:"total_confidence"`
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
