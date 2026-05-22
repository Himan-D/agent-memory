package analytics

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"agent-memory/internal/memory/types"
)

type MemoryServiceStats interface {
	QueryGraph(query string, params map[string]interface{}) ([]map[string]interface{}, error)
	GetMemoryStats(ctx context.Context, userID, orgID string) (*types.MemoryStats, error)
}

// PersistedAnalytics holds the counter values that survive restarts.
type PersistedAnalytics struct {
	SearchCount     int64     `json:"search_count"`
	TotalResults    int64     `json:"total_results"`
	ZeroResultCount int64     `json:"zero_result_count"`
	LastFlush       time.Time `json:"last_flush"`
}

// AnalyticsPersistence manages saving and loading analytics counters to/from disk.
type AnalyticsPersistence struct {
	filePath string
	mu       sync.Mutex
}

// NewAnalyticsPersistence creates a persistence helper for the given file path.
func NewAnalyticsPersistence(filePath string) *AnalyticsPersistence {
	return &AnalyticsPersistence{filePath: filePath}
}

// Load reads persisted counter values from disk. Returns zero values if the
// file does not yet exist.
func (p *AnalyticsPersistence) Load() (*PersistedAnalytics, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	data, err := os.ReadFile(p.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return &PersistedAnalytics{}, nil
		}
		return nil, fmt.Errorf("analytics persistence: load: %w", err)
	}
	var pa PersistedAnalytics
	if err := json.Unmarshal(data, &pa); err != nil {
		return nil, fmt.Errorf("analytics persistence: unmarshal: %w", err)
	}
	return &pa, nil
}

// Flush writes the provided counter values atomically to disk.
func (p *AnalyticsPersistence) Flush(pa *PersistedAnalytics) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	pa.LastFlush = time.Now()
	data, err := json.MarshalIndent(pa, "", "  ")
	if err != nil {
		return fmt.Errorf("analytics persistence: marshal: %w", err)
	}

	// Write to a temp file then rename for atomicity.
	tmp := p.filePath + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return fmt.Errorf("analytics persistence: write tmp: %w", err)
	}
	if err := os.Rename(tmp, p.filePath); err != nil {
		return fmt.Errorf("analytics persistence: rename: %w", err)
	}
	return nil
}

type Service struct {
	memorySvc       MemoryServiceStats
	searchCount     atomic.Int64
	totalResults    atomic.Int64
	zeroResultCount atomic.Int64
	persistence     *AnalyticsPersistence
	stopCh          chan struct{}
}

// NewService creates a Service with purely in-memory counters (no persistence).
func NewService(memorySvc MemoryServiceStats) *Service {
	return &Service{
		memorySvc: memorySvc,
		stopCh:    make(chan struct{}),
	}
}

// NewPersistentService creates a Service that loads prior counters from
// filePath on startup and flushes to disk every 60 seconds.
func NewPersistentService(memorySvc MemoryServiceStats, filePath string) (*Service, error) {
	if dir := analyticsDir(filePath); dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("analytics: mkdir: %w", err)
		}
	}

	p := NewAnalyticsPersistence(filePath)
	svc := &Service{
		memorySvc:   memorySvc,
		persistence: p,
		stopCh:      make(chan struct{}),
	}

	// Restore counters from disk.
	if pa, err := p.Load(); err == nil {
		svc.searchCount.Store(pa.SearchCount)
		svc.totalResults.Store(pa.TotalResults)
		svc.zeroResultCount.Store(pa.ZeroResultCount)
	}

	go svc.flushLoop()

	return svc, nil
}

// analyticsDir returns the directory portion of a file path.
func analyticsDir(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == '\\' {
			return path[:i]
		}
	}
	return ""
}

// FlushToDisk writes the current counter values to the persistence file.
func (s *Service) FlushToDisk() error {
	if s.persistence == nil {
		return nil
	}
	return s.persistence.Flush(&PersistedAnalytics{
		SearchCount:     s.searchCount.Load(),
		TotalResults:    s.totalResults.Load(),
		ZeroResultCount: s.zeroResultCount.Load(),
	})
}

