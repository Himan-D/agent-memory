package recommendation

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"agent-memory/internal/llm"
)

// MemoryAction represents a predicted engagement type for a memory.
type MemoryAction string

const (
	ActionUseful     MemoryAction = "useful"     // Agent will use this memory for a task
	ActionAccurate   MemoryAction = "accurate"   // Memory is factually correct
	ActionActionable MemoryAction = "actionable" // Memory contains actionable information
	ActionDerivable  MemoryAction = "derivable"  // New insights can be derived from it
	ActionReferenced MemoryAction = "referenced" // Agent will reference it in conversation
	ActionSkipped    MemoryAction = "skipped"    // Agent will skip/ignore this
	ActionBlocked    MemoryAction = "blocked"    // Agent should block this source
)

// ActionWeights defines the importance of each predicted action in the final score.
// Based on X's weighted scorer: Final Score = Σ (weight_i × P(action_i))
var DefaultActionWeights = map[MemoryAction]float64{
	ActionUseful:     3.0,
	ActionAccurate:   2.0,
	ActionActionable: 2.5,
	ActionDerivable:  1.5,
	ActionReferenced: 1.0,
	ActionSkipped:    -1.0,
	ActionBlocked:    -5.0,
}

// EngagementPrediction holds predicted probabilities for each action type.
type EngagementPrediction struct {
	Action      MemoryAction
	Probability float64 // 0.0 to 1.0
}

// MemoryPredictions holds all predictions for a single memory.
type MemoryPredictions struct {
	CandidateID string
	Predictions map[MemoryAction]float64
	Summary     string // Human-readable summary of why this memory scored high
}

// PhoenixMemoryScorer implements X's Phoenix scoring pattern adapted for agent memories.
// Uses LLM to predict multi-action engagement probabilities, then computes weighted score.
type PhoenixMemoryScorer struct {
	llmProvider llm.Provider
	weights     map[MemoryAction]float64
}

// NewPhoenixMemoryScorer creates a new Phoenix-style scorer.
func NewPhoenixMemoryScorer(llm llm.Provider, weights map[MemoryAction]float64) *PhoenixMemoryScorer {
	if weights == nil {
		weights = DefaultActionWeights
	}
	return &PhoenixMemoryScorer{
		llmProvider: llm,
		weights:     weights,
	}
}

func (s *PhoenixMemoryScorer) Name() string {
	return "phoenix_memory_scorer"
}

func (s *PhoenixMemoryScorer) Score(ctx context.Context, query *QueryContext, candidate *MemoryCandidate) error {
	memoryContent, _ := candidate.Metadata["content"].(string)
	memoryType, _ := candidate.Metadata["type"].(string)
	memorySource, _ := candidate.Metadata["source"].(string)
	authorID, _ := candidate.Metadata["author_id"].(string)
	recency, _ := candidate.Metadata["recency"].(float64)

	// Build engagement history summary for context
	historySummary := s.buildHistorySummary(query)

	prompt := fmt.Sprintf(`You are evaluating how relevant a memory will be for an AI agent's current context.

AGENT PROFILE:
- User ID: %s
- Agent ID: %s
- Recent Engagement History: %s

MEMORY TO EVALUATE:
- Content: %s
- Type: %s
- Source: %s
- Author ID: %s
- Recency Score: %.2f (0=stale, 1=fresh)

PREDICT the probability (0.0 to 1.0) for EACH of these actions the agent might take:
- useful: Agent will use this memory for a current or upcoming task
- accurate: Memory is factually correct and trustworthy
- actionable: Memory contains information the agent can act on immediately
- derivable: New insights or connections can be derived from this memory
- referenced: Agent will reference this memory in conversation or reasoning
- skipped: Agent will skip or ignore this memory as irrelevant
- blocked: Agent should block or filter out this source

Return ONLY a JSON object with probabilities for each action. No explanation.

Example format:
{"useful": 0.85, "accurate": 0.92, "actionable": 0.60, "derivable": 0.45, "referenced": 0.70, "skipped": 0.10, "blocked": 0.02}`,
		query.UserID,
		query.AgentID,
		historySummary,
		memoryContent,
		memoryType,
		memorySource,
		authorID,
		recency,
	)

	if s.llmProvider == nil {
		// Fallback: use heuristic scoring when no LLM is available
		predictions := s.heuristicScore(candidate, query)
		s.applyWeightedScore(candidate, predictions)
		candidate.Metadata["scorer"] = "phoenix_heuristic_fallback"
		return nil
	}

	resp, err := s.llmProvider.Complete(ctx, &llm.CompletionRequest{
		Model: "gpt-4o-mini",
		Messages: []llm.Message{
			{Role: "system", Content: "You predict agent engagement with memories. Return only JSON."},
			{Role: "user", Content: prompt},
		},
		Temperature: 0.1,
		MaxTokens:   200,
	})
	if err != nil {
		// Fallback to heuristic on LLM error
		predictions := s.heuristicScore(candidate, query)
		s.applyWeightedScore(candidate, predictions)
		candidate.Metadata["scorer"] = "phoenix_heuristic_fallback"
		candidate.Metadata["llm_error"] = err.Error()
		return nil
	}

	predictions := s.parsePredictions(resp.Content)
	s.applyWeightedScore(candidate, predictions)
	candidate.Metadata["scorer"] = "phoenix_llm"
	candidate.Metadata["predictions"] = predictions

	return nil
}

