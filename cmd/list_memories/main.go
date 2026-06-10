package main

import (
	"fmt"
	"log"
	"os"

	"agent-memory/internal/config"
	"agent-memory/internal/memory/neo4j"
)

func main() {
	cfg := config.Load()
	if os.Getenv("NEO4J_PASSWORD") != "" {
		cfg.Neo4j.Password = os.Getenv("NEO4J_PASSWORD")
	} else {
		cfg.Neo4j.Password = "password123"
	}
	client, err := neo4j.NewClient(cfg.Neo4j)
	if err != nil {
		log.Fatalf("connect neo4j: %v", err)
	}
	defer client.Close()

	memories, err := client.GetMemoriesByOrg("benchmark")
	if err != nil {
		log.Fatalf("get memories: %v", err)
	}

	fmt.Printf("--- Total benchmark memories: %d ---\n", len(memories))
	for _, m := range memories {
		fmt.Printf("ID: %s | User: %s | Org: %s | Content: %s\n", m.ID, m.UserID, m.OrgID, m.Content)
	}
}
