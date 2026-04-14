package memify

import (
	"context"
	"fmt"

	"agent-memory/internal/llm"
	"agent-memory/internal/memory/datapoint"
)

type Enricher interface {
	Enrich(ctx context.Context, input *datapoint.ExtractedData) (*datapoint.ExtractedData, error)
}

type Memify struct {
	llm    llm.Provider
	pipes  []Pipeline
	config Config
}

type Config struct {
	EnableTripletEmbeddings bool
	EnableCodingRules       bool
	EnableCustomRules       bool
	Rules                   []string
}

type Pipeline interface {
	Run(ctx context.Context, data *datapoint.ExtractedData) (*datapoint.ExtractedData, error)
}

func New(llmClient llm.Provider, cfg Config) *Memify {
	m := &Memify{
		llm:    llmClient,
		config: cfg,
		pipes:  make([]Pipeline, 0),
	}

	m.registerDefaultPipelines()
	return m
}

func (m *Memify) registerDefaultPipelines() {
	if m.config.EnableTripletEmbeddings {
		m.pipes = append(m.pipes, NewTripletEmbeddingPipeline(m.llm))
	}

	if m.config.EnableCodingRules {
		m.pipes = append(m.pipes, NewCodingRulesPipeline(m.llm))
	}

	if m.config.EnableCustomRules && len(m.config.Rules) > 0 {
		m.pipes = append(m.pipes, NewCustomRulesPipeline(m.llm, m.config.Rules))
	}
}

func (m *Memify) Memify(ctx context.Context, data *datapoint.ExtractedData) (*datapoint.ExtractedData, error) {
	result := data

	for _, pipe := range m.pipes {
		enriched, err := pipe.Run(ctx, result)
		if err != nil {
			continue
		}
		result = enriched
	}

	return result, nil
}

type TripletEmbeddingPipeline struct {
	llm llm.Provider
}

func NewTripletEmbeddingPipeline(llmClient llm.Provider) *TripletEmbeddingPipeline {
	return &TripletEmbeddingPipeline{llm: llmClient}
}

func (p *TripletEmbeddingPipeline) Run(ctx context.Context, data *datapoint.ExtractedData) (*datapoint.ExtractedData, error) {
	for _, rel := range data.Relations {
		rel.Metadata["triplet_embedding"] = fmt.Sprintf("%s_%s_%s", rel.Properties["source_id"], rel.Properties["relation_type"], rel.Properties["target_id"])
	}
	return data, nil
}

type CodingRulesPipeline struct {
	llm llm.Provider
}

func NewCodingRulesPipeline(llmClient llm.Provider) *CodingRulesPipeline {
	return &CodingRulesPipeline{llm: llmClient}
}

func (p *CodingRulesPipeline) Run(ctx context.Context, data *datapoint.ExtractedData) (*datapoint.ExtractedData, error) {
	for _, entity := range data.Entities {
		if entity.Properties["entity_type"] == "FUNCTION" || entity.Properties["entity_type"] == "METHOD" {
			entity.Metadata["coding_rule"] = true
		}
	}
	return data, nil
}

type CustomRulesPipeline struct {
	llm   llm.Provider
	rules []string
}

func NewCustomRulesPipeline(llmClient llm.Provider, rules []string) *CustomRulesPipeline {
	return &CustomRulesPipeline{llm: llmClient, rules: rules}
}

func (p *CustomRulesPipeline) Run(ctx context.Context, data *datapoint.ExtractedData) (*datapoint.ExtractedData, error) {
	return data, nil
}

type NodeSet struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Type        string                 `json:"type"`
	Nodes       []string               `json:"nodes"`
	Metadata    map[string]interface{} `json:"metadata"`
	CreatedAt   string                 `json:"created_at"`
}

type NodeSetManager struct {
	nodeSets map[string]*NodeSet
}

func NewNodeSetManager() *NodeSetManager {
	return &NodeSetManager{
		nodeSets: make(map[string]*NodeSet),
	}
}

func (m *NodeSetManager) Create(set *NodeSet) error {
	if set.ID == "" {
		return fmt.Errorf("NodeSet ID is required")
	}
	m.nodeSets[set.ID] = set
	return nil
}

func (m *NodeSetManager) Get(id string) (*NodeSet, bool) {
	set, ok := m.nodeSets[id]
	return set, ok
}

func (m *NodeSetManager) List() []*NodeSet {
	sets := make([]*NodeSet, 0, len(m.nodeSets))
	for _, s := range m.nodeSets {
		sets = append(sets, s)
	}
	return sets
}

func (m *NodeSetManager) AddNode(setID, nodeID string) error {
	set, ok := m.nodeSets[setID]
	if !ok {
		return fmt.Errorf("NodeSet not found: %s", setID)
	}
	set.Nodes = append(set.Nodes, nodeID)
	return nil
}

func (m *NodeSetManager) RemoveNode(setID, nodeID string) error {
	set, ok := m.nodeSets[setID]
	if !ok {
		return fmt.Errorf("NodeSet not found: %s", setID)
	}

	for i, n := range set.Nodes {
		if n == nodeID {
			set.Nodes = append(set.Nodes[:i], set.Nodes[i+1:]...)
			break
		}
	}
	return nil
}

func (m *NodeSetManager) GetNodes(setID string) ([]string, error) {
	set, ok := m.nodeSets[setID]
	if !ok {
		return nil, fmt.Errorf("NodeSet not found: %s", setID)
	}
	return set.Nodes, nil
}

const (
	NodeSetTypeCodingRules = "coding_agent_rules"
	NodeSetTypeConcepts    = "concepts"
	NodeSetTypeEntities    = "entities"
)

func NewCodingRulesNodeSet() *NodeSet {
	return &NodeSet{
		ID:          "coding_agent_rules",
		Name:        "Coding Rules",
		Description: "Coding rules and patterns extracted from code",
		Type:        NodeSetTypeCodingRules,
		Nodes:       make([]string, 0),
		Metadata:    make(map[string]interface{}),
	}
}
