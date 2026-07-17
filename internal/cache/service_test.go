package cache

import (
	"context"
	"fmt"
	"testing"
)

// --- Mock implementations ---

type mockSkillProvider struct {
	skills map[string]*Skill
}

func (m *mockSkillProvider) GetSkill(_ context.Context, id string) (*Skill, error) {
	if s, ok := m.skills[id]; ok {
		return s, nil
	}
	return nil, fmt.Errorf("skill not found: %s", id)
}

func (m *mockSkillProvider) ListSkills(_ context.Context) ([]*Skill, error) {
	result := make([]*Skill, 0, len(m.skills))
	for _, s := range m.skills {
		result = append(result, s)
	}
	return result, nil
}

// --- Test helpers ---

func newTestService() *Service {
	sp := &mockSkillProvider{
		skills: map[string]*Skill{
			"s1": {ID: "s1", Name: "Summarizer", Domain: "nlp", Trigger: "summarize", Action: "Summarize text", Prompt: "You are a summarizer"},
			"s2": {ID: "s2", Name: "Translator", Domain: "nlp", Trigger: "translate", Action: "Translate text", Prompt: "You are a translator"},
			"s3": {ID: "s3", Name: "CodeReview", Domain: "dev", Trigger: "review", Action: "Review code", Prompt: "You are a code reviewer"},
		},
	}
	return NewService(sp, nil)
}

// --- Tests ---

func TestWarmCache(t *testing.T) {
	svc := newTestService()
	if err := svc.WarmCache(context.Background()); err != nil {
		t.Fatalf("WarmCache: %v", err)
	}

	stats := svc.GetStats()
	if stats.CachedSkills != 3 {
		t.Errorf("expected 3 cached skills, got %d", stats.CachedSkills)
	}
}

func TestBlendCacheHit(t *testing.T) {
	svc := newTestService()
	svc.WarmCache(context.Background()) //nolint:errcheck

	result, err := svc.Blend(context.Background(), &BlendRequest{
		DynamicContent: "Please summarize this document",
		SkillIDs:       []string{"s1", "s2"},
		Mode:           "full",
	})
	if err != nil {
		t.Fatalf("Blend: %v", err)
	}

	if result.CacheHits != 2 {
		t.Errorf("expected 2 cache hits, got %d", result.CacheHits)
	}
	if result.CacheMisses != 0 {
		t.Errorf("expected 0 cache misses, got %d", result.CacheMisses)
	}
	if result.TokensSaved <= 0 {
		t.Error("expected tokens saved > 0")
	}
}

func TestBlendCacheMiss(t *testing.T) {
	svc := newTestService()
	// Do NOT warm the cache — should get a miss then on-demand fill.

	result, err := svc.Blend(context.Background(), &BlendRequest{
		DynamicContent: "Hello",
		SkillIDs:       []string{"s1"},
		Mode:           "full",
	})
	if err != nil {
		t.Fatalf("Blend: %v", err)
	}

	if result.CacheMisses != 1 {
		t.Errorf("expected 1 miss, got %d", result.CacheMisses)
	}

	// Second call should hit the cache that was filled on-demand.
	result2, err := svc.Blend(context.Background(), &BlendRequest{
		DynamicContent: "Hello again",
		SkillIDs:       []string{"s1"},
		Mode:           "full",
	})
	if err != nil {
		t.Fatalf("Blend (second): %v", err)
	}
	if result2.CacheHits != 1 {
		t.Errorf("expected 1 hit on second call, got %d", result2.CacheHits)
	}
}

func TestInvalidate(t *testing.T) {
	svc := newTestService()
	svc.WarmCache(context.Background()) //nolint:errcheck

	svc.Invalidate("s1")
	if _, exists := svc.GetCachedSkill("s1"); exists {
		t.Error("s1 should be invalidated")
	}

	stats := svc.GetStats()
	if stats.CachedSkills != 2 {
		t.Errorf("expected 2 cached skills after invalidate, got %d", stats.CachedSkills)
	}
}

func TestInvalidateAll(t *testing.T) {
	svc := newTestService()
	svc.WarmCache(context.Background()) //nolint:errcheck
	svc.InvalidateAll()

	stats := svc.GetStats()
	if stats.CachedSkills != 0 {
		t.Errorf("expected 0 cached skills after invalidate all, got %d", stats.CachedSkills)
	}
}

