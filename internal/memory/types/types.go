package types

import (
	"time"
)

type MemoryType string

const (
	MemoryTypeConversation MemoryType = "conversation"
	MemoryTypeSession      MemoryType = "session"
	MemoryTypeUser         MemoryType = "user"
	MemoryTypeOrg          MemoryType = "org"
)

type ImportanceLevel string

const (
	ImportanceCritical ImportanceLevel = "critical"
	ImportanceHigh     ImportanceLevel = "high"
	ImportanceMedium   ImportanceLevel = "medium"
	ImportanceLow      ImportanceLevel = "low"
)

type MemoryLinkType string

const (
	MemoryLinkParent  MemoryLinkType = "parent"
	MemoryLinkRelated MemoryLinkType = "related"
	MemoryLinkReply   MemoryLinkType = "reply"
	MemoryLinkCite    MemoryLinkType = "cite"
)

type FeedbackType string

const (
	FeedbackPositive     FeedbackType = "positive"
	FeedbackNegative     FeedbackType = "negative"
	FeedbackVeryNegative FeedbackType = "very_negative"
)

type MemoryStatus string

const (
	MemoryStatusActive   MemoryStatus = "active"
	MemoryStatusArchived MemoryStatus = "archived"
	MemoryStatusDeleted  MemoryStatus = "deleted"
)

type Message struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id,omitempty"`
	SessionID string    `json:"session_id"`
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

type Entity struct {
	ID         string                 `json:"id"`
	TenantID   string                 `json:"tenant_id,omitempty"`
	Type       string                 `json:"type"`
	Name       string                 `json:"name"`
	Properties map[string]interface{} `json:"properties"`
	Embedding  []float32              `json:"embedding,omitempty"`
	CreatedAt  time.Time              `json:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at"`
	LastSynced *time.Time             `json:"last_synced,omitempty"`
}

type Relation struct {
	ID       string                 `json:"id"`
	TenantID string                 `json:"tenant_id,omitempty"`
	FromID   string                 `json:"from_id"`
	ToID     string                 `json:"to_id"`
	Type     string                 `json:"type"`
	Weight   float64                `json:"weight"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type Session struct {
	ID        string                 `json:"id"`
	TenantID  string                 `json:"tenant_id,omitempty"`
	AgentID   string                 `json:"agent_id"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

type Path struct {
	Nodes []Entity   `json:"nodes"`
	Edges []Relation `json:"edges"`
}

type MemoryResult struct {
	Entity   Entity  `json:"entity"`
	Score    float32 `json:"score"`
	Text     string  `json:"text"`
	Source   string  `json:"source"`
	MemoryID string  `json:"memory_id,omitempty"`
	Metadata *Memory `json:"metadata,omitempty"`
}

type Memory struct {
	ID               string                 `json:"id"`
	TenantID         string                 `json:"tenant_id,omitempty"`
	UserID           string                 `json:"user_id,omitempty"`
	OrgID            string                 `json:"org_id,omitempty"`
	AgentID          string                 `json:"agent_id,omitempty"`
	SessionID        string                 `json:"session_id,omitempty"`
	Type             MemoryType             `json:"type"`
	Content          string                 `json:"content"`
	MemoryType       string                 `json:"memory_type,omitempty"`
	Category         string                 `json:"category,omitempty"`
	Tags             []string               `json:"tags,omitempty"`
	Importance       ImportanceLevel        `json:"importance"`
	EntityID         string                 `json:"entity_id,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
	Status           MemoryStatus           `json:"status"`
	Immutable        bool                   `json:"immutable"`
	ExpirationDate   *time.Time             `json:"expiration_date,omitempty"`
	FeedbackScore    FeedbackType           `json:"feedback_score,omitempty"`
	ParentMemoryID   string                 `json:"parent_memory_id,omitempty"`
	RelatedMemoryIDs []string               `json:"related_memory_ids,omitempty"`
	Version          int                    `json:"version"`
	AccessCount      int64                  `json:"access_count"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
	LastAccessed     *time.Time             `json:"last_accessed,omitempty"`
}

type MemoryHistory struct {
	ID        string                 `json:"id"`
	MemoryID  string                 `json:"memory_id"`
	Action    HistoryAction          `json:"action"`
	OldValue  string                 `json:"old_value,omitempty"`
	NewValue  string                 `json:"new_value,omitempty"`
	ChangedBy string                 `json:"changed_by,omitempty"`
	Reason    string                 `json:"reason,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
}

type MemoryLink struct {
	ID       string                 `json:"id"`
	TenantID string                 `json:"tenant_id,omitempty"`
	FromID   string                 `json:"from_id"`
	ToID     string                 `json:"to_id"`
	Type     MemoryLinkType         `json:"type"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	Weight   float64                `json:"weight"`
}

type MemoryVersion struct {
	ID        string                 `json:"id"`
	MemoryID  string                 `json:"memory_id"`
	Version   int                    `json:"version"`
	Content   string                 `json:"content"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	CreatedBy string                 `json:"created_by,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
}

type HistoryAction string

const (
	HistoryActionCreate   HistoryAction = "create"
	HistoryActionUpdate   HistoryAction = "update"
	HistoryActionDelete   HistoryAction = "delete"
	HistoryActionArchive  HistoryAction = "archive"
	HistoryActionFeedback HistoryAction = "feedback"
)

type Feedback struct {
	ID        string       `json:"id"`
	MemoryID  string       `json:"memory_id"`
	Type      FeedbackType `json:"type"`
	Comment   string       `json:"comment,omitempty"`
	SessionID string       `json:"session_id,omitempty"`
	UserID    string       `json:"user_id,omitempty"`
	CreatedAt time.Time    `json:"created_at"`
}

type SearchFilter struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"`
	Value    interface{} `json:"value"`
}

