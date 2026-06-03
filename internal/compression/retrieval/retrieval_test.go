package retrieval

import (
	"context"
	"fmt"
	"testing"

	"agent-memory/internal/memory/types"
)

type mockGraphStore struct {
	entities   map[string]*types.Entity
	relations  map[string][]types.Relation
}

func (m *mockGraphStore) GetEntityRelations(entityID string, relType string) ([]types.Relation, error) {
	return m.relations[entityID], nil
}

func (m *mockGraphStore) GetEntity(id string) (*types.Entity, error) {
	if e, ok := m.entities[id]; ok {
		return e, nil
	}
	return nil, fmt.Errorf("entity not found: %s", id)
}

type mockVectorStore struct {
	results []types.MemoryResult
}

func (m *mockVectorStore) Search(ctx context.Context, query []float32, limit int, threshold float32, filters map[string]interface{}) ([]types.MemoryResult, error) {
	return m.results, nil
}

type mockMemoryService struct {
	results []types.MemoryResult
}

func (m *mockMemoryService) SearchMemories(ctx context.Context, req *types.SearchRequest) ([]types.MemoryResult, error) {
	return m.results, nil
}

func TestNewSpreadingActivation_Defaults(t *testing.T) {
	sa := NewSpreadingActivation(nil, nil, nil)

	if sa.initialBudget != 1.0 {
		t.Errorf("expected initialBudget=1.0, got %f", sa.initialBudget)
	}
	if sa.decayFactor != 0.85 {
		t.Errorf("expected decayFactor=0.85, got %f", sa.decayFactor)
	}
	if sa.threshold != 0.1 {
		t.Errorf("expected threshold=0.1, got %f", sa.threshold)
	}
	if sa.maxHops != 3 {
		t.Errorf("expected maxHops=3, got %d", sa.maxHops)
	}
}

func TestPropagate_AccumulatesActivation(t *testing.T) {
	graph := &mockGraphStore{
		entities: map[string]*types.Entity{
			"A": {ID: "A", Name: "NodeA"},
			"B": {ID: "B", Name: "NodeB"},
			"C": {ID: "C", Name: "NodeC"},
		},
		relations: map[string][]types.Relation{
			"A": {
				{FromID: "A", ToID: "B", Type: "related"},
				{FromID: "A", ToID: "C", Type: "related"},
			},
			"B": {
				{FromID: "B", ToID: "C", Type: "related"},
			},
		},
	}

	sa := NewSpreadingActivation(nil, graph, nil)
	sa.threshold = 0.01

	activationMap := map[string]float64{
		"A": 1.0,
	}

	newMap := sa.propagate(context.Background(), activationMap)

	activatedB := newMap["B"]
	activatedC := newMap["C"]

	if activatedB == 0 {
		t.Error("expected node B to receive activation from A")
	}
	if activatedC == 0 {
		t.Error("expected node C to receive activation from A")
	}

	expectedB := 1.0 * 0.85
	if activatedB < expectedB-0.01 || activatedB > expectedB+0.01 {
		t.Errorf("expected B activation ~= %f, got %f", expectedB, activatedB)
	}

	expectedCfromA := 1.0 * 0.85
	if activatedC < expectedCfromA-0.01 {
		t.Errorf("expected C activation >= %f (from A alone), got %f", expectedCfromA, activatedC)
	}
}

func TestPropagate_PreservesExistingNodes(t *testing.T) {
	graph := &mockGraphStore{
		entities: map[string]*types.Entity{
			"A": {ID: "A", Name: "NodeA"},
			"B": {ID: "B", Name: "NodeB"},
		},
		relations: map[string][]types.Relation{
			"A": {
				{FromID: "A", ToID: "B", Type: "related"},
			},
		},
	}

	sa := NewSpreadingActivation(nil, graph, nil)
	sa.threshold = 0.01

	activationMap := map[string]float64{
		"A": 1.0,
		"B": 0.5,
	}

	newMap := sa.propagate(context.Background(), activationMap)

	if _, ok := newMap["A"]; !ok {
		t.Error("expected node A to be preserved after propagation")
	}
	if _, ok := newMap["B"]; !ok {
		t.Error("expected node B to be preserved after propagation")
	}

	expectedA := 1.0 * 0.85
	if newMap["A"] < expectedA-0.01 || newMap["A"] > expectedA+0.01 {
		t.Errorf("expected A decayed to ~= %f, got %f", expectedA, newMap["A"])
	}
}

func TestPropagate_DecayApplied(t *testing.T) {
	graph := &mockGraphStore{
		entities:  map[string]*types.Entity{},
		relations: map[string][]types.Relation{},
	}

	sa := NewSpreadingActivation(nil, graph, nil)
	sa.threshold = 0.01
	sa.decayFactor = 0.5

	activationMap := map[string]float64{
		"X": 1.0,
	}

	newMap := sa.propagate(context.Background(), activationMap)

	decay := newMap["X"]
	expected := 1.0 * 0.5
	if decay < expected-0.01 || decay > expected+0.01 {
		t.Errorf("expected decayed value ~= %f, got %f", expected, decay)
	}
}

func TestRankByActivation_Sorted(t *testing.T) {
	graph := &mockGraphStore{
		entities: map[string]*types.Entity{
			"low":    {ID: "low", Name: "LowNode"},
			"medium": {ID: "medium", Name: "MediumNode"},
			"high":   {ID: "high", Name: "HighNode"},
		},
	}

	sa := NewSpreadingActivation(nil, graph, nil)
	sa.threshold = 0.1

	activationMap := map[string]float64{
		"low":    0.2,
		"high":   0.9,
		"medium": 0.5,
	}

	nodes := sa.rankByActivation(context.Background(), activationMap)

	if len(nodes) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(nodes))
	}

	if nodes[0].Score < nodes[1].Score {
		t.Errorf("expected descending order: node[0].Score (%f) >= node[1].Score (%f)", nodes[0].Score, nodes[1].Score)
	}
	if nodes[1].Score < nodes[2].Score {
		t.Errorf("expected descending order: node[1].Score (%f) >= node[2].Score (%f)", nodes[1].Score, nodes[2].Score)
	}

	if nodes[0].ID != "high" {
		t.Errorf("expected highest-scored node first, got %s", nodes[0].ID)
	}
}

func TestPropagate_BelowThreshold_Removed(t *testing.T) {
	graph := &mockGraphStore{
		entities:  map[string]*types.Entity{},
		relations: map[string][]types.Relation{},
	}

	sa := NewSpreadingActivation(nil, graph, nil)
	sa.threshold = 0.5

	activationMap := map[string]float64{
		"weak":   0.3,
		"strong": 1.0,
	}

	newMap := sa.propagate(context.Background(), activationMap)

	if _, ok := newMap["weak"]; ok {
		t.Error("expected weak node to be removed after propagation (below threshold)")
	}
	if _, ok := newMap["strong"]; !ok {
		t.Error("expected strong node to survive propagation")
	}
}