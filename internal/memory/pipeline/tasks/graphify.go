package tasks

import (
	"context"
	"encoding/json"
	"fmt"

	"agent-memory/internal/memory/datapoint"
)

type GraphifyTask struct {
	graphStore GraphStore
}

type GraphStore interface {
	CreateNode(label string, properties map[string]interface{}) error
	CreateRelationship(fromID, toID, relType string, properties map[string]interface{}) error
}

type GraphifyTaskConfig struct {
	NodeLabel string
}

func NewGraphifyTask(graphStore GraphStore, cfg GraphifyTaskConfig) *GraphifyTask {
	label := cfg.NodeLabel
	if label == "" {
		label = "Entity"
	}
	return &GraphifyTask{
		graphStore: graphStore,
	}
}

func (t *GraphifyTask) Name() string {
	return "graphify"
}

func (t *GraphifyTask) Execute(ctx context.Context, input *datapoint.DataPoint) (*datapoint.DataPoint, error) {
	if t.graphStore == nil {
		return input, nil
	}

	switch input.Type {
	case datapoint.DataPointTypeEntity:
		return t.createEntityNode(input)
	case datapoint.DataPointTypeRelation:
		return t.createRelationEdge(input)
	case datapoint.DataPointTypeConcept:
		return t.createConceptNodes(input)
	default:
		return input, nil
	}
}

func (t *GraphifyTask) createEntityNode(input *datapoint.DataPoint) (*datapoint.DataPoint, error) {
	entityType := input.Properties["entity_type"]
	if entityType == "" {
		entityType = "Entity"
	}

	properties := map[string]interface{}{
		"id":     input.ID,
		"name":   input.Content,
		"type":   entityType,
		"source": input.Source.Name,
	}

	for k, v := range input.Properties {
		properties[k] = v
	}

	for k, v := range input.Metadata {
		properties[k] = v
	}

	if err := t.graphStore.CreateNode(entityType, properties); err != nil {
		return nil, fmt.Errorf("failed to create entity node: %w", err)
	}

	input.Metadata["graph_node_created"] = true
	return input, nil
}

func (t *GraphifyTask) createRelationEdge(input *datapoint.DataPoint) (*datapoint.DataPoint, error) {
	sourceID := input.Properties["source_id"]
	targetID := input.Properties["target_id"]
	relType := input.Properties["relation_type"]

	if sourceID == "" || targetID == "" {
		return nil, fmt.Errorf("relation missing source_id or target_id")
	}

	if relType == "" {
		relType = "RELATED_TO"
	}

	properties := map[string]interface{}{
		"id":   input.ID,
		"type": relType,
	}

	for k, v := range input.Properties {
		if k != "source_id" && k != "target_id" && k != "relation_type" {
			properties[k] = v
		}
	}

	if err := t.graphStore.CreateRelationship(sourceID, targetID, relType, properties); err != nil {
		return nil, fmt.Errorf("failed to create relationship: %w", err)
	}

	input.Metadata["graph_edge_created"] = true
	return input, nil
}

func (t *GraphifyTask) createConceptNodes(input *datapoint.DataPoint) (*datapoint.DataPoint, error) {
	var extraction ExtractionResult

	data, err := json.Marshal(input.Metadata)
	if err != nil {
		return input, nil
	}

	if err := json.Unmarshal(data, &extraction); err != nil {
		return input, nil
	}

	for _, entity := range extraction.Entities {
		dp := datapoint.NewEntity(entity.Name, entity.Type, entity.Properties)
		dp.SetSource("concept", input.Source.Name, "")
		if _, err := t.createEntityNode(dp); err != nil {
			continue
		}
	}

	for _, rel := range extraction.Relations {
		dp := datapoint.NewRelation(rel.Source, rel.Target, rel.Type, rel.Properties)
		dp.SetSource("concept", input.Source.Name, "")
		if _, err := t.createRelationEdge(dp); err != nil {
			continue
		}
	}

	input.Metadata["graph_nodes_created"] = len(extraction.Entities)
	input.Metadata["graph_edges_created"] = len(extraction.Relations)
	return input, nil
}

func (t *GraphifyTask) Validate(input *datapoint.DataPoint) error {
	if input == nil {
		return fmt.Errorf("input is nil")
	}
	return nil
}

type NullGraphStore struct{}

func (n *NullGraphStore) CreateNode(label string, properties map[string]interface{}) error {
	return nil
}

func (n *NullGraphStore) CreateRelationship(fromID, toID, relType string, properties map[string]interface{}) error {
	return nil
}