type FilterLogic string

const (
	FilterLogicAnd FilterLogic = "AND"
	FilterLogicOr  FilterLogic = "OR"
	FilterLogicNot FilterLogic = "NOT"
)

type SearchFilters struct {
	Logic  FilterLogic     `json:"logic"`
	Rules  []SearchFilter  `json:"rules"`
	Nested []SearchFilters `json:"nested,omitempty"`
}

type SearchRequest struct {
	Query      string         `json:"query"`
	Limit      int            `json:"limit"`
	Offset     int            `json:"offset"`
	Threshold  float32        `json:"threshold"`
	Filters    *SearchFilters `json:"filters,omitempty"`
	MemoryType MemoryType     `json:"memory_type,omitempty"`
	UserID     string         `json:"user_id,omitempty"`
	OrgID      string         `json:"org_id,omitempty"`
	AgentID    string         `json:"agent_id,omitempty"`
	Category   string         `json:"category,omitempty"`
	Rerank     bool           `json:"rerank"`
	RerankTopK int            `json:"rerank_top_k"`
}

type BatchUpdateRequest struct {
	IDs      []string               `json:"ids"`
	Action   string                 `json:"action"`
	Content  string                 `json:"content,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type BatchDeleteRequest struct {
	IDs      []string `json:"ids"`
	UserID   string   `json:"user_id,omitempty"`
	OrgID    string   `json:"org_id,omitempty"`
	Category string   `json:"category,omitempty"`
}

type EntityMemory struct {
	EntityID   string  `json:"entity_id"`
	MemoryID   string  `json:"memory_id"`
	Content    string  `json:"content"`
	EntityType string  `json:"entity_type"`
	Score      float32 `json:"score"`
}

type UnifiedMemory interface {
	AddToContext(sessionID string, msg Message) error
	GetContext(sessionID string, limit int) ([]Message, error)
	ClearContext(sessionID string) error
	CreateSession(agentID string, metadata map[string]interface{}) (*Session, error)

	AddEntity(entity Entity) (*Entity, error)
	GetEntity(id string) (*Entity, error)
	AddRelation(fromID, toID, relType string, props map[string]interface{}) error
	QueryGraph(cypher string, params map[string]interface{}) ([]map[string]interface{}, error)
	Traverse(fromEntityID string, depth int) ([]Path, error)
	GetEntityRelations(entityID string, relType string) ([]Relation, error)

	StoreEmbedding(text string, entityID string, metadata map[string]interface{}) (string, error)
	SearchSemantic(query string, limit int, scoreThreshold float32, filters map[string]interface{}) ([]MemoryResult, error)
	UpdateMemory(id string, text string, metadata map[string]interface{}) error
	DeleteMemory(id string) error

	SyncEntityToVector(entityID string) error
	BatchSyncEntities(entityIDs []string) error

	Close() error
}

type Project struct {
	ID                 string                 `json:"id"`
	Name               string                 `json:"name"`
	Description        string                 `json:"description,omitempty"`
	UserID             string                 `json:"user_id,omitempty"`
	OrgID              string                 `json:"org_id,omitempty"`
	CustomInstructions string                 `json:"custom_instructions,omitempty"`
	Settings           ProjectSettings        `json:"settings"`
	Metadata           map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt          time.Time              `json:"created_at"`
	UpdatedAt          time.Time              `json:"updated_at"`
}

type ProjectSettings struct {
	MemoryTypes        []MemoryType   `json:"memory_types,omitempty"`
	Categories         []string       `json:"categories,omitempty"`
	EmbeddingModel     string         `json:"embedding_model,omitempty"`
	RerankingEnabled   bool           `json:"reranking_enabled"`
	ConflictResolution bool           `json:"conflict_resolution"`
	AutoExpiration     *time.Duration `json:"auto_expiration,omitempty"`
	MaxMemoriesPerUser int            `json:"max_memories_per_user,omitempty"`
}

type Webhook struct {
	ID        string                 `json:"id"`
	ProjectID string                 `json:"project_id"`
	URL       string                 `json:"url"`
	Events    []WebhookEvent         `json:"events"`
	Secret    string                 `json:"secret,omitempty"`
	Active    bool                   `json:"active"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
}

