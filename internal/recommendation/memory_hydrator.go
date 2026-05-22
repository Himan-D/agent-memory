package recommendation

import (
	"context"
	"time"
)

// MemoryHydrator enriches candidates with core memory data from the store.
type MemoryHydrator struct {
	store MemoryDetailStore
}

// MemoryDetailStore provides detailed memory retrieval.
type MemoryDetailStore interface {
	GetMemoryWithMetadata(ctx context.Context, id string) (*MemoryDetail, error)
}

// MemoryDetail holds all metadata for a memory.
type MemoryDetail struct {
	ID        string
	Content   string
	Type      string
	Source    string
	AuthorID  string
	CreatedAt time.Time
	UpdatedAt time.Time
	Tags      []string
	Entities  []string
	ProjectID string
	TenantID  string
}

// NewMemoryHydrator creates a hydrator that fetches memory details.
func NewMemoryHydrator(store MemoryDetailStore) *MemoryHydrator {
	return &MemoryHydrator{store: store}
}

func (h *MemoryHydrator) Name() string {
	return "memory_hydrator"
}

func (h *MemoryHydrator) Hydrate(ctx context.Context, query *QueryContext, candidate *MemoryCandidate) error {
	if h.store == nil {
		return nil
	}

	detail, err := h.store.GetMemoryWithMetadata(ctx, candidate.ID)
	if err != nil {
		return err
	}

	candidate.Metadata["content"] = detail.Content
	candidate.Metadata["type"] = detail.Type
	candidate.Metadata["source"] = detail.Source
	candidate.Metadata["author_id"] = detail.AuthorID
	candidate.Metadata["created_at"] = detail.CreatedAt
	candidate.Metadata["updated_at"] = detail.UpdatedAt
	candidate.Metadata["tags"] = detail.Tags
	candidate.Metadata["entities"] = detail.Entities
	candidate.Metadata["project_id"] = detail.ProjectID

	// Compute recency score (0 to 1)
	age := time.Since(detail.CreatedAt)
	maxAge := 30 * 24 * time.Hour
	if age > maxAge {
		candidate.Metadata["recency"] = 0.0
	} else {
		candidate.Metadata["recency"] = 1.0 - (float64(age) / float64(maxAge))
	}

	return nil
}

// AuthorHydrator enriches candidates with author metadata.
type AuthorHydrator struct {
	authorStore AuthorStore
}

// AuthorStore provides author information.
type AuthorStore interface {
	GetAuthorInfo(ctx context.Context, authorID string) (*AuthorInfo, error)
}

// AuthorInfo holds author metadata.
type AuthorInfo struct {
	ID                 string
	Name               string
	Type               string // "agent", "human", "system"
	VerificationStatus string
	FollowerCount      int
}

// NewAuthorHydrator creates an author hydrator.
func NewAuthorHydrator(store AuthorStore) *AuthorHydrator {
	return &AuthorHydrator{authorStore: store}
}

func (h *AuthorHydrator) Name() string {
	return "author_hydrator"
}

func (h *AuthorHydrator) Hydrate(ctx context.Context, query *QueryContext, candidate *MemoryCandidate) error {
	authorID, _ := candidate.Metadata["author_id"].(string)
	if authorID == "" || h.authorStore == nil {
		return nil
	}

	info, err := h.authorStore.GetAuthorInfo(ctx, authorID)
	if err != nil {
		return nil // Non-fatal: author info is optional
	}

	candidate.Metadata["author_name"] = info.Name
	candidate.Metadata["author_type"] = info.Type
	candidate.Metadata["author_verified"] = info.VerificationStatus == "verified"
	candidate.Metadata["author_followers"] = info.FollowerCount

	return nil
}

// EngagementHydrator adds the user's past engagement with this memory.
type EngagementHydrator struct {
	engagementStore EngagementStore
}

// EngagementStore provides engagement history.
type EngagementStore interface {
	GetEngagement(ctx context.Context, userID string, memoryID string) (*EngagementInfo, error)
}

// EngagementInfo holds user engagement data for a memory.
type EngagementInfo struct {
	MemoryID   string
	Actions    []string
	LastAction string
	Timestamp  time.Time
	DwellMs    int64
	Count      int
}

// NewEngagementHydrator creates an engagement hydrator.
func NewEngagementHydrator(store EngagementStore) *EngagementHydrator {
	return &EngagementHydrator{engagementStore: store}
}

func (h *EngagementHydrator) Name() string {
	return "engagement_hydrator"
}

func (h *EngagementHydrator) Hydrate(ctx context.Context, query *QueryContext, candidate *MemoryCandidate) error {
	if h.engagementStore == nil {
		return nil
	}

	engagement, err := h.engagementStore.GetEngagement(ctx, query.UserID, candidate.ID)
	if err != nil {
		return nil // No engagement data is fine
	}

	candidate.Metadata["engagement_count"] = engagement.Count
	candidate.Metadata["last_engagement_action"] = engagement.LastAction
	candidate.Metadata["last_engagement_time"] = engagement.Timestamp
	candidate.Metadata["total_dwell_ms"] = engagement.DwellMs

	return nil
}