func TestSummaryMode(t *testing.T) {
	svc := newTestService()
	svc.WarmCache(context.Background()) //nolint:errcheck

	resultFull, err := svc.Blend(context.Background(), &BlendRequest{
		DynamicContent: "test",
		SkillIDs:       []string{"s1"},
		Mode:           "full",
	})
	if err != nil {
		t.Fatalf("Blend full: %v", err)
	}

	resultSummary, err := svc.Blend(context.Background(), &BlendRequest{
		DynamicContent: "test",
		SkillIDs:       []string{"s1"},
		Mode:           "summary",
	})
	if err != nil {
		t.Fatalf("Blend summary: %v", err)
	}

	if len(resultSummary.BlendedPrompt) >= len(resultFull.BlendedPrompt) {
		t.Error("summary mode should produce shorter prompt than full mode")
	}
}

func TestGetStats(t *testing.T) {
	svc := newTestService()
	svc.WarmCache(context.Background()) //nolint:errcheck

	// Generate some hits.
	svc.Blend(context.Background(), &BlendRequest{DynamicContent: "a", SkillIDs: []string{"s1"}, Mode: "full"})       //nolint:errcheck
	svc.Blend(context.Background(), &BlendRequest{DynamicContent: "b", SkillIDs: []string{"s1", "s2"}, Mode: "full"}) //nolint:errcheck

	stats := svc.GetStats()
	if stats.TotalHits != 3 {
		t.Errorf("expected 3 total hits, got %d", stats.TotalHits)
	}
	if stats.HitRate != 1.0 {
		t.Errorf("expected 100%% hit rate, got %f", stats.HitRate)
	}
	if stats.TotalTokensSaved <= 0 {
		t.Error("expected tokens saved > 0")
	}
}

func TestCosineSimilarity(t *testing.T) {
	a := []float32{1, 0, 0}
	b := []float32{1, 0, 0}
	if s := cosineSimilarity(a, b); s < 0.99 {
		t.Errorf("identical vectors should have similarity ~1.0, got %f", s)
	}

	c := []float32{0, 1, 0}
	if s := cosineSimilarity(a, c); s > 0.01 {
		t.Errorf("orthogonal vectors should have similarity ~0.0, got %f", s)
	}
}

func TestHashContent(t *testing.T) {
	h1 := hashContent("hello")
	h2 := hashContent("hello")
	h3 := hashContent("world")

	if h1 != h2 {
		t.Error("same content should produce same hash")
	}
	if h1 == h3 {
		t.Error("different content should produce different hash")
	}
}

func TestMatchSkillsNoEmbedder(t *testing.T) {
	svc := newTestService()
	_, err := svc.MatchSkills(context.Background(), "summarize this", 5)
	if err == nil {
		t.Error("expected error when no embedding provider configured")
	}
}

func TestWarmCacheIdempotent(t *testing.T) {
	svc := newTestService()
	svc.WarmCache(context.Background()) //nolint:errcheck
	svc.WarmCache(context.Background()) //nolint:errcheck

	stats := svc.GetStats()
	if stats.CachedSkills != 3 {
		t.Errorf("expected 3 cached skills after double warm, got %d", stats.CachedSkills)
	}
}

func TestBlendEmptySkillIDs(t *testing.T) {
	svc := newTestService()
	result, err := svc.Blend(context.Background(), &BlendRequest{
		DynamicContent: "just the query",
		SkillIDs:       []string{},
		Mode:           "full",
	})
	if err != nil {
		t.Fatalf("Blend with empty skills: %v", err)
	}
	if result.CacheHits != 0 {
		t.Errorf("expected 0 hits, got %d", result.CacheHits)
	}
	if result.BlendedPrompt != "just the query" {
		t.Errorf("expected prompt to equal dynamic content only, got %q", result.BlendedPrompt)
	}
}

func TestGetCachedSkill(t *testing.T) {
	svc := newTestService()
	svc.WarmCache(context.Background()) //nolint:errcheck

	cs, ok := svc.GetCachedSkill("s1")
	if !ok {
		t.Fatal("expected s1 to be cached")
	}
	if cs.Name != "Summarizer" {
		t.Errorf("expected name Summarizer, got %q", cs.Name)
	}
	if cs.FullContent == "" {
		t.Error("expected non-empty full content")
	}
	if cs.Summary == "" {
		t.Error("expected non-empty summary")
	}
	if cs.TokenCount <= 0 {
		t.Error("expected token count > 0")
	}
}