type WebhookEvent string

const (
	WebhookEventMemoryCreated    WebhookEvent = "memory.created"
	WebhookEventMemoryUpdated    WebhookEvent = "memory.updated"
	WebhookEventMemoryDeleted    WebhookEvent = "memory.deleted"
	WebhookEventMemoryArchived   WebhookEvent = "memory.archived"
	WebhookEventFeedbackAdded    WebhookEvent = "feedback.added"
	WebhookEventConflictResolved WebhookEvent = "conflict.resolved"
)

type WebhookPayload struct {
	Event     WebhookEvent `json:"event"`
	Timestamp time.Time    `json:"timestamp"`
	Data      interface{}  `json:"data"`
}

type ConflictInfo struct {
	ExistingMemory *Memory `json:"existing_memory"`
	NewContent     string  `json:"new_content"`
	Similarity     float32 `json:"similarity"`
	Resolution     string  `json:"resolution"`
}

type MemoryAnalytics struct {
	TotalMemories        int64            `json:"total_memories"`
	ActiveMemories       int64            `json:"active_memories"`
	ArchivedMemories     int64            `json:"archived_memories"`
	ExpiredMemories      int64            `json:"expired_memories"`
	ByCategory           map[string]int64 `json:"by_category"`
	ByType               map[string]int64 `json:"by_type"`
	ByFeedbackScore      map[string]int64 `json:"by_feedback_score"`
	AvgFeedbackScore     float64          `json:"avg_feedback_score"`
	MemoriesWithFeedback int64            `json:"memories_with_feedback"`
	TotalFeedback        int64            `json:"total_feedback"`
	PositiveFeedback     int64            `json:"positive_feedback"`
	NegativeFeedback     int64            `json:"negative_feedback"`
}

