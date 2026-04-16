package tier

import (
	"context"
	"fmt"
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
	config      *TierConfig
	vectorStore VectorStore
	cacheStore CacheStore
}

type TierConfig struct {
	Policy          TierPolicy
	WorkingMaxTokens int   `env:"tier_working_max_tokens" envDefault:"4096"`
	HotMaxTokens     int   `env:"tier_hot_tokens" envDefault:"32768"`
	HotRetentionDays int   `env:"tier_hot_retention_days" envDefault:"7"`
	ArchiveThreshold int   `env:"tier_archive_threshold" envDefault:"100"`
}

type VectorStore interface {
	Search(ctx context.Context, query string, limit int, threshold float32, filters map[string]interface{}) ([]types.MemoryResult, error)
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
			Policy:          TierPolicyBalanced,
			WorkingMaxTokens: 4096,
			HotMaxTokens:    32768,
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

func (r *MemoryRouter) DetermineTier(ctx context.Context, memory *types.Memory) (MemoryTier, error) {
	tokenCount := estimateTokens(memory.Content)

	if tokenCount <= r.config.WorkingMaxTokens {
		return TierWorking, nil
	}

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

	return TierCold, nil
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
		err := r.cacheStore.Del(ctx, fmt.Sprintf("hot:%s", id))
		if err != nil {
			return err
		}
	}

	return nil
}

func estimateTokens(text string) int {
	return len(text) * 4 / 3
}

type TierStats struct {
	WorkingCount int `json:"working_count"`
	HotCount   int `json:"hot_count"`
	ColdCount  int `json:"cold_count"`
}

func NewTierStats() *TierStats {
	return &TierStats{
		WorkingCount: 0,
		HotCount:   0,
		ColdCount:  0,
	}
}