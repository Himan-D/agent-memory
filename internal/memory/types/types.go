package types

import (
	"encoding/json"
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

type Fact struct {
	Fact        string
	Type        string
	Confidence  float64
	Verified    bool
	SourceMemID string
	Claim       string
	Subject     string
	Predicate   string
	Object      string
}

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
	FeedbackNeutral      FeedbackType = "neutral"
	FeedbackCorrection   FeedbackType = "correction"
	FeedbackVeryNegative FeedbackType = "very_negative"
)

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
	ID         string                 `json:"id"`
	TenantID   string                 `json:"tenant_id,omitempty"`
	FromID     string                 `json:"from_id"`
	ToID       string                 `json:"to_id"`
	Type       string                 `json:"type"`
	Properties map[string]interface{} `json:"properties"`
	Weight     float64                `json:"weight"`
	CreatedAt  time.Time              `json:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at"`
}

type MemoryStatus string

const (
	MemoryStatusActive   MemoryStatus = "active"
	MemoryStatusArchived MemoryStatus = "archived"
	MemoryStatusDeleted  MemoryStatus = "deleted"
	MemoryStatusPending  MemoryStatus = "pending"
)

// Validity statuses for conflict resolution.
const (
	ValidityCurrent           = "current"
	ValiditySuperseded        = "superseded"
	ValidityHistoricallyValid = "historically_valid"
	ValidityUnknown           = "unknown"
)

// Pool types for dual-pool memory management.
const (
	PoolExploitation = "exploitation"
	PoolExploration  = "exploration"
)

type Path struct {
	Nodes []Entity   `json:"nodes"`
	Edges []Relation `json:"edges"`
}

type MemoryResult struct {
	Entity          Entity          `json:"entity"`
	Score           float32         `json:"score"`
	Text            string          `json:"text"`
	Source          string          `json:"source"`
	MemoryID        string          `json:"memory_id,omitempty"`
	Metadata        *Memory         `json:"metadata,omitempty"`
	TemporalContext TemporalContext `json:"temporal_context,omitempty"`
	DecayMultiplier float64         `json:"decay_multiplier,omitempty"`
	SessionID       string          `json:"session_id,omitempty"`
}

type TemporalContext string

const (
	TemporalCurrent    TemporalContext = "current"
	TemporalHistorical TemporalContext = "historical"
	TemporalUpcoming   TemporalContext = "upcoming"
)