type SearchResultWithFeedback struct {
	Memory        *Memory `json:"memory"`
	Score         float32 `json:"score"`
	FeedbackBoost float32 `json:"feedback_boost"`
	FinalScore    float32 `json:"final_score"`
}

type PaginationParams struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
}

type PaginatedResponse struct {
	Items      interface{} `json:"items"`
	Page       int         `json:"page"`
	PageSize   int         `json:"page_size"`
	TotalItems int64       `json:"total_items"`
	TotalPages int         `json:"total_pages"`
	HasMore    bool        `json:"has_more"`
}

type MemoryExport struct {
	Version    string     `json:"version"`
	ExportedAt time.Time  `json:"exported_at"`
	Memories   []Memory   `json:"memories"`
	Entities   []Entity   `json:"entities,omitempty"`
	Relations  []Relation `json:"relations,omitempty"`
}

type MemoryImport struct {
	Memories  []Memory   `json:"memories"`
	Entities  []Entity   `json:"entities,omitempty"`
	Relations []Relation `json:"relations,omitempty"`
	Overwrite bool       `json:"overwrite"`
	MergeMode string     `json:"merge_mode"`
}

type HybridSearchRequest struct {
	Query         string          `json:"query"`
	SemanticLimit int             `json:"semantic_limit"`
	KeywordLimit  int             `json:"keyword_limit"`
	Threshold     float32         `json:"threshold"`
	Boost         float32         `json:"boost"`
	Rerank        bool            `json:"rerank"`
	RerankLimit   int             `json:"rerank_limit"`
	Filters       *SearchFilters  `json:"filters,omitempty"`
	MemoryType    MemoryType      `json:"memory_type,omitempty"`
	UserID        string          `json:"user_id,omitempty"`
	OrgID         string          `json:"org_id,omitempty"`
	AgentID       string          `json:"agent_id,omitempty"`
	Category      string          `json:"category,omitempty"`
	Tags          []string        `json:"tags,omitempty"`
	Importance    ImportanceLevel `json:"importance,omitempty"`
	DateFrom      *time.Time      `json:"date_from,omitempty"`
	DateTo        *time.Time      `json:"date_to,omitempty"`
}

type MemoryInsight struct {
	Type        string                 `json:"type"`
	Description string                 `json:"description"`
	Memories    []string               `json:"memory_ids"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

type MemoryStats struct {
	TotalMemories   int64            `json:"total_memories"`
	ByCategory      map[string]int64 `json:"by_category"`
	ByType          map[string]int64 `json:"by_type"`
	ByImportance    map[string]int64 `json:"by_importance"`
	ByStatus        map[string]int64 `json:"by_status"`
	AvgAccessCount  float64          `json:"avg_access_count"`
	TopTags         []TagCount       `json:"top_tags"`
	RecentMemories  int64            `json:"recent_memories"`
	ExpiredMemories int64            `json:"expired_memories"`
}

type TagCount struct {
	Tag   string `json:"tag"`
	Count int64  `json:"count"`
}

// ==================== Procedural Memory Types ====================

type Skill struct {
	ID            string                 `json:"id"`
	TenantID      string                 `json:"tenant_id,omitempty"`
	GroupID       string                 `json:"group_id,omitempty"`
	Name          string                 `json:"name"`
	Domain        string                 `json:"domain"`
	Trigger       string                 `json:"trigger"`
	Action        string                 `json:"action"`
	Confidence    float32                `json:"confidence"`
	UsageCount    int64                  `json:"usage_count"`
	SourceMemory  string                 `json:"source_memory,omitempty"`
	CreatedBy     string                 `json:"created_by,omitempty"`
	Verified      bool                   `json:"verified"`
	HumanReviewed bool                   `json:"human_reviewed"`
	Version       int                    `json:"version"`
	Tags          []string               `json:"tags,omitempty"`
	Examples      []string               `json:"examples,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
	LastUsed      *time.Time             `json:"last_used,omitempty"`
}

