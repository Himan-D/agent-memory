// Package rollback: Neo4j-backed Ledger implementation.
//
// The Neo4j ledger persists rollback entries across process restarts. Each
// entry is stored as a (:RollbackEntry) node keyed by pipeline_run_id
// (uniqueness constraint recommended in production migrations):
//
//   CREATE CONSTRAINT rollback_run_id IF NOT EXISTS
//   FOR (r:RollbackEntry) REQUIRE r.pipeline_run_id IS UNIQUE;
//
// The ledger is the source of truth for ingestion safety: when an
// ingestion pipeline run is committed, the entry's status flips to
// "committed" and the recorded ids are no longer eligible for rollback.
// When a run fails mid-flight, the entry stays "pending" and the operator
// can call RollbackPipelineRun to delete the partial writes.
package rollback

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// graphExec is the minimal graph-store surface used by Neo4jLedger. It
// matches memory.GraphStore.QueryGraph so production code can pass
// *memory.neo4j.Client directly.
type graphExec interface {
	QueryGraph(cypher string, params map[string]interface{}) ([]map[string]interface{}, error)
}

// Neo4jLedger is a Neo4j-backed Ledger.
type Neo4jLedger struct {
	graph graphExec
	now   func() time.Time
}

// NewNeo4jLedger wraps a graphExec.
func NewNeo4jLedger(g graphExec) *Neo4jLedger {
	return &Neo4jLedger{graph: g, now: func() time.Time { return time.Now().UTC() }}
}

// --- Ledger interface ---

func (l *Neo4jLedger) Record(ctx context.Context, entry LedgerEntry) error {
	if l.graph == nil {
		return errors.New("neo4j ledger: graph executor is nil")
	}
	if entry.PipelineRunID == "" {
		return errors.New("rollback: PipelineRunID is required")
	}
	now := l.now()
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = now
	}
	if entry.Status == "" {
		entry.Status = StatusPending
	}

	cypher := `
MERGE (r:RollbackEntry {pipeline_run_id: $pipeline_run_id})
  ON CREATE SET r.node_ids       = $node_ids,
                r.edge_ids       = $edge_ids,
                r.vector_ids     = $vector_ids,
                r.status         = $status,
                r.note           = $note,
                r.created_at     = datetime($created_at),
                r.updated_at     = datetime($updated_at)
  ON MATCH  SET r.node_ids       = $node_ids,
                r.edge_ids       = $edge_ids,
                r.vector_ids     = $vector_ids,
                r.status         = $status,
                r.note           = $note,
                r.updated_at     = datetime($updated_at)
`
	params := map[string]interface{}{
		"pipeline_run_id": entry.PipelineRunID,
		"node_ids":        entry.NodeIDs,
		"edge_ids":        entry.EdgeIDs,
		"vector_ids":      entry.VectorIDs,
		"status":          string(entry.Status),
		"note":            entry.Note,
		"created_at":      entry.CreatedAt.Format(time.RFC3339Nano),
		"updated_at":      now.Format(time.RFC3339Nano),
	}
	if _, err := l.graph.QueryGraph(cypher, params); err != nil {
		return fmt.Errorf("neo4j ledger: record: %w", err)
	}
	return nil
}

func (l *Neo4jLedger) MarkCommitted(ctx context.Context, pipelineRunID string) error {
	return l.updateStatus(ctx, pipelineRunID, StatusCommitted, "")
}

func (l *Neo4jLedger) MarkFailed(ctx context.Context, pipelineRunID string, note string) error {
	return l.updateStatus(ctx, pipelineRunID, StatusFailed, note)
}

