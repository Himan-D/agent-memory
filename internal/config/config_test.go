package config

import (
	"os"
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	os.Clearenv()

	cfg := Load()

	if cfg.Neo4j.URI != "bolt://localhost:7687" {
		t.Errorf("expected default Neo4j URI, got %s", cfg.Neo4j.URI)
	}
	if cfg.Qdrant.URL != "http://localhost:6333" {
		t.Errorf("expected default Qdrant URL, got %s", cfg.Qdrant.URL)
	}
	if cfg.App.HTTPPort != ":8080" {
		t.Errorf("expected default HTTP port :8080, got %s", cfg.App.HTTPPort)
	}
	if cfg.App.MessageBuffer != 100 {
		t.Errorf("expected default MessageBuffer 100, got %d", cfg.App.MessageBuffer)
	}
}

func TestLoadFromEnv(t *testing.T) {
	os.Setenv("NEO4J_URI", "bolt://custom:7687")
	os.Setenv("QDRANT_URL", "custom:6333")
	os.Setenv("OPENAI_API_KEY", "sk-test")
	os.Setenv("HTTP_PORT", ":9090")
	defer os.Unsetenv("NEO4J_URI")
	defer os.Unsetenv("QDRANT_URL")
	defer os.Unsetenv("OPENAI_API_KEY")
	defer os.Unsetenv("HTTP_PORT")

	cfg := Load()

	if cfg.Neo4j.URI != "bolt://custom:7687" {
		t.Errorf("expected bolt://custom:7687, got %s", cfg.Neo4j.URI)
	}
	if cfg.Qdrant.URL != "custom:6333" {
		t.Errorf("expected custom:6333, got %s", cfg.Qdrant.URL)
	}
	if cfg.OpenAI.APIKey != "sk-test" {
		t.Errorf("expected sk-test, got %s", cfg.OpenAI.APIKey)
	}
	if cfg.App.HTTPPort != ":9090" {
		t.Errorf("expected :9090, got %s", cfg.App.HTTPPort)
	}
}

func TestAppConfigDefaults(t *testing.T) {
	cfg := Load()

	if cfg.App.SyncInterval != time.Hour {
		t.Errorf("expected SyncInterval 1h, got %v", cfg.App.SyncInterval)
	}
	if cfg.App.ContextWindow != 50 {
		t.Errorf("expected ContextWindow 50, got %d", cfg.App.ContextWindow)
	}
	if cfg.App.BatchSize != 1000 {
		t.Errorf("expected BatchSize 1000, got %d", cfg.App.BatchSize)
	}
	if cfg.App.BufferTimeout != 5*time.Second {
		t.Errorf("expected BufferTimeout 5s, got %v", cfg.App.BufferTimeout)
	}
}

func TestLLMConfigDefaults(t *testing.T) {
	os.Clearenv()
	cfg := Load()

	if cfg.LLM.Provider != "openai" {
		t.Errorf("expected provider openai, got %s", cfg.LLM.Provider)
	}
	if cfg.LLM.Model != "gpt-4o" {
		t.Errorf("expected model gpt-4o, got %s", cfg.LLM.Model)
	}
	if cfg.LLM.MaxTokens != 4096 {
		t.Errorf("expected max tokens 4096, got %d", cfg.LLM.MaxTokens)
	}
	if cfg.LLM.Temperature != 0.7 {
		t.Errorf("expected temperature 0.7, got %f", cfg.LLM.Temperature)
	}
	if cfg.LLM.RetryMax != 3 {
		t.Errorf("expected retry max 3, got %d", cfg.LLM.RetryMax)
	}
	if cfg.LLM.RetryTimeout != 30 {
		t.Errorf("expected retry timeout 30, got %d", cfg.LLM.RetryTimeout)
	}
}

func TestLLMConfigFromEnv(t *testing.T) {
	os.Setenv("LLM_PROVIDER", "anthropic")
	os.Setenv("LLM_MODEL", "claude-3")
	os.Setenv("LLM_MAX_TOKENS", "8192")
	os.Setenv("LLM_RETRY_MAX", "5")
	os.Setenv("LLM_RETRY_TIMEOUT", "60")
	defer os.Unsetenv("LLM_PROVIDER")
	defer os.Unsetenv("LLM_MODEL")
	defer os.Unsetenv("LLM_MAX_TOKENS")
	defer os.Unsetenv("LLM_RETRY_MAX")
	defer os.Unsetenv("LLM_RETRY_TIMEOUT")

	cfg := Load()

	if cfg.LLM.Provider != "anthropic" {
		t.Errorf("expected provider anthropic, got %s", cfg.LLM.Provider)
	}
	if cfg.LLM.Model != "claude-3" {
		t.Errorf("expected model claude-3, got %s", cfg.LLM.Model)
	}
	if cfg.LLM.MaxTokens != 8192 {
		t.Errorf("expected max tokens 8192, got %d", cfg.LLM.MaxTokens)
	}
	if cfg.LLM.RetryMax != 5 {
		t.Errorf("expected retry max 5, got %d", cfg.LLM.RetryMax)
	}
	if cfg.LLM.RetryTimeout != 60 {
		t.Errorf("expected retry timeout 60, got %d", cfg.LLM.RetryTimeout)
	}
}

