package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"agent-memory/internal/memory"
	"agent-memory/internal/memory/types"
	"github.com/google/uuid"
)

type MCPClient struct {
	memSvc *memory.Service
	apiKey string
	userID string
}

type AddMemoryParams struct {
	Content  string                 `json:"content"`
	UserID   string                 `json:"userId,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type RecallParams struct {
	Query string `json:"query"`
	Limit int    `json:"limit,omitempty"`
}

type ProfileParams struct {
	ContainerTag string `json:"containerTag,omitempty"`
}

type ContextResult struct {
	UserID      string   `json:"userId"`
	Preferences []string `json:"preferences"`
	RecentActivity []string `json:"recentActivity"`
	Intent      string   `json:"intent"`
}

func NewMCPClient(memSvc *memory.Service, apiKey string, userID string) *MCPClient {
	return &MCPClient{
		memSvc: memSvc,
		apiKey: apiKey,
		userID: userID,
	}
}

func (c *MCPClient) AddMemory(params AddMemoryParams) (string, error) {
	if params.Content == "" {
		return "", fmt.Errorf("content is required")
	}

	userID := params.UserID
	if userID == "" {
		userID = c.userID
	}
	if userID == "" {
		userID = "default"
	}

	metadata := params.Metadata
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	metadata["source"] = "mcp"
	metadata["timestamp"] = time.Now().Unix()

	mem := &types.Memory{
		ID:       uuid.New().String(),
		Content:  params.Content,
		UserID:   userID,
		TenantID: "default",
		Type:     types.MemoryTypeUser,
		Metadata: metadata,
	}

	ctx := context.Background()
	created, err := c.memSvc.CreateMemory(ctx, mem)
	if err != nil {
		return "", fmt.Errorf("failed to create memory: %w", err)
	}

	return fmt.Sprintf("Memory added: %s", created.ID), nil
}

func (c *MCPClient) Recall(params RecallParams) (string, error) {
	query := params.Query
	if query == "" {
		return "", fmt.Errorf("query is required")
	}

	limit := params.Limit
	if limit == 0 {
		limit = 5
	}

	userID := c.userID
	if userID == "" {
		userID = "default"
	}

	ctx := context.Background()

	results, err := c.memSvc.SearchMemories(ctx, &types.SearchRequest{
		Query:  query,
		UserID: userID,
		Limit:  limit,
	})
	if err != nil {
		return "", fmt.Errorf("search failed: %w", err)
	}

	var output strings.Builder
	for i, r := range results {
		output.WriteString(fmt.Sprintf("[%d] %s (relevance: %.2f)\n", i+1, r.Text, r.Score))
	}

	if output.Len() == 0 {
		return "No relevant memories found.", nil
	}

	return output.String(), nil
}

func (c *MCPClient) GetContext() (string, error) {
	userID := c.userID
	if userID == "" {
		userID = "default"
	}

	ctx := context.Background()

	profile, err := c.buildProfile(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("failed to build profile: %w", err)
	}

	return formatContext(profile), nil
}

func (c *MCPClient) buildProfile(ctx context.Context, userID string) (*ContextResult, error) {
	memories, err := c.memSvc.GetMemoriesByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	profile := &ContextResult{
		UserID:        userID,
		Preferences:  []string{},
		RecentActivity: []string{},
	}

	recentCount := 0
	for _, mem := range memories {
		if recentCount >= 10 {
			break
		}

		content := strings.TrimSpace(mem.Content)
		if len(content) > 100 {
			content = content[:100] + "..."
		}
		profile.RecentActivity = append(profile.RecentActivity, content)
		recentCount++
	}

	return profile, nil
}

func formatContext(p *ContextResult) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("User: %s\n\n", p.UserID))

	if len(p.Preferences) > 0 {
		b.WriteString("Preferences:\n")
		for _, pref := range p.Preferences {
			b.WriteString(fmt.Sprintf("- %s\n", pref))
		}
		b.WriteString("\n")
	}

	if len(p.RecentActivity) > 0 {
		b.WriteString("Recent Activity:\n")
		for _, activity := range p.RecentActivity {
			b.WriteString(fmt.Sprintf("- %s\n", activity))
		}
	}

	if b.Len() == 0 {
		return "No context available."
	}

	return b.String()
}

type SearchResult struct {
	MemoryID string  `json:"memoryId"`
	Content string  `json:"content"`
	Score   float32 `json:"score"`
}

func (c *MCPClient) Search(params RecallParams) ([]SearchResult, error) {
	query := params.Query
	if query == "" {
		return nil, fmt.Errorf("query is required")
	}

	limit := params.Limit
	if limit == 0 {
		limit = 5
	}

	results, err := c.memSvc.SearchMemories(context.Background(), &types.SearchRequest{
		Query:  query,
		UserID: c.userID,
		Limit:  limit,
	})
	if err != nil {
		return nil, err
	}

	var searchResults []SearchResult
	for _, r := range results {
		searchResults = append(searchResults, SearchResult{
			MemoryID: r.MemoryID,
			Content: r.Text,
			Score:   r.Score,
		})
	}

	return searchResults, nil
}

func (c *MCPClient) WhoAmI() (string, error) {
	return fmt.Sprintf(`{"userId": "%s", "role": "user", "status": "active"}`, c.userID), nil
}

type Memory struct {
	ID        string                 `json:"id"`
	Content  string                 `json:"content"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt time.Time             `json:"createdAt"`
}