func (l *Neo4jLedger) updateStatus(_ context.Context, pipelineRunID string, status Status, note string) error {
	if l.graph == nil {
		return errors.New("neo4j ledger: graph executor is nil")
	}
	cypher := `
MATCH (r:RollbackEntry {pipeline_run_id: $pipeline_run_id})
SET r.status = $status,
    r.note = $note,
    r.updated_at = datetime($updated_at)
RETURN r.pipeline_run_id AS id
`
	rows, err := l.graph.QueryGraph(cypher, map[string]interface{}{
		"pipeline_run_id": pipelineRunID,
		"status":          string(status),
		"note":            note,
		"updated_at":      l.now().Format(time.RFC3339Nano),
	})
	if err != nil {
		return fmt.Errorf("neo4j ledger: update status: %w", err)
	}
	if len(rows) == 0 {
		return fmt.Errorf("neo4j ledger: unknown run %s", pipelineRunID)
	}
	return nil
}

func (l *Neo4jLedger) Get(_ context.Context, pipelineRunID string) (LedgerEntry, error) {
	if l.graph == nil {
		return LedgerEntry{}, errors.New("neo4j ledger: graph executor is nil")
	}
	rows, err := l.graph.QueryGraph(`
MATCH (r:RollbackEntry {pipeline_run_id: $pipeline_run_id})
RETURN r.node_ids AS node_ids, r.edge_ids AS edge_ids,
       r.vector_ids AS vector_ids, r.status AS status,
       r.note AS note, r.created_at AS created_at,
       r.updated_at AS updated_at
`, map[string]interface{}{"pipeline_run_id": pipelineRunID})
	if err != nil {
		return LedgerEntry{}, fmt.Errorf("neo4j ledger: get: %w", err)
	}
	if len(rows) == 0 {
		return LedgerEntry{}, fmt.Errorf("neo4j ledger: unknown run %s", pipelineRunID)
	}
	return entryFromRow(pipelineRunID, rows[0]), nil
}

// Rollback marks the entry as rolled back. Unlike InMemoryLedger, this
// implementation does NOT delete the recorded ids from the graph here --
// the caller supplies a Deleter via RollbackWithDeleter to do that. This
// separation lets the caller compose Neo4j node deletion with Qdrant
// vector deletion as a single logical operation.
func (l *Neo4jLedger) Rollback(_ context.Context, pipelineRunID string) ([]string, error) {
	if l.graph == nil {
		return nil, errors.New("neo4j ledger: graph executor is nil")
	}
	entry, err := l.Get(context.Background(), pipelineRunID)
	if err != nil {
		return nil, err
	}
	if entry.Status == StatusCommitted {
		return nil, fmt.Errorf("neo4j ledger: run %s already committed; cannot rollback", pipelineRunID)
	}
	if entry.Status == StatusRolled {
		return entry.NodeIDs, nil
	}
	if err := l.updateStatus(context.Background(), pipelineRunID, StatusRolled, ""); err != nil {
		return nil, err
	}
	return entry.NodeIDs, nil
}

// entryFromRow converts a Cypher result row into a LedgerEntry.
func entryFromRow(pipelineRunID string, r map[string]interface{}) LedgerEntry {
	e := LedgerEntry{
		PipelineRunID: pipelineRunID,
	}
	if v, ok := r["status"].(string); ok {
		e.Status = Status(v)
	}
	if v, ok := r["note"].(string); ok {
		e.Note = v
	}
	if v, ok := r["node_ids"].([]interface{}); ok {
		e.NodeIDs = toStringSlice(v)
	}
	if v, ok := r["edge_ids"].([]interface{}); ok {
		e.EdgeIDs = toStringSlice(v)
	}
	if v, ok := r["vector_ids"].([]interface{}); ok {
		e.VectorIDs = toStringSlice(v)
	}
	if t, ok := r["created_at"].(time.Time); ok {
		e.CreatedAt = t
	}
	if t, ok := r["updated_at"].(time.Time); ok {
		e.UpdatedAt = t
	}
	return e
}

func toStringSlice(in []interface{}) []string {
	out := make([]string, 0, len(in))
	for _, v := range in {
		if s, ok := v.(string); ok {
			out = append(out, s)
		}
	}
	return out
}