func (s *PhoenixMemoryScorer) buildHistorySummary(query *QueryContext) string {
	if len(query.EngagementHistory) == 0 {
		return "No engagement history available"
	}

	// Summarize recent engagements
	var parts []string
	actionCounts := make(map[string]int)
	for _, e := range query.EngagementHistory {
		actionCounts[e.Action]++
	}
	for action, count := range actionCounts {
		parts = append(parts, fmt.Sprintf("%s: %d", action, count))
	}
	return strings.Join(parts, ", ")
}

func (s *PhoenixMemoryScorer) parsePredictions(content string) map[MemoryAction]float64 {
	// Extract JSON from response
	content = strings.TrimSpace(content)
	if idx := strings.Index(content, "{"); idx >= 0 {
		content = content[idx:]
	}
	if idx := strings.LastIndex(content, "}"); idx >= 0 {
		content = content[:idx+1]
	}

	predictions := make(map[MemoryAction]float64)
	if err := json.Unmarshal([]byte(content), &predictions); err != nil {
		// Return empty predictions on parse error
		return predictions
	}
	return predictions
}

func (s *PhoenixMemoryScorer) applyWeightedScore(candidate *MemoryCandidate, predictions map[MemoryAction]float64) {
	var totalScore float64
	for action, prob := range predictions {
		weight := s.weights[action]
		totalScore += weight * prob
	}

	// Normalize to 0-1 range for consistent scoring
	maxPossible := 0.0
	for _, w := range s.weights {
		if w > 0 {
			maxPossible += w
		}
	}
	if maxPossible > 0 {
		candidate.Score = totalScore / maxPossible
	}

	// Clamp to [0, 1]
	if candidate.Score < 0 {
		candidate.Score = 0
	}
	if candidate.Score > 1 {
		candidate.Score = 1
	}
}

func (s *PhoenixMemoryScorer) heuristicScore(candidate *MemoryCandidate, query *QueryContext) map[MemoryAction]float64 {
	// Simple heuristic fallback when LLM is unavailable
	predictions := make(map[MemoryAction]float64)

	content, _ := candidate.Metadata["content"].(string)
	recency, _ := candidate.Metadata["recency"].(float64)
	source, _ := candidate.Metadata["source"].(string)

	// Recency strongly influences usefulness
	predictions[ActionUseful] = recency * 0.7

	// Check if memory is from a followed source (higher accuracy trust)
	for _, fid := range query.FollowingIDs {
		if source == fid {
			predictions[ActionAccurate] = 0.8
			break
		}
	}
	if predictions[ActionAccurate] == 0 {
		predictions[ActionAccurate] = 0.5
	}

	// Content length heuristic for actionability
	if len(content) > 50 {
		predictions[ActionActionable] = 0.6
	} else {
		predictions[ActionActionable] = 0.3
	}

	// Check engagement history for similar memories
	for _, e := range query.EngagementHistory {
		if e.MemoryID == candidate.ID && e.Action == "use" {
			predictions[ActionUseful] = 0.9
			predictions[ActionReferenced] = 0.8
			break
		}
	}

	// Default values
	if _, ok := predictions[ActionDerivable]; !ok {
		predictions[ActionDerivable] = 0.4
	}
	if _, ok := predictions[ActionReferenced]; !ok {
		predictions[ActionReferenced] = 0.5
	}
	predictions[ActionSkipped] = 0.2
	predictions[ActionBlocked] = 0.0

	// Check blocked sources
	for _, bid := range query.BlockedIDs {
		if source == bid {
			predictions[ActionBlocked] = 1.0
			predictions[ActionUseful] = 0.0
		}
	}

	return predictions
}
