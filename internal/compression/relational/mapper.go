package relational

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"agent-memory/internal/llm"
)

type RelationalMapper struct {
	llmClient    llm.Provider
	entityCache *EntityCache
	relTypes   []RelationType
}

type EntityCache struct {
	mu      sync.RWMutex
	entities map[string]*CachedEntity
}

type CachedEntity struct {
	Name       string
	Type       string
	Properties map[string]interface{}
	Confidence float64
	LastSeen   int64
}

type RelationType string

const (
	RelTypeKnows      RelationType = "KNOWS"
	RelTypeUses       RelationType = "USES"
	RelTypePartOf    RelationType = "PART_OF"
	RelTypeDepends   RelationType = "DEPENDS_ON"
	RelTypeSimilar   RelationType = "SIMILAR_TO"
	RelTypeImplements RelationType = "IMPLEMENTS"
	RelTypeCreatedBy RelationType = "CREATED_BY"
)

type Entity struct {
	Name     string      `json:"name"`
	Type     string      `json:"type"`
	Role     string      `json:"role,omitempty"`
	Metadata interface{} `json:"metadata,omitempty"`
}

type Relationship struct {
	From     string      `json:"from"`
	To       string      `json:"to"`
	Type     RelationType `json:"type"`
	Weight   float64     `json:"weight"`
	Metadata interface{} `json:"metadata,omitempty"`
}

type RelationalGraph struct {
	Entities     []Entity     `json:"entities"`
	Relationships []Relationship `json:"relationships"`
}

var defaultRelationTypes = []RelationType{
	RelTypeKnows, RelTypeUses, RelTypePartOf, RelTypeDepends,
	RelTypeSimilar, RelTypeImplements, RelTypeCreatedBy,
}

func NewRelationalMapper(client llm.Provider) *RelationalMapper {
	return &RelationalMapper{
		llmClient:  client,
		entityCache: &EntityCache{entities: make(map[string]*CachedEntity)},
		relTypes:   defaultRelationTypes,
	}
}

func (r *RelationalMapper) ExtractRelations(ctx context.Context, memories []string) (*RelationalGraph, error) {
	if len(memories) == 0 {
		return &RelationalGraph{}, nil
	}

	combinedMemories := strings.Join(memories, "\n---\n")

	prompt := r.buildExtractionPrompt(combinedMemories)

	resp, err := r.llmClient.Complete(ctx, &llm.CompletionRequest{
		Model: "gpt-4o-mini",
		Messages: []llm.Message{
			{Role: "system", Content: r.getSystemPrompt()},
			{Role: "user", Content: prompt},
		},
		Temperature: 0.3,
		MaxTokens: 4000,
	})

	if err != nil {
		return nil, fmt.Errorf("LLM completion failed: %w", err)
	}

	entities, relationships := r.parseLLMResponse(resp.Content)

	for i := range entities {
		r.entityCache.mu.Lock()
		r.entityCache.entities[entities[i].Name] = &CachedEntity{
			Name:       entities[i].Name,
			Type:       entities[i].Type,
			Confidence: 0.9,
		}
		r.entityCache.mu.Unlock()
	}

	return &RelationalGraph{
		Entities:      entities,
		Relationships: relationships,
	}, nil
}

func (r *RelationalMapper) getSystemPrompt() string {
	return `You are an expert at analyzing text to extract entities and their relationships.
Your task is to identify:
1. ENTITIES: People, concepts, tools, technologies, organizations mentioned
2. RELATIONSHIPS: How entities relate to each other

For each text, extract all entities and determine relationships between them.

Output a structured analysis with:
- clear entity identification
- relationship direction and weight
- confidence scores

Focus on technical, scientific, and factual relationships.`
}

func (r *RelationalMapper) buildExtractionPrompt(memories string) string {
	return fmt.Sprintf(`Analyze the following memories and extract all entities and their relationships.

Memories:
%s

For each entity found, determine:
1. Its type (PERSON, CONCEPT, TOOL, TECHNOLOGY, ORGANIZATION, LOCATION, ACTION, etc.)
2. Its role in the context
3. Relationships to other entities (KNOWS, USES, DEPENDS_ON, PART_OF, SIMILAR_TO, CREATED_BY, etc.)

IMPORTANT RELATIONSHIP RULES:
- KNOWS: Person knows about concept/tool
- USES: Person/tool uses another tool/technology  
- DEPENDS_ON: Concept depends on another concept
- PART_OF: Entity is part of larger entity
- SIMILAR_TO: Two concepts are similar/related
- IMPLEMENTES: Tool implements a concept
- CREATED_BY: Entity created by another entity

Provide detailed analysis with multiple relationships. Be thorough.`, memories)
}

