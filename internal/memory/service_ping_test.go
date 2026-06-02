package memory

import (
	"context"
	"testing"

	"agent-memory/internal/config"
)

// TestPingNeo4jWithoutStores verifies that a service started without Neo4j
// does not panic when PingNeo4j is called (typed-nil interface regression).
func TestPingNeo4jWithoutStores(t *testing.T) {
	cfg := config.Load()
	cfg.Neo4j.URI = "bolt://127.0.0.1:1" // unreachable; NewClient fails
	cfg.Qdrant.URL = "http://127.0.0.1:1"

	svc, err := NewService(cfg)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	defer svc.Close()

	if err := svc.PingNeo4j(context.Background()); err == nil {
		t.Fatal("expected error when neo4j is unavailable")
	}
}
