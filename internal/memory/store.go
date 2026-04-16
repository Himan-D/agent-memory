package memory

import (
	"context"
	"time"

	"agent-memory/internal/memory/types"
)

type GraphStore interface {
	Close() error
	Ping(ctx context.Context) error

	CreateMemory(mem *types.Memory) error
	BatchCreateMemories(memories []*types.Memory) error
	GetMemory(id string) (*types.Memory, error)
	UpdateMemory(mem *types.Memory) error
	DeleteMemory(id string) error
	BatchDeleteMemories(ids []string) error
	UpdateMemoryAccess(id string, ts time.Time) error
	UpdateMemoryFeedbackScore(id string, fbType types.FeedbackType) error
	GetMemoriesByUser(userID string) ([]*types.Memory, error)
	GetMemoriesByOrg(orgID string) ([]*types.Memory, error)
	GetAllMemories() ([]*types.Memory, error)
	GetExpiredMemories() ([]*types.Memory, error)
	RecordHistory(memID, action, oldContent, newContent, userID, comment string) error
	GetMemoryHistory(memID string) ([]types.MemoryHistory, error)

	AdvancedSearch(filters *types.SearchFilters) ([]*types.Memory, error)
	BulkDeleteByFilter(userID, orgID, category string) (int, error)

	CreateSession(agentID string, metadata map[string]interface{}) (*types.Session, error)
	ListSessions() ([]*types.Session, error)
	GetMessages(sessionID string, limit int) ([]types.Message, error)
	ClearMessages(sessionID string) error

	AddEntity(entity types.Entity) error
	GetEntity(id string) (*types.Entity, error)
	ListEntities(tenantID string, limit int) ([]types.Entity, error)
	AddRelation(fromID, toID, relType string, props map[string]interface{}) error
	QueryGraph(cypher string, params map[string]interface{}) ([]map[string]interface{}, error)
	Traverse(fromEntityID string, depth int) ([]types.Path, error)
	GetEntityRelations(entityID string, relType string) ([]types.Relation, error)
	LinkMemoryEntity(memoryID, entityID string) error
	GetMemoryIDsByEntity(entityID string) ([]string, error)
	GetMemoriesByIDs(ids []string) ([]*types.Memory, error)
	BatchUpdateSyncTime(entityIDs []string) error

	CreateFeedback(feedback *types.Feedback) error
	GetFeedbackByType(fbType types.FeedbackType, limit int) ([]*types.Feedback, error)

	CreateMemoryLink(link *types.MemoryLink) error
	GetMemoryLinks(memoryID string) ([]types.MemoryLink, error)
	DeleteMemoryLink(linkID string) error

	CreateMemoryVersion(version *types.MemoryVersion) error
	GetMemoryVersions(memoryID string) ([]types.MemoryVersion, error)

	CreateSkill(ctx context.Context, skill *types.Skill) error
	ListSkills(ctx context.Context, tenantID, domain string, limit, offset int) ([]*types.Skill, error)
	GetSkill(ctx context.Context, skillID string) (*types.Skill, error)
	UpdateSkill(ctx context.Context, skill *types.Skill) error
	DeleteSkill(ctx context.Context, skillID string) error
	GetSkillsByTrigger(ctx context.Context, trigger string, limit int) ([]*types.Skill, error)
	GetSkillsByDomain(ctx context.Context, domain string, limit int) ([]*types.Skill, error)
	IncrementSkillUsage(ctx context.Context, skillID string) error
	CreateSkillReview(ctx context.Context, review *types.SkillReview) error

	CreateAgent(ctx context.Context, agent *types.Agent) error
	GetAgent(ctx context.Context, agentID string) (*types.Agent, error)
	UpdateAgent(ctx context.Context, agent *types.Agent) error
	DeleteAgent(ctx context.Context, agentID string) error
	ListAgents(ctx context.Context, tenantID string, limit, offset int) ([]*types.Agent, int64, error)

	CreateAgentGroup(ctx context.Context, group *types.AgentGroup) error
	GetAgentGroup(ctx context.Context, groupID string) (*types.AgentGroup, error)
	UpdateAgentGroup(ctx context.Context, group *types.AgentGroup) error
	DeleteAgentGroup(ctx context.Context, groupID string) error
	ListAgentGroups(ctx context.Context, tenantID string, limit, offset int) ([]*types.AgentGroup, int64, error)
	AddAgentToGroup(ctx context.Context, agentID, groupID string, role types.MemberRole) error
	RemoveAgentFromGroup(ctx context.Context, agentID, groupID string) error
	GetGroupSkills(ctx context.Context, groupID string, limit int) ([]*types.Skill, error)
	GetGroupMemories(ctx context.Context, groupID string) ([]*types.Memory, error)
	ShareMemoryToGroup(ctx context.Context, memoryID, groupID, role string) error

	ListPendingReviews(ctx context.Context, tenantID string) ([]*types.SkillReview, error)
	GetReview(ctx context.Context, reviewID string) (*types.SkillReview, error)
	ProcessReview(ctx context.Context, reviewID string, approved bool, notes string) error

	CreateChain(ctx context.Context, chain *types.SkillChain) error
	GetChain(ctx context.Context, chainID string) (*types.SkillChain, error)
	ListChains(ctx context.Context, tenantID string, query *types.ChainQuery) ([]*types.SkillChain, error)
	UpdateChain(ctx context.Context, chain *types.SkillChain) error
	DeleteChain(ctx context.Context, chainID string) error
	GetChainExecutions(ctx context.Context, chainID string, limit int) ([]*types.ChainExecution, error)
	UpdateChainExecution(ctx context.Context, exec *types.ChainExecution) error
	IncrementChainUsage(ctx context.Context, chainID string) error
}

type VectorStore interface {
	StoreEmbedding(ctx context.Context, text string, id string, embedding []float32, metadata map[string]interface{}) (string, error)
	Search(ctx context.Context, query []float32, limit int, threshold float32, filters map[string]interface{}) ([]types.MemoryResult, error)
	UpdateMemory(ctx context.Context, id string, text string, metadata map[string]interface{}) error
	DeleteMemory(ctx context.Context, id string) error
	UpdateVector(ctx context.Context, id string, embedding []float32) error
	Ping(ctx context.Context) error
	Close() error
}
