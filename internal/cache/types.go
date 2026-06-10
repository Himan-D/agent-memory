package cache

import (
	"time"
)

// CachedSkill represents a pre-computed skill representation
type CachedSkill struct {
	SkillID        string                 `json:"skill_id"`
	Name           string                 `json:"name"`
	Domain         string                 `json:"domain"`
	ContentHash    string                 `json:"content_hash"`
	Summary        string                 `json:"summary"`      // Level 1: short summary for relevance detection
	FullContent    string                 `json:"full_content"` // Level 2: full instructions
	Embedding      []float32              `json:"embedding"`    // Pre-computed embedding for similarity
	TokenCount     int                    `json:"token_count"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	CachedAt       time.Time              `json:"cached_at"`
	LastAccessedAt time.Time              `json:"last_accessed_at"`
	HitCount       int64                  `json:"hit_count"`
}

// CacheStats tracks cache performance metrics
type CacheStats struct {
	TotalHits        int64   `json:"total_hits"`
	TotalMisses      int64   `json:"total_misses"`
	HitRate          float64 `json:"hit_rate"`
	TotalTokensSaved int64   `json:"total_tokens_saved"`
	CachedSkills     int     `json:"cached_skills"`
	CacheSize        int64   `json:"cache_size_bytes"`
	AvgHitLatencyMs  float64 `json:"avg_hit_latency_ms"`
	AvgMissLatencyMs float64 `json:"avg_miss_latency_ms"`
}

// BlendRequest represents a request to blend cached skills into a prompt
type BlendRequest struct {
	DynamicContent string   `json:"dynamic_content"` // User query + conversation history
	SkillIDs       []string `json:"skill_ids"`       // Skills to blend in
	MaxTokens      int      `json:"max_tokens"`
	Mode           string   `json:"mode"` // "summary" or "full"
}

// BlendResult is the output of a cache blend operation
type BlendResult struct {
	BlendedPrompt string   `json:"blended_prompt"`
	CacheHits     int      `json:"cache_hits"`
	CacheMisses   int      `json:"cache_misses"`
	TokensSaved   int      `json:"tokens_saved"`
	SkillsUsed    []string `json:"skills_used"`
	LatencyMs     float64  `json:"latency_ms"`
}

// SkillMatch represents a skill matched by relevance to a query
type SkillMatch struct {
	SkillID string  `json:"skill_id"`
	Name    string  `json:"name"`
	Score   float64 `json:"score"`
	Level   int     `json:"level"` // 1=summary matched, 2=full content needed
}