type Memory struct {
	ID                 string                 `json:"id"`
	ContentHash        string                 `json:"content_hash,omitempty"`
	TenantID           string                 `json:"tenant_id,omitempty"`
	UserID             string                 `json:"user_id,omitempty"`
	OrgID              string                 `json:"org_id,omitempty"`
	AgentID            string                 `json:"agent_id,omitempty"`
	SessionID          string                 `json:"session_id,omitempty"`
	Type               MemoryType             `json:"type"`
	Content            string                 `json:"content"`
	Compressed         string                 `json:"compressed,omitempty"`
	MemoryType         string                 `json:"memory_type,omitempty"`
	Category           string                 `json:"category,omitempty"`
	Tags               []string               `json:"tags,omitempty"`
	Importance         ImportanceLevel        `json:"importance"`
	EntityID           string                 `json:"entity_id,omitempty"`
	Metadata           map[string]interface{} `json:"metadata,omitempty"`
	Status             MemoryStatus           `json:"status"`
	Immutable          bool                   `json:"immutable"`
	ExpirationDate     *time.Time             `json:"expiration_date,omitempty"`
	FeedbackScore      FeedbackType           `json:"feedback_score,omitempty"`
	ParentMemoryID     string                 `json:"parent_memory_id,omitempty"`
	RelatedMemoryIDs   []string               `json:"related_memory_ids,omitempty"`
	Version            int                    `json:"version"`
	CompressionRatio   float64                `json:"compression_ratio,omitempty"`
	AccessCount        int64                  `json:"access_count"`
	CreatedAt          time.Time              `json:"created_at"`
	UpdatedAt          time.Time              `json:"updated_at"`
	LastAccessed       *time.Time             `json:"last_accessed,omitempty"`
	CustomInstructions string                 `json:"custom_instructions,omitempty"`
	StateKey           string                 `json:"state_key,omitempty"`
	DecayMultiplier    float64                `json:"decay_multiplier,omitempty"`
	IsLatest           *bool                  `json:"is_latest,omitempty"`
	SupersedesIDs      []string               `json:"supersedes_ids,omitempty"`
	PreviousVersionID  string                 `json:"previous_version_id,omitempty"`
	Tier               string                 `json:"tier,omitempty"`

	// Memory Worth (MW) scoring — outcome-linked importance
	SuccessCount int64   `json:"success_count"`
	FailureCount int64   `json:"failure_count"`
	WorthScore   float64 `json:"worth_score"`

	// Temporal phase rotation
	VolatilityScore float64 `json:"volatility_score"`
	PhaseAngle      float64 `json:"phase_angle"`

	// Conflict validity
	ValidityStatus string `json:"validity_status"` // current, superseded, historically_valid, unknown

	// Provenance tracking
	ProvenanceEdges []string `json:"provenance_edges,omitempty"` // memory IDs this was derived from
	QValue          float64  `json:"q_value"`                    // TD(lambda) credit assignment score

	// Dual pool
	PoolType string `json:"pool_type"` // exploitation, exploration

	// Retrieval stats for UCB
	RetrievalCount  int64      `json:"retrieval_count"`
	LastRetrievedAt *time.Time `json:"last_retrieved_at,omitempty"`

	// Source monitoring (MEMTIER paper — +33pp on LongMemEval-S)
	SourceType      string  `json:"source_type"`      // observed, told, inferred, external
	SourceAuthority float64 `json:"source_authority"` // 0.0-1.0, derived from source type

	// Dimensional fields (DimMem paper — 24% token reduction)
	Dimensions *MemoryDimensions `json:"dimensions,omitempty"`

	// Polarity (PolarMem paper — negation memory)
	Polarity string `json:"polarity"` // positive, negative, unknown

	// Prospective memory (reminders)
	RemindAt        *time.Time `json:"remind_at,omitempty"`
	RemindCondition string     `json:"remind_condition,omitempty"`

	// Three-granularity storage (TriMem paper)
	RawSegment         string `json:"raw_segment,omitempty"`
	SynthesizedProfile string `json:"synthesized_profile,omitempty"`

	// Graph type (GAM paper — event/topic dual graph)
	GraphLayer string `json:"graph_layer"` // event, topic
}

// MemoryDimensions holds dimensional fields for structured memory storage.
// Based on DimMem paper — 24% token reduction.
type MemoryDimensions struct {
	TimeRef  string   `json:"time_ref,omitempty"` // temporal reference ("yesterday", "2024-01-15")
	Location string   `json:"location,omitempty"` // spatial reference
	Reason   string   `json:"reason,omitempty"`   // why this was stored
	Purpose  string   `json:"purpose,omitempty"`  // intended use
	Keywords []string `json:"keywords,omitempty"` // extracted key terms
}

// Source types (MEMTIER paper).
const (
	SourceObserved = "observed" // agent directly observed/experienced
	SourceTold     = "told"     // user explicitly stated
	SourceInferred = "inferred" // derived from other memories
	SourceExternal = "external" // from external API/document
)

// SourceAuthorityScores maps source types to their default authority scores.
var SourceAuthorityScores = map[string]float64{
	SourceObserved: 1.0,
	SourceTold:     0.85,
	SourceInferred: 0.6,
	SourceExternal: 0.4,
}

// Polarity constants (PolarMem paper).
const (
	PolarityPositive = "positive"
	PolarityNegative = "negative"
	PolarityUnknown  = "unknown"
)

// Graph layer constants (GAM paper).
const (
	GraphLayerEvent = "event"
	GraphLayerTopic = "topic"
)

