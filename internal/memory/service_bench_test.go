package memory

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"agent-memory/internal/config"
	"agent-memory/internal/memory/types"
)

// mockVectorStore implements VectorStore for benchmarking.
type mockVectorStore struct {
	SearchFunc func(ctx context.Context, query []float32, limit int, threshold float32, filters map[string]interface{}) ([]types.MemoryResult, error)
}

func (m *mockVectorStore) StoreEmbedding(ctx context.Context, text string, id string, embedding []float32, metadata map[string]interface{}) (string, error) {
	return id, nil
}
func (m *mockVectorStore) Search(ctx context.Context, query []float32, limit int, threshold float32, filters map[string]interface{}) ([]types.MemoryResult, error) {
	if m.SearchFunc != nil {
		return m.SearchFunc(ctx, query, limit, threshold, filters)
	}
	return []types.MemoryResult{}, nil
}
func (m *mockVectorStore) UpdateMemory(ctx context.Context, id string, text string, metadata map[string]interface{}) error {
	return nil
}
func (m *mockVectorStore) DeleteMemory(ctx context.Context, id string) error { return nil }
func (m *mockVectorStore) UpdateVector(ctx context.Context, id string, embedding []float32) error {
	return nil
}
func (m *mockVectorStore) Ping(ctx context.Context) error { return nil }
func (m *mockVectorStore) Close() error            { return nil }

// mockGraphStore implements GraphStore for benchmarking.
type mockGraphStore struct {
	GetMemoryFunc       func(id string) (*types.Memory, error)
	GetMemoriesByIDsFunc func(ids []string) ([]*types.Memory, error)
}

