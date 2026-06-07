package memory

import (
	"context"
	"testing"

	"agent-memory/internal/config"
)

func TestNewServiceWithoutNeo4jUsesNilGraphInterface(t *testing.T) {
	cfg := config.Load()
	cfg.Neo4j.URI = "bolt://127.0.0.1:1"
	cfg.Neo4j.User = "neo4j"
	cfg.Neo4j.Password = "invalid"
	cfg.Qdrant.URL = "http://127.0.0.1:1"

	svc, err := NewService(cfg)
	if err != nil {
		t.Fatalf("NewService returned error: %v", err)
	}
	defer svc.Close()

	if svc.graph != nil {
		t.Fatal("expected graph interface to be nil when neo4j is unavailable")
	}
	if svc.vector != nil {
		t.Fatal("expected vector interface to be nil when qdrant is unavailable")
	}

	if err := svc.PingNeo4j(context.Background()); err == nil {
		t.Fatal("expected PingNeo4j error when neo4j is unavailable")
	}
	if err := svc.PingQdrant(context.Background()); err == nil {
		t.Fatal("expected PingQdrant error when qdrant is unavailable")
	}
}
