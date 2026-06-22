package tier

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"agent-memory/internal/memory/types"
)

type MemoryTier string

const (
	TierWorking MemoryTier = "working"
	TierHot     MemoryTier = "hot"
	TierCold    MemoryTier = "cold"
	TierArchive MemoryTier = "archive"
)

type TierPolicy string

const (
	TierPolicyAggressive   TierPolicy = "aggressive"
	TierPolicyBalanced     TierPolicy = "balanced"
	TierPolicyConservative TierPolicy = "conservative"
)

type MemoryTierConfig struct {
	WorkingMaxTokens int
	HotMaxTokens     int
	HotRetentionDays int
	ArchiveThreshold int
}

type MemoryRouter struct {
	config       *TierConfig
	vectorStore  VectorStore
	cacheStore   CacheStore
	archiveStore ArchiveStore
}

type TierConfig struct {
	Policy           TierPolicy
	WorkingMaxTokens int `env:"tier_working_max_tokens" envDefault:"4096"`
	HotMaxTokens     int `env:"tier_hot_tokens" envDefault:"32768"`
	HotRetentionDays int `env:"tier_hot_retention_days" envDefault:"7"`
	ArchiveThreshold int `env:"tier_archive_threshold" envDefault:"100"`
}

type VectorStore interface {
	Search(ctx context.Context, query []float32, limit int, threshold float32, filters map[string]interface{}) ([]types.MemoryResult, error)
}

type CacheStore interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value string, ttl time.Duration) error
	Del(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
}

func NewMemoryRouter(cfg *TierConfig) *MemoryRouter {
	if cfg == nil {
		cfg = &TierConfig{
			Policy:           TierPolicyBalanced,
			WorkingMaxTokens: 4096,
			HotMaxTokens:     32768,
			HotRetentionDays: 7,
		}
	}

	return &MemoryRouter{
		config: cfg,
	}
}

func (r *MemoryRouter) SetVectorStore(store VectorStore) {
	r.vectorStore = store
}

func (r *MemoryRouter) SetCacheStore(store CacheStore) {
	r.cacheStore = store
}

func (r *MemoryRouter) SetArchiveStore(store ArchiveStore) {
	r.archiveStore = store
}

// CacheStore returns the underlying cache store (nil when not configured).
func (r *MemoryRouter) CacheStore() CacheStore {
	return r.cacheStore
}

func (r *MemoryRouter) DetermineTier(ctx context.Context, memory *types.Memory) (MemoryTier, error) {
	tokenCount := estimateTokens(memory.Content)

	// Fast-path: memory was previously archived.
	if r.cacheStore != nil {
		if archived, err := r.cacheStore.Exists(ctx, fmt.Sprintf("archive:%s", memory.ID)); err == nil && archived {
			return TierArchive, nil
		}
	}

	allowedTiers := r.GetTierKeys(r.config.Policy)

	if _, ok := allowedTiers[TierWorking]; ok && tokenCount <= r.config.WorkingMaxTokens {
		return TierWorking, nil
	}

	if _, ok := allowedTiers[TierHot]; ok {
		if r.cacheStore != nil {
			exists, err := r.cacheStore.Exists(ctx, fmt.Sprintf("hot:%s", memory.ID))
			if err == nil && exists {
				return TierHot, nil
			}
		}

		if r.config.isRecent(memory) && tokenCount <= r.config.HotMaxTokens {
			if r.cacheStore != nil {
				r.cacheStore.Set(ctx, fmt.Sprintf("hot:%s", memory.ID), memory.Content, r.config.HotTTL())
			}
			return TierHot, nil
		}
	}

	if _, ok := allowedTiers[TierArchive]; ok {
		if r.config.ArchiveThreshold > 0 {
			archiveCutoff := time.Duration(r.config.ArchiveThreshold) * 24 * time.Hour
			if time.Since(memory.UpdatedAt) > archiveCutoff {
				return TierArchive, nil
			}
		}
	}

	if _, ok := allowedTiers[TierCold]; ok {
		return TierCold, nil
	}

	return TierHot, nil
}

func (r *MemoryRouter) SetTierPolicy(policy TierPolicy) {
	r.config.Policy = policy
}

func (r *MemoryRouter) GetTierPolicy() TierPolicy {
	return r.config.Policy
}

func (r *TierConfig) isRecent(memory *types.Memory) bool {
	recentDuration := time.Duration(r.HotRetentionDays) * 24 * time.Hour
	return time.Since(memory.UpdatedAt) < recentDuration
}

func (r *TierConfig) HotTTL() time.Duration {
	return time.Duration(r.HotRetentionDays) * 24 * time.Hour
}

