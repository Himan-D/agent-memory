package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"

	"agent-memory/internal/config"
	"agent-memory/internal/memory"
	"agent-memory/internal/project"
	"agent-memory/internal/sync"
	"agent-memory/internal/webhook"
)

func main() {
	godotenv.Load()

	cfg := config.Load()
	log.Println("=== Agent Memory System ===")
	log.Printf("Neo4j:  %s", cfg.Neo4j.URI)
	log.Printf("Qdrant: %s", cfg.Qdrant.URL)

	memSvc, err := memory.NewService(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize memory service: %v", err)
	}
	defer memSvc.Close()

	projSvc := project.NewService(cfg)
	whSvc := webhook.NewService(cfg)

	mode := os.Getenv("SERVER_MODE")
	if mode == "mcp-stdio" {
		log.Println("Starting MCP server (stdio mode)...")
		RunMCPServer(memSvc, projSvc, whSvc)
		return
	}

	log.Printf("HTTP:   %s", cfg.App.HTTPPort)
	log.Println()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	syncer := sync.New(memSvc, cfg.App.SyncInterval, cfg.App.BatchSize)
	go syncer.Start(ctx)

	apiServer := NewAPIServer(cfg, memSvc, projSvc, whSvc)

	go func() {
		if err := apiServer.RunUntilShutdown(); err != nil {
			log.Printf("API server error: %v", err)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("Shutting down...")
	syncer.Stop()
	cancel()
	apiServer.Stop()
	log.Println("Goodbye!")
}
