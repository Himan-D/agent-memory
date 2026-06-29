package neo4j

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type fakeGraph struct {
	cyphers []string
	params  []map[string]interface{}
	rows    []map[string]interface{}
	err     error
}

func (f *fakeGraph) QueryGraph(cypher string, params map[string]interface{}) ([]map[string]interface{}, error) {
	f.cyphers = append(f.cyphers, cypher)
	f.params = append(f.params, params)
	if f.err != nil {
		return nil, f.err
	}
	return f.rows, nil
}

type fakeVectorWriter struct {
	calls    int
	payloads []VectorPayload
	err      error
}

func (f *fakeVectorWriter) WriteBatch(ctx context.Context, payloads []VectorPayload) error {
	f.calls++
	f.payloads = append(f.payloads, payloads...)
	return f.err
}

type fakeExtractor struct {
	prefix string
	err    error
}

func (f *fakeExtractor) Extract(ctx context.Context, item map[string]interface{}) (VectorPayload, error) {
	if f.err != nil {
		return VectorPayload{}, f.err
	}
	return VectorPayload{
		ID:     f.prefix + ":" + item["id"].(string),
		Vector: []float32{0.1, 0.2, 0.3},
		Metadata: map[string]interface{}{
			"source": item["id"],
		},
	}, nil
}

func TestHybridWriter_DisabledReturnsErr(t *testing.T) {
	h := NewHybridWriter(HybridWriteConfig{Enabled: false, Graph: &fakeGraph{}, VectorWriter: &fakeVectorWriter{}})
	if _, err := h.AddNodesWithVectors(context.Background(), []map[string]interface{}{{"id": "n1"}}); !errors.Is(err, ErrHybridDisabled) {
		t.Fatalf("expected ErrHybridDisabled, got %v", err)
	}
}

func TestHybridWriter_NilGraphErrors(t *testing.T) {
	h := NewHybridWriter(HybridWriteConfig{Enabled: true, Graph: nil, VectorWriter: &fakeVectorWriter{}})
	if _, err := h.AddNodesWithVectors(context.Background(), []map[string]interface{}{{"id": "n1"}}); err == nil {
		t.Fatal("expected error for nil graph")
	}
}

func TestHybridWriter_NilVectorWriterErrors(t *testing.T) {
	fg := &fakeGraph{}
	h := NewHybridWriter(HybridWriteConfig{Enabled: true, Graph: fg, VectorWriter: nil, NodePayloadExtractor: &fakeExtractor{}})
	if _, err := h.AddNodesWithVectors(context.Background(), []map[string]interface{}{{"id": "n1"}}); err == nil {
		t.Fatal("expected error for nil vector writer")
	}
}

func TestHybridWriter_AddNodesHappyPath(t *testing.T) {
	fg := &fakeGraph{}
	fv := &fakeVectorWriter{}
	h := NewHybridWriter(HybridWriteConfig{
		Enabled:              true,
		Graph:                fg,
		VectorWriter:         fv,
		NodePayloadExtractor: &fakeExtractor{prefix: "node"},
	})
	nodes := []map[string]interface{}{
		{"id": "n1", "label": "Memory", "content": "x"},
		{"id": "n2", "label": "Memory", "content": "y"},
	}
	ids, err := h.AddNodesWithVectors(context.Background(), nodes)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 2 || ids[0] != "n1" || ids[1] != "n2" {
		t.Fatalf("unexpected ids: %v", ids)
	}
	if len(fg.cyphers) != 2 {
		t.Fatalf("expected 2 cypher calls (one per node), got %d", len(fg.cyphers))
	}
	if fv.calls != 1 {
		t.Fatalf("expected 1 vector batch call, got %d", fv.calls)
	}
	if len(fv.payloads) != 2 {
		t.Fatalf("expected 2 payloads, got %d", len(fv.payloads))
	}
}

func TestHybridWriter_AddNodesGraphErrorPropagates(t *testing.T) {
	fg := &fakeGraph{err: errors.New("neo4j down")}
	fv := &fakeVectorWriter{}
	h := NewHybridWriter(HybridWriteConfig{
		Enabled:              true,
		Graph:                fg,
		VectorWriter:         fv,
		NodePayloadExtractor: &fakeExtractor{},
	})
	if _, err := h.AddNodesWithVectors(context.Background(), []map[string]interface{}{{"id": "n1"}}); err == nil {
		t.Fatal("expected error from graph failure")
	}
	if fv.calls != 0 {
		t.Fatal("vector writer should not be called when graph write fails")
	}
}

