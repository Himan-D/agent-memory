package memory

import (
	"context"
	"testing"

	"agent-memory/internal/config"
)

func TestNewService_UnavailableStores(t *testing.T) {
	cfg := config.Load()
	cfg.Neo4j.URI = "bolt://127.0.0.1:1"
	cfg.Qdrant.URL = "http://127.0.0.1:1"

	svc, err := NewService(cfg)
	if err != nil {
		t.Fatalf("NewService should degrade gracefully: %v", err)
	}
	defer svc.Close()

	if svc.GetGraph() != nil {
		t.Fatal("graph store should be nil when neo4j is unavailable")
	}
	if svc.GetVector() != nil {
		t.Fatal("vector store should be nil when qdrant is unavailable")
	}
	if svc.GetNeo4jClient() != nil {
		t.Fatal("neo4j client should be nil when neo4j is unavailable")
	}

	ctx := context.Background()
	if err := svc.PingNeo4j(ctx); err == nil {
		t.Fatal("PingNeo4j should fail when neo4j is unavailable")
	}
	if err := svc.PingQdrant(ctx); err == nil {
		t.Fatal("PingQdrant should fail when qdrant is unavailable")
	}
}