func (m *mockGraphStore) Close() error                                   { return nil }
func (m *mockGraphStore) Ping(ctx context.Context) error                  { return nil }
func (m *mockGraphStore) CreateMemory(mem *types.Memory) error            { return nil }
func (m *mockGraphStore) BatchCreateMemories(memories []*types.Memory) error { return nil }
func (m *mockGraphStore) GetMemory(id string) (*types.Memory, error) {
	if m.GetMemoryFunc != nil {
		return m.GetMemoryFunc(id)
	}
	return nil, nil
}
func (m *mockGraphStore) UpdateMemory(mem *types.Memory) error { return nil }
func (m *mockGraphStore) DeleteMemory(id string) error        { return nil }
func (m *mockGraphStore) BatchDeleteMemories(ids []string) error { return nil }
func (m *mockGraphStore) UpdateMemoryAccess(id string, ts time.Time) error { return nil }
func (m *mockGraphStore) UpdateMemoryFeedbackScore(id string, fbType types.FeedbackType) error {
	return nil
}
func (m *mockGraphStore) GetMemoriesByUser(userID string) ([]*types.Memory, error) { return nil, nil }
func (m *mockGraphStore) GetMemoriesByOrg(orgID string) ([]*types.Memory, error)   { return nil, nil }
func (m *mockGraphStore) GetMemoriesByHash(userID, hash string) (string, error)    { return "", nil }
func (m *mockGraphStore) GetAllMemories() ([]*types.Memory, error)                 { return nil, nil }
func (m *mockGraphStore) GetExpiredMemories() ([]*types.Memory, error)             { return nil, nil }
func (m *mockGraphStore) RecordHistory(memID, action, oldContent, newContent, userID, comment string) error {
	return nil
}
func (m *mockGraphStore) GetMemoryHistory(memID string) ([]types.MemoryHistory, error) { return nil, nil }
func (m *mockGraphStore) AdvancedSearch(filters *types.SearchFilters) ([]*types.Memory, error) {
	return nil, nil
}
func (m *mockGraphStore) BulkDeleteByFilter(userID, orgID, category string) (int, error) {
	return 0, nil
}
func (m *mockGraphStore) CreateSession(agentID string, metadata map[string]interface{}) (*types.Session, error) {
	return nil, nil
}
func (m *mockGraphStore) ListSessions() ([]*types.Session, error) { return nil, nil }
func (m *mockGraphStore) GetMessages(sessionID string, limit int) ([]types.Message, error) {
	return nil, nil
}
func (m *mockGraphStore) ClearMessages(sessionID string) error { return nil }
func (m *mockGraphStore) AddEntity(entity types.Entity) error   { return nil }
func (m *mockGraphStore) GetEntity(id string) (*types.Entity, error) { return nil, nil }
func (m *mockGraphStore) ListEntities(tenantID string, limit int) ([]types.Entity, error) {
	return nil, nil
}
func (m *mockGraphStore) AddRelation(fromID, toID, relType string, props map[string]interface{}) error {
	return nil
}
func (m *mockGraphStore) DeleteRelation(fromID, toID, relType string) error { return nil }
func (m *mockGraphStore) QueryGraph(cypher string, params map[string]interface{}) ([]map[string]interface{}, error) {
	return nil, nil
}
func (m *mockGraphStore) Traverse(fromEntityID string, depth int) ([]types.Path, error) { return nil, nil }
func (m *mockGraphStore) GetEntityRelations(entityID string, relType string) ([]types.Relation, error) {
	return nil, nil
}
func (m *mockGraphStore) LinkMemoryEntity(memoryID, entityID string) error { return nil }
func (m *mockGraphStore) SearchByContent(query string, limit int) ([]types.MemoryResult, error) {
	return nil, nil
}
func (m *mockGraphStore) SearchByEntities(entities []string, limit int) ([]types.MemoryResult, error) {
	return nil, nil
}
func (m *mockGraphStore) GetMemoryIDsByEntity(entityID string) ([]string, error) { return nil, nil }
func (m *mockGraphStore) GetEntitiesByMemory(memoryID string) ([]types.Entity, error) { return nil, nil }
func (m *mockGraphStore) GetMemoriesPaginated(req *types.SearchRequest) ([]*types.Memory, int64, error) {
	return nil, 0, nil
}
func (m *mockGraphStore) GetMemoriesByIDs(ids []string) ([]*types.Memory, error) {
	if m.GetMemoriesByIDsFunc != nil {
		return m.GetMemoriesByIDsFunc(ids)
	}
	return nil, nil
}
func (m *mockGraphStore) BatchUpdateSyncTime(entityIDs []string) error { return nil }
func (m *mockGraphStore) CreateFeedback(feedback *types.Feedback) error { return nil }
func (m *mockGraphStore) GetFeedbackByType(fbType types.FeedbackType, limit int) ([]*types.Feedback, error) {
	return nil, nil
}
func (m *mockGraphStore) GetFeedbackByMemory(ctx context.Context, memoryID string) ([]*types.Feedback, error) {
	return nil, nil
}
func (m *mockGraphStore) CreateMemoryLink(link *types.MemoryLink) error { return nil }
func (m *mockGraphStore) GetMemoryLinks(memoryID string) ([]types.MemoryLink, error) { return nil, nil }
func (m *mockGraphStore) DeleteMemoryLink(linkID string) error { return nil }
func (m *mockGraphStore) CreateMemoryVersion(version *types.MemoryVersion) error { return nil }
func (m *mockGraphStore) GetMemoryVersions(memoryID string) ([]types.MemoryVersion, error) {
	return nil, nil
}
func (m *mockGraphStore) CreateSkill(ctx context.Context, skill *types.Skill) error { return nil }
func (m *mockGraphStore) ListSkills(ctx context.Context, tenantID, domain string, limit, offset int) ([]*types.Skill, error) {
	return nil, nil
}
func (m *mockGraphStore) GetSkill(ctx context.Context, skillID string) (*types.Skill, error) {
	return nil, nil
}
func (m *mockGraphStore) UpdateSkill(ctx context.Context, skill *types.Skill) error { return nil }
func (m *mockGraphStore) DeleteSkill(ctx context.Context, skillID string) error { return nil }
func (m *mockGraphStore) GetSkillsByTrigger(ctx context.Context, trigger string, limit int) ([]*types.Skill, error) {
	return nil, nil
}
func (m *mockGraphStore) GetSkillsByDomain(ctx context.Context, domain string, limit int) ([]*types.Skill, error) {
	return nil, nil
}
func (m *mockGraphStore) GetSimilarSkills(ctx context.Context, skillID string, limit int) ([]*types.Skill, error) {
	return nil, nil
}
func (m *mockGraphStore) IncrementSkillUsage(ctx context.Context, skillID string) error { return nil }
func (m *mockGraphStore) CreateSkillReview(ctx context.Context, review *types.SkillReview) error {
	return nil
}
func (m *mockGraphStore) CreateAgent(ctx context.Context, agent *types.Agent) error { return nil }
func (m *mockGraphStore) GetAgent(ctx context.Context, agentID string) (*types.Agent, error) {
	return nil, nil
}
func (m *mockGraphStore) UpdateAgent(ctx context.Context, agent *types.Agent) error { return nil }
func (m *mockGraphStore) DeleteAgent(ctx context.Context, agentID string) error { return nil }
func (m *mockGraphStore) ListAgents(ctx context.Context, tenantID string, limit, offset int) ([]*types.Agent, int64, error) {
	return nil, 0, nil
}
func (m *mockGraphStore) CreateAgentGroup(ctx context.Context, group *types.AgentGroup) error {
	return nil
}
func (m *mockGraphStore) GetAgentGroup(ctx context.Context, groupID string) (*types.AgentGroup, error) {
	return nil, nil
}
func (m *mockGraphStore) UpdateAgentGroup(ctx context.Context, group *types.AgentGroup) error {
	return nil
}
func (m *mockGraphStore) DeleteAgentGroup(ctx context.Context, groupID string) error { return nil }
func (m *mockGraphStore) ListAgentGroups(ctx context.Context, tenantID string, limit, offset int) ([]*types.AgentGroup, int64, error) {
	return nil, 0, nil
}
func (m *mockGraphStore) AddAgentToGroup(ctx context.Context, agentID, groupID string, role types.MemberRole) error {
	return nil
}
func (m *mockGraphStore) RemoveAgentFromGroup(ctx context.Context, agentID, groupID string) error {
	return nil
}
func (m *mockGraphStore) GetGroupSkills(ctx context.Context, groupID string, limit int) ([]*types.Skill, error) {
	return nil, nil
}
func (m *mockGraphStore) GetGroupMemories(ctx context.Context, groupID string) ([]*types.Memory, error) {
	return nil, nil
}
func (m *mockGraphStore) ShareMemoryToGroup(ctx context.Context, memoryID, groupID, role string) error {
	return nil
}
func (m *mockGraphStore) ListPendingReviews(ctx context.Context, tenantID string) ([]*types.SkillReview, error) {
	return nil, nil
}
func (m *mockGraphStore) GetReview(ctx context.Context, reviewID string) (*types.SkillReview, error) {
	return nil, nil
}
func (m *mockGraphStore) ProcessReview(ctx context.Context, reviewID string, approved bool, notes string) error {
	return nil
}
func (m *mockGraphStore) CreateChain(ctx context.Context, chain *types.SkillChain) error { return nil }
func (m *mockGraphStore) GetChain(ctx context.Context, chainID string) (*types.SkillChain, error) {
	return nil, nil
}
func (m *mockGraphStore) ListChains(ctx context.Context, tenantID string, query *types.ChainQuery) ([]*types.SkillChain, error) {
	return nil, nil
}
func (m *mockGraphStore) UpdateChain(ctx context.Context, chain *types.SkillChain) error { return nil }
func (m *mockGraphStore) DeleteChain(ctx context.Context, chainID string) error { return nil }
func (m *mockGraphStore) GetChainExecutions(ctx context.Context, chainID string, limit int) ([]*types.ChainExecution, error) {
	return nil, nil
}
func (m *mockGraphStore) UpdateChainExecution(ctx context.Context, exec *types.ChainExecution) error {
	return nil
}
func (m *mockGraphStore) IncrementChainUsage(ctx context.Context, chainID string) error { return nil }

