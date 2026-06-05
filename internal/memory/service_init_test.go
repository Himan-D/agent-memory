package memory

import (
	"context"
	"testing"

	"agent-memory/internal/config"
)

func TestNewService_UnavailableStoresUseNilInterfaces(t *testing.T) {
	cfg := config.Load()
	cfg.Neo4j.URI = "bolt://127.0.0.1:1"
	cfg.Qdrant.URL = "http://127.0.0.1:1"

	svc, err := NewService(cfg)
	if err != nil {
		t.Fatalf("NewService returned error: %v", err)
	}
	defer svc.Close()

	if svc.GetGraph() != nil {
		t.Fatal("graph interface should be nil when neo4j client init fails")
	}
	if svc.GetVector() != nil {
		t.Fatal("vector interface should be nil when qdrant client init fails")
	}

	ctx := context.Background()
	if err := svc.PingNeo4j(ctx); err == nil {
		t.Fatal("expected PingNeo4j error when neo4j is unavailable")
	}
	if err := svc.PingQdrant(ctx); err == nil {
		t.Fatal("expected PingQdrant error when qdrant is unavailable")
	}
}
