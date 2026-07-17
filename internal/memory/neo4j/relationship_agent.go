package neo4j

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"agent-memory/internal/config"
	"agent-memory/internal/llm"
	"agent-memory/internal/memory/types"
)

// RelationshipAgent automatically discovers and creates relationships between entities
// using graph algorithms, vector similarity, and LLM analysis
type RelationshipAgent struct {
	client    *Client
	llmClient llm.Provider
	config    *config.Config
	stopCh    chan struct{}
	running   bool
	runningMu sync.Mutex
}

// NewRelationshipAgent creates a new relationship discovery agent
func NewRelationshipAgent(client *Client, llmClient llm.Provider, cfg *config.Config) *RelationshipAgent {
	return &RelationshipAgent{
		client:    client,
		llmClient: llmClient,
		config:    cfg,
		stopCh:    make(chan struct{}),
	}
}

// Start begins the background relationship discovery process
func (a *RelationshipAgent) Start(ctx context.Context) {
	a.runningMu.Lock()
	defer a.runningMu.Unlock()
	if a.running {
		return
	}
	// Skip starting when Neo4j is unavailable. Every code path in this
	// agent touches the graph client, so a nil client means we cannot do
	// any work; failing fast here prevents the periodic ticker from
	// panicking later in the background goroutine.
	if a.client == nil {
		return
	}
	a.running = true
	go a.run(ctx)
}

func (a *RelationshipAgent) Stop() {
	a.runningMu.Lock()
	defer a.runningMu.Unlock()
	if !a.running {
		return
	}
	close(a.stopCh)
	a.running = false
}

func (a *RelationshipAgent) run(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-a.stopCh:
			return
		case <-ticker.C:
			a.discoverRelationships(ctx)
		}
	}
}

// discoverRelationships finds potential relationships and creates them
func (a *RelationshipAgent) discoverRelationships(ctx context.Context) error {
	entities, err := a.getEntitiesForAnalysis(ctx)
	if err != nil {
		return fmt.Errorf("failed to get entities: %w", err)
	}

	for _, entity := range entities {
		if err := a.analyzeEntityRelationships(ctx, entity); err != nil {
			continue
		}
	}

	return nil
}

// getEntitiesForAnalysis returns entities that need relationship analysis
func (a *RelationshipAgent) getEntitiesForAnalysis(ctx context.Context) ([]*types.Entity, error) {
	session, cleanup, err := a.client.AcquireSession(ctx)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	result, err := session.Run(ctx, `
		MATCH (e:Entity)
		WHERE NOT EXISTS((e)-[:RELATED_TO]-(:Entity))
		RETURN e.id as id, e.name as name, e.type as type
		LIMIT 50
	`, nil)
	if err != nil {
		return nil, err
	}
	defer result.Consume(ctx)

	var entities []*types.Entity
	for result.Next(ctx) {
		record := result.Record()
		id, _ := record.Get("id")
		name, _ := record.Get("name")
		entityType, _ := record.Get("type")

		entities = append(entities, &types.Entity{
			ID:   id.(string),
			Name: name.(string),
			Type: entityType.(string),
		})
	}

	return entities, nil
}

// analyzeEntityRelationships uses LLM to find and create relationships for an entity
func (a *RelationshipAgent) analyzeEntityRelationships(ctx context.Context, entity *types.Entity) error {
	relatedEntities, err := a.findRelatedEntities(ctx, entity)
	if err != nil {
		return err
	}

	if len(relatedEntities) == 0 {
		return nil
	}

	relationships, err := a.determineRelationshipsWithLLM(ctx, entity, relatedEntities)
	if err != nil {
		return err
	}

	for _, rel := range relationships {
		if err := a.createRelationship(ctx, rel); err != nil {
			continue
		}
	}

	return nil
}

// findRelatedEntities finds entities that might be related using graph traversal
func (a *RelationshipAgent) findRelatedEntities(ctx context.Context, entity *types.Entity) ([]*types.Entity, error) {
	session, cleanup, err := a.client.AcquireSession(ctx)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	result, err := session.Run(ctx, `
		MATCH (e:Entity {id: $id})-[*1..2]-(related:Entity)
		WHERE related.id <> $id
		RETURN DISTINCT related.id as id, related.name as name, related.type as type
		LIMIT 20
	`, map[string]interface{}{
		"id": entity.ID,
	})
	if err != nil {
		return nil, err
	}
	defer result.Consume(ctx)

	var entities []*types.Entity
	for result.Next(ctx) {
		record := result.Record()
		id, _ := record.Get("id")
		name, _ := record.Get("name")
		entityType, _ := record.Get("type")

		entities = append(entities, &types.Entity{
			ID:   id.(string),
			Name: name.(string),
			Type: entityType.(string),
		})
	}

	return entities, nil
}

