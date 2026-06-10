package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// SkillProvider is the interface the cache service uses to fetch skills.
type SkillProvider interface {
	GetSkill(ctx context.Context, id string) (*Skill, error)
	ListSkills(ctx context.Context) ([]*Skill, error)
}

// Skill is a minimal representation used inside the cache package.
// It is intentionally self-contained to avoid importing internal packages.
type Skill struct {
	ID      string
	Name    string
	Domain  string
	Trigger string
	Action  string
	Prompt  string
}

// EmbeddingProvider computes a vector embedding for a piece of text.
type EmbeddingProvider interface {
	Embed(ctx context.Context, text string) ([]float32, error)
}

// Service is the CacheBlend-inspired skill cache.
type Service struct {
	mu            sync.RWMutex
	cache         map[string]*CachedSkill // skillID -> cached skill
	skillProvider SkillProvider
	embedProvider EmbeddingProvider

	// Atomic counters for thread-safe stats.
	hits           atomic.Int64
	misses         atomic.Int64
	tokensSaved    atomic.Int64
	hitLatencySum  atomic.Int64
	missLatencySum atomic.Int64
}

// NewService creates a new cache Service.
// sp is required; ep is optional (similarity-based MatchSkills requires it).
func NewService(sp SkillProvider, ep EmbeddingProvider) *Service {
	return &Service{
		cache:         make(map[string]*CachedSkill),
		skillProvider: sp,
		embedProvider: ep,
	}
}

// WarmCache pre-computes cached representations for all skills.
func (s *Service) WarmCache(ctx context.Context) error {
	skills, err := s.skillProvider.ListSkills(ctx)
	if err != nil {
		return fmt.Errorf("cache warm: list skills: %w", err)
	}

	for _, skill := range skills {
		// Partial cache is better than none — continue on error.
		_ = s.cacheSkill(ctx, skill)
	}
	return nil
}

// cacheSkill pre-computes and stores a single skill's cached representation.
func (s *Service) cacheSkill(ctx context.Context, skill *Skill) error {
	content := buildSkillContent(skill)
	hash := hashContent(content)

	// Skip re-computation when content has not changed.
	s.mu.RLock()
	existing, exists := s.cache[skill.ID]
	s.mu.RUnlock()

	if exists && existing.ContentHash == hash {
		return nil
	}

	summary := generateSummary(content)

	var embedding []float32
	if s.embedProvider != nil {
		var err error
		embedding, err = s.embedProvider.Embed(ctx, summary)
		if err != nil {
			// Non-fatal: cache works without embeddings.
			embedding = nil
		}
	}

	cached := &CachedSkill{
		SkillID:        skill.ID,
		Name:           skill.Name,
		Domain:         skill.Domain,
		ContentHash:    hash,
		Summary:        summary,
		FullContent:    content,
		Embedding:      embedding,
		TokenCount:     estimateTokens(content),
		CachedAt:       time.Now(),
		LastAccessedAt: time.Now(),
	}

	s.mu.Lock()
	s.cache[skill.ID] = cached
	s.mu.Unlock()

	return nil
}

// Blend takes dynamic content and skill IDs, returns a blended prompt with
// cache statistics.
func (s *Service) Blend(ctx context.Context, req *BlendRequest) (*BlendResult, error) {
	start := time.Now()

	var parts []string
	var hits, misses, tokensSaved int
	var usedSkills []string

	// Dynamic content always goes first (recomputed each time).
	parts = append(parts, req.DynamicContent)

	for _, skillID := range req.SkillIDs {
		s.mu.RLock()
		cached, exists := s.cache[skillID]
		s.mu.RUnlock()

		if exists {
			hits++
			s.hits.Add(1)
			tokensSaved += cached.TokenCount
			s.tokensSaved.Add(int64(cached.TokenCount))

			s.mu.Lock()
			cached.LastAccessedAt = time.Now()
			cached.HitCount++
			s.mu.Unlock()

			if req.Mode == "summary" {
				parts = append(parts, cached.Summary)
			} else {
				parts = append(parts, cached.FullContent)
			}
			usedSkills = append(usedSkills, skillID)
		} else {
			misses++
			s.misses.Add(1)

			// On-demand cache fill for a miss.
			if s.skillProvider != nil {
				skill, err := s.skillProvider.GetSkill(ctx, skillID)
				if err == nil {
					_ = s.cacheSkill(ctx, skill)

					s.mu.RLock()
					nowCached, ok := s.cache[skillID]
					s.mu.RUnlock()

					if ok {
						if req.Mode == "summary" {
							parts = append(parts, nowCached.Summary)
						} else {
							parts = append(parts, nowCached.FullContent)
						}
						usedSkills = append(usedSkills, skillID)
					}
				}
			}
		}
	}

	latency := time.Since(start)

	if hits > 0 {
		s.hitLatencySum.Add(latency.Milliseconds())
	} else {
		s.missLatencySum.Add(latency.Milliseconds())
	}

	return &BlendResult{
		BlendedPrompt: strings.Join(parts, "\n\n---\n\n"),
		CacheHits:     hits,
		CacheMisses:   misses,
		TokensSaved:   tokensSaved,
		SkillsUsed:    usedSkills,
		LatencyMs:     float64(latency.Microseconds()) / 1000.0,
	}, nil
}

