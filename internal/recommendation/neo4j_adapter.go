package recommendation

import (
	"context"
	"fmt"
	"time"

	"agent-memory/internal/memory"
)

// Neo4jStoreAdapter bridges the Neo4j GraphStore to recommendation interfaces.
type Neo4jStoreAdapter struct {
	graphStore  memory.GraphStore
	vectorStore memory.VectorStore
	embedder    Embedder
	neo4jClient Neo4jQuerier
}

// Neo4jQuerier provides direct Neo4j query access for recommendation-specific queries.
type Neo4jQuerier interface {
	Query(ctx context.Context, cypher string, params map[string]interface{}) ([]map[string]interface{}, error)
}

// NewNeo4jStoreAdapter creates a Neo4j-backed adapter for the recommendation system.
func NewNeo4jStoreAdapter(graph memory.GraphStore, vector memory.VectorStore, embedder Embedder) *Neo4jStoreAdapter {
	return &Neo4jStoreAdapter{
		graphStore:  graph,
		vectorStore: vector,
		embedder:    embedder,
	}
}

// GetMemoriesByIDs implements MemoryStore.
func (a *Neo4jStoreAdapter) GetMemoriesByIDs(ctx context.Context, ids []string) ([]interface{}, error) {
	if a.graphStore == nil || len(ids) == 0 {
		return nil, nil
	}
	memories, err := a.graphStore.GetMemoriesByIDs(ids)
	if err != nil {
		return nil, err
	}
	result := make([]interface{}, len(memories))
	for i, m := range memories {
		result[i] = m
	}
	return result, nil
}

// GetRecentMemoriesByEntity implements MemoryStore.
// Returns memory IDs created by the given entity within the maxAge window.
// If entityID is empty, returns all recent memories.
func (a *Neo4jStoreAdapter) GetRecentMemoriesByEntity(ctx context.Context, entityID string, limit int, maxAge time.Duration) ([]string, error) {
	if a.graphStore == nil {
		return []string{}, nil
	}

	// Fallback: get all memories and filter (works without direct Neo4j query)
	return a.fallbackRecentMemories(entityID, limit, maxAge)
}

func (a *Neo4jStoreAdapter) fallbackRecentMemories(entityID string, limit int, maxAge time.Duration) ([]string, error) {
	// Get all memories and filter client-side
	memories, err := a.graphStore.GetAllMemories()
	if err != nil {
		return []string{}, nil
	}

	cutoff := time.Now().Add(-maxAge)
	var ids []string
	for _, m := range memories {
		if len(ids) >= limit {
			break
		}
		if m.Status != "active" {
			continue
		}
		if !m.CreatedAt.Before(cutoff) {
			// Filter by entity if specified
			if entityID == "" || m.EntityID == entityID {
				ids = append(ids, m.ID)
			}
		}
	}

	return ids, nil
}

// GetMemoryWithMetadata implements MemoryDetailStore.
func (a *Neo4jStoreAdapter) GetMemoryWithMetadata(ctx context.Context, id string) (*MemoryDetail, error) {
	if a.graphStore == nil {
		return nil, fmt.Errorf("graph store not available")
	}

	mem, err := a.graphStore.GetMemory(id)
	if err != nil {
		return nil, err
	}
	if mem == nil {
		return nil, fmt.Errorf("memory %s not found", id)
	}

	detail := &MemoryDetail{
		ID:        mem.ID,
		Content:   mem.Content,
		Type:      string(mem.Type),
		CreatedAt: mem.CreatedAt,
		UpdatedAt: mem.UpdatedAt,
		Tags:      mem.Tags,
	}

	// Extract author from metadata
	if meta := mem.Metadata; meta != nil {
		if source, ok := meta["source"].(string); ok {
			detail.Source = source
		}
		if authorID, ok := meta["author_id"].(string); ok {
			detail.AuthorID = authorID
		}
		if projectID, ok := meta["project_id"].(string); ok {
			detail.ProjectID = projectID
		}
	}

	// Extract entity IDs from relations
	relations, err := a.graphStore.GetMemoryLinks(id)
	if err == nil {
		var entities []string
		for _, link := range relations {
			entities = append(entities, link.ToID)
		}
		if len(entities) > 0 {
			detail.Entities = entities
		}
	}

	return detail, nil
}

// SearchByVector implements VectorSearchStore.
func (a *Neo4jStoreAdapter) SearchByVector(ctx context.Context, vector []float32, limit int, filters map[string]interface{}) ([]string, error) {
	if a.vectorStore == nil || vector == nil {
		return []string{}, nil
	}

	results, err := a.vectorStore.Search(ctx, vector, limit, 0.0, filters)
	if err != nil {
		return []string{}, err
	}

	var ids []string
	for _, r := range results {
		if r.MemoryID != "" {
			ids = append(ids, r.MemoryID)
		}
	}

	return ids, nil
}

