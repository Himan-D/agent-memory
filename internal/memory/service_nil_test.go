package memory

import (
	"context"
	"testing"

	"agent-memory/internal/config"
)

func TestNewService_UnavailableStoresAreNilInterfaces(t *testing.T) {
	cfg := config.Load()
	cfg.Neo4j.URI = "bolt://127.0.0.1:1"
	cfg.Qdrant.URL = "http://127.0.0.1:1"

	svc, err := NewService(cfg)
	if err != nil {
		t.Fatalf("NewService should not fail when stores are unavailable: %v", err)
	}
	defer svc.Close()

	if svc.graph != nil {
		t.Fatal("graph interface should be nil when neo4j is unavailable")
	}
	if svc.vector != nil {
		t.Fatal("vector interface should be nil when qdrant is unavailable")
	}
	if svc.neo4jClient != nil {
		t.Fatal("neo4jClient should be nil when neo4j is unavailable")
	}

	ctx := context.Background()
	if err := svc.PingNeo4j(ctx); err == nil {
		t.Fatal("PingNeo4j should return error when neo4j is unavailable")
	}
	if err := svc.PingQdrant(ctx); err == nil {
		t.Fatal("PingQdrant should return error when qdrant is unavailable")
	}
}
