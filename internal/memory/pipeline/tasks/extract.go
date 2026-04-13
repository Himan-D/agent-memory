package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"agent-memory/internal/llm"
	"agent-memory/internal/memory/datapoint"
)

type ExtractTask struct {
	llm         llm.Provider
	entityTypes []string
}

type ExtractTaskConfig struct {
	EntityTypes []string
	Model       string
}

func NewExtractTask(llmClient llm.Provider, cfg ExtractTaskConfig) *ExtractTask {
	entityTypes := cfg.EntityTypes
	if len(entityTypes) == 0 {
		entityTypes = []string{"PERSON", "ORGANIZATION", "LOCATION", "CONCEPT", "EVENT", "OBJECT"}
	}
	return &ExtractTask{
		llm:         llmClient,
		entityTypes: entityTypes,
	}
}

func (t *ExtractTask) Name() string {
	return "extract"
}

type ExtractionResult struct {
	Entities  []Entity   `json:"entities"`
	Relations []Relation `json:"relations"`
}

type Entity struct {
	Name       string            `json:"name"`
	Type       string            `json:"type"`
	Properties map[string]string `json:"properties,omitempty"`
}

type Relation struct {
	Source     string            `json:"source"`
	Target     string            `json:"target"`
	Type       string            `json:"relation_type"`
	Properties map[string]string `json:"properties,omitempty"`
}

func (t *ExtractTask) Execute(ctx context.Context, input *datapoint.DataPoint) (*datapoint.DataPoint, error) {
	if t.llm == nil {
		return t.mockExtract(input)
	}

	prompt := t.buildExtractionPrompt(input.Content)

	resp, err := t.llm.Complete(ctx, &llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: "user", Content: prompt},
		},
		Model:       "gpt-4o",
		MaxTokens:   4000,
		Temperature: 0.1,
	})
	if err != nil {
		return nil, fmt.Errorf("extraction failed: %w", err)
	}

	result, err := t.parseExtraction(resp.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse extraction: %w", err)
	}

	return t.buildDataPoint(input, result)
}

func (t *ExtractTask) buildExtractionPrompt(content string) string {
	entityTypes := strings.Join(t.entityTypes, ", ")
	return fmt.Sprintf(`Extract entities and relations from the following text.

Extract the following entity types: %s

For each entity, provide:
- name: The entity name
- type: The entity type from the list above
- properties: Any additional properties (optional)

For each relation, provide:
- source: The subject entity name
- target: The object entity name  
- relation_type: The relationship type (e.g., WORKS_FOR, LOCATED_IN, CREATED_BY)

Return the results as a JSON object with "entities" and "relations" arrays.

Text:
%s

JSON output:`, entityTypes, content)
}

func (t *ExtractTask) parseExtraction(content string) (*ExtractionResult, error) {
	var result ExtractionResult

	jsonStr := t.extractJSON(content)
	if jsonStr == "" {
		return &result, nil
	}

	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		var entities []Entity
		var relations []Relation

		simple := struct {
			Entities  []Entity   `json:"entities"`
			Relations []Relation `json:"relations"`
		}{Entities: entities, Relations: relations}

		if err2 := json.Unmarshal([]byte(jsonStr), &simple); err2 != nil {
			return nil, err
		}
		result = ExtractionResult{
			Entities:  simple.Entities,
			Relations: simple.Relations,
		}
	}

	return &result, nil
}

func (t *ExtractTask) extractJSON(content string) string {
	start := strings.Index(content, "{")
	end := strings.LastIndex(content, "}")
	if start == -1 || end == -1 {
		start = strings.Index(content, "[")
		end = strings.LastIndex(content, "]")
	}
	if start == -1 || end == -1 {
		return content
	}
	return content[start : end+1]
}