type ProceduralData struct {
	Trigger        string   `json:"trigger"`
	Steps          []string `json:"steps"`
	Preconditions  []string `json:"preconditions,omitempty"`
	Postconditions []string `json:"postconditions,omitempty"`
	Examples       []string `json:"examples,omitempty"`
	Confidence     float32  `json:"confidence"`
}

type SkillReview struct {
	ID         string       `json:"id"`
	TenantID   string       `json:"tenant_id,omitempty"`
	SkillID    string       `json:"skill_id"`
	Status     ReviewStatus `json:"status"`
	ReviewedBy string       `json:"reviewed_by,omitempty"`
	Notes      string       `json:"notes,omitempty"`
	Decision   string       `json:"decision,omitempty"`
	CreatedAt  time.Time    `json:"created_at"`
	ReviewedAt *time.Time   `json:"reviewed_at,omitempty"`
}

type ReviewStatus string

const (
	ReviewStatusPending  ReviewStatus = "pending"
	ReviewStatusApproved ReviewStatus = "approved"
	ReviewStatusRejected ReviewStatus = "rejected"
)

type SkillSynthesis struct {
	ID             string    `json:"id"`
	TenantID       string    `json:"tenant_id,omitempty"`
	GroupID        string    `json:"group_id,omitempty"`
	SourceSkillIDs []string  `json:"source_skill_ids"`
	ResultSkill    *Skill    `json:"result_skill"`
	Status         string    `json:"status"`
	Reason         string    `json:"reason"`
	CreatedAt      time.Time `json:"created_at"`
}

type SkillQuery struct {
	Domain        string   `json:"domain,omitempty"`
	Trigger       string   `json:"trigger,omitempty"`
	Tags          []string `json:"tags,omitempty"`
	Verified      *bool    `json:"verified,omitempty"`
	MinConfidence float32  `json:"min_confidence,omitempty"`
	Limit         int      `json:"limit"`
	Offset        int      `json:"offset"`
}

// ==================== Multi-Agent Types ====================

type Agent struct {
	ID          string                 `json:"id"`
	TenantID    string                 `json:"tenant_id,omitempty"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Config      AgentConfig            `json:"config,omitempty"`
	Status      AgentStatus            `json:"status"`
	Groups      []string               `json:"groups,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	LastActive  *time.Time             `json:"last_active,omitempty"`
}

type AgentConfig struct {
	MaxMemories   int      `json:"max_memories,omitempty"`
	AutoExtract   bool     `json:"auto_extract"`
	SharingPolicy string   `json:"sharing_policy"`
	SkillDomains  []string `json:"skill_domains,omitempty"`
}

type AgentStatus string

const (
	AgentStatusActive    AgentStatus = "active"
	AgentStatusInactive  AgentStatus = "inactive"
	AgentStatusSuspended AgentStatus = "suspended"
)

