package tier

import (
	"context"
	"testing"
	"time"

	"agent-memory/internal/memory/types"
)

type mockCacheStore struct {
	data  map[string]string
	err   map[string]error
	exist map[string]bool
}

func newMockCacheStore() *mockCacheStore {
	return &mockCacheStore{
		data:  make(map[string]string),
		err:   make(map[string]error),
		exist: make(map[string]bool),
	}
}

func (m *mockCacheStore) Get(ctx context.Context, key string) (string, error) {
	if err, ok := m.err[key]; ok {
		return "", err
	}
	if val, ok := m.data[key]; ok {
		return val, nil
	}
	return "", nil
}

func (m *mockCacheStore) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	m.data[key] = value
	m.exist[key] = true
	return nil
}

func (m *mockCacheStore) Del(ctx context.Context, key string) error {
	delete(m.data, key)
	delete(m.exist, key)
	return nil
}

func (m *mockCacheStore) Exists(ctx context.Context, key string) (bool, error) {
	if err, ok := m.err[key]; ok {
		return false, err
	}
	return m.exist[key], nil
}

func TestNewMemoryRouter_NilConfig(t *testing.T) {
	router := NewMemoryRouter(nil)
	if router == nil {
		t.Fatal("NewMemoryRouter(nil) returned nil")
	}
	if router.config == nil {
		t.Error("expected default config to be set")
	}
	if router.config.Policy != TierPolicyBalanced {
		t.Errorf("expected default policy 'balanced', got %s", router.config.Policy)
	}
	if router.config.WorkingMaxTokens != 4096 {
		t.Errorf("expected default WorkingMaxTokens 4096, got %d", router.config.WorkingMaxTokens)
	}
	if router.config.HotMaxTokens != 32768 {
		t.Errorf("expected default HotMaxTokens 32768, got %d", router.config.HotMaxTokens)
	}
	if router.config.HotRetentionDays != 7 {
		t.Errorf("expected default HotRetentionDays 7, got %d", router.config.HotRetentionDays)
	}
}

func TestNewMemoryRouter_CustomConfig(t *testing.T) {
	cfg := &TierConfig{
		Policy:          TierPolicyAggressive,
		WorkingMaxTokens: 2048,
		HotMaxTokens:    16384,
		HotRetentionDays: 3,
	}
	router := NewMemoryRouter(cfg)
	if router.config.Policy != TierPolicyAggressive {
		t.Errorf("expected policy 'aggressive', got %s", router.config.Policy)
	}
	if router.config.WorkingMaxTokens != 2048 {
		t.Errorf("expected WorkingMaxTokens 2048, got %d", router.config.WorkingMaxTokens)
	}
}

func TestDetermineTier_ShortContent_Working(t *testing.T) {
	router := NewMemoryRouter(nil)
	ctx := context.Background()

	memory := &types.Memory{
		ID:        "mem1",
		Content:   "short content",
		UpdatedAt: time.Now(),
	}

	tier, err := router.DetermineTier(ctx, memory)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if tier != TierWorking {
		optimalTokens := estimateTokens(memory.Content)
		t.Errorf("expected TierWorking for short content (%d tokens), got %s", optimalTokens, tier)
	}
}

func TestDetermineTier_LongContent_WithCache_Hot(t *testing.T) {
	cfg := &TierConfig{
		Policy:          TierPolicyBalanced,
		WorkingMaxTokens: 10,
		HotMaxTokens:    10000,
		HotRetentionDays: 7,
	}
	router := NewMemoryRouter(cfg)
	cache := newMockCacheStore()
	router.SetCacheStore(cache)
	ctx := context.Background()

	longContent := make([]byte, 500)
	for i := range longContent {
		longContent[i] = 'a'
	}

	cache.Set(ctx, "hot:mem2", "cached content", time.Hour*24*7)

	memory := &types.Memory{
		ID:        "mem2",
		Content:   string(longContent),
		UpdatedAt: time.Now(),
	}

	tier, err := router.DetermineTier(ctx, memory)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tier != TierHot {
		t.Errorf("expected TierHot for cached content, got %s", tier)
	}
}

