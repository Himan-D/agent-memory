package profiles

import (
	"context"

	"agent-memory/internal/memory/types"
)

func (s *Service) GetRecommendations(ctx context.Context, userID string, limit int) ([]*Recommendation, error) {
	if limit <= 0 {
		limit = 5
	}

	needs, err := s.PredictUserNeeds(ctx, userID)
	if err != nil || len(needs) == 0 {
		return s.defaultRecommendations(ctx, userID, limit)
	}

	var recs []*Recommendation

	for _, need := range needs {
		rec := &Recommendation{
			Type:   "content",
			Score:  0.8,
			Reason: "Based on your recent " + need + " queries",
		}

		switch need {
		case "information":
			rec.Content = "Try searching for more detailed articles in the knowledge base"
			rec.Type = "suggestion"
		case "task":
			rec.Content = "Consider breaking down complex tasks into smaller steps"
			rec.Type = "tip"
		case "problem":
			rec.Content = "Check the troubleshooting section for common solutions"
			rec.Type = "help"
		case "optimization":
			rec.Content = "Review your recent work for potential improvements"
			rec.Type = "suggestion"
		case "learning":
			rec.Content = "Explore related topics to expand your knowledge"
			rec.Type = "exploration"
		case "research":
			rec.Content = "Use advanced filters to narrow down your search results"
			rec.Type = "tip"
		default:
			rec.Content = "Continue exploring new features"
			rec.Type = "general"
		}

		recs = append(recs, rec)
	}

	bd, err := s.BuildBehaviorProfile(ctx, userID)
	if err == nil && bd != nil {
		if bd.InteractionRate < 0.3 {
			recs = append(recs, &Recommendation{
				Type:    "engagement",
				Content: "Increase your interaction frequency for better personalization",
				Score:   0.6,
				Reason:  "Low interaction rate detected",
			})
		}
		if len(bd.TopCategories) > 0 {
			recs = append(recs, &Recommendation{
				Type:    "content",
				Content: "Explore more " + bd.TopCategories[0] + " content",
				Score:   0.7,
				Reason:  "Based on your interest in " + bd.TopCategories[0],
			})
		}
	}

	if len(recs) > limit {
		recs = recs[:limit]
	}

	return recs, nil
}

func (s *Service) defaultRecommendations(ctx context.Context, userID string, limit int) ([]*Recommendation, error) {
	memSvc, ok := s.memSvc.(interface{ GetMemoriesByUser(ctx context.Context, userID string, limit int) ([]*types.Memory, error) })
	if !ok {
		return []*Recommendation{}, nil
	}

	memories, err := memSvc.GetMemoriesByUser(ctx, userID, 10)
	if err != nil || len(memories) == 0 {
		return []*Recommendation{
			{Type: "onboarding", Content: "Create your first memory to get started", Score: 1.0, Reason: "No memories found"},
		}, nil
	}

	return []*Recommendation{
		{Type: "suggestion", Content: "Search your existing memories for relevant information", Score: 0.9, Reason: "You have memories to explore"},
		{Type: "tip", Content: "Add more details when creating memories for better retrieval", Score: 0.7, Reason: "Improve memory quality"},
	}, nil
}

func (s *Service) AnalyzeUserEngagement(ctx context.Context, userID string) (float32, error) {
	bd, err := s.BuildBehaviorProfile(ctx, userID)
	if err != nil {
		return 0, err
	}

	score := float32(bd.InteractionRate) * 0.4

	if bd.TotalMemories > 100 {
		score += 0.3
	} else if bd.TotalMemories > 50 {
		score += 0.2
	} else if bd.TotalMemories > 10 {
		score += 0.1
	}

	if bd.ActiveDaysPerWeek >= 5 {
		score += 0.2
	} else if bd.ActiveDaysPerWeek >= 3 {
		score += 0.1
	}

	if bd.TotalSearches > bd.TotalMemories {
		score += 0.1
	}

	return score, nil
}

func (s *Service) AnalyzeUserBehaviorPattern(ctx context.Context, userID string) (string, error) {
	bd, err := s.BuildBehaviorProfile(ctx, userID)
	if err != nil {
		return "unknown", err
	}

	if bd.TotalMemories < 5 {
		return "new_user", nil
	}

	if bd.TotalSearches > bd.TotalMemories*2 {
		return "researcher", nil
	}

	if bd.InteractionRate > 0.8 {
		return "power_user", nil
	}

	if bd.ActiveDaysPerWeek >= 5 {
		return "regular", nil
	}

	return "casual", nil
}

func (s *Service) EstimateUserRetentionRisk(ctx context.Context, userID string) (float32, error) {
	bd, err := s.BuildBehaviorProfile(ctx, userID)
	if err != nil {
		return 0.5, err
	}

	risk := float32(bd.RetentionRate)

	if bd.TotalMemories < 5 {
		risk += 0.3
	}

	if bd.ActiveDaysPerWeek < 2 {
		risk += 0.2
	}

	if bd.InteractionRate < 0.2 {
		risk += 0.2
	}

	if risk > 1.0 {
		risk = 1.0
	}

	return risk, nil
}