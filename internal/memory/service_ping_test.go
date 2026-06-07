package memory

import (
	"context"
	"testing"

	"agent-memory/internal/config"
)

func TestPingStoresWhenUnavailable(t *testing.T) {
	cfg := config.Load()
	cfg.Neo4j.URI = "bolt://127.0.0.1:1"
	cfg.Qdrant.URL = "http://127.0.0.1:1"

	svc, err := NewService(cfg)
	if err != nil {
		t.Fatalf("NewService returned error: %v", err)
	}
	defer svc.Close()

	ctx := context.Background()

	if err := svc.PingNeo4j(ctx); err == nil {
		t.Fatal("expected PingNeo4j to fail when neo4j is unavailable")
	}
	if err := svc.PingQdrant(ctx); err == nil {
		t.Fatal("expected PingQdrant to fail when qdrant is unavailable")
	}
}
