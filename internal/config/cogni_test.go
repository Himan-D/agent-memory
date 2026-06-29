package config

import (
	"os"
	"testing"
	"time"
)

func TestDefaultCogniConfig_AllFeaturesDisabled(t *testing.T) {
	c := DefaultCogniConfig()
	if c.EnableCogniWiring {
		t.Fatal("default EnableCogniWiring should be false")
	}
	if c.Hybrid.Enabled {
		t.Fatal("default Hybrid.Enabled should be false")
	}
	if c.Session.TTL != 168*time.Hour {
		t.Fatalf("default Session.TTL=%v, want 168h", c.Session.TTL)
	}
	if c.Distillation.NoveltyThreshold != 0.92 {
		t.Fatalf("default NoveltyThreshold=%v, want 0.92", c.Distillation.NoveltyThreshold)
	}
}

func TestLoadCogniConfigFromEnv_DefaultsWhenUnset(t *testing.T) {
	// Unset every relevant env var so the loader falls back to defaults.
	for _, k := range []string{
		"ENABLE_COGNI_WIRING", "SESSION_REDIS_URL", "SESSION_TTL",
		"SESSION_KEY_PREFIX", "IMPROVEMENT_QUEUE_STREAM",
		"DISTILLATION_NOVELTY_THRESHOLD", "DISTILLATION_MIN_ENTITIES",
		"DISTILLATION_MAX_STATEMENT_CHARS", "HYBRID_WRITE_ENABLED",
		"HYBRID_COMMIT_TIMEOUT",
	} {
		os.Unsetenv(k)
	}
	c := LoadCogniConfigFromEnv()
	d := DefaultCogniConfig()
	if c.EnableCogniWiring != d.EnableCogniWiring {
		t.Fatal("EnableCogniWiring default mismatch")
	}
	if c.Session.TTL != d.Session.TTL {
		t.Fatal("Session.TTL default mismatch")
	}
	if c.Distillation.NoveltyThreshold != d.Distillation.NoveltyThreshold {
		t.Fatal("Distillation.NoveltyThreshold default mismatch")
	}
}

func TestLoadCogniConfigFromEnv_ReadsAllVars(t *testing.T) {
	os.Setenv("ENABLE_COGNI_WIRING", "true")
	os.Setenv("SESSION_REDIS_URL", "redis://localhost:6379/0")
	os.Setenv("SESSION_TTL", "24h")
	os.Setenv("SESSION_KEY_PREFIX", "myapp:session:")
	os.Setenv("IMPROVEMENT_QUEUE_STREAM", "improve:queue")
	os.Setenv("DISTILLATION_NOVELTY_THRESHOLD", "0.85")
	os.Setenv("DISTILLATION_MIN_ENTITIES", "2")
	os.Setenv("DISTILLATION_MAX_STATEMENT_CHARS", "1000")
	os.Setenv("HYBRID_WRITE_ENABLED", "true")
	os.Setenv("HYBRID_COMMIT_TIMEOUT", "30s")
	defer func() {
		for _, k := range []string{
			"ENABLE_COGNI_WIRING", "SESSION_REDIS_URL", "SESSION_TTL",
			"SESSION_KEY_PREFIX", "IMPROVEMENT_QUEUE_STREAM",
			"DISTILLATION_NOVELTY_THRESHOLD", "DISTILLATION_MIN_ENTITIES",
			"DISTILLATION_MAX_STATEMENT_CHARS", "HYBRID_WRITE_ENABLED",
			"HYBRID_COMMIT_TIMEOUT",
		} {
			os.Unsetenv(k)
		}
	}()

	c := LoadCogniConfigFromEnv()
	if !c.EnableCogniWiring {
		t.Fatal("EnableCogniWiring should be true")
	}
	if c.Session.RedisURL != "redis://localhost:6379/0" {
		t.Fatalf("Session.RedisURL=%q", c.Session.RedisURL)
	}
	if c.Session.TTL != 24*time.Hour {
		t.Fatalf("Session.TTL=%v, want 24h", c.Session.TTL)
	}
	if c.Session.KeyPrefix != "myapp:session:" {
		t.Fatalf("Session.KeyPrefix=%q", c.Session.KeyPrefix)
	}
	if c.Improvement.QueueStream != "improve:queue" {
		t.Fatalf("Improvement.QueueStream=%q", c.Improvement.QueueStream)
	}
	if c.Distillation.NoveltyThreshold != 0.85 {
		t.Fatalf("NoveltyThreshold=%v", c.Distillation.NoveltyThreshold)
	}
	if c.Distillation.MinEntities != 2 {
		t.Fatalf("MinEntities=%v", c.Distillation.MinEntities)
	}
	if c.Distillation.MaxStatementChars != 1000 {
		t.Fatalf("MaxStatementChars=%v", c.Distillation.MaxStatementChars)
	}
	if !c.Hybrid.Enabled {
		t.Fatal("Hybrid.Enabled should be true")
	}
	if c.Hybrid.CommitTimeout != 30*time.Second {
		t.Fatalf("CommitTimeout=%v", c.Hybrid.CommitTimeout)
	}
}

func TestLoadCogniConfigFromEnv_InvalidValuesFallBack(t *testing.T) {
	os.Setenv("SESSION_TTL", "not-a-duration")
	os.Setenv("DISTILLATION_NOVELTY_THRESHOLD", "not-a-float")
	os.Setenv("HYBRID_WRITE_ENABLED", "yes-please")
	defer func() {
		os.Unsetenv("SESSION_TTL")
		os.Unsetenv("DISTILLATION_NOVELTY_THRESHOLD")
		os.Unsetenv("HYBRID_WRITE_ENABLED")
	}()
	c := LoadCogniConfigFromEnv()
	if c.Session.TTL != 168*time.Hour {
		t.Fatalf("invalid SESSION_TTL should fall back to default, got %v", c.Session.TTL)
	}
	if c.Distillation.NoveltyThreshold != 0.92 {
		t.Fatalf("invalid NOVELTY_THRESHOLD should fall back, got %v", c.Distillation.NoveltyThreshold)
	}
	if c.Hybrid.Enabled {
		t.Fatal("invalid HYBRID_WRITE_ENABLED should fall back to false")
	}
}

func TestEnvBool(t *testing.T) {
	cases := []struct {
		in   string
		def  bool
		want bool
	}{
		{"", true, true},
		{"", false, false},
		{"true", false, true},
		{"1", false, true},
		{"yes", false, true},
		{"on", false, true},
		{"false", true, false},
		{"no", true, false},
		{"random", true, false},
	}
	for _, c := range cases {
		os.Setenv("__TEST_ENV_BOOL__", c.in)
		if got := envBool("__TEST_ENV_BOOL__", c.def); got != c.want {
			t.Errorf("envBool(%q, %v) = %v, want %v", c.in, c.def, got, c.want)
		}
	}
	os.Unsetenv("__TEST_ENV_BOOL__")
}
