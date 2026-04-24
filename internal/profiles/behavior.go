package profiles

import (
	"context"
	"strings"
	"time"

	"agent-memory/internal/memory/types"
)

func (s *Service) BuildBehaviorProfile(ctx context.Context, userID string) (*BehaviorData, error) {
	memSvc, ok := s.memSvc.(interface{ GetMemoriesByUser(ctx context.Context, userID string, limit int) ([]*types.Memory, error) })
	if !ok {
		return &BehaviorData{LastUpdated: time.Now()}, nil
	}

	memories, err := memSvc.GetMemoriesByUser(ctx, userID, 1000)
	if err != nil {
		return &BehaviorData{LastUpdated: time.Now()}, nil
	}

	bd := &BehaviorData{
		TotalMemories:      int64(len(memories)),
		FeatureUsage:      make(map[string]int),
		SearchPatterns:    []string{},
		LastUpdated:       time.Now(),
	}

	var categories map[string]int
	var agents map[string]int
	var sessions []time.Time

	for _, mem := range memories {
		if mem.Type != "" {
			bd.FeatureUsage[string(mem.Type)]++
		}

		memLower := strings.ToLower(mem.Content)
		if strings.Contains(memLower, "search") || strings.Contains(memLower, "find") || strings.Contains(memLower, "lookup") {
			bd.TotalSearches++
			if len(bd.SearchPatterns) < 10 {
				if len(mem.Content) > 50 {
					bd.SearchPatterns = append(bd.SearchPatterns, mem.Content[:50])
				}
			}
		}

		if mem.AgentID != "" {
			if agents == nil {
				agents = make(map[string]int)
			}
			agents[mem.AgentID]++
		}

		if mem.CreatedAt.After(time.Now().AddDate(0, -1, 0)) {
			sessions = append(sessions, mem.CreatedAt)
		}
	}

	if categories != nil {
		bd.TopCategories = topN(categories, 5)
	}
	if agents != nil {
		bd.TopAgents = topN(agents, 5)
	}

	bd.TotalSessions = int64(len(sessions))

	if len(sessions) > 0 {
		hours := make([]int, len(sessions))
		for i, t := range sessions {
			hours[i] = t.Hour()
		}
		bd.PreferredTime = preferredTimeOfDay(hours)
		bd.ActiveDaysPerWeek = activeDaysPerWeek(sessions)
	}

	if bd.TotalMemories > 0 {
		bd.InteractionRate = float32(bd.TotalMemories) / 30.0
		if bd.InteractionRate > 1.0 {
			bd.InteractionRate = 1.0
		}
	}

	return bd, nil
}

func topN(m map[string]int, n int) []string {
	type kv struct {
		k string
		v int
	}
	var ss []kv
	for k, v := range m {
		ss = append(ss, kv{k, v})
	}
	for i := 0; i < len(ss)-1; i++ {
		for j := i + 1; j < len(ss); j++ {
			if ss[j].v > ss[i].v {
				ss[i], ss[j] = ss[j], ss[i]
			}
		}
	}
	var result []string
	for i := 0; i < n && i < len(ss); i++ {
		result = append(result, ss[i].k)
	}
	return result
}

func preferredTimeOfDay(hours []int) string {
	if len(hours) == 0 {
		return "unknown"
	}
	morning, afternoon, evening, night := 0, 0, 0, 0
	for _, h := range hours {
		switch {
		case h >= 6 && h < 12:
			morning++
		case h >= 12 && h < 17:
			afternoon++
		case h >= 17 && h < 21:
			evening++
		default:
			night++
		}
	}
	max := morning
	timeStr := "morning"
	if afternoon > max {
		max = afternoon
		timeStr = "afternoon"
	}
	if evening > max {
		max = evening
		timeStr = "evening"
	}
	if night > max {
		timeStr = "night"
	}
	return timeStr
}

func activeDaysPerWeek(sessions []time.Time) int {
	if len(sessions) == 0 {
		return 0
	}
	days := make(map[int]int)
	weekAgo := time.Now().AddDate(0, 0, -7)
	for _, s := range sessions {
		if s.After(weekAgo) {
			day := int(s.Sub(weekAgo).Hours() / 24)
			days[day]++
		}
	}
	return len(days)
}