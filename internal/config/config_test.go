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
	if cfg.Qdrant.URL != "localhost:6333" {
		t.Errorf("expected default Qdrant URL, got %s", cfg.Qdrant.URL)
	}
	if cfg.App.HTTPPort != ":8080" {
		t.Errorf("expected default HTTP port :8080, got %s", cfg.App.HTTPPort)
	}
	if cfg.App.MessageBuffer != 50 {
		t.Errorf("expected default MessageBuffer 50, got %d", cfg.App.MessageBuffer)
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
