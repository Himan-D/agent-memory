package tier

import (
	"context"
	"log"
	"time"

	"agent-memory/internal/memory/types"
)

// MemoryFetcher provides paginated access to memories for the migration job.
type MemoryFetcher interface {
	GetAllMemoriesPaginated(ctx context.Context, limit, offset int) ([]*types.Memory, error)
	UpdateMemoryTier(ctx context.Context, memoryID string, tier string) error
}

// Migrator runs a periodic tier migration loop, promoting and demoting memories
// between Working, Hot, Cold, and Archive tiers based on access patterns and age.
type Migrator struct {
	router   *MemoryRouter
	fetcher  MemoryFetcher
	interval time.Duration
	batch    int
}

// NewMigrator creates a Migrator. interval controls how often the scan runs;
// reasonable starting values are 15m–60m in production.
func NewMigrator(router *MemoryRouter, fetcher MemoryFetcher, interval time.Duration) *Migrator {
	return &Migrator{
		router:   router,
		fetcher:  fetcher,
		interval: interval,
		batch:    100,
	}
}

// Run starts the migration loop. It runs once immediately on start, then on
// each tick. Blocks until ctx is cancelled, returning ctx.Err().
func (m *Migrator) Run(ctx context.Context) error {
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	// Run once immediately so we don't wait a full interval on startup.
	m.migrate(ctx)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			m.migrate(ctx)
		}
	}
}

// migrate performs one full scan of all memories and moves those whose
// determined tier differs from their current stored tier.
func (m *Migrator) migrate(ctx context.Context) {
	offset := 0
	moved := 0

	for {
		memories, err := m.fetcher.GetAllMemoriesPaginated(ctx, m.batch, offset)
		if err != nil {
			log.Printf("tier-migrator: fetch page (offset=%d) error: %v", offset, err)
			break
		}
		if len(memories) == 0 {
			break
		}

		for _, mem := range memories {
			currentTier := mem.Tier
			if currentTier == "" {
				currentTier = string(TierWorking)
			}

			targetTier, err := m.router.DetermineTier(ctx, mem)
			if err != nil {
				log.Printf("tier-migrator: determine tier for %s: %v", mem.ID, err)
				continue
			}

			if string(targetTier) == currentTier {
				continue
			}

			// Execute tier transition side-effects before updating the record.
			switch targetTier {
			case TierHot:
				// Promote: write into the hot cache so future Exists() calls hit.
				if cs := m.router.CacheStore(); cs != nil {
					if setErr := cs.Set(ctx, "hot:"+mem.ID, mem.Content, m.router.config.HotTTL()); setErr != nil {
						log.Printf("tier-migrator: hot cache set %s: %v", mem.ID, setErr)
					}
				}

			case TierCold:
				// Demote from hot: evict the hot cache key.
				if err := m.router.MigrateToCold(ctx, []string{mem.ID}); err != nil {
					log.Printf("tier-migrator: cold migration %s: %v", mem.ID, err)
					continue
				}

			case TierArchive:
				// Demote to archive: evict cache keys and write to archive store.
				if err := m.router.MigrateToArchive(ctx, []string{mem.ID}); err != nil {
					log.Printf("tier-migrator: archive migration %s: %v", mem.ID, err)
					continue
				}
			}

			// Persist the new tier on the memory record.
			if err := m.fetcher.UpdateMemoryTier(ctx, mem.ID, string(targetTier)); err != nil {
				log.Printf("tier-migrator: update tier %s → %s: %v", mem.ID, targetTier, err)
				continue
			}

			moved++
		}

		offset += len(memories)
		if len(memories) < m.batch {
			break // last page
		}
	}

	if moved > 0 {
		log.Printf("tier-migrator: migrated %d memories", moved)
	}
}
