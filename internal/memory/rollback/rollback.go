// Package rollback provides a ledger for safe, recoverable ingestion. It is
// modeled after Cognee's upsert_nodes/edges + relational rollback ledger
// pattern: before mutating the graph or vector store, callers record the ids
// they intend to touch; on failure, RollbackPipelineRun deletes the recorded
// ids.
//
// This package is purely additive. It does not modify the existing memory
// service or any call sites. The default InMemoryLedger is suitable for
// tests and single-process deployments; production deployments can plug in
// a persistent implementation (Postgres, Neo4j) without changing call
// sites.
package rollback

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// LedgerEntry records one planned mutation. The NodeIDs and EdgeIDs are the
// ids the caller intends to write; VectorIDs are the corresponding vector
// store ids.
type LedgerEntry struct {
	PipelineRunID string
	NodeIDs       []string
	EdgeIDs       []string
	VectorIDs     []string
	Status        Status
	CreatedAt     time.Time
	UpdatedAt     time.Time
	Note          string
}

// Status is the lifecycle state of a ledger entry.
type Status string

const (
	StatusPending   Status = "pending"
	StatusCommitted Status = "committed"
	StatusRolled    Status = "rolled_back"
	StatusFailed    Status = "failed"
)

// Ledger is the interface for a rollback ledger.
type Ledger interface {
	Record(ctx context.Context, entry LedgerEntry) error
	MarkCommitted(ctx context.Context, pipelineRunID string) error
	MarkFailed(ctx context.Context, pipelineRunID string, note string) error
	Get(ctx context.Context, pipelineRunID string) (LedgerEntry, error)
	Rollback(ctx context.Context, pipelineRunID string) ([]string, error)
}

// Deleter is the callback used by Rollback to remove graph + vector ids. It
// is supplied by the caller (typically the memory service) so the rollback
// package stays storage-agnostic.
type Deleter func(ctx context.Context, nodeIDs, edgeIDs, vectorIDs []string) error

// InMemoryLedger is a goroutine-safe Ledger suitable for tests and CLI use.
type InMemoryLedger struct {
	mu      sync.RWMutex
	entries map[string]LedgerEntry
}

// NewInMemoryLedger returns an empty ledger.
func NewInMemoryLedger() *InMemoryLedger {
	return &InMemoryLedger{entries: make(map[string]LedgerEntry)}
}

func (l *InMemoryLedger) Record(_ context.Context, entry LedgerEntry) error {
	if entry.PipelineRunID == "" {
		return errors.New("rollback: PipelineRunID is required")
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now().UTC()
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = now
	}
	entry.UpdatedAt = now
	if entry.Status == "" {
		entry.Status = StatusPending
	}
	l.entries[entry.PipelineRunID] = entry
	return nil
}

func (l *InMemoryLedger) MarkCommitted(_ context.Context, pipelineRunID string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	e, ok := l.entries[pipelineRunID]
	if !ok {
		return fmt.Errorf("rollback: unknown run %s", pipelineRunID)
	}
	e.Status = StatusCommitted
	e.UpdatedAt = time.Now().UTC()
	l.entries[pipelineRunID] = e
	return nil
}

func (l *InMemoryLedger) MarkFailed(_ context.Context, pipelineRunID string, note string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	e, ok := l.entries[pipelineRunID]
	if !ok {
		return fmt.Errorf("rollback: unknown run %s", pipelineRunID)
	}
	e.Status = StatusFailed
	e.Note = note
	e.UpdatedAt = time.Now().UTC()
	l.entries[pipelineRunID] = e
	return nil
}

func (l *InMemoryLedger) Get(_ context.Context, pipelineRunID string) (LedgerEntry, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	e, ok := l.entries[pipelineRunID]
	if !ok {
		return LedgerEntry{}, fmt.Errorf("rollback: unknown run %s", pipelineRunID)
	}
	return e, nil
}

func (l *InMemoryLedger) Rollback(ctx context.Context, pipelineRunID string) ([]string, error) {
	l.mu.Lock()
	e, ok := l.entries[pipelineRunID]
	if !ok {
		l.mu.Unlock()
		return nil, fmt.Errorf("rollback: unknown run %s", pipelineRunID)
	}
	if e.Status == StatusCommitted {
		l.mu.Unlock()
		return nil, fmt.Errorf("rollback: run %s already committed; cannot rollback", pipelineRunID)
	}
	if e.Status == StatusRolled {
		l.mu.Unlock()
		return e.NodeIDs, nil
	}
	e.Status = StatusRolled
	e.UpdatedAt = time.Now().UTC()
	l.entries[pipelineRunID] = e
	l.mu.Unlock()
	return e.NodeIDs, nil
}

// RollbackWithDeleter records the entry, calls the supplied deleter to
// remove the recorded ids, and marks the entry as rolled back. The deleter
// may be nil for dry-run rollback that just updates the ledger.
func RollbackWithDeleter(ctx context.Context, l Ledger, pipelineRunID string, del Deleter) error {
	e, err := l.Get(ctx, pipelineRunID)
	if err != nil {
		return err
	}
	if del != nil {
		if err := del(ctx, e.NodeIDs, e.EdgeIDs, e.VectorIDs); err != nil {
			// Mark failed and surface the error so callers can retry.
			_ = l.MarkFailed(ctx, pipelineRunID, err.Error())
			return fmt.Errorf("rollback: deleter: %w", err)
		}
	}
	if _, err := l.Rollback(ctx, pipelineRunID); err != nil {
		return err
	}
	return nil
}