func BenchmarkSearchMemoriesExpansion(b *testing.B) {
	// Mock OpenAI Embedding API
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Input interface{} `json:"input"`
		}
		json.NewDecoder(r.Body).Decode(&req)

		var data []map[string]interface{}
		switch input := req.Input.(type) {
		case string:
			data = append(data, map[string]interface{}{
				"embedding": make([]float32, 1536),
				"index":     0,
			})
		case []interface{}:
			for i := range input {
				data = append(data, map[string]interface{}{
					"embedding": make([]float32, 1536),
					"index":     i,
				})
			}
		}

		resp := map[string]interface{}{
			"data": data,
			"usage": map[string]interface{}{
				"total_tokens": 10,
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cfg := &config.Config{
		OpenAI: config.OpenAIConfig{
			APIKey:  "test-key",
			BaseURL: server.URL,
		},
		Memory: config.MemoryConfig{
			MultiSignalEnabled: false,
		},
		App: config.AppConfig{
			BufferTimeout: 5 * time.Second,
		},
	}

	svc, _ := NewService(cfg)
	svc.vector = &mockVectorStore{
		SearchFunc: func(ctx context.Context, query []float32, limit int, threshold float32, filters map[string]interface{}) ([]types.MemoryResult, error) {
			// Simulate some work
			time.Sleep(10 * time.Millisecond)
			return []types.MemoryResult{
				{MemoryID: "mem-1", Score: 0.9, Text: "Result"},
			}, nil
		},
	}
	svc.graph = &mockGraphStore{
		GetMemoriesByIDsFunc: func(ids []string) ([]*types.Memory, error) {
			return []*types.Memory{{ID: "mem-1", Content: "Result", CreatedAt: time.Now()}}, nil
		},
	}

	req := &types.SearchRequest{
		Query: "Tell me about Alice", // Triggers expansion in Prospector
		Limit: 5,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := svc.SearchMemories(context.Background(), req)
		if err != nil {
			b.Fatal(err)
		}
	}
}
