package sync

import (
	"context"
	"fmt"
	"time"

	"agent-memory/internal/memory"
	"agent-memory/internal/memory/types"
)

// Syncer handles periodic synchronization between Neo4j and Qdrant
type Syncer struct {
	memory    *memory.Service
	interval  time.Duration
	batchSize int
	done      chan struct{}
}

func New(m *memory.Service, interval time.Duration, batchSize int) *Syncer {
	return &Syncer{
		memory:    m,
		interval:  interval,
		batchSize: batchSize,
		done:      make(chan struct{}),
	}
}

func (s *Syncer) Start(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	fmt.Printf("syncer: started with interval=%v batch=%d\n", s.interval, s.batchSize)

	for {
		select {
		case <-ctx.Done():
			fmt.Println("syncer: stopped by context")
			return
		case <-s.done:
			fmt.Println("syncer: stopped")
			return
		case <-ticker.C:
			if err := s.runSync(ctx); err != nil {
				fmt.Printf("syncer: error: %v\n", err)
			}
		}
	}
}

func (s *Syncer) Stop() {
	close(s.done)
}

func (s *Syncer) runSync(ctx context.Context) error {
	fmt.Println("syncer: running periodic sync...")

	results, err := s.memory.QueryGraph("MATCH (e:Entity) WHERE e.last_synced IS NULL RETURN e.id AS id LIMIT 100", map[string]interface{}{})
	if err != nil {
		return fmt.Errorf("query unsynced entities: %w", err)
	}

	if len(results) == 0 {
		fmt.Println("syncer: no entities to sync")
		return nil
	}

	entityIDs := make([]string, 0, len(results))
	for _, result := range results {
		if id, ok := result["id"].(string); ok {
			entityIDs = append(entityIDs, id)
		}
	}

	if len(entityIDs) == 0 {
		return nil
	}

	// Convert string IDs to Entity objects
	var entities []types.Entity
	for _, id := range entityIDs {
		entities = append(entities, types.Entity{
			ID:   id,
			Name: "Synced Entity",
			Type: "user",
		})
	}

	fmt.Printf("syncer: syncing %d entities...\n", len(entities))
	if err := s.memory.BatchSyncEntities(entityIDs); err != nil {
		return fmt.Errorf("batch sync: %w", err)
	}

	fmt.Printf("syncer: synced %d entities successfully\n", len(entityIDs))
	return nil
}