func (c *MCPClient) GetMemories(limit int) ([]Memory, error) {
	if limit == 0 {
		limit = 10
	}

	memories, err := c.memSvc.GetMemoriesByUser(context.Background(), c.userID)
	if err != nil {
		return nil, err
	}

	if len(memories) > limit {
		memories = memories[:limit]
	}

	var result []Memory
	for _, m := range memories {
		result = append(result, Memory{
			ID:        m.ID,
			Content:  m.Content,
			Metadata: m.Metadata,
			CreatedAt: m.CreatedAt,
		})
	}

	return result, nil
}

func (c *MCPClient) DeleteMemory(memoryID string) error {
	return c.memSvc.DeleteMemory(context.Background(), memoryID)
}

func (c *MCPClient) GetProfile() (map[string]interface{}, error) {
	ctx := context.Background()
	
	memories, err := c.memSvc.GetMemoriesByUser(ctx, c.userID)
	if err != nil {
		return nil, err
	}

	preferences := extractPreferences(memories)
	
	return map[string]interface{}{
		"userId":       c.userID,
		"preferences":  preferences,
		"memoryCount": len(memories),
	}, nil
}

func extractPreferences(memories []*types.Memory) []string {
	var prefs []string
	
	keywords := []string{"prefer", "like", "love", "hate", "dislike", "favorite", "best", "worst"}
	
	for _, mem := range memories {
		content := strings.ToLower(mem.Content)
		
		for _, keyword := range keywords {
			if strings.Contains(content, keyword) {
				if len(mem.Content) > 100 {
					prefs = append(prefs, mem.Content[:100])
				} else {
					prefs = append(prefs, mem.Content)
				}
				break
			}
		}
		
		if len(prefs) >= 5 {
			break
		}
	}
	
	return prefs
}

func (c *MCPClient) GetRecentMemories(limit int) ([]Memory, error) {
	if limit == 0 {
		limit = 5
	}

	memories, err := c.memSvc.GetMemoriesByUser(context.Background(), c.userID)
	if err != nil {
		return nil, err
	}

	var result []Memory
	for i := 0; i < len(memories) && i < limit; i++ {
		result = append(result, Memory{
			ID:        memories[i].ID,
			Content:  memories[i].Content,
			CreatedAt: memories[i].CreatedAt,
		})
	}

	return result, nil
}

func WriteMCPConfig() string {
	config := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"hystersis": map[string]interface{}{
				"command": "npx",
				"args": []string{"-y", "hystersis-mcp", "--api-key", "YOUR_API_KEY"},
			},
		},
	}

	b, _ := json.MarshalIndent(config, "", "  ")
	return string(b)
}