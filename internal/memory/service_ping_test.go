package memory

import (
	"context"
	"testing"

	"agent-memory/internal/config"
)

func TestPingStoresWhenUnavailable(t *testing.T) {
	cfg := &config.Config{
		Neo4j: config.Neo4jConfig{
			URI:      "bolt://127.0.0.1:1",
			User:     "neo4j",
			Password: "invalid",
		},
		Qdrant: config.QdrantConfig{
			URL: "http://127.0.0.1:1",
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
	}
	if err := svc.PingQdrant(ctx); err == nil {
		t.Fatal("expected PingQdrant error when qdrant unavailable")
	}
}
