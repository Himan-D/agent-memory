package memory

import (
	"context"
	"strings"
	"testing"
	"time"

	"agent-memory/internal/config"
)

func TestPingStoresWhenUnavailable(t *testing.T) {
	cfg := &config.Config{
		Neo4j: config.Neo4jConfig{
			URI:      "bolt://127.0.0.1:59999",
			User:     "neo4j",
			Password: "test",
		},
		Qdrant: config.QdrantConfig{
			URL: "http://127.0.0.1:59999",
		},
		App: config.AppConfig{
			MessageBuffer: 100,
			BufferTimeout: 5 * time.Second,
		},
	}

	svc, err := NewService(cfg)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	defer svc.Close()

	ctx := context.Background()

	if err := svc.PingNeo4j(ctx); err == nil {
		t.Fatal("expected PingNeo4j error when neo4j unavailable")
	} else if !strings.Contains(err.Error(), "neo4j not configured") {
		t.Fatalf("unexpected neo4j ping error: %v", err)
	}

	if err := svc.PingQdrant(ctx); err == nil {
		t.Fatal("expected PingQdrant error when qdrant unavailable")
	} else if !strings.Contains(err.Error(), "qdrant not configured") {
		t.Fatalf("unexpected qdrant ping error: %v", err)
	}
}