func TestDetermineTier_RecentContent_Hot(t *testing.T) {
	cfg := &TierConfig{
		Policy:          TierPolicyBalanced,
		WorkingMaxTokens: 10,
		HotMaxTokens:    10000,
		HotRetentionDays: 7,
	}
	router := NewMemoryRouter(cfg)
	cache := newMockCacheStore()
	router.SetCacheStore(cache)
	ctx := context.Background()

	longContent := make([]byte, 500)
	for i := range longContent {
		longContent[i] = 'a'
	}

	memory := &types.Memory{
		ID:        "mem3",
		Content:   string(longContent),
		UpdatedAt: time.Now(),
	}

	tier, err := router.DetermineTier(ctx, memory)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tier != TierHot {
		t.Errorf("expected TierHot for recent content within token limit, got %s", tier)
	}
}

func TestDetermineTier_OldContent_Cold(t *testing.T) {
	cfg := &TierConfig{
		Policy:          TierPolicyBalanced,
		WorkingMaxTokens: 10,
		HotMaxTokens:    10000,
		HotRetentionDays: 7,
	}
	router := NewMemoryRouter(cfg)
	ctx := context.Background()

	longContent := make([]byte, 500)
	for i := range longContent {
		longContent[i] = 'a'
	}

	memory := &types.Memory{
		ID:        "mem4",
		Content:   string(longContent),
		UpdatedAt: time.Now().Add(-30 * 24 * time.Hour),
	}

	tier, err := router.DetermineTier(ctx, memory)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tier != TierCold {
		t.Errorf("expected TierCold for old content, got %s", tier)
	}
}

func TestDetermineTier_NilCache(t *testing.T) {
	cfg := &TierConfig{
		Policy:          TierPolicyBalanced,
		WorkingMaxTokens: 10,
		HotMaxTokens:    10000,
		HotRetentionDays: 7,
	}
	router := NewMemoryRouter(cfg)
	ctx := context.Background()

	longContent := make([]byte, 500)
	for i := range longContent {
		longContent[i] = 'a'
	}

	recentMemory := &types.Memory{
		ID:        "mem-recent",
		Content:   string(longContent),
		UpdatedAt: time.Now(),
	}

	tier, err := router.DetermineTier(ctx, recentMemory)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tier != TierHot {
		t.Errorf("expected TierHot for recent content without cache, got %s", tier)
	}
}

func TestSetGetTierPolicy(t *testing.T) {
	router := NewMemoryRouter(nil)

	if router.GetTierPolicy() != TierPolicyBalanced {
		t.Errorf("expected default policy 'balanced', got %s", router.GetTierPolicy())
	}

	tests := []struct {
		policy TierPolicy
	}{
		{TierPolicyAggressive},
		{TierPolicyBalanced},
		{TierPolicyConservative},
	}

	for _, tt := range tests {
		router.SetTierPolicy(tt.policy)
		if router.GetTierPolicy() != tt.policy {
			t.Errorf("expected policy %s, got %s", tt.policy, router.GetTierPolicy())
		}
	}
}

func TestGetTierKeys(t *testing.T) {
	router := NewMemoryRouter(nil)

	tests := []struct {
		policy       TierPolicy
		expectedTiers []MemoryTier
	}{
		{TierPolicyAggressive, []MemoryTier{TierWorking, TierHot}},
		{TierPolicyBalanced, []MemoryTier{TierWorking, TierHot, TierCold}},
		{TierPolicyConservative, []MemoryTier{TierWorking, TierHot, TierCold, TierArchive}},
	}

	for _, tt := range tests {
		t.Run(string(tt.policy), func(t *testing.T) {
			keys := router.GetTierKeys(tt.policy)
			if len(keys) != len(tt.expectedTiers) {
				t.Errorf("expected %d tiers for policy %s, got %d", len(tt.expectedTiers), tt.policy, len(keys))
			}
			for _, tier := range tt.expectedTiers {
				if _, ok := keys[tier]; !ok {
					t.Errorf("expected tier %s for policy %s", tier, tt.policy)
				}
			}
		})
	}
}

func TestMigrateToCold_NilCache(t *testing.T) {
	router := NewMemoryRouter(nil)
	ctx := context.Background()

	err := router.MigrateToCold(ctx, []string{"mem1", "mem2"})
	if err != nil {
		t.Errorf("expected no error with nil cache, got %v", err)
	}
}

