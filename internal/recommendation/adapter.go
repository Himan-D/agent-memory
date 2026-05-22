package recommendation

import (
	"context"
	"time"
)

// MemoryAdapter bridges the internal memory service to the recommendation interfaces.
type MemoryAdapter struct {
	memSvc MemoryServiceInterface
}

// MemoryServiceInterface is a subset of memory.Service methods needed by recommendation.
type MemoryServiceInterface interface {
	GetMemoriesByIDs(ctx context.Context, ids []string, userID string, projectID string) ([]interface{}, error)
	GetMemory(ctx context.Context, id string, userID string, projectID string) (interface{}, error)
}

// NewMemoryAdapter creates an adapter from a memory service.
func NewMemoryAdapter(memSvc MemoryServiceInterface) *MemoryAdapter {
	return &MemoryAdapter{memSvc: memSvc}
}

// GetMemoriesByIDs implements MemoryStore.
func (a *MemoryAdapter) GetMemoriesByIDs(ctx context.Context, ids []string) ([]interface{}, error) {
	if a.memSvc == nil || len(ids) == 0 {
		return nil, nil
	}
	return a.memSvc.GetMemoriesByIDs(ctx, ids, "", "")
}

// GetRecentMemoriesByEntity implements MemoryStore.
func (a *MemoryAdapter) GetRecentMemoriesByEntity(ctx context.Context, entityID string, limit int, maxAge time.Duration) ([]string, error) {
	// This would query Neo4j for recent memories by entity
	// For now, return empty as the spreading activation source handles discovery
	return []string{}, nil
}

// GetMemoryWithMetadata implements MemoryDetailStore.
func (a *MemoryAdapter) GetMemoryWithMetadata(ctx context.Context, id string) (*MemoryDetail, error) {
	if a.memSvc == nil {
		return nil, nil
	}

	mem, err := a.memSvc.GetMemory(ctx, id, "", "")
	if err != nil {
		return nil, err
	}

	// Convert the memory interface to MemoryDetail
	// The actual conversion depends on the memory type returned by the service
	detail := &MemoryDetail{
		ID:      id,
		Content: "",
		Type:    "memory",
	}

	if mem != nil {
		// Try to extract fields from the memory (type assertion would depend on actual type)
		type memoryLike interface {
			GetID() string
			GetText() string
			GetType() string
			GetCreatedAt() time.Time
		}
		if ml, ok := mem.(memoryLike); ok {
			detail.ID = ml.GetID()
			detail.Content = ml.GetText()
			detail.Type = ml.GetType()
			detail.CreatedAt = ml.GetCreatedAt()
		}
	}

	return detail, nil
}

// SearchByVector implements VectorSearchStore.
func (a *MemoryAdapter) SearchByVector(ctx context.Context, vector []float32, limit int, filters map[string]interface{}) ([]string, error) {
	// This would call Qdrant via the memory service
	// For now, return empty as the multi-signal retrieval handles this
	return []string{}, nil
}

// SearchBySpreadingActivation implements GraphSearchStore.
func (a *MemoryAdapter) SearchBySpreadingActivation(ctx context.Context, seedIDs []string, hops int, threshold float64) ([]string, error) {
	// This would use the spreading activation retrieval
	// For now, return empty
	return []string{}, nil
}

// SearchByTopic implements GraphSearchStore.
func (a *MemoryAdapter) SearchByTopic(ctx context.Context, topicIDs []string, limit int) ([]string, error) {
	// This would query Neo4j for topic-based memories
	return []string{}, nil
}