// MatchSkills finds skills relevant to a query using embedding similarity.
// Requires an EmbeddingProvider; returns an error if none is configured.
func (s *Service) MatchSkills(ctx context.Context, query string, topK int) ([]SkillMatch, error) {
	if s.embedProvider == nil {
		return nil, fmt.Errorf("embedding provider not configured")
	}

	queryEmb, err := s.embedProvider.Embed(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("cache match: embed query: %w", err)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	var matches []SkillMatch
	for _, cached := range s.cache {
		if cached.Embedding == nil {
			continue
		}
		score := cosineSimilarity(queryEmb, cached.Embedding)
		if score > 0.3 {
			level := 1
			if score > 0.7 {
				level = 2 // High relevance: load full content.
			}
			matches = append(matches, SkillMatch{
				SkillID: cached.SkillID,
				Name:    cached.Name,
				Score:   score,
				Level:   level,
			})
		}
	}

	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Score > matches[j].Score
	})

	if topK > 0 && len(matches) > topK {
		matches = matches[:topK]
	}

	return matches, nil
}

// Invalidate removes a skill from the cache.
func (s *Service) Invalidate(skillID string) {
	s.mu.Lock()
	delete(s.cache, skillID)
	s.mu.Unlock()
}

// InvalidateAll clears the entire cache.
func (s *Service) InvalidateAll() {
	s.mu.Lock()
	s.cache = make(map[string]*CachedSkill)
	s.mu.Unlock()
}

// GetStats returns current cache performance metrics.
func (s *Service) GetStats() CacheStats {
	totalHits := s.hits.Load()
	totalMisses := s.misses.Load()
	total := totalHits + totalMisses

	var hitRate float64
	if total > 0 {
		hitRate = float64(totalHits) / float64(total)
	}

	var avgHitLatency, avgMissLatency float64
	if totalHits > 0 {
		avgHitLatency = float64(s.hitLatencySum.Load()) / float64(totalHits)
	}
	if totalMisses > 0 {
		avgMissLatency = float64(s.missLatencySum.Load()) / float64(totalMisses)
	}

	s.mu.RLock()
	cachedCount := len(s.cache)
	var cacheSize int64
	for _, c := range s.cache {
		cacheSize += int64(len(c.FullContent) + len(c.Summary))
	}
	s.mu.RUnlock()

	return CacheStats{
		TotalHits:        totalHits,
		TotalMisses:      totalMisses,
		HitRate:          hitRate,
		TotalTokensSaved: s.tokensSaved.Load(),
		CachedSkills:     cachedCount,
		CacheSize:        cacheSize,
		AvgHitLatencyMs:  avgHitLatency,
		AvgMissLatencyMs: avgMissLatency,
	}
}

// GetCachedSkill returns a cached skill by ID if it exists.
func (s *Service) GetCachedSkill(skillID string) (*CachedSkill, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cached, ok := s.cache[skillID]
	return cached, ok
}

// --- Helpers ---

func buildSkillContent(skill *Skill) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("# Skill: %s\n", skill.Name))
	if skill.Domain != "" {
		b.WriteString(fmt.Sprintf("Domain: %s\n", skill.Domain))
	}
	if skill.Trigger != "" {
		b.WriteString(fmt.Sprintf("Trigger: %s\n", skill.Trigger))
	}
	if skill.Action != "" {
		b.WriteString(fmt.Sprintf("\n## Action\n%s\n", skill.Action))
	}
	if skill.Prompt != "" {
		b.WriteString(fmt.Sprintf("\n## System Prompt\n%s\n", skill.Prompt))
	}
	return b.String()
}

func generateSummary(content string) string {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if len(line) > 200 {
			return line[:200] + "..."
		}
		return line
	}
	if len(content) > 200 {
		return content[:200] + "..."
	}
	return content
}

func hashContent(content string) string {
	h := sha256.Sum256([]byte(content))
	return hex.EncodeToString(h[:])
}

func estimateTokens(text string) int {
	// Rough estimate: ~4 chars per token for English.
	return len(text) / 4
}

func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dotProduct / (sqrtNewton(normA) * sqrtNewton(normB))
}

// sqrtNewton is a Newton's method sqrt to keep the package dependency-free.
func sqrtNewton(x float64) float64 {
	if x <= 0 {
		return 0
	}
	z := x
	for i := 0; i < 20; i++ {
		z = (z + x/z) / 2
	}
	return z
}