// Concept represents a higher-level concept node in the knowledge graph.
type Concept struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	TenantID    string                 `json:"tenant_id,omitempty"`
	Properties  map[string]interface{} `json:"properties,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
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
	Query            string         `json:"query"`
	Limit            int            `json:"limit"`
	Offset           int            `json:"offset"`
	Threshold        float32        `json:"threshold"`
	Filters          *SearchFilters `json:"filters,omitempty"`
	MemoryType       MemoryType     `json:"memory_type,omitempty"`
	TenantID         string         `json:"tenant_id,omitempty"` // server-injected; do not trust clients
	UserID           string         `json:"user_id,omitempty"`
	OrgID            string         `json:"org_id,omitempty"`
	AgentID          string         `json:"agent_id,omitempty"`
	Category         string         `json:"category,omitempty"`
	Rerank           bool           `json:"rerank"`
	RerankTopK       int            `json:"rerank_top_k"`
	Mode             string         `json:"mode,omitempty"`
	TimeReference    *time.Time     `json:"time_reference,omitempty"`
	SessionDiversity int            `json:"session_diversity,omitempty"`
	PrivacyFilter    bool           `json:"privacy_filter,omitempty"`
	FusionMode       string         `json:"fusion_mode,omitempty"`
	// MaxTokens caps the total token budget of returned results (~4 chars/token).
	// Zero means no budget enforcement (default 7000 applied in service).
	MaxTokens int `json:"max_tokens,omitempty"`
}

type MemoryUpdate struct {
	ID       string                 `json:"id"`
	Content  string                 `json:"content,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	Tags     []string               `json:"tags,omitempty"`
}

