package analytics

import (
	"context"
	"time"

	"agent-memory/internal/memory/types"
)

type MemoryServiceStats interface {
	QueryGraph(query string, params map[string]interface{}) ([]map[string]interface{}, error)
	GetMemoryStats(ctx context.Context, userID, orgID string) (*types.MemoryStats, error)
}

type Service struct {
	memorySvc MemoryServiceStats
}

func NewService(memorySvc MemoryServiceStats) *Service {
	return &Service{memorySvc: memorySvc}
}

type DashboardResponse struct {
	Period          string                    `json:"period"`
	GeneratedAt     time.Time                 `json:"generated_at"`
	MemoryGrowth    MemoryGrowthMetrics       `json:"memory_growth"`
	SearchAnalytics SearchAnalyticsMetrics    `json:"search_analytics"`
	SkillMetrics    SkillEffectivenessMetrics `json:"skill_metrics"`
	AgentActivity   []AgentActivityMetrics    `json:"agent_activity"`
	Retention       RetentionMetrics          `json:"retention"`
}

type MemoryGrowthMetrics struct {
	TotalCreated  int64            `json:"total_created"`
	TotalArchived int64            `json:"total_archived"`
	TotalDeleted  int64            `json:"total_deleted"`
	ByCategory    map[string]int64 `json:"by_category"`
	ByType        map[string]int64 `json:"by_type"`
	ByImportance  map[string]int64 `json:"by_importance"`
	DailyTrend    []DailyCount     `json:"daily_trend,omitempty"`
}

type DailyCount struct {
	Date  string `json:"date"`
	Count int64  `json:"count"`
}

type SearchAnalyticsMetrics struct {
	TotalSearches      int64          `json:"total_searches"`
	AvgResultsPerQuery float64        `json:"avg_results_per_query"`
	TopQueries         []QueryCount   `json:"top_queries"`
	ZeroResultQueries  int64          `json:"zero_result_queries"`
	TopRecallMemories  []MemoryRecall `json:"top_recall_memories"`
}

type QueryCount struct {
	Query string `json:"query"`
	Count int64  `json:"count"`
}

type MemoryRecall struct {
	MemoryID    string `json:"memory_id"`
	Content     string `json:"content"`
	RecallCount int64  `json:"recall_count"`
}

type SkillEffectivenessMetrics struct {
	TotalSkills    int64             `json:"total_skills"`
	ActiveSkills   int64             `json:"active_skills"`
	TopSkills      []SkillUsage      `json:"top_skills"`
	ChainUsage     ChainUsageMetrics `json:"chain_usage"`
	AvgConfidence  float64           `json:"avg_confidence"`
	SkillsByDomain map[string]int64  `json:"skills_by_domain"`
}

type SkillUsage struct {
	SkillID     string  `json:"skill_id"`
	Name        string  `json:"name"`
	UsageCount  int64   `json:"usage_count"`
	SuccessRate float64 `json:"success_rate"`
	Confidence  float32 `json:"confidence"`
}

type ChainUsageMetrics struct {
	TotalChains      int64   `json:"total_chains"`
	TotalExecutions  int64   `json:"total_executions"`
	SuccessRate      float64 `json:"success_rate"`
	AvgStepsPerChain float64 `json:"avg_steps_per_chain"`
}

type AgentActivityMetrics struct {
	AgentID          string     `json:"agent_id"`
	AgentName        string     `json:"agent_name,omitempty"`
	SessionCount     int64      `json:"session_count"`
	MemoryCount      int64      `json:"memory_count"`
	SkillInvocations int64      `json:"skill_invocations"`
	LastActive       *time.Time `json:"last_active,omitempty"`
}

type RetentionMetrics struct {
	Period             string  `json:"period"`
	ActiveUsers        int64   `json:"active_users"`
	ReturningUsers     int64   `json:"returning_users"`
	RetentionRate      float64 `json:"retention_rate"`
	AvgMemoriesPerUser float64 `json:"avg_memories_per_user"`
}

func (s *Service) GetDashboard(ctx context.Context, tenantID, period string) (*DashboardResponse, error) {
	if period == "" {
		period = "7d"
	}

	dashboard := &DashboardResponse{
		Period:      period,
		GeneratedAt: time.Now(),
	}

	memoryStats, err := s.memorySvc.GetMemoryStats(ctx, tenantID, "")
	if err == nil && memoryStats != nil {
		dashboard.MemoryGrowth = MemoryGrowthMetrics{
			TotalCreated: memoryStats.TotalMemories,
			ByCategory:   memoryStats.ByCategory,
			ByType:       memoryStats.ByType,
		}
	}

	agentActivity, err := s.getAgentActivity(ctx, tenantID)
	if err == nil {
		dashboard.AgentActivity = agentActivity
	}

	skillMetrics, err := s.getSkillMetrics(ctx, tenantID)
	if err == nil {
		dashboard.SkillMetrics = *skillMetrics
	}

	retentionMetrics, err := s.getRetentionMetrics(ctx, tenantID)
	if err == nil {
		dashboard.Retention = *retentionMetrics
	}

	return dashboard, nil
}