// LoadFromDisk restores counters from the persistence file.
func (s *Service) LoadFromDisk() error {
	if s.persistence == nil {
		return nil
	}
	pa, err := s.persistence.Load()
	if err != nil {
		return err
	}
	s.searchCount.Store(pa.SearchCount)
	s.totalResults.Store(pa.TotalResults)
	s.zeroResultCount.Store(pa.ZeroResultCount)
	return nil
}

// Stop signals the background flush goroutine to exit and performs a final flush.
func (s *Service) Stop() {
	select {
	case <-s.stopCh:
		// already closed
	default:
		close(s.stopCh)
	}
	_ = s.FlushToDisk()
}

func (s *Service) flushLoop() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			_ = s.FlushToDisk()
		case <-s.stopCh:
			return
		}
	}
}

// RecordSearch tracks search activity with in-memory atomic counters.
// Call this asynchronously after every search operation.
func (s *Service) RecordSearch(resultCount int) {
	s.searchCount.Add(1)
	s.totalResults.Add(int64(resultCount))
	if resultCount == 0 {
		s.zeroResultCount.Add(1)
	}
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

	memoryCount, err := s.getMemoryCount(ctx, tenantID)
	if err == nil {
		dashboard.MemoryGrowth = MemoryGrowthMetrics{
			TotalCreated: memoryCount,
			ByCategory:   make(map[string]int64),
			ByType:       make(map[string]int64),
		}
	}

	searchAnalytics, err := s.getSearchAnalytics(ctx, tenantID)
	if err == nil {
		dashboard.SearchAnalytics = *searchAnalytics
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

func (s *Service) getMemoryCount(ctx context.Context, tenantID string) (int64, error) {
	query := `
		MATCH (m:Memory)
		WHERE m.tenant_id = $tenantID OR m.tenant_id IS NULL OR m.tenant_id = ""
		RETURN count(m) AS total
	`
	results, err := s.memorySvc.QueryGraph(query, map[string]interface{}{"tenantID": tenantID})
	if err != nil {
		return 0, err
	}
	if len(results) > 0 {
		if count, ok := results[0]["total"].(int64); ok {
			return count, nil
		}
	}
	return 0, nil
}

func (s *Service) getSearchAnalytics(ctx context.Context, tenantID string) (*SearchAnalyticsMetrics, error) {
	m := &SearchAnalyticsMetrics{}

	// Use in-memory atomic counters as the primary source (always accurate)
	totalSearches := s.searchCount.Load()
	totalResults := s.totalResults.Load()
	m.TotalSearches = totalSearches
	m.ZeroResultQueries = s.zeroResultCount.Load()
	if totalSearches > 0 {
		m.AvgResultsPerQuery = float64(totalResults) / float64(totalSearches)
	}

	// Supplement with Neo4j SearchEvent nodes if any exist
	query := `
		MATCH (s:SearchEvent)
		WHERE s.tenant_id = $tenantID OR $tenantID = "" OR s.tenant_id IS NULL
		RETURN count(s) AS total_searches,
			   avg(s.result_count) AS avg_results
		LIMIT 1
	`
	results, err := s.memorySvc.QueryGraph(query, map[string]interface{}{"tenantID": tenantID})
	if err == nil && len(results) > 0 {
		r := results[0]
		if ts, ok := r["total_searches"].(int64); ok && ts > totalSearches {
			m.TotalSearches = ts // use whichever is larger
		}
		if ar, ok := r["avg_results"].(float64); ok && m.AvgResultsPerQuery == 0 {
			m.AvgResultsPerQuery = ar
		}
	}

	return m, nil
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

	// Calculate AvgConfidence from the skill list (was always 0 before)
	if len(metrics.TopSkills) > 0 {
		var totalConf float64
		for _, sk := range metrics.TopSkills {
			totalConf += float64(sk.Confidence)
		}
		metrics.AvgConfidence = totalConf / float64(len(metrics.TopSkills))
	}

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
