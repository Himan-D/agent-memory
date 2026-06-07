package memory

import (
	"context"
	"testing"

	"agent-memory/internal/config"
)

func TestPingNeo4jWhenUnavailable(t *testing.T) {
	cfg := config.Load()
	cfg.Neo4j.URI = "bolt://127.0.0.1:1" // unlikely to be listening
	cfg.Neo4j.User = "neo4j"
	cfg.Neo4j.Password = "test"
	cfg.Qdrant.URL = "http://127.0.0.1:1"

	svc, err := NewService(cfg)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	defer svc.Close()

	if svc.graph != nil {
		t.Fatal("graph interface should be nil when neo4j is unavailable")
	}

	err = svc.PingNeo4j(context.Background())
	if err == nil {
		t.Fatal("expected error when neo4j is not configured")
	}
}

func TestPingQdrantWhenUnavailable(t *testing.T) {
	cfg := config.Load()
	cfg.Neo4j.URI = "bolt://127.0.0.1:1"
	cfg.Neo4j.User = "neo4j"
	cfg.Neo4j.Password = "test"
	cfg.Qdrant.URL = "http://127.0.0.1:1"

	svc, err := NewService(cfg)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	defer svc.Close()

	if svc.vector != nil {
		t.Fatal("vector interface should be nil when qdrant is unavailable")
	}

	err = svc.PingQdrant(context.Background())
	if err == nil {
		t.Fatal("expected error when qdrant is not configured")
	}
}