func TestMemoryConfigDefaults(t *testing.T) {
	os.Clearenv()
	cfg := Load()

	if !cfg.Memory.ProcessingEnabled {
		t.Error("expected processing enabled by default")
	}
	if !cfg.Memory.AutoExtractFacts {
		t.Error("expected auto extract facts by default")
	}
	if !cfg.Memory.AutoExtractEntities {
		t.Error("expected auto extract entities by default")
	}
	if cfg.Memory.DefaultImportance != "medium" {
		t.Errorf("expected default importance medium, got %s", cfg.Memory.DefaultImportance)
	}
	if !cfg.Memory.ConflictResolution {
		t.Error("expected conflict resolution enabled by default")
	}
	if !cfg.Memory.CacheEnabled {
		t.Error("expected cache enabled by default")
	}
	if cfg.Memory.CacheTTL != 3600 {
		t.Errorf("expected cache TTL 3600, got %d", cfg.Memory.CacheTTL)
	}
}

func TestMemoryConfigFromEnv(t *testing.T) {
	os.Setenv("MEMORY_PROCESSING_ENABLED", "false")
	os.Setenv("MEMORY_DEFAULT_IMPORTANCE", "high")
	os.Setenv("MEMORY_CACHE_ENABLED", "false")
	os.Setenv("MEMORY_CACHE_TTL", "7200")
	defer os.Unsetenv("MEMORY_PROCESSING_ENABLED")
	defer os.Unsetenv("MEMORY_DEFAULT_IMPORTANCE")
	defer os.Unsetenv("MEMORY_CACHE_ENABLED")
	defer os.Unsetenv("MEMORY_CACHE_TTL")

	cfg := Load()

	if cfg.Memory.ProcessingEnabled {
		t.Error("expected processing disabled")
	}
	if cfg.Memory.DefaultImportance != "high" {
		t.Errorf("expected default importance high, got %s", cfg.Memory.DefaultImportance)
	}
	if cfg.Memory.CacheEnabled {
		t.Error("expected cache disabled")
	}
	if cfg.Memory.CacheTTL != 7200 {
		t.Errorf("expected cache TTL 7200, got %d", cfg.Memory.CacheTTL)
	}
}

func TestCompactionConfigDefaults(t *testing.T) {
	os.Clearenv()
	cfg := Load()

	if !cfg.Compaction.Enabled {
		t.Error("expected compaction enabled by default")
	}
	if cfg.Compaction.Interval != "24h" {
		t.Errorf("expected interval 24h, got %s", cfg.Compaction.Interval)
	}
	if cfg.Compaction.MinMemories != 100 {
		t.Errorf("expected min memories 100, got %d", cfg.Compaction.MinMemories)
	}
	if cfg.Compaction.ImportanceThreshold != "low" {
		t.Errorf("expected importance threshold low, got %s", cfg.Compaction.ImportanceThreshold)
	}
	if cfg.Compaction.SummarizeThreshold != 1000 {
		t.Errorf("expected summarize threshold 1000, got %d", cfg.Compaction.SummarizeThreshold)
	}
	if !cfg.Compaction.ArchiveOld {
		t.Error("expected archive old enabled by default")
	}
	if cfg.Compaction.ArchiveAfterDays != 30 {
		t.Errorf("expected archive after days 30, got %d", cfg.Compaction.ArchiveAfterDays)
	}
	if !cfg.Compaction.Deduplicate {
		t.Error("expected deduplicate enabled by default")
	}
	if cfg.Compaction.SimilarityThreshold != 0.92 {
		t.Errorf("expected similarity threshold 0.92, got %f", cfg.Compaction.SimilarityThreshold)
	}
}

func TestCompactionConfigFromEnv(t *testing.T) {
	os.Setenv("COMPACTION_ENABLED", "false")
	os.Setenv("COMPACTION_INTERVAL", "12h")
	os.Setenv("COMPACTION_MIN_MEMORIES", "50")
	os.Setenv("COMPACTION_SIMILARITY_THRESHOLD", "0.85")
	defer os.Unsetenv("COMPACTION_ENABLED")
	defer os.Unsetenv("COMPACTION_INTERVAL")
	defer os.Unsetenv("COMPACTION_MIN_MEMORIES")
	defer os.Unsetenv("COMPACTION_SIMILARITY_THRESHOLD")

	cfg := Load()

	if cfg.Compaction.Enabled {
		t.Error("expected compaction disabled")
	}
	if cfg.Compaction.Interval != "12h" {
		t.Errorf("expected interval 12h, got %s", cfg.Compaction.Interval)
	}
	if cfg.Compaction.MinMemories != 50 {
		t.Errorf("expected min memories 50, got %d", cfg.Compaction.MinMemories)
	}
	if cfg.Compaction.SimilarityThreshold != 0.85 {
		t.Errorf("expected similarity threshold 0.85, got %f", cfg.Compaction.SimilarityThreshold)
	}
}
