package memory

import (
	"context"
	"testing"

	"agent-memory/internal/config"
)

func TestNewService_UnavailableStoresDoNotPanicOnPing(t *testing.T) {
	cfg := config.Load()
	cfg.Neo4j.URI = "bolt://127.0.0.1:1"
	cfg.Neo4j.User = "neo4j"
	cfg.Neo4j.Password = "changeme"
	cfg.Qdrant.URL = "http://127.0.0.1:1"

	svc, err := NewService(cfg)
	if err != nil {
		t.Fatalf("NewService should tolerate unavailable stores: %v", err)
	}
	defer svc.Close()

	ctx := context.Background()

	if svc.graph != nil {
		t.Fatal("graph should be nil when neo4j is unavailable")
	}
	if svc.vector != nil {
		t.Fatal("vector should be nil when qdrant is unavailable")
	}

	if err := svc.PingNeo4j(ctx); err == nil {
		t.Fatal("expected error when neo4j is unavailable")
	}
	if err := svc.PingQdrant(ctx); err == nil {
		t.Fatal("expected error when qdrant is unavailable")
	}
}
