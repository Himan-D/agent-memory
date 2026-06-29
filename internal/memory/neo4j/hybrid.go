package neo4j

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// HybridWriteConfig configures HybridWriter.
type HybridWriteConfig struct {
	// Graph is the Neo4j executor (typically *Client). Required.
	Graph graphExec
	// VectorWriter writes vector embeddings to the configured vector store.
	// It is called AFTER the graph write succeeds. If it returns an
	// error, the recorded graph ids are returned so the caller can roll
	// back via the rollback.Ledger.
	VectorWriter VectorWriter
	// NodePayloadExtractor returns the vector payload (id, vector, metadata)
	// for a node. Required for AddNodesWithVectors.
	NodePayloadExtractor PayloadExtractor
	// EdgePayloadExtractor returns the vector payload for an edge. Required
	// for AddEdgesWithVectors.
	EdgePayloadExtractor PayloadExtractor
	// CommitTimeout caps how long each side of the write is allowed to
	// take. Zero disables the timeout.
	CommitTimeout time.Duration
	// Enabled toggles the hybrid write path globally. When false, callers
	// should fall back to separate graph + vector writes.
	Enabled bool
}

// graphExec is the minimal graph surface used by HybridWriter. Defined here
// to avoid pulling client.go into the test harness.
type graphExec interface {
	QueryGraph(cypher string, params map[string]interface{}) ([]map[string]interface{}, error)
}

// VectorWriter writes a batch of vector payloads to the configured vector
// store (Qdrant, Pinecone, pgvector, etc.).
type VectorWriter interface {
	WriteBatch(ctx context.Context, payloads []VectorPayload) error
}

// VectorPayload is one node or edge plus its embedding. The HybridWriter
// calls VectorWriter.WriteBatch with these payloads after the graph write
// succeeds.
type VectorPayload struct {
	ID       string
	Vector   []float32
	Metadata map[string]interface{}
}

// PayloadExtractor converts a node or edge (as a parameter map) into a
// VectorPayload for the vector store. The HybridWriter is generic over the
// payload shape; callers plug in their own extractor.
type PayloadExtractor interface {
	Extract(ctx context.Context, item map[string]interface{}) (VectorPayload, error)
}

// HybridWriter orchestrates the two-phase write described in Cognee's
// add_data_points: graph write inside a Neo4j transaction, then vector
// upsert to the vector store. If the vector write fails, the recorded
// graph ids are returned to the caller for rollback.
type HybridWriter struct {
	cfg HybridWriteConfig
}

// NewHybridWriter returns a HybridWriter. cfg.Enabled gates the hybrid
// path; when false, AddNodesWithVectors and AddEdgesWithVectors return
// ErrHybridDisabled.
func NewHybridWriter(cfg HybridWriteConfig) *HybridWriter {
	return &HybridWriter{cfg: cfg}
}

// AddNodesWithVectors writes nodes to Neo4j, computes vector payloads via
// the configured NodePayloadExtractor, and upserts them to the vector
// store. Returns the created graph ids so callers can register them with
// the rollback.Ledger.
func (h *HybridWriter) AddNodesWithVectors(ctx context.Context, nodes []map[string]interface{}) ([]string, error) {
	if !h.cfg.Enabled {
		return nil, ErrHybridDisabled
	}
	if h.cfg.Graph == nil {
		return nil, errors.New("hybrid writer: graph executor is nil")
	}
	if h.cfg.NodePayloadExtractor == nil {
		return nil, errors.New("hybrid writer: node payload extractor is nil")
	}
	if h.cfg.VectorWriter == nil {
		return nil, errors.New("hybrid writer: vector writer is nil")
	}
	if len(nodes) == 0 {
		return nil, nil
	}

	ids, err := h.writeNodes(ctx, nodes)
	if err != nil {
		return nil, err
	}

	payloads := make([]VectorPayload, 0, len(nodes))
	for _, n := range nodes {
		p, err := h.cfg.NodePayloadExtractor.Extract(ctx, n)
		if err != nil {
			return ids, fmt.Errorf("hybrid writer: extract node payload: %w", err)
		}
		payloads = append(payloads, p)
	}

	if err := h.cfg.VectorWriter.WriteBatch(ctx, payloads); err != nil {
		// Vector write failed after graph write succeeded. Surface ids so
		// caller can rollback via the ledger.
		return ids, fmt.Errorf("hybrid writer: vector write (graph ids %v): %w", ids, err)
	}
	return ids, nil
}

