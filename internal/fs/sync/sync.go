package sync

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"agent-memory/internal/fs/vfs"
	"agent-memory/internal/memory/types"
)

// SyncEngine manages background sync between filesystem and memory service
// Pattern: follows sync/syncer.go
type SyncEngine struct {
	svc      vfs.ServiceInterface
	cache    vfs.CacheInterface
	opts     *SyncOptions
	mu       sync.RWMutex
	stopChan chan struct{}
	wg       sync.WaitGroup
	stats    SyncStats
}

// SyncOptions configures the sync engine
type SyncOptions struct {
	PullInterval time.Duration // Default: 30s
	PushInterval time.Duration // Default: 5s
	ScanInterval time.Duration // Default: 5min
	MaxRetries   int           // Default: 3
	RetryBackoff time.Duration // Default: 1s
}

// DefaultSyncOptions returns sensible defaults
func DefaultSyncOptions() *SyncOptions {
	return &SyncOptions{
		PullInterval: 30 * time.Second,
		PushInterval: 5 * time.Second,
		ScanInterval: 5 * time.Minute,
		MaxRetries:   3,
		RetryBackoff: 1 * time.Second,
	}
}

// SyncStats tracks sync statistics
type SyncStats struct {
	PullCount    int64
	PushCount    int64
	ScanCount    int64
	ErrorCount   int64
	LastPullTime time.Time
	LastPushTime time.Time
	LastScanTime time.Time
}

// NewSyncEngine creates a new sync engine
// Pattern: like NewSyncer() in sync/syncer.go
func NewSyncEngine(svc vfs.ServiceInterface, cache vfs.CacheInterface, opts *SyncOptions) *SyncEngine {
	if opts == nil {
		opts = DefaultSyncOptions()
	}
	return &SyncEngine{
		svc:      svc,
		cache:    cache,
		opts:     opts,
		stopChan: make(chan struct{}),
	}
}

// Start begins all background sync loops
// Pattern: like Syncer.Start() in sync/syncer.go
func (e *SyncEngine) Start(ctx context.Context) {
	log.Println("Sync engine starting...")

	// Loop A: Delta pull (every 30s)
	e.wg.Add(1)
	go e.pullLoop(ctx)

	// Loop B: Push worker (coalesces rapid writes)
	e.wg.Add(1)
	go e.pushLoop(ctx)

	// Loop C: Deletion scan (every 5min)
	e.wg.Add(1)
	go e.scanLoop(ctx)

	log.Println("Sync engine started")
}

// Stop gracefully stops all sync loops
func (e *SyncEngine) Stop() {
	log.Println("Sync engine stopping...")
	close(e.stopChan)
	e.wg.Wait()
	log.Println("Sync engine stopped")
}

// ---- Loop A: Delta Pull ----

// pullLoop pulls changes from the service periodically
// Pattern: like runDeltaLoop() in smfs
func (e *SyncEngine) pullLoop(ctx context.Context) {
	defer e.wg.Done()

	ticker := time.NewTicker(e.opts.PullInterval)
	defer ticker.Stop()

	emptyStreak := 0

	for {
		select {
		case <-e.stopChan:
			return
		case <-ticker.C:
			n := e.pullDelta(ctx)
			e.mu.Lock()
			e.stats.PullCount++
			e.stats.LastPullTime = time.Now()
			e.mu.Unlock()

			if n == 0 {
				emptyStreak++
			} else {
				emptyStreak = 0
			}

			// Adaptive interval based on empty streaks
			if emptyStreak > 10 {
				// Slow down to 5min if nothing new
				ticker.Reset(5 * time.Minute)
			} else if emptyStreak == 0 {
				// Speed up to 10s if active
				ticker.Reset(10 * time.Second)
			}
		}
	}
}

// pullDelta fetches recent memories/entities/skills
func (e *SyncEngine) pullDelta(ctx context.Context) int {
	// Pull recent memories
	results, err := e.svc.SearchMemories(ctx, &types.SearchRequest{
		Limit: 100,
	})
	if err != nil {
		log.Printf("Pull error: %v", err)
		e.mu.Lock()
		e.stats.ErrorCount++
		e.mu.Unlock()
		return 0
	}

	// Cache recent results
	for _, r := range results {
		key := fmt.Sprintf("mem:%s", r.MemoryID)
		e.cache.Set(key, r)
	}

	return len(results)
}

// ---- Loop B: Push Worker ----

// pushLoop processes pending writes
// Pattern: like pushWorker() in smfs
func (e *SyncEngine) pushLoop(ctx context.Context) {
	defer e.wg.Done()

	ticker := time.NewTicker(e.opts.PushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-e.stopChan:
			return
		case <-ticker.C:
			e.pushPending(ctx)
		}
	}
}

// pushPending sends pending writes to the service
func (e *SyncEngine) pushPending(ctx context.Context) {
	// In production: implement a push queue in filesystem.go
	// For now, just track stats
	e.mu.Lock()
	e.stats.PushCount++
	e.stats.LastPushTime = time.Now()
	e.mu.Unlock()
}

// ---- Loop C: Deletion Scan ----

// scanLoop scans for deleted items periodically
func (e *SyncEngine) scanLoop(ctx context.Context) {
	defer e.wg.Done()

	ticker := time.NewTicker(e.opts.ScanInterval)
	defer ticker.Stop()

	for {
		select {
		case <-e.stopChan:
			return
		case <-ticker.C:
			e.scanDeletions(ctx)
		}
	}
}

// scanDeletions removes items that no longer exist in the service
func (e *SyncEngine) scanDeletions(ctx context.Context) {
	// In production: compare local cache with service state
	// Delete cached items that no longer exist
	e.mu.Lock()
	e.stats.ScanCount++
	e.stats.LastScanTime = time.Now()
	e.mu.Unlock()
}

// ---- Stats ----

// GetStats returns current sync statistics
func (e *SyncEngine) GetStats() SyncStats {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.stats
}