func (r *RelationalMapper) parseLLMResponse(content string) ([]Entity, []Relationship) {
	var entities []Entity
	var relationships []Relationship

	lines := strings.Split(content, "\n")
	var currentEntity string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") {
			parts := strings.Split(strings.TrimPrefix(line, "- *"), ":")

			name := strings.TrimSpace(parts[0])
			if len(parts) > 1 {
				details := strings.TrimSpace(parts[1])

				entityType := "CONCEPT"
				if strings.Contains(details, "PERSON") {
					entityType = "PERSON"
				} else if strings.Contains(details, "TOOL") || strings.Contains(details, "technology") {
					entityType = "TOOL"
				} else if strings.Contains(details, "organization") {
					entityType = "ORGANIZATION"
				} else if strings.Contains(details, "location") {
					entityType = "LOCATION"
				}

				entities = append(entities, Entity{
					Name: name,
					Type: entityType,
				})

				if currentEntity != "" && currentEntity != name {
					relationships = append(relationships, Relationship{
						From:   currentEntity,
						To:     name,
						Type:   RelTypeKnows,
						Weight: 0.8,
					})
				}
				currentEntity = name
			}
		}
	}

	if len(entities) == 0 {
		entities = r.extractSimpleEntities(content)
	}

	return entities, relationships
}

func (r *RelationalMapper) extractSimpleEntities(content string) []Entity {
	var entities []Entity

	words := strings.Fields(content)
	var seen map[string]bool = make(map[string]bool)

_keywords := []string{"machine", "learning", "AI", "neural", "network", "model", "data", "algorithm", "python", "training", "inference"}

	for _, word := range words {
		clean := strings.Trim(strings.ToLower(word), ".,!?;:\"'()[]")
		if _, exists := seen[clean]; exists {
			continue
		}

		for _, kw := range _keywords {
			if strings.Contains(clean, kw) && len(clean) > 2 {
				seen[clean] = true
				entityType := "CONCEPT"
				if clean == "python" {
					entityType = "TOOL"
				}

				entities = append(entities, Entity{
					Name: clean,
					Type: entityType,
				})
			}
		}
	}

	return entities
}

func (r *RelationalMapper) CompressWithRelations(ctx context.Context, memories []string) (string, float64, error) {
	graph, err := r.ExtractRelations(ctx, memories)
	if err != nil {
		return "", 0, err
	}

	if len(graph.Entities) == 0 {
		combined := strings.Join(memories, " ")
		return combined, 0.0, nil
	}

	compressed := r.buildCompressedSummary(graph, memories)

	originalTokens := 0
	for _, mem := range memories {
		originalTokens += len(strings.Fields(mem))
	}

	compressedTokens := len(strings.Fields(compressed))
	reduction := 1.0 - float64(compressedTokens)/float64(originalTokens)

	return compressed, reduction, nil
}

func (r *RelationalMapper) buildCompressedSummary(graph *RelationalGraph, originals []string) string {
	var summary strings.Builder

	summary.WriteString("=== COMPRESSED MEMORY GRAPH ===\n")

	summary.WriteString(fmt.Sprintf("Entities (%d):\n", len(graph.Entities)))
	for _, e := range graph.Entities {
		summary.WriteString(fmt.Sprintf("- %s [%s]\n", e.Name, e.Type))
	}

	summary.WriteString(fmt.Sprintf("\nRelationships (%d):\n", len(graph.Relationships)))
	for _, rel := range graph.Relationships {
		summary.WriteString(fmt.Sprintf("- %s --(%s)--> %s\n", rel.From, rel.Type, rel.To))
	}

	summary.WriteString("\n=== KEY FACTS ===\n")
	for _, mem := range originals {
		lines := strings.Split(mem, ".")
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if len(trimmed) > 20 && len(trimmed) < 200 {
				summary.WriteString("- ")
				summary.WriteString(trimmed)
				summary.WriteString("\n")
			}
		}
	}

	return summary.String()
}

func (r *RelationalMapper) FindPath(fromEntity, toEntity string, graph *RelationalGraph) []string {
	var path []string
	var visited map[string]bool = make(map[string]bool)

	var dfs func(current string, depth int)
	dfs = func(current string, depth int) {
		if current == toEntity {
			path = append(path, current)
			return
		}

		if depth > 5 || visited[current] {
			return
		}

		visited[current] = true
		path = append(path, current)

		for _, rel := range graph.Relationships {
			if rel.From == current {
				dfs(rel.To, depth+1)
				if len(path) > 0 && path[len(path)-1] == toEntity {
					return
				}
				path = path[:len(path)-1]
			}
		}

		visited[current] = false
	}

	dfs(fromEntity, 0)

	return path
}

func (r *RelationalMapper) GetSimilarEntities(entityName string, graph *RelationalGraph) []string {
	var similar []string

	for _, rel := range graph.Relationships {
		if (rel.From == entityName || rel.To == entityName) && rel.Type == RelTypeSimilar {
			if rel.From == entityName {
				similar = append(similar, rel.To)
			} else {
				similar = append(similar, rel.From)
			}
		}
	}

	return similar
}

func (r *RelationalMapper) SuggestCompressions(entities []Entity) []string {
	var suggestions []string

	techEntities := make(map[string][]string)
	personEntities := make(map[string][]string)

	for _, e := range entities {
		switch e.Type {
		case "CONCEPT", "TOOL", "TECHNOLOGY":
			techEntities[e.Name] = append(techEntities[e.Name], e.Name)
		case "PERSON":
			personEntities[e.Name] = append(personEntities[e.Name], e.Name)
		}
	}

	for e := range techEntities {
		if len(techEntities[e]) > 1 {
			suggestions = append(suggestions, fmt.Sprintf("Group related: %v", techEntities[e]))
		}
	}

	return suggestions
}