// determineRelationshipsWithLLM uses LLM to analyze entities and determine relationships
func (a *RelationshipAgent) determineRelationshipsWithLLM(ctx context.Context, source *types.Entity, candidates []*types.Entity) ([]*RelationshipSuggestion, error) {
	if a.llmClient == nil {
		return nil, nil
	}

	candidateDescriptions := make([]string, 0, len(candidates))
	for i, c := range candidates {
		candidateDescriptions = append(candidateDescriptions, fmt.Sprintf("%d. %s (Type: %s, ID: %s)", i+1, c.Name, c.Type, c.ID))
	}

	prompt := fmt.Sprintf(`Analyze the relationship between entities and suggest meaningful relationships.

Source Entity:
- Name: %s
- Type: %s
- ID: %s

Candidate Entities:
%s

For each candidate that has a meaningful relationship with the source entity, respond in JSON format:
[
  {
    "target_id": "<entity_id>",
    "relation_type": "<RELATION_TYPE>",
    "confidence": <0.0-1.0>,
    "reason": "<brief explanation>"
  }
]

Only include relationships where confidence >= 0.7. Valid relation types: KNOWS, HAS, RELATED_TO, DEPENDS_ON, USES, CREATED_BY, PART_OF, IMPROVES, CONFLICTS, FOLLOWS, LIKES, DISLIKES, SUBSCRIBED, MEMBER_OF, OWNS, WORKS_WITH, MANAGES, CONTRADICTS, IMPLIES, MERGES, SUPPORTS, REFUTES, SPECIALIZES, GENERALIZES, ENTAILS

Respond with ONLY the JSON array, no other text.`, source.Name, source.Type, source.ID, strings.Join(candidateDescriptions, "\n"))

	response, err := a.llmClient.Complete(ctx, &llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: "user", Content: prompt},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("LLM completion failed: %w", err)
	}

	var suggestions []*RelationshipSuggestion
	if err := parseJSONResponse(response.Content, &suggestions); err != nil {
		return nil, fmt.Errorf("failed to parse LLM response: %w", err)
	}

	filtered := make([]*RelationshipSuggestion, 0)
	for _, s := range suggestions {
		if s.Confidence >= 0.7 {
			s.SourceID = source.ID
			filtered = append(filtered, s)
		}
	}

	return filtered, nil
}

// RelationshipSuggestion represents a suggested relationship from LLM
type RelationshipSuggestion struct {
	SourceID     string  `json:"source_id"`
	TargetID     string  `json:"target_id"`
	RelationType string  `json:"relation_type"`
	Confidence   float64 `json:"confidence"`
	Reason       string  `json:"reason"`
}

// createRelationship creates a relationship in Neo4j
func (a *RelationshipAgent) createRelationship(ctx context.Context, rel *RelationshipSuggestion) error {
	session, cleanup, err := a.client.AcquireSession(ctx)
	if err != nil {
		return err
	}
	defer cleanup()

	if err := ValidateRelationType(rel.RelationType); err != nil {
		return err
	}

	_, err = session.Run(ctx, fmt.Sprintf(`
		MATCH (source:Entity {id: $sourceId})
		MATCH (target:Entity {id: $targetId})
		MERGE (source)-[r:%s {confidence: $confidence, reason: $reason, auto_created: true, created_at: datetime()}]->(target)
		RETURN r
	`, rel.RelationType), map[string]interface{}{
		"sourceId":   rel.SourceID,
		"targetId":   rel.TargetID,
		"confidence": rel.Confidence,
		"reason":     rel.Reason,
	})
	if err != nil {
		return err
	}

	return nil
}

// parseJSONResponse parses JSON from LLM response, handling potential markdown code blocks
func parseJSONResponse(text string, v interface{}) error {
	text = strings.TrimSpace(text)
	if strings.HasPrefix(text, "```json") {
		text = strings.TrimPrefix(text, "```json")
		text = strings.TrimSuffix(text, "```")
		text = strings.TrimSpace(text)
	} else if strings.HasPrefix(text, "```") {
		text = strings.TrimPrefix(text, "```")
		text = strings.TrimSuffix(text, "```")
		text = strings.TrimSpace(text)
	}

	return json.Unmarshal([]byte(text), v)
}

// TriggerAnalysis manually triggers relationship analysis for an entity
func (a *RelationshipAgent) TriggerAnalysis(ctx context.Context, entityID string) error {
	entity, err := a.getEntityByID(ctx, entityID)
	if err != nil {
		return err
	}

	return a.analyzeEntityRelationships(ctx, entity)
}

func (a *RelationshipAgent) getEntityByID(ctx context.Context, id string) (*types.Entity, error) {
	session, cleanup, err := a.client.AcquireSession(ctx)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	result, err := session.Run(ctx, `
		MATCH (e:Entity {id: $id})
		RETURN e.id as id, e.name as name, e.type as type
	`, map[string]interface{}{"id": id})
	if err != nil {
		return nil, err
	}
	defer result.Consume(ctx)

	if result.Next(ctx) {
		record := result.Record()
		id, _ := record.Get("id")
		name, _ := record.Get("name")
		entityType, _ := record.Get("type")

		return &types.Entity{
			ID:   id.(string),
			Name: name.(string),
			Type: entityType.(string),
		}, nil
	}

	return nil, fmt.Errorf("entity not found: %s", id)
}
