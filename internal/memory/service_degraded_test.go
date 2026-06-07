package memory

import (
	"context"
	"testing"

	"agent-memory/internal/config"
)

func TestNewServiceDegradedStoresNilInterfaces(t *testing.T) {
	cfg := config.Load()
	cfg.Neo4j.URI = "bolt://127.0.0.1:1"
	cfg.Neo4j.User = "neo4j"
	cfg.Neo4j.Password = "changeme"
	cfg.Qdrant.URL = "http://127.0.0.1:1"

	svc, err := NewService(cfg)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	defer svc.Close()

	if svc.graph != nil {
		t.Fatal("graph interface should be nil when neo4j is unavailable")
	}
	if svc.vector != nil {
		t.Fatal("vector interface should be nil when qdrant is unavailable")
	}
	if svc.apiKeys != nil {
		t.Fatal("apiKeys interface should be nil when neo4j is unavailable")
	}

	ctx := context.Background()
	if err := svc.PingNeo4j(ctx); err == nil {
		t.Fatal("PingNeo4j should fail when neo4j is unavailable")
	}
	if err := svc.PingQdrant(ctx); err == nil {
		t.Fatal("PingQdrant should fail when qdrant is unavailable")
	}
}