func (s *Service) getAgentActivity(ctx context.Context, tenantID string) ([]AgentActivityMetrics, error) {
	query := `
		MATCH (a:Agent)
		WHERE a.tenant_id = $tenantID OR $tenantID = ""
		OPTIONAL MATCH (a)-[:CREATED]->(s:Session)
		OPTIONAL MATCH (a)-[:CREATED]->(m:Memory)
		OPTIONAL MATCH (a)-[:USED]->(sk:Skill)
		RETURN a.id AS agent_id, a.name AS agent_name,
			   count(DISTINCT s) AS session_count,
			   count(DISTINCT m) AS memory_count,
			   count(DISTINCT sk) AS skill_invocations,
			   max(a.updated_at) AS last_active
		LIMIT 50
	`

	results, err := s.memorySvc.QueryGraph(query, map[string]interface{}{"tenantID": tenantID})
	if err != nil {
		return nil, err
	}

	var activity []AgentActivityMetrics
	for _, r := range results {
		metrics := AgentActivityMetrics{
			AgentID: getString(r, "agent_id"),
		}
		if name, ok := r["agent_name"].(string); ok {
			metrics.AgentName = name
		}
		if sc, ok := r["session_count"].(int64); ok {
			metrics.SessionCount = sc
		}
		if mc, ok := r["memory_count"].(int64); ok {
			metrics.MemoryCount = mc
		}
		if si, ok := r["skill_invocations"].(int64); ok {
			metrics.SkillInvocations = si
		}
		activity = append(activity, metrics)
	}

	return activity, nil
}

func (s *Service) getSkillMetrics(ctx context.Context, tenantID string) (*SkillEffectivenessMetrics, error) {
	query := `
		MATCH (sk:Skill)
		WHERE sk.tenant_id = $tenantID OR $tenantID = ""
		RETURN sk.id AS skill_id, sk.name AS name,
			   sk.usage_count AS usage_count,
			   sk.confidence AS confidence,
			   sk.domain AS domain,
			   sk.verified AS verified
		ORDER BY sk.usage_count DESC
		LIMIT 20
	`

	results, err := s.memorySvc.QueryGraph(query, map[string]interface{}{"tenantID": tenantID})
	if err != nil {
		return nil, err
	}

	metrics := &SkillEffectivenessMetrics{}
	byDomain := make(map[string]int64)

	for _, r := range results {
		usage := SkillUsage{
			SkillID: getString(r, "skill_id"),
			Name:    getString(r, "name"),
		}
		if uc, ok := r["usage_count"].(int64); ok {
			usage.UsageCount = uc
			metrics.TotalSkills++
		}
		if conf, ok := r["confidence"].(float32); ok {
			usage.Confidence = conf
		}
		domain := getString(r, "domain")
		if domain != "" {
			byDomain[domain]++
		}
		metrics.TopSkills = append(metrics.TopSkills, usage)
	}

	metrics.SkillsByDomain = byDomain
	return metrics, nil
}

func getString(r map[string]interface{}, key string) string {
	if v, ok := r[key].(string); ok {
		return v
	}
	return ""
}

func (s *Service) getRetentionMetrics(ctx context.Context, tenantID string) (*RetentionMetrics, error) {
	activeUsersQuery := `
		MATCH (u:User)
		WHERE u.tenant_id = $tenantID OR $tenantID = ""
		WITH count(u) AS total_users
		MATCH (u:User)-[:CREATED]->(m:Memory)
		WHERE u.tenant_id = $tenantID OR $tenantID = ""
		WITH total_users, count(DISTINCT u) AS active_users,
			 collect(DISTINCT u.id) AS user_ids
		RETURN total_users, active_users,
			   CASE WHEN total_users > 0 THEN toFloat(active_users) / total_users ELSE 0.0 END AS retention_rate
	`

	results, err := s.memorySvc.QueryGraph(activeUsersQuery, map[string]interface{}{"tenantID": tenantID})
	if err != nil {
		return nil, err
	}

	metrics := &RetentionMetrics{}

	if len(results) > 0 {
		r := results[0]
		if tu, ok := r["total_users"].(int64); ok {
			metrics.ActiveUsers = tu
		}
		if au, ok := r["active_users"].(int64); ok {
			metrics.ReturningUsers = au
		}
		if rr, ok := r["retention_rate"].(float64); ok {
			metrics.RetentionRate = rr * 100
		}
	}

	avgMemoriesQuery := `
		MATCH (u:User)-[:CREATED]->(m:Memory)
		WHERE u.tenant_id = $tenantID OR $tenantID = ""
		WITH u, count(m) AS memory_count
		RETURN avg(memory_count) AS avg_memories
	`

	avgResults, err := s.memorySvc.QueryGraph(avgMemoriesQuery, map[string]interface{}{"tenantID": tenantID})
	if err == nil && len(avgResults) > 0 {
		if am, ok := avgResults[0]["avg_memories"].(float64); ok {
			metrics.AvgMemoriesPerUser = am
		}
	}

	return metrics, nil
}