func (t *ExtractTask) buildDataPoint(input *datapoint.DataPoint, result *ExtractionResult) (*datapoint.DataPoint, error) {
	data, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}

	dp := datapoint.New(string(data), datapoint.DataPointTypeConcept)
	dp.SetSource("extraction", input.Source.Name, "")
	dp.Metadata["entities_count"] = len(result.Entities)
	dp.Metadata["relations_count"] = len(result.Relations)
	dp.Metadata["entities"] = result.Entities
	dp.Metadata["relations"] = result.Relations

	return dp, nil
}

func (t *ExtractTask) Validate(input *datapoint.DataPoint) error {
	if input == nil {
		return fmt.Errorf("input is nil")
	}
	if input.Content == "" {
		return fmt.Errorf("input content is empty")
	}
	return nil
}

func (t *ExtractTask) mockExtract(input *datapoint.DataPoint) (*datapoint.DataPoint, error) {
	result := &ExtractionResult{
		Entities: []Entity{
			{Name: "Mock Entity", Type: "CONCEPT"},
		},
		Relations: []Relation{},
	}
	return t.buildDataPoint(input, result)
}

type ExtractAllTask struct {
	llm         llm.Provider
	entityTypes []string
}

func NewExtractAllTask(llmClient llm.Provider, cfg ExtractTaskConfig) *ExtractAllTask {
	entityTypes := cfg.EntityTypes
	if len(entityTypes) == 0 {
		entityTypes = []string{"PERSON", "ORGANIZATION", "LOCATION", "CONCEPT", "EVENT", "OBJECT"}
	}
	return &ExtractAllTask{
		llm:         llmClient,
		entityTypes: entityTypes,
	}
}

func (t *ExtractAllTask) Name() string {
	return "extract_all"
}

func (t *ExtractAllTask) Execute(ctx context.Context, input *datapoint.DataPoint) (*datapoint.ExtractedData, error) {
	if input.Type != datapoint.DataPointTypeChunk {
		return nil, fmt.Errorf("input must be a chunk")
	}

	result := datapoint.NewExtractedData()

	if t.llm == nil {
		return result, nil
	}

	prompt := t.buildExtractionPrompt(input.Content)

	resp, err := t.llm.Complete(ctx, &llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: "user", Content: prompt},
		},
		Model:       "gpt-4o",
		MaxTokens:   4000,
		Temperature: 0.1,
	})
	if err != nil {
		return nil, fmt.Errorf("extraction failed: %w", err)
	}

	extraction, err := t.parseExtraction(resp.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse extraction: %w", err)
	}

	for _, entity := range extraction.Entities {
		dp := datapoint.NewEntity(entity.Name, entity.Type, entity.Properties)
		dp.SetSource("extraction", input.Source.Name, "")
		result.AddEntity(dp)
	}

	for _, rel := range extraction.Relations {
		dp := datapoint.NewRelation(rel.Source, rel.Target, rel.Type, rel.Properties)
		dp.SetSource("extraction", input.Source.Name, "")
		result.AddRelation(dp)
	}

	return result, nil
}

func (t *ExtractAllTask) Validate(input *datapoint.DataPoint) error {
	if input == nil {
		return fmt.Errorf("input is nil")
	}
	if input.Type != datapoint.DataPointTypeChunk {
		return fmt.Errorf("input must be a chunk, got %s", input.Type)
	}
	return nil
}

func (t *ExtractAllTask) buildExtractionPrompt(content string) string {
	entityTypes := strings.Join(t.entityTypes, ", ")
	return fmt.Sprintf(`Extract all entities and relations from the following text.

Entity types: %s

Return JSON with "entities" (array of {name, type, properties}) and "relations" (array of {source, target, relation_type}).

Text:
%s

JSON:`, entityTypes, content)
}

func (t *ExtractAllTask) parseExtraction(content string) (*ExtractionResult, error) {
	var result ExtractionResult
	jsonStr := t.extractJSON(content)
	if jsonStr == "" {
		return &result, nil
	}
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (t *ExtractAllTask) extractJSON(content string) string {
	start := strings.Index(content, "{")
	end := strings.LastIndex(content, "}")
	if start == -1 || end == -1 {
		return content
	}
	return content[start : end+1]
}
