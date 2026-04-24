package profiles

import (
	"context"
	"strings"
	"time"

	"agent-memory/internal/memory/types"
)

var intentPatterns = map[string][]string{
	"information":    {"what is", "how does", "explain", "tell me about", "define", "describe"},
	"task":         {"do", "make", "create", "write", "build", "generate", "implement"},
	"problem":      {"fix", "error", "bug", "issue", "broken", "failed", "doesn't work"},
	"optimization": {"improve", "better", "optimize", "faster", "efficient", "refactor"},
	"learning":     {"learn", "study", "understand", "practice", "exercise"},
	"planning":     {"plan", "schedule", "roadmap", "milestone", "goal"},
	"communication": {"tell", "ask", "notify", "message", "email", "contact"},
	"research":     {"search", "find", "lookup", "explore", "discover"},
}

func (s *Service) DetectUserIntent(ctx context.Context, userID string, latestContent string) (string, error) {
	if latestContent == "" {
		return "unknown", nil
	}

	lower := strings.ToLower(latestContent)

	for intent, patterns := range intentPatterns {
		for _, pattern := range patterns {
			if strings.Contains(lower, pattern) {
				return intent, nil
			}
		}
	}

	return "general", nil
}

func (s *Service) PredictUserNeeds(ctx context.Context, userID string) ([]string, error) {
	memSvc, ok := s.memSvc.(interface{ GetMemoriesByUser(ctx context.Context, userID string, limit int) ([]*types.Memory, error) })
	if !ok {
		return []string{}, nil
	}

	memories, err := memSvc.GetMemoriesByUser(ctx, userID, 50)
	if err != nil || len(memories) == 0 {
		return []string{}, nil
	}

	var needs []string
	topicCounts := make(map[string]int)

	for _, mem := range memories {
		if mem.CreatedAt.After(time.Now().AddDate(0, -1, 0)) {
			content := strings.ToLower(mem.Content)
			for intent, patterns := range intentPatterns {
				for _, pattern := range patterns {
					if strings.Contains(content, pattern) {
						topicCounts[intent]++
					}
				}
			}
		}
	}

	for intent, count := range topicCounts {
		if count >= 3 {
			needs = append(needs, intent)
		}
	}

	if len(needs) == 0 {
		needs = append(needs, "information")
	}

	return needs, nil
}

func (s *Service) EstimateUserTrust(ctx context.Context, userID string) (float32, error) {
	memSvc, ok := s.memSvc.(interface{ GetMemoriesByUser(ctx context.Context, userID string, limit int) ([]*types.Memory, error) })
	if !ok {
		return 0.5, nil
	}

	memories, err := memSvc.GetMemoriesByUser(ctx, userID, 100)
	if err != nil {
		return 0.5, nil
	}

	if len(memories) < 10 {
		return 0.3, nil
	}

	positive := 0
	for _, mem := range memories {
		if mem.Importance == types.ImportanceHigh || mem.Importance == types.ImportanceCritical {
			positive++
		}
	}

	return float32(positive) / float32(len(memories)), nil
}