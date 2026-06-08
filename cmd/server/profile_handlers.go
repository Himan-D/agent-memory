package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"agent-memory/internal/memory/types"
	"agent-memory/internal/sources"
)

type profileResponse struct {
	UserID             string                 `json:"user_id,omitempty"`
	OrgID              string                 `json:"org_id,omitempty"`
	MemoryCount        int                    `json:"memory_count"`
	Preferences        map[string][]string    `json:"preferences"`
	RecentActivity     []activityItem         `json:"recent_activity"`
	FrequentCategories map[string]int         `json:"frequent_categories"`
	TopTags            map[string]int         `json:"top_tags"`
	TopSourceTopics    []string               `json:"top_source_topics"`
	Signals            map[string]interface{} `json:"signals"`
	UpdatedAt          time.Time              `json:"updated_at"`
}

type activityItem struct {
	ID        string    `json:"id"`
	Content   string    `json:"content"`
	Category  string    `json:"category,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type agentContextResponse struct {
	Role      string           `json:"role"`
	Content   string           `json:"content"`
	Profile   *profileResponse `json:"profile"`
	Memories  []activityItem   `json:"memories"`
	UpdatedAt time.Time        `json:"updated_at"`
}

func (s *APIServer) profileHandler(w http.ResponseWriter, r *http.Request) {
	memories, err := s.profileMemories(r)
	if err != nil {
		safeHTTPError(w, r, fmt.Errorf("profile memories: %w", err), http.StatusInternalServerError)
		return
	}
	profile := buildProfile(r.URL.Query().Get("user_id"), r.URL.Query().Get("org_id"), memories, parseIntParam(r, "recent_limit", 10))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(profile)
}

func (s *APIServer) agentContextHandler(w http.ResponseWriter, r *http.Request) {
	memories, err := s.profileMemories(r)
	if err != nil {
		safeHTTPError(w, r, fmt.Errorf("context memories: %w", err), http.StatusInternalServerError)
		return
	}
	limit := parseIntParam(r, "limit", 12)
	profile := buildProfile(r.URL.Query().Get("user_id"), r.URL.Query().Get("org_id"), memories, limit)
	contextMemories := recentActivity(memories, limit)
	resp := agentContextResponse{
		Role:      "system",
		Content:   renderAgentContext(profile, contextMemories),
		Profile:   profile,
		Memories:  contextMemories,
		UpdatedAt: time.Now().UTC(),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *APIServer) profileMemories(r *http.Request) ([]*types.Memory, error) {
	userID := r.URL.Query().Get("user_id")
	orgID := r.URL.Query().Get("org_id")
	switch {
	case userID != "":
		return s.memSvc.GetMemoriesByUser(r.Context(), userID)
	case orgID != "":
		return s.memSvc.GetMemoriesByOrg(r.Context(), orgID)
	default:
		return s.memSvc.GetAllMemories(r.Context())
	}
}

func buildProfile(userID, orgID string, memories []*types.Memory, recentLimit int) *profileResponse {
	categoryCounts := map[string]int{}
	tagCounts := map[string]int{}
	preferences := map[string][]string{
		"likes":         {},
		"dislikes":      {},
		"goals":         {},
		"communication": {},
		"tools":         {},
		"topics":        {},
	}
	sourceTopics := map[string]int{}

	for _, mem := range memories {
		if mem == nil {
			continue
		}
		if mem.Category != "" {
			categoryCounts[mem.Category]++
		}
		for _, tag := range mem.Tags {
			tag = strings.TrimSpace(strings.ToLower(tag))
			if tag != "" {
				tagCounts[tag]++
			}
		}
		classifyPreference(preferences, mem.Content)
		if mem.Category == sources.CategorySource || mem.Category == sources.CategorySourceChunk {
			for _, topic := range topicTokens(mem.Content) {
				sourceTopics[topic]++
			}
		}
	}

	return &profileResponse{
		UserID:             userID,
		OrgID:              orgID,
		MemoryCount:        len(memories),
		Preferences:        dedupePreferenceMap(preferences),
		RecentActivity:     recentActivity(memories, recentLimit),
		FrequentCategories: categoryCounts,
		TopTags:            topCounts(tagCounts, 12),
		TopSourceTopics:    topKeys(sourceTopics, 12),
		Signals: map[string]interface{}{
			"has_sources":        categoryCounts[sources.CategorySource] > 0 || categoryCounts[sources.CategorySourceChunk] > 0,
			"category_count":     len(categoryCounts),
			"tag_count":          len(tagCounts),
			"source_topic_count": len(sourceTopics),
		},
		UpdatedAt: time.Now().UTC(),
	}
}

func recentActivity(memories []*types.Memory, limit int) []activityItem {
	if limit <= 0 {
		limit = 10
	}
	cp := append([]*types.Memory(nil), memories...)
	sort.Slice(cp, func(i, j int) bool {
		return cp[i].CreatedAt.After(cp[j].CreatedAt)
	})
	if len(cp) > limit {
		cp = cp[:limit]
	}
	items := make([]activityItem, 0, len(cp))
	for _, mem := range cp {
		if mem == nil {
			continue
		}
		items = append(items, activityItem{
			ID:        mem.ID,
			Content:   truncateText(mem.Content, 240),
			Category:  mem.Category,
			CreatedAt: mem.CreatedAt,
		})
	}
	return items
}

func classifyPreference(preferences map[string][]string, content string) {
	lower := strings.ToLower(content)
	switch {
	case strings.Contains(lower, "likes ") || strings.Contains(lower, "prefers ") || strings.Contains(lower, "preference"):
		preferences["likes"] = append(preferences["likes"], truncateText(content, 180))
	case strings.Contains(lower, "dislikes ") || strings.Contains(lower, "does not like") || strings.Contains(lower, "avoid"):
		preferences["dislikes"] = append(preferences["dislikes"], truncateText(content, 180))
	case strings.Contains(lower, "goal") || strings.Contains(lower, "wants to") || strings.Contains(lower, "needs to"):
		preferences["goals"] = append(preferences["goals"], truncateText(content, 180))
	case strings.Contains(lower, "communicate") || strings.Contains(lower, "tone") || strings.Contains(lower, "writing style"):
		preferences["communication"] = append(preferences["communication"], truncateText(content, 180))
	case strings.Contains(lower, "uses ") || strings.Contains(lower, "tool") || strings.Contains(lower, "stack"):
		preferences["tools"] = append(preferences["tools"], truncateText(content, 180))
	}
	for _, token := range topicTokens(content) {
		preferences["topics"] = append(preferences["topics"], token)
	}
}

func dedupePreferenceMap(in map[string][]string) map[string][]string {
	out := map[string][]string{}
	for key, values := range in {
		seen := map[string]bool{}
		for _, value := range values {
			value = strings.TrimSpace(value)
			if value == "" || seen[value] {
				continue
			}
			seen[value] = true
			out[key] = append(out[key], value)
			if len(out[key]) >= 10 {
				break
			}
		}
	}
	return out
}

func topicTokens(content string) []string {
	words := strings.FieldsFunc(strings.ToLower(content), func(r rune) bool {
		return !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9') && r != '-' && r != '_'
	})
	stop := map[string]bool{"the": true, "and": true, "for": true, "with": true, "from": true, "that": true, "this": true, "memory": true, "user": true, "source": true}
	counts := map[string]int{}
	for _, word := range words {
		if len(word) < 4 || stop[word] {
			continue
		}
		counts[word]++
	}
	return topKeys(counts, 8)
}

func topCounts(in map[string]int, limit int) map[string]int {
	out := map[string]int{}
	for _, key := range topKeys(in, limit) {
		out[key] = in[key]
	}
	return out
}

func topKeys(in map[string]int, limit int) []string {
	type kv struct {
		Key   string
		Count int
	}
	items := make([]kv, 0, len(in))
	for k, v := range in {
		items = append(items, kv{k, v})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Count == items[j].Count {
			return items[i].Key < items[j].Key
		}
		return items[i].Count > items[j].Count
	})
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, item.Key)
	}
	return out
}

func renderAgentContext(profile *profileResponse, memories []activityItem) string {
	var b strings.Builder
	b.WriteString("Hystersis memory context\n")
	if profile.UserID != "" {
		b.WriteString("User: " + profile.UserID + "\n")
	}
	if profile.OrgID != "" {
		b.WriteString("Org: " + profile.OrgID + "\n")
	}
	b.WriteString(fmt.Sprintf("Memories available: %d\n", profile.MemoryCount))
	if len(profile.TopSourceTopics) > 0 {
		b.WriteString("Source topics: " + strings.Join(profile.TopSourceTopics, ", ") + "\n")
	}
	if len(profile.Preferences["goals"]) > 0 {
		b.WriteString("Goals:\n")
		for _, goal := range profile.Preferences["goals"] {
			b.WriteString("- " + goal + "\n")
		}
	}
	if len(memories) > 0 {
		b.WriteString("Recent memories:\n")
		for _, mem := range memories {
			b.WriteString("- " + mem.Content + "\n")
		}
	}
	return b.String()
}

func truncateText(text string, max int) string {
	text = strings.TrimSpace(text)
	if max <= 0 || len(text) <= max {
		return text
	}
	return strings.TrimSpace(text[:max]) + "..."
}
