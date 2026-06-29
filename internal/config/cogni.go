package config

import (
	"os"
	"strconv"
	"time"
)

// CogniConfig groups the environment-tunable settings for the Cognee-
// inspired wiring (session store, distillation, improvement pipeline).
//
// All fields are optional. Missing values fall back to production-safe
// defaults in the consuming packages (Redis-backed session store degrades
// to in-memory, etc.).
//
// This struct is intentionally defined in a separate file so the existing
// Config struct in config.go is not modified. Callers either embed it
// directly or use LoadCogniConfigFromEnv to populate it.
type CogniConfig struct {
	EnableCogniWiring bool

	Session      SessionConfig
	Improvement  ImprovementConfig
	Distillation DistillationConfig
	Hybrid       HybridConfig
}

// SessionConfig configures the Redis-backed session store. RedisURL is
// read by TieredStore + RedisStore at construction time.
type SessionConfig struct {
	RedisURL  string
	TTL       time.Duration
	KeyPrefix string
}

// ImprovementConfig configures the 6-stage improvement pipeline.
type ImprovementConfig struct {
	QueueStream string
}

// DistillationConfig configures the session distiller.
type DistillationConfig struct {
	NoveltyThreshold  float64
	MinEntities       int
	MaxStatementChars int
}

// HybridConfig configures the hybrid (graph + vector) write path on Neo4j.
type HybridConfig struct {
	Enabled       bool
	CommitTimeout time.Duration
}

// DefaultCogniConfig returns production-safe defaults. All features
// disabled so the struct is a true no-op when used as-is.
func DefaultCogniConfig() CogniConfig {
	return CogniConfig{
		EnableCogniWiring: false,
		Session: SessionConfig{
			TTL:       168 * time.Hour,
			KeyPrefix: "session:",
		},
		Improvement: ImprovementConfig{
			QueueStream: "improvement:queue",
		},
		Distillation: DistillationConfig{
			NoveltyThreshold:  0.92,
			MinEntities:       1,
			MaxStatementChars: 500,
		},
		Hybrid: HybridConfig{
			Enabled:       false,
			CommitTimeout: 10 * time.Second,
		},
	}
}

// LoadCogniConfigFromEnv reads the CogniConfig from environment variables.
// Missing variables fall back to DefaultCogniConfig values.
func LoadCogniConfigFromEnv() CogniConfig {
	c := DefaultCogniConfig()
	c.EnableCogniWiring = envBool("ENABLE_COGNI_WIRING", false)
	if v := os.Getenv("SESSION_REDIS_URL"); v != "" {
		c.Session.RedisURL = v
	}
	if v := os.Getenv("SESSION_TTL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			c.Session.TTL = d
		}
	}
	if v := os.Getenv("SESSION_KEY_PREFIX"); v != "" {
		c.Session.KeyPrefix = v
	}
	if v := os.Getenv("IMPROVEMENT_QUEUE_STREAM"); v != "" {
		c.Improvement.QueueStream = v
	}
	if v := os.Getenv("DISTILLATION_NOVELTY_THRESHOLD"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			c.Distillation.NoveltyThreshold = f
		}
	}
	if v := os.Getenv("DISTILLATION_MIN_ENTITIES"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			c.Distillation.MinEntities = i
		}
	}
	if v := os.Getenv("DISTILLATION_MAX_STATEMENT_CHARS"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			c.Distillation.MaxStatementChars = i
		}
	}
	c.Hybrid.Enabled = envBool("HYBRID_WRITE_ENABLED", false)
	if v := os.Getenv("HYBRID_COMMIT_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			c.Hybrid.CommitTimeout = d
		}
	}
	return c
}

// envBool parses "true"/"1" as true, anything else as false.
func envBool(key string, def bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	switch v {
	case "true", "1", "yes", "on":
		return true
	default:
		return false
	}
}
