package memory

import (
	"context"
	"testing"

	"agent-memory/internal/config"
)

func TestNewServiceUnavailableBackendsUseNilInterfaces(t *testing.T) {
	cfg := config.Load()
	cfg.Neo4j.URI = "bolt://127.0.0.1:59999"
	cfg.Qdrant.URL = "http://127.0.0.1:59998"

	svc, err := NewService(cfg)
	if err != nil {
		t.Fatalf("NewService should tolerate unavailable backends: %v", err)
	}
	defer svc.Close()

	if svc.GetGraph() != nil {
		t.Fatal("graph interface must be nil when neo4j client init fails")
	}
	if svc.GetVector() != nil {
		t.Fatal("vector interface must be nil when qdrant client init fails")
	}

	ctx := context.Background()
	if err := svc.PingNeo4j(ctx); err == nil {
		t.Fatal("PingNeo4j should return error when graph store is unavailable")
	}
	if err := svc.PingQdrant(ctx); err == nil {
		t.Fatal("PingQdrant should return error when vector store is unavailable")
	}
}