func (r *MemoryRouter) GetTierKeys(policy TierPolicy) map[MemoryTier]struct{} {
	keys := make(map[MemoryTier]struct{})

	switch policy {
	case TierPolicyAggressive:
		keys[TierWorking] = struct{}{}
		keys[TierHot] = struct{}{}
	case TierPolicyBalanced:
		keys[TierWorking] = struct{}{}
		keys[TierHot] = struct{}{}
		keys[TierCold] = struct{}{}
	case TierPolicyConservative:
		keys[TierWorking] = struct{}{}
		keys[TierHot] = struct{}{}
		keys[TierCold] = struct{}{}
		keys[TierArchive] = struct{}{}
	}

	return keys
}

func (r *MemoryRouter) MigrateToCold(ctx context.Context, memoryIDs []string) error {
	if r.cacheStore == nil {
		return nil
	}

	for _, id := range memoryIDs {
		if err := r.cacheStore.Del(ctx, fmt.Sprintf("hot:%s", id)); err != nil {
			return err
		}
	}

	return nil
}

// MigrateToArchive evicts memories from both the hot cache and writes them
// to the archive store if available. If no archive store is configured,
// memories are only evicted from cache.
func (r *MemoryRouter) MigrateToArchive(ctx context.Context, memoryIDs []string) error {
	if r.cacheStore == nil && r.archiveStore == nil {
		return nil
	}

	for _, id := range memoryIDs {
		if r.cacheStore != nil {
			_ = r.cacheStore.Del(ctx, fmt.Sprintf("hot:%s", id))
			_ = r.cacheStore.Del(ctx, fmt.Sprintf("cold:%s", id))
			if err := r.cacheStore.Set(ctx, fmt.Sprintf("archive:%s", id), "1", 0); err != nil {
				return fmt.Errorf("tier: mark archive %s: %w", id, err)
			}
		}

		if r.archiveStore != nil {
			var data []byte
			if r.cacheStore != nil {
				if content, err := r.cacheStore.Get(ctx, fmt.Sprintf("hot:%s", id)); err == nil && content != "" {
					data = []byte(content)
				}
			}
			if len(data) == 0 {
				data = []byte(fmt.Sprintf(`{"id":"%s","archived_at":"%s"}`, id, time.Now().Format(time.RFC3339)))
			}
			if err := r.archiveStore.Write(ctx, id, data); err != nil {
				return fmt.Errorf("tier: archive write %s: %w", id, err)
			}
		}
	}

	return nil
}

func estimateTokens(text string) int {
	return len(text) * 4 / 3
}

type TierStats struct {
	WorkingCount int `json:"working_count"`
	HotCount     int `json:"hot_count"`
	ColdCount    int `json:"cold_count"`
	ArchiveCount int `json:"archive_count"`
}

func NewTierStats() *TierStats {
	return &TierStats{
		WorkingCount: 0,
		HotCount:     0,
		ColdCount:    0,
		ArchiveCount: 0,
	}
}

type ArchiveStore interface {
	Write(ctx context.Context, memoryID string, data []byte) error
	Read(ctx context.Context, memoryID string) ([]byte, error)
	Delete(ctx context.Context, memoryID string) error
	List(ctx context.Context) ([]string, error)
}

type FilesystemArchive struct {
	baseDir string
}

func NewFilesystemArchive(baseDir string) *FilesystemArchive {
	return &FilesystemArchive{baseDir: baseDir}
}

func (f *FilesystemArchive) Write(ctx context.Context, memoryID string, data []byte) error {
	if err := os.MkdirAll(f.baseDir, 0755); err != nil {
		return fmt.Errorf("archive mkdir: %w", err)
	}
	path := filepath.Join(f.baseDir, memoryID+".json")
	return os.WriteFile(path, data, 0644)
}

func (f *FilesystemArchive) Read(ctx context.Context, memoryID string) ([]byte, error) {
	path := filepath.Join(f.baseDir, memoryID+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("archive read %s: %w", memoryID, err)
	}
	return data, nil
}

func (f *FilesystemArchive) Delete(ctx context.Context, memoryID string) error {
	path := filepath.Join(f.baseDir, memoryID+".json")
	return os.Remove(path)
}

func (f *FilesystemArchive) List(ctx context.Context) ([]string, error) {
	entries, err := os.ReadDir(f.baseDir)
	if err != nil {
		return nil, fmt.Errorf("archive list: %w", err)
	}
	var ids []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(name, ".json") {
			ids = append(ids, strings.TrimSuffix(name, ".json"))
		}
	}
	return ids, nil
}