// SearchBySpreadingActivation implements GraphSearchStore.
// Uses the Neo4j graph to propagate activation from seed nodes.
func (a *Neo4jStoreAdapter) SearchBySpreadingActivation(ctx context.Context, seedIDs []string, hops int, threshold float64) ([]string, error) {
	if len(seedIDs) == 0 || a.graphStore == nil {
		return []string{}, nil
	}

	if a.neo4jClient == nil {
		return a.spreadingActivationCypher(ctx, seedIDs, hops, threshold)
	}

	return a.spreadingActivationDirect(ctx, seedIDs, hops, threshold)
}

func (a *Neo4jStoreAdapter) spreadingActivationCypher(ctx context.Context, seedIDs []string, hops int, threshold float64) ([]string, error) {
	if a.graphStore == nil {
		return []string{}, nil
	}

	// Use a multi-hop traversal Cypher query to find related memories
	// This is a simplified version of spreading activation using Neo4j's native graph traversal
	cypher := `
		UNWIND $seedIDs AS seedID
		MATCH (seed:Memory {id: seedID})
		OPTIONAL MATCH path = (seed)-[*1..` + fmt.Sprintf("%d", hops) + `]-(related:Memory)
		WHERE related.status = 'active'
		AND related.id <> seed.id
		RETURN DISTINCT related.id AS id,
		       length(path) AS hopCount,
		       count(*) AS connectionCount
		ORDER BY hopCount ASC, connectionCount DESC
		LIMIT 100
	`

	querier, ok := a.graphStore.(Neo4jQuerier)
	if !ok {
		return []string{}, nil
	}

	results, err := querier.Query(ctx, cypher, map[string]interface{}{
		"seedIDs": seedIDs,
	})
	if err != nil {
		return []string{}, err
	}

	var ids []string
	for _, r := range results {
		if id, ok := r["id"].(string); ok {
			ids = append(ids, id)
		}
	}

	return ids, nil
}

func (a *Neo4jStoreAdapter) spreadingActivationDirect(ctx context.Context, seedIDs []string, hops int, threshold float64) ([]string, error) {
	// Direct spreading activation implementation
	activationMap := make(map[string]float64)
	currentFrontier := make(map[string]float64)

	// Initialize seeds with full activation
	for _, id := range seedIDs {
		currentFrontier[id] = 1.0
		activationMap[id] = 1.0
	}

	decayFactor := 0.85

	for hop := 0; hop < hops; hop++ {
		nextFrontier := make(map[string]float64)

		for nodeID, activation := range currentFrontier {
			if activation < threshold {
				continue
			}

			// Get neighbors of this node
			relations, err := a.graphStore.GetEntityRelations(nodeID, "")
			if err != nil {
				continue
			}

			for _, rel := range relations {
				neighborID := rel.ToID
				newActivation := activation * decayFactor

				if newActivation > threshold && newActivation > activationMap[neighborID] {
					nextFrontier[neighborID] = newActivation
					activationMap[neighborID] = newActivation
				}
			}
		}

		currentFrontier = nextFrontier
		if len(currentFrontier) == 0 {
			break
		}
	}

	// Convert to sorted list (exclude seeds)
	var ids []string
	for id, score := range activationMap {
		if score >= threshold {
			// Check if this is a seed
			isSeed := false
			for _, seedID := range seedIDs {
				if id == seedID {
					isSeed = true
					break
				}
			}
			if !isSeed {
				ids = append(ids, id)
			}
		}
	}

	return ids, nil
}

// SearchByTopic implements GraphSearchStore.
func (a *Neo4jStoreAdapter) SearchByTopic(ctx context.Context, topicIDs []string, limit int) ([]string, error) {
	if len(topicIDs) == 0 || a.graphStore == nil {
		return []string{}, nil
	}

	// Query memories linked to topic entities
	cypher := `
		UNWIND $topicIDs AS topicID
		MATCH (topic:Entity {id: topicID})-[:RELATED_TO]-(m:Memory)
		WHERE m.status = 'active'
		RETURN DISTINCT m.id AS id
		ORDER BY m.created_at DESC
		LIMIT $limit
	`

	querier, ok := a.graphStore.(Neo4jQuerier)
	if !ok {
		return []string{}, nil
	}

	results, err := querier.Query(ctx, cypher, map[string]interface{}{
		"topicIDs": topicIDs,
		"limit":    limit,
	})
	if err != nil {
		return []string{}, err
	}

	var ids []string
	for _, r := range results {
		if id, ok := r["id"].(string); ok {
			ids = append(ids, id)
		}
	}

	return ids, nil
}

// Embed implements Embedder (for prompt-based search).
func (a *Neo4jStoreAdapter) Embed(ctx context.Context, text string) ([]float32, error) {
	if a.embedder == nil {
		return nil, fmt.Errorf("embedder not available")
	}
	return a.embedder.Embed(ctx, text)
}