func TestMigrateToCold_WithCache(t *testing.T) {
	router := NewMemoryRouter(nil)
	cache := newMockCacheStore()
	router.SetCacheStore(cache)
	ctx := context.Background()

	cache.Set(ctx, "hot:mem1", "content1", time.Hour)
	cache.Set(ctx, "hot:mem2", "content2", time.Hour)

	err := router.MigrateToCold(ctx, []string{"mem1", "mem2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	exists, _ := cache.Exists(ctx, "hot:mem1")
	if exists {
		t.Error("expected hot:mem1 to be deleted after migration")
	}
	exists, _ = cache.Exists(ctx, "hot:mem2")
	if exists {
		t.Error("expected hot:mem2 to be deleted after migration")
	}
}

func TestMigrateToCold_EmptyList(t *testing.T) {
	router := NewMemoryRouter(nil)
	cache := newMockCacheStore()
	router.SetCacheStore(cache)
	ctx := context.Background()

	err := router.MigrateToCold(ctx, []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		minToken int
		maxToken int
	}{
		{"empty", "", 0, 0},
		{"short", "hello", 6, 8},
		{"medium", "this is a test string", 28, 32},
		{"long", "a longer piece of text that should produce more tokens", 68, 76},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens := estimateTokens(tt.text)
			if tt.text == "" {
				if tokens != 0 {
					t.Errorf("expected 0 tokens for empty string, got %d", tokens)
				}
				return
			}
			if tokens < tt.minToken || tokens > tt.maxToken {
				t.Errorf("estimateTokens(%q) = %d, want range [%d, %d]", tt.text, tokens, tt.minToken, tt.maxToken)
			}
		})
	}
}

func TestEstimateTokens_Calculation(t *testing.T) {
	text := "1234567890"
	tokens := estimateTokens(text)
	expected := len(text) * 4 / 3
	if tokens != expected {
		t.Errorf("estimateTokens(%q) = %d, want %d", text, tokens, expected)
	}
}

func TestTierHotTTL(t *testing.T) {
	tests := []struct {
		days int
		ttl  time.Duration
	}{
		{1, 24 * time.Hour},
		{7, 7 * 24 * time.Hour},
		{30, 30 * 24 * time.Hour},
	}

	for _, tt := range tests {
		cfg := &TierConfig{HotRetentionDays: tt.days}
		ttl := cfg.HotTTL()
		if ttl != tt.ttl {
			t.Errorf("HotTTL() with %d days = %v, want %v", tt.days, ttl, tt.ttl)
		}
	}
}

func TestIsRecent(t *testing.T) {
	tests := []struct {
		name    string
		days    int
		age     time.Duration
		recent  bool
	}{
		{"within retention", 7, 6 * 24 * time.Hour, true},
		{"at boundary", 7, 7 * 24 * time.Hour, false},
		{"beyond retention", 7, 30 * 24 * time.Hour, false},
		{"just now", 7, time.Minute, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &TierConfig{HotRetentionDays: tt.days}
			memory := &types.Memory{
				UpdatedAt: time.Now().Add(-tt.age),
			}
			result := cfg.isRecent(memory)
			if result != tt.recent {
				t.Errorf("isRecent() with age=%v, retention=%d days = %v, want %v", tt.age, tt.days, result, tt.recent)
			}
		})
	}
}

func TestNewTierStats(t *testing.T) {
	stats := NewTierStats()
	if stats == nil {
		t.Fatal("NewTierStats() returned nil")
	}
	if stats.WorkingCount != 0 {
		t.Errorf("expected WorkingCount=0, got %d", stats.WorkingCount)
	}
	if stats.HotCount != 0 {
		t.Errorf("expected HotCount=0, got %d", stats.HotCount)
	}
	if stats.ColdCount != 0 {
		t.Errorf("expected ColdCount=0, got %d", stats.ColdCount)
	}
}

func TestSetVectorStore(t *testing.T) {
	router := NewMemoryRouter(nil)
	if router.vectorStore != nil {
		t.Error("expected nil vectorStore initially")
	}
}

func TestSetCacheStore(t *testing.T) {
	router := NewMemoryRouter(nil)
	if router.cacheStore != nil {
		t.Error("expected nil cacheStore initially")
	}

	cache := newMockCacheStore()
	router.SetCacheStore(cache)
	if router.cacheStore == nil {
		t.Error("expected cacheStore to be set")
	}
}