package profiles

import (
	"context"
	"strings"
	"time"

	"agent-memory/internal/memory/types"
)

func (s *Service) UpdateUserPreferences(ctx context.Context, userID string, preferences map[string]interface{}) error {
	profile, err := s.GetUserProfile(ctx, userID)
	if err != nil {
		return err
	}

	for k, v := range preferences {
		profile.Preferences[k] = v
	}
	profile.UpdatedAt = time.Now()

	return s.saveProfile(ctx, profile)
}

func (s *Service) LearnPreferences(ctx context.Context, userID string) (map[string]interface{}, error) {
	memSvc, ok := s.memSvc.(interface{ GetMemoriesByUser(ctx context.Context, userID string, limit int) ([]*types.Memory, error) })
	if !ok {
		return make(map[string]interface{}), nil
	}

	memories, err := memSvc.GetMemoriesByUser(ctx, userID, 200)
	if err != nil || len(memories) == 0 {
		return make(map[string]interface{}), nil
	}

	prefs := make(map[string]interface{})

	timePrefs := make(map[string]int)
	formatPrefs := make(map[string]int)
	channelPrefs := make(map[string]int)

	timeWords := []string{"morning", "afternoon", "evening", "night", "deadline", "asap", "urgent"}
	formatWords := []string{"json", "text", "markdown", "table", "list", "summary", "detail"}
	channelWords := []string{"email", "slack", "chat", "message", "notification", "sms"}

	for _, mem := range memories {
		content := strings.ToLower(mem.Content)

		for _, w := range timeWords {
			if strings.Contains(content, w) {
				timePrefs[w]++
			}
		}
		for _, w := range formatWords {
			if strings.Contains(content, w) {
				formatPrefs[w]++
			}
		}
		for _, w := range channelWords {
			if strings.Contains(content, w) {
				channelPrefs[w]++
			}
		}
	}

	if len(timePrefs) > 0 {
		prefs["communication_time"] = topKey(timePrefs)
	}
	if len(formatPrefs) > 0 {
		prefs["preferred_format"] = topKey(formatPrefs)
	}
	if len(channelPrefs) > 0 {
		prefs["preferred_channel"] = topKey(channelPrefs)
	}

	prefs["last_updated"] = time.Now().Unix()

	return prefs, nil
}

func topKey(m map[string]int) string {
	max := 0
	var topKey string
	for k, v := range m {
		if v > max {
			max = v
			topKey = k
		}
	}
	return topKey
}

func (s *Service) saveProfile(ctx context.Context, profile *UserProfile) error {
	if s.graph != nil {
		return s.graph.UpdateProfile(ctx, profile)
	}
	return nil
}