type BatchUpdateRequest struct {
	IDs      []string               `json:"ids"`
	Action   string                 `json:"action"`
	Content  string                 `json:"content,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	Updates  []MemoryUpdate         `json:"updates,omitempty"`
}

type BatchDeleteRequest struct {
	IDs      []string `json:"ids"`
	UserID   string   `json:"user_id,omitempty"`
	OrgID    string   `json:"org_id,omitempty"`
	AgentID  string   `json:"agent_id,omitempty"`
	Category string   `json:"category,omitempty"`
}

type Project struct {
	ID                 string                 `json:"id"`
	Name               string                 `json:"name"`
	Description        string                 `json:"description,omitempty"`
	TenantID           string                 `json:"tenant_id,omitempty"`
	UserID             string                 `json:"user_id,omitempty"`
	OrgID              string                 `json:"org_id,omitempty"`
	CustomInstructions string                 `json:"custom_instructions,omitempty"`
	Settings           ProjectSettings        `json:"settings"`
	Metadata           map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt          time.Time              `json:"created_at"`
	UpdatedAt          time.Time              `json:"updated_at"`
}

type ProjectSettings struct {
	MemoryTypes        []MemoryType `json:"memory_types,omitempty"`
	RerankingEnabled   bool         `json:"reranking_enabled"`
	ConflictResolution bool         `json:"conflict_resolution"`
	Categories         []string     `json:"categories,omitempty"`
	MaxMemoriesPerUser int          `json:"max_memories_per_user,omitempty"`
}

type APIKey struct {
	ID        string    `json:"id"`
	Key       string    `json:"key"`
	ProjectID string    `json:"project_id"`
	CreatedAt time.Time `json:"created_at"`
}

type Session struct {
	ID        string                 `json:"id"`
	AgentID   string                 `json:"agent_id"`
	Metadata  map[string]interface{} `json:"metadata"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

type Message struct {
	ID        string    `json:"id"`
	SessionID string    `json:"session_id"`
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

type EntityMemory struct {
	Entity   Entity   `json:"entity"`
	Memories []Memory `json:"memories"`
}

type GraphStoreConfig struct {
	URI      string
	User     string
	Password string
}

type VectorStoreConfig struct {
	URL    string
	APIKey string
}

type OpenAIConfig struct {
	APIKey string
	Model  string
}

type AppConfig struct {
	Port          int
	LogLevel      string
	MessageBuffer int
	BufferTimeout time.Duration
	RedisURL      string
}

type AuthConfig struct {
	Enabled bool
	Secret  string
}

type LLMConfig struct {
	Provider string
	APIKey   string
	BaseURL  string
}

type MemoryConfig struct {
	ProcessingEnabled   bool
	AutoExtractFacts    bool
	AutoExtractEntities bool
	DefaultImportance   ImportanceLevel
	MultiSignalEnabled  bool
}

type CompactionConfig struct {
	Enabled  bool
	Interval time.Duration
}

type CompressionConfig struct {
	Enabled             bool
	WorkerCount         int
	FastModel           string
	VerifyModel         string
	ComplexityThreshold float64
	TierPolicy          string
}

type RerankerConfig struct {
	Provider string
	APIKey   string
	Model    string
}

type Pagination struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"pageSize"`
	TotalItems int64 `json:"totalItems"`
	TotalPages int   `json:"totalPages"`
}

type MemoryListResponse struct {
	Memories   []Memory   `json:"memories"`
	Pagination Pagination `json:"pagination"`
}

type SearchResponse struct {
	Results []MemoryResult `json:"results"`
}

type MemorySummary struct {
	UserID       string   `json:"user_id"`
	Summary      string   `json:"summary"`
	KeyPoints    []string `json:"key_points"`
	MemoryCount  int      `json:"memory_count"`
	TokenSavings float64  `json:"token_savings"`
}

type HealthStatus struct {
	Status    string            `json:"status"`
	Version   string            `json:"version"`
	Uptime    string            `json:"uptime"`
	Services  map[string]string `json:"services"`
	Timestamp time.Time         `json:"timestamp"`
}

type PaginationInfo struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	TotalItems int64 `json:"total_items"`
	TotalPages int   `json:"total_pages"`
	HasMore    bool  `json:"has_more"`
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
	TenantID      string          `json:"tenant_id,omitempty"` // server-injected; do not trust clients
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

type SkillChain struct {
	ID           string                 `json:"id"`
	TenantID     string                 `json:"tenant_id,omitempty"`
	Name         string                 `json:"name"`
	Description  string                 `json:"description,omitempty"`
	Trigger      string                 `json:"trigger"`
	Steps        []ChainStep            `json:"steps"`
	Conditions   []ChainCondition       `json:"conditions,omitempty"`
	Status       ChainStatus            `json:"status"`
	Version      int                    `json:"version"`
	Tags         []string               `json:"tags,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	UsageCount   int64                  `json:"usage_count"`
	SuccessCount int64                  `json:"success_count"`
	AvgDuration  int64                  `json:"avg_duration"`
	LastUsed     *time.Time             `json:"last_used,omitempty"`
	Confidence   float32                `json:"confidence"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

type ChainStep struct {
	SkillID    string `json:"skill_id"`
	Order      int    `json:"order"`
	ContinueIf string `json:"continue_if,omitempty"`
}

type ChainCondition struct {
	Type     string `json:"type"`
	Property string `json:"property"`
	Operator string `json:"operator"`
	Value    string `json:"value"`
}

type ChainStatus string

const (
	ChainStatusActive    ChainStatus = "active"
	ChainStatusInactive  ChainStatus = "inactive"
	ChainStatusDraft     ChainStatus = "draft"
	ChainStatusCompleted ChainStatus = "completed"
	ChainStatusFailed    ChainStatus = "failed"
)

type ChainExecution struct {
	ID          string                 `json:"id"`
	ChainID     string                 `json:"chain_id"`
	Status      ChainStatus            `json:"status"`
	Results     []ChainStepResult      `json:"results"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	StartedAt   time.Time              `json:"started_at"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	Error       string                 `json:"error,omitempty"`
}

type ChainStepResult struct {
	StepID    string      `json:"step_id"`
	SkillID   string      `json:"skill_id"`
	Output    interface{} `json:"output"`
	Status    string      `json:"status"`
	Duration  float64     `json:"duration"`
	Timestamp time.Time   `json:"timestamp"`
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

type ContradictionResult struct {
	OldMemoryID  string  `json:"old_memory_id"`
	NewMemoryID  string  `json:"new_memory_id"`
	OldContent   string  `json:"old_content"`
	NewContent   string  `json:"new_content"`
	Similarity   float64 `json:"similarity"`
	IsContradict bool    `json:"is_contradict"`
	Action       string  `json:"action"`
}

type HookEvent string

const (
	HookPreCreate  HookEvent = "pre_create"
	HookPostCreate HookEvent = "post_create"
	HookPreUpdate  HookEvent = "pre_update"
	HookPostUpdate HookEvent = "post_update"
	HookPreDelete  HookEvent = "pre_delete"
	HookPostDelete HookEvent = "post_delete"
	HookPreSearch  HookEvent = "pre_search"
	HookPostSearch HookEvent = "post_search"
)

type Hook struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Trigger   string                 `json:"trigger"`
	Action    string                 `json:"action"`
	Config    map[string]interface{} `json:"config"`
	Enabled   bool                   `json:"enabled"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// Advanced search types
type AdvancedSearchRequest struct {
	Query         string                 `json:"query"`
	Filters       *SearchFilters         `json:"filters,omitempty"`
	SemanticBoost float32                `json:"semantic_boost"`
	KeywordBoost  float32                `json:"keyword_boost"`
	SearchType    string                 `json:"search_type"`
	Limit         int                    `json:"limit"`
	Offset        int                    `json:"offset"`
	SortBy        string                 `json:"sort_by"`
	SortOrder     string                 `json:"sort_order"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

type CreateOptions struct {
	AutoExtract     bool                   `json:"auto_extract"`
	AutoIndex       bool                   `json:"auto_index"`
	Priority        string                 `json:"priority"`
	Tags            []string               `json:"tags"`
	Metadata        map[string]interface{} `json:"metadata"`
	Expiration      *time.Time             `json:"expiration"`
	Compression     bool                   `json:"compression"`
	RetentionPolicy string                 `json:"retention_policy"`
}

type HookPayload struct {
	Event     HookEvent              `json:"event"`
	MemoryID  string                 `json:"memory_id,omitempty"`
	Content   string                 `json:"content,omitempty"`
	UserID    string                 `json:"user_id,omitempty"`
	SessionID string                 `json:"session_id,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

type CompactionResult struct {
	DeletedCount    int      `json:"deleted_count"`
	SummarizedCount int      `json:"summarized_count"`
	ArchivedCount   int      `json:"archived_count"`
	MemoryIDs       []string `json:"memory_ids,omitempty"`
}

type SkillListRequest struct {
	Domain string `json:"domain,omitempty"`
	Limit  int    `json:"limit"`
	Offset int    `json:"offset"`
}

type SkillSuggestionRequest struct {
	Trigger string `json:"trigger"`
	Context string `json:"context,omitempty"`
	Limit   int    `json:"limit"`
}

type Review struct {
	ID         string                 `json:"id"`
	SkillID    string                 `json:"skill_id"`
	ReviewerID string                 `json:"reviewer_id"`
	Status     string                 `json:"status"`
	Notes      string                 `json:"notes,omitempty"`
	Rating     int                    `json:"rating,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt  time.Time              `json:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at"`
}

type ImportRequest struct {
	Data      json.RawMessage `json:"data"`
	Format    string          `json:"format"`
	Overwrite bool            `json:"overwrite"`
}

type WebhookEvent string

const (
	WebhookEventMemoryCreated  WebhookEvent = "memory.created"
	WebhookEventMemoryUpdated  WebhookEvent = "memory.updated"
	WebhookEventMemoryDeleted  WebhookEvent = "memory.deleted"
	WebhookEventMemoryArchived WebhookEvent = "memory.archived"

	WebhookEventEntityCreated     WebhookEvent = "entity.created"
	WebhookEventEntityUpdated     WebhookEvent = "entity.updated"
	WebhookEventEntityDeleted     WebhookEvent = "entity.deleted"
	WebhookEventSessionCreated    WebhookEvent = "session.created"
	WebhookEventSessionEnded      WebhookEvent = "session.ended"
	WebhookEventSkillExecuted     WebhookEvent = "skill.executed"
	WebhookEventAlertTriggered    WebhookEvent = "alert.triggered"
	WebhookEventSearchPerformed   WebhookEvent = "search.performed"
	WebhookEventAgentConnected    WebhookEvent = "agent.connected"
	WebhookEventAgentDisconnected WebhookEvent = "agent.disconnected"
)

var AllWebhookEvents = []WebhookEvent{
	WebhookEventMemoryCreated, WebhookEventMemoryUpdated, WebhookEventMemoryDeleted, WebhookEventMemoryArchived,
	WebhookEventEntityCreated, WebhookEventEntityUpdated, WebhookEventEntityDeleted,
	WebhookEventSessionCreated, WebhookEventSessionEnded,
	WebhookEventSkillExecuted, WebhookEventAlertTriggered, WebhookEventSearchPerformed,
	WebhookEventAgentConnected, WebhookEventAgentDisconnected,
}

type Agent struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	OrgID       string                 `json:"org_id"`
	TenantID    string                 `json:"tenant_id,omitempty"`
	Status      AgentStatus            `json:"status"`
	Config      AgentConfig            `json:"config"`
	Groups      []string               `json:"groups,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	LastActive  *time.Time             `json:"last_active,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

type AgentConfig struct {
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

type MemberRole string

const (
	MemberRoleAdmin       MemberRole = "admin"
	MemberRoleMember      MemberRole = "member"
	MemberRoleViewer      MemberRole = "viewer"
	MemberRoleContributor MemberRole = "contributor"
)

type ReviewStatus string

const (
	ReviewStatusPending  ReviewStatus = "pending"
	ReviewStatusApproved ReviewStatus = "approved"
	ReviewStatusRejected ReviewStatus = "rejected"
)

type SkillReview struct {
	ID         string       `json:"id"`
	SkillID    string       `json:"skill_id"`
	TenantID   string       `json:"tenant_id"`
	Status     ReviewStatus `json:"status"`
	ReviewedBy string       `json:"reviewed_by,omitempty"`
	Decision   string       `json:"decision,omitempty"`
	Notes      string       `json:"notes,omitempty"`
	ReviewedAt *time.Time   `json:"reviewed_at,omitempty"`
	CreatedAt  time.Time    `json:"created_at"`
}

type ChainQuery struct {
	Trigger string `json:"trigger"`
	Limit   int    `json:"limit,omitempty"`
	Offset  int    `json:"offset,omitempty"`
}

type ChainExecutionRequest struct {
	ChainID string                 `json:"chain_id"`
	Params  map[string]interface{} `json:"params"`
}

type LicenseTier string

const (
	LicenseTierFree       LicenseTier = "free"
	LicenseTierDeveloper  LicenseTier = "developer"
	LicenseTierTeam       LicenseTier = "team"
	LicenseTierPro        LicenseTier = "pro"
	LicenseTierEnterprise LicenseTier = "enterprise"
	LicenseTierOpenSource LicenseTier = "opensource"
)

type Entitlement struct {
	MaxMemories        int      `json:"max_memories"`
	MaxAgents          int      `json:"max_agents"`
	MaxGroups          int      `json:"max_groups"`
	MaxSkills          int      `json:"max_skills"`
	Features           []string `json:"features"`
	SupportLevel       string   `json:"support_level"`
	HumanReviewEnabled bool     `json:"human_review_enabled"`
	AuditLogging       bool     `json:"audit_logging"`
	CustomDomains      bool     `json:"custom_domains"`
}

var DefaultEntitlements = map[LicenseTier]Entitlement{
	LicenseTierFree: {
		MaxMemories:        1000,
		MaxAgents:          3,
		MaxGroups:          1,
		MaxSkills:          5,
		Features:           []string{FeatureProceduralMemory},
		SupportLevel:       "community",
		HumanReviewEnabled: false,
		AuditLogging:       false,
	},
	LicenseTierOpenSource: {
		MaxMemories:        10000,
		MaxAgents:          5, // Positive value required for AGPL multi-agent check
		MaxGroups:          3,
		MaxSkills:          10,
		Features:           []string{FeatureProceduralMemory, FeatureMultiAgent},
		SupportLevel:       "community",
		HumanReviewEnabled: false,
		AuditLogging:       false,
	},
	LicenseTierDeveloper: {
		MaxMemories:        50000,
		MaxAgents:          10,
		MaxGroups:          5,
		MaxSkills:          20,
		Features:           []string{FeatureProceduralMemory, FeatureMultiAgent},
		SupportLevel:       "developer",
		HumanReviewEnabled: false,
		AuditLogging:       false,
	},
	LicenseTierTeam: {
		MaxMemories:        500000,
		MaxAgents:          50,
		MaxGroups:          20,
		MaxSkills:          100,
		Features:           []string{FeatureProceduralMemory, FeatureMultiAgent, FeatureSharedMemoryPool, FeatureHumanReview, FeatureAuditLogging},
		SupportLevel:       "priority",
		HumanReviewEnabled: true,
		AuditLogging:       true,
	},
	LicenseTierPro: {
		MaxMemories:        100000,
		MaxAgents:          20,
		MaxGroups:          10,
		MaxSkills:          50,
		Features:           []string{FeatureProceduralMemory, FeatureMultiAgent, FeatureSharedMemoryPool, FeatureHumanReview},
		SupportLevel:       "priority",
		HumanReviewEnabled: true,
		AuditLogging:       false,
	},
	LicenseTierEnterprise: {
		MaxMemories:        -1, // -1 is unlimited
		MaxAgents:          -1,
		MaxGroups:          -1,
		MaxSkills:          -1,
		Features:           []string{FeatureProceduralMemory, FeatureMultiAgent, FeatureSharedMemoryPool, FeatureHumanReview, FeatureAuditLogging, FeatureIndustryModules, FeatureCustomBranding, FeaturePrioritySupport},
		SupportLevel:       "dedicated",
		HumanReviewEnabled: true,
		AuditLogging:       true,
		CustomDomains:      true,
	},
}

const (
	FeatureProceduralMemory = "procedural_memory"
	FeatureMultiAgent       = "multi_agent"
	FeatureHumanReview      = "human_review"
	FeatureAuditLogging     = "audit_logging"
	FeatureIndustryModules  = "industry_modules"
	FeatureSharedMemoryPool = "shared_memory_pool"
	FeatureCustomBranding   = "custom_branding"
	FeaturePrioritySupport  = "priority_support"
)

type License struct {
	ID          string      `json:"id"`
	Key         string      `json:"key"`
	TenantID    string      `json:"tenant_id"`
	Tier        LicenseTier `json:"tier"`
	ExpiresAt   time.Time   `json:"expires_at"`
	ActivatedAt time.Time   `json:"activated_at"`
}

type MemoryPoolEvent struct {
	Type      string                 `json:"type"`
	GroupID   string                 `json:"group_id"`
	AgentID   string                 `json:"agent_id"`
	MemoryID  string                 `json:"memory_id,omitempty"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

type AgentMember struct {
	AgentID  string     `json:"agent_id"`
	GroupID  string     `json:"group_id"`
	Role     MemberRole `json:"role"`
	JoinedAt time.Time  `json:"joined_at"`
}

type GroupPolicy struct {
	AllowCrossAgentMemory bool `json:"allow_cross_agent_memory"`
	SkillSharingEnabled   bool `json:"skill_sharing_enabled"`
	RequireHumanReview    bool `json:"require_human_review"`
}

type AgentGroup struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Description  string                 `json:"description,omitempty"`
	OrgID        string                 `json:"org_id"`
	TenantID     string                 `json:"tenant_id,omitempty"`
	Domain       string                 `json:"domain,omitempty"`
	Policy       GroupPolicy            `json:"policy"`
	MemoryPoolID string                 `json:"memory_pool_id,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	Members      []AgentMember          `json:"members,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

type Webhook struct {
	ID             string                 `json:"id"`
	ProjectID      string                 `json:"project_id"`
	TenantID       string                 `json:"tenant_id,omitempty"`
	URL            string                 `json:"url"`
	Events         []WebhookEvent         `json:"events"`
	Fields         []string               `json:"fields,omitempty"`
	Active         bool                   `json:"active"`
	Secret         string                 `json:"secret,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
	VerifiedAt     *time.Time             `json:"verified_at,omitempty"`
	SuccessCount   int64      `json:"success_count"`
	FailureCount   int64      `json:"failure_count"`
	LastDeliveryAt *time.Time `json:"last_delivery_at,omitempty"`
	// LastTriggered mirrors LastDeliveryAt for older dashboard clients.
	LastTriggered  *time.Time `json:"last_triggered,omitempty"`
	LastStatusCode int        `json:"last_status_code,omitempty"`
}

type WebhookPayload struct {
	Event     WebhookEvent `json:"event"`
	MemoryID  string       `json:"memory_id,omitempty"`
	Data      interface{}  `json:"data,omitempty"`
	Timestamp time.Time    `json:"timestamp"`
}

type Conflict struct {
	MemoryA    string  `json:"memory_a"`
	MemoryB    string  `json:"memory_b"`
	Type       string  `json:"type"`
	Confidence float64 `json:"confidence"`
}

type PaginatedResponse struct {
	Items      interface{} `json:"items"`
	Page       int         `json:"page"`
	PageSize   int         `json:"pageSize"`
	TotalItems int64       `json:"totalItems"`
	TotalPages int         `json:"totalPages"`
	HasMore    bool        `json:"hasMore"`
}

type PaginationParams struct {
	Page     int `json:"page"`
	PageSize int `json:"pageSize"`
}

type MemoryProcessingResult struct {
	ProcessedContent string `json:"processed_content"`
	Importance       string `json:"importance"`
	ShouldStore      bool   `json:"should_store"`
}

type SkillExtractionResult struct {
	Skills []*Skill `json:"skills"`
}

// BatchEmbeddingItem represents a single item for batch vector store upsert.
type BatchEmbeddingItem struct {
	ID        string
	Text      string
	Embedding []float32
	Metadata  map[string]interface{}
}