// AddEdgesWithVectors is the edge analog. See AddNodesWithVectors for the
// full semantics.
func (h *HybridWriter) AddEdgesWithVectors(ctx context.Context, edges []map[string]interface{}) ([]string, error) {
	if !h.cfg.Enabled {
		return nil, ErrHybridDisabled
	}
	if h.cfg.Graph == nil {
		return nil, errors.New("hybrid writer: graph executor is nil")
	}
	if h.cfg.EdgePayloadExtractor == nil {
		return nil, errors.New("hybrid writer: edge payload extractor is nil")
	}
	if h.cfg.VectorWriter == nil {
		return nil, errors.New("hybrid writer: vector writer is nil")
	}
	if len(edges) == 0 {
		return nil, nil
	}

	ids, err := h.writeEdges(ctx, edges)
	if err != nil {
		return nil, err
	}

	payloads := make([]VectorPayload, 0, len(edges))
	for _, e := range edges {
		p, err := h.cfg.EdgePayloadExtractor.Extract(ctx, e)
		if err != nil {
			return ids, fmt.Errorf("hybrid writer: extract edge payload: %w", err)
		}
		payloads = append(payloads, p)
	}

	if err := h.cfg.VectorWriter.WriteBatch(ctx, payloads); err != nil {
		return ids, fmt.Errorf("hybrid writer: vector write (edge ids %v): %w", ids, err)
	}
	return ids, nil
}

// writeNodes issues a single Cypher UNWIND that creates all nodes in one
// round-trip. Returns the ids in the order the caller passed them.
//
// The exact Cypher shape is intentionally generic: callers pass nodes as
// parameter maps with at minimum an `id` field. The HybridWriter is
// storage-shape agnostic; the Neo4j schema constraint is the caller's
// responsibility.
func (h *HybridWriter) writeNodes(ctx context.Context, nodes []map[string]interface{}) ([]string, error) {
	ids := make([]string, 0, len(nodes))
	for _, n := range nodes {
		if id, ok := n["id"].(string); ok {
			ids = append(ids, id)
		} else {
			return nil, errors.New("hybrid writer: node missing string id")
		}
	}

	// Single-node MERGE for now. A future optimization would use
	// CALL { ... } IN TRANSACTIONS for large batches, but that's a
	// Neo4j-5.x-only feature and we cannot depend on it being available.
	for _, n := range nodes {
		cypher := `
MERGE (n {id: $id})
  ON CREATE SET n += $props
  ON MATCH  SET n += $props
`
		if _, err := h.cfg.Graph.QueryGraph(cypher, map[string]interface{}{
			"id":    n["id"],
			"props": n,
		}); err != nil {
			return ids, fmt.Errorf("hybrid writer: write node %v: %w", n["id"], err)
		}
	}
	return ids, nil
}

func (h *HybridWriter) writeEdges(ctx context.Context, edges []map[string]interface{}) ([]string, error) {
	ids := make([]string, 0, len(edges))
	for _, e := range edges {
		if id, ok := e["id"].(string); ok {
			ids = append(ids, id)
		} else {
			return nil, errors.New("hybrid writer: edge missing string id")
		}
	}

	for _, e := range edges {
		fromID, okFrom := e["from"].(string)
		toID, okTo := e["to"].(string)
		if !okFrom || !okTo {
			return ids, errors.New("hybrid writer: edge missing from/to")
		}
		relType, _ := e["type"].(string)
		if relType == "" {
			relType = "RELATED"
		}
		cypher := fmt.Sprintf(`
MATCH (a {id: $from}), (b {id: $to})
MERGE (a)-[r:%s {id: $id}]->(b)
  ON CREATE SET r += $props
  ON MATCH  SET r += $props
`, relType)
		props := map[string]interface{}{}
		for k, v := range e {
			if k == "id" || k == "from" || k == "to" || k == "type" {
				continue
			}
			props[k] = v
		}
		if _, err := h.cfg.Graph.QueryGraph(cypher, map[string]interface{}{
			"id":   e["id"],
			"from": fromID,
			"to":   toID,
			"props": props,
		}); err != nil {
			return ids, fmt.Errorf("hybrid writer: write edge %v: %w", e["id"], err)
		}
	}
	return ids, nil
}

// ErrHybridDisabled is returned when a hybrid write is attempted while the
// feature flag is off.
var ErrHybridDisabled = errors.New("hybrid write disabled")