type AgentGroup struct {
	ID           string                 `json:"id"`
	TenantID     string                 `json:"tenant_id,omitempty"`
	Name         string                 `json:"name"`
	Description  string                 `json:"description,omitempty"`
	Domain       string                 `json:"domain,omitempty"`
	Members      []AgentMember          `json:"members"`
	Policy       GroupPolicy            `json:"policy"`
	MemoryPoolID string                 `json:"memory_pool_id,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

type AgentMember struct {
	AgentID  string     `json:"agent_id"`
	Role     MemberRole `json:"role"`
	JoinedAt time.Time  `json:"joined_at"`
}

type MemberRole string

const (
	MemberRoleAdmin       MemberRole = "admin"
	MemberRoleContributor MemberRole = "contributor"
	MemberRoleReader      MemberRole = "reader"
)

type GroupPolicy struct {
	AllowCrossAgentMemory bool `json:"allow_cross_agent_memory"`
	RequireHumanReview    bool `json:"require_human_review"`
	AutoSyncEnabled       bool `json:"auto_sync_enabled"`
	SyncIntervalSeconds   int  `json:"sync_interval_seconds"`
	MaxSharedMemories     int  `json:"max_shared_memories"`
	SkillSharingEnabled   bool `json:"skill_sharing_enabled"`
}

type SharedMemory struct {
	ID        string     `json:"id"`
	GroupID   string     `json:"group_id"`
	MemoryID  string     `json:"memory_id"`
	SharedBy  string     `json:"shared_by"`
	SharedAt  time.Time  `json:"shared_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

type MemoryPoolEvent struct {
	Type      string      `json:"type"`
	GroupID   string      `json:"group_id"`
	AgentID   string      `json:"agent_id"`
	MemoryID  string      `json:"memory_id,omitempty"`
	SkillID   string      `json:"skill_id,omitempty"`
	Data      interface{} `json:"data,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

// ==================== Skill Chain Types ====================

type SkillChain struct {
	ID           string                 `json:"id"`
	TenantID     string                 `json:"tenant_id,omitempty"`
	Name         string                 `json:"name"`
	Description  string                 `json:"description,omitempty"`
	Trigger      string                 `json:"trigger"`
	Steps        []ChainStep            `json:"steps"`
	Conditions   []ChainCondition       `json:"conditions,omitempty"`
	Confidence   float32                `json:"confidence"`
	UsageCount   int64                  `json:"usage_count"`
	SuccessCount int64                  `json:"success_count"`
	AvgDuration  int64                  `json:"avg_duration_ms"`
	Tags         []string               `json:"tags,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
	LastUsed     *time.Time             `json:"last_used,omitempty"`
}

type ChainStep struct {
	SkillID    string `json:"skill_id"`
	SkillName  string `json:"skill_name,omitempty"`
	Order      int    `json:"order"`
	ContinueIf string `json:"continue_if,omitempty"`
	TimeoutMs  int    `json:"timeout_ms,omitempty"`
}

type ChainCondition struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"`
	Value    interface{} `json:"value"`
	Action   string      `json:"action"`
}