func TestHybridWriter_AddNodesVectorErrorReturnsGraphIDs(t *testing.T) {
	fg := &fakeGraph{}
	fv := &fakeVectorWriter{err: errors.New("qdrant down")}
	h := NewHybridWriter(HybridWriteConfig{
		Enabled:              true,
		Graph:                fg,
		VectorWriter:         fv,
		NodePayloadExtractor: &fakeExtractor{},
	})
	_, err := h.AddNodesWithVectors(context.Background(), []map[string]interface{}{
		{"id": "n1"},
		{"id": "n2"},
	})
	if err == nil {
		t.Fatal("expected error from vector failure")
	}
	if !strings.Contains(err.Error(), "n1") || !strings.Contains(err.Error(), "n2") {
		t.Fatalf("error should include graph ids for rollback, got %v", err)
	}
}

func TestHybridWriter_AddEdgesHappyPath(t *testing.T) {
	fg := &fakeGraph{}
	fv := &fakeVectorWriter{}
	h := NewHybridWriter(HybridWriteConfig{
		Enabled:              true,
		Graph:                fg,
		VectorWriter:         fv,
		EdgePayloadExtractor: &fakeExtractor{prefix: "edge"},
	})
	edges := []map[string]interface{}{
		{"id": "e1", "from": "n1", "to": "n2", "type": "KNOWS"},
	}
	ids, err := h.AddEdgesWithVectors(context.Background(), edges)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 1 || ids[0] != "e1" {
		t.Fatalf("unexpected ids: %v", ids)
	}
	if len(fg.cyphers) != 1 {
		t.Fatalf("expected 1 cypher, got %d", len(fg.cyphers))
	}
	c := fg.cyphers[0]
	if !strings.Contains(c, "MERGE (a)-[r:KNOWS") {
		t.Fatalf("expected relationship type KNOWS in cypher: %s", c)
	}
	if !strings.Contains(c, "MATCH (a {id: $from})") {
		t.Fatalf("expected MATCH by from id: %s", c)
	}
}

func TestHybridWriter_AddEdgesMissingFromTo(t *testing.T) {
	fg := &fakeGraph{}
	fv := &fakeVectorWriter{}
	h := NewHybridWriter(HybridWriteConfig{
		Enabled:              true,
		Graph:                fg,
		VectorWriter:         fv,
		EdgePayloadExtractor: &fakeExtractor{},
	})
	_, err := h.AddEdgesWithVectors(context.Background(), []map[string]interface{}{
		{"id": "e1", "from": "n1"}, // missing "to"
	})
	if err == nil {
		t.Fatal("expected error for missing to")
	}
}

func TestHybridWriter_AddEdgesDefaultRelType(t *testing.T) {
	fg := &fakeGraph{}
	fv := &fakeVectorWriter{}
	h := NewHybridWriter(HybridWriteConfig{
		Enabled:              true,
		Graph:                fg,
		VectorWriter:         fv,
		EdgePayloadExtractor: &fakeExtractor{},
	})
	_, err := h.AddEdgesWithVectors(context.Background(), []map[string]interface{}{
		{"id": "e1", "from": "n1", "to": "n2"}, // no type
	})
	if err != nil {
		t.Fatal(err)
	}
	c := fg.cyphers[0]
	if !strings.Contains(c, "MERGE (a)-[r:RELATED") {
		t.Fatalf("expected default rel type RELATED, got %s", c)
	}
}

func TestHybridWriter_AddNodesEmptyList(t *testing.T) {
	fg := &fakeGraph{}
	fv := &fakeVectorWriter{}
	h := NewHybridWriter(HybridWriteConfig{
		Enabled:              true,
		Graph:                fg,
		VectorWriter:         fv,
		NodePayloadExtractor: &fakeExtractor{},
	})
	ids, err := h.AddNodesWithVectors(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 0 {
		t.Fatalf("expected empty ids for empty input, got %v", ids)
	}
	if len(fg.cyphers) != 0 {
		t.Fatalf("expected no cypher calls for empty input, got %d", len(fg.cyphers))
	}
}

func TestHybridWriter_AddNodesMissingID(t *testing.T) {
	fg := &fakeGraph{}
	fv := &fakeVectorWriter{}
	h := NewHybridWriter(HybridWriteConfig{
		Enabled:              true,
		Graph:                fg,
		VectorWriter:         fv,
		NodePayloadExtractor: &fakeExtractor{},
	})
	_, err := h.AddNodesWithVectors(context.Background(), []map[string]interface{}{
		{"label": "Memory"},
	})
	if err == nil {
		t.Fatal("expected error for missing id")
	}
}

func TestHybridWriter_ExtractorError(t *testing.T) {
	fg := &fakeGraph{}
	fv := &fakeVectorWriter{}
	h := NewHybridWriter(HybridWriteConfig{
		Enabled:              true,
		Graph:                fg,
		VectorWriter:         fv,
		NodePayloadExtractor: &fakeExtractor{err: errors.New("extract fail")},
	})
	if _, err := h.AddNodesWithVectors(context.Background(), []map[string]interface{}{{"id": "n1"}}); err == nil {
		t.Fatal("expected error from extractor")
	}
}