type ChainExecution struct {
	ID          string                 `json:"id"`
	ChainID     string                 `json:"chain_id"`
	Status      ChainStatus            `json:"status"`
	Results     []ChainStepResult      `json:"results"`
	StartedAt   time.Time              `json:"started_at"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	Error       string                 `json:"error,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

type ChainStepResult struct {
	StepOrder  int    `json:"step_order"`
	SkillID    string `json:"skill_id"`
	Success    bool   `json:"success"`
	Output     string `json:"output,omitempty"`
	Error      string `json:"error,omitempty"`
	DurationMs int64  `json:"duration_ms"`
}

type ChainStatus string

const (
	ChainStatusPending   ChainStatus = "pending"
	ChainStatusRunning   ChainStatus = "running"
	ChainStatusCompleted ChainStatus = "completed"
	ChainStatusFailed    ChainStatus = "failed"
	ChainStatusCancelled ChainStatus = "cancelled"
)

type ChainQuery struct {
	Trigger       string   `json:"trigger,omitempty"`
	Tags          []string `json:"tags,omitempty"`
	MinConfidence float32  `json:"min_confidence,omitempty"`
	Limit         int      `json:"limit"`
	Offset        int      `json:"offset"`
}

type ChainExecutionRequest struct {
	ChainID   string                 `json:"chain_id"`
	Context   map[string]interface{} `json:"context"`
	TimeoutMs int                    `json:"timeout_ms,omitempty"`
}

// ==================== License Types ====================

type LicenseTier string

const (
	LicenseTierOpenSource LicenseTier = "agpl"
	LicenseTierDeveloper  LicenseTier = "developer"
	LicenseTierTeam       LicenseTier = "team"
	LicenseTierEnterprise LicenseTier = "enterprise"
)

type License struct {
	ID        string                 `json:"id"`
	Tier      LicenseTier            `json:"tier"`
	TenantID  string                 `json:"tenant_id"`
	Key       string                 `json:"key,omitempty"`
	ExpiresAt *time.Time             `json:"expires_at,omitempty"`
	Features  []string               `json:"features"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
}

type Entitlement struct {
	Tier               LicenseTier `json:"tier"`
	MaxAgents          int         `json:"max_agents"`
	MaxGroups          int         `json:"max_groups"`
	MaxSkills          int         `json:"max_skills"`
	HumanReviewEnabled bool        `json:"human_review_enabled"`
	AuditLogging       bool        `json:"audit_logging"`
	CustomDomains      bool        `json:"custom_domains"`
	SupportLevel       string      `json:"support_level"`
}

type FeatureFlag struct {
	Name    string      `json:"name"`
	Enabled bool        `json:"enabled"`
	Tier    LicenseTier `json:"tier_required"`
}

const (
	FeatureProceduralMemory = "procedural_memory"
	FeatureMultiAgent       = "multi_agent"
	FeatureSharedMemoryPool = "shared_memory_pool"
	FeatureHumanReview      = "human_review"
	FeatureAuditLogging     = "audit_logging"
	FeatureIndustryModules  = "industry_modules"
	FeatureCustomBranding   = "custom_branding"
	FeaturePrioritySupport  = "priority_support"
)

var DefaultEntitlements = map[LicenseTier]Entitlement{
	LicenseTierOpenSource: {
		Tier:               LicenseTierOpenSource,
		MaxAgents:          3,
		MaxGroups:          1,
		MaxSkills:          100,
		HumanReviewEnabled: false,
		AuditLogging:       false,
		CustomDomains:      false,
		SupportLevel:       "community",
	},
	LicenseTierDeveloper: {
		Tier:               LicenseTierDeveloper,
		MaxAgents:          10,
		MaxGroups:          5,
		MaxSkills:          1000,
		HumanReviewEnabled: false,
		AuditLogging:       false,
		CustomDomains:      false,
		SupportLevel:       "email",
	},
	LicenseTierTeam: {
		Tier:               LicenseTierTeam,
		MaxAgents:          50,
		MaxGroups:          20,
		MaxSkills:          10000,
		HumanReviewEnabled: true,
		AuditLogging:       true,
		CustomDomains:      false,
		SupportLevel:       "priority",
	},
	LicenseTierEnterprise: {
		Tier:               LicenseTierEnterprise,
		MaxAgents:          -1, // unlimited
		MaxGroups:          -1,
		MaxSkills:          -1,
		HumanReviewEnabled: true,
		AuditLogging:       true,
		CustomDomains:      true,
		SupportLevel:       "dedicated",
	},
}

// ==================== Audit Types ====================

type AuditLog struct {
	ID         string                 `json:"id"`
	TenantID   string                 `json:"tenant_id,omitempty"`
	ActorID    string                 `json:"actor_id"`
	ActorType  string                 `json:"actor_type"`
	Action     string                 `json:"action"`
	Resource   string                 `json:"resource"`
	ResourceID string                 `json:"resource_id,omitempty"`
	Details    map[string]interface{} `json:"details,omitempty"`
	IPAddress  string                 `json:"ip_address,omitempty"`
	Timestamp  time.Time              `json:"timestamp"`
}

type AuditEvent string

const (
	AuditEventSkillCreated     AuditEvent = "skill.created"
	AuditEventSkillApproved    AuditEvent = "skill.approved"
	AuditEventSkillRejected    AuditEvent = "skill.rejected"
	AuditEventSkillSynthesized AuditEvent = "skill.synthesized"
	AuditEventAgentJoinedGroup AuditEvent = "agent.joined_group"
	AuditEventAgentLeftGroup   AuditEvent = "agent.left_group"
	AuditEventMemoryShared     AuditEvent = "memory.shared"
	AuditEventLicenseChecked   AuditEvent = "license.checked"
)
