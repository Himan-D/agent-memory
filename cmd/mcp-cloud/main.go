package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"agent-memory/internal/mcp"
)

var (
	port = flag.String("port", "8090", "Port to listen on")
)

func main() {
	flag.Parse()

	// Create memory service shim (proxies to upstream API)
	memSvc := createMemoryService()

	// Create cloud MCP server
	cloud := mcp.NewCloudServer(memSvc, ":"+*port)

	// Graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Received shutdown signal")
		shutdownCtx, shutdownCancel := context.WithTimeout(ctx, 10*time.Second)
		defer shutdownCancel()
		if err := cloud.Stop(shutdownCtx); err != nil {
			log.Printf("Shutdown error: %v", err)
		}
	}()

	log.Printf("Cloud MCP server starting on :%s", *port)
	if err := cloud.Start(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}
}

// createMemoryService creates a shim that proxies to the upstream memory API.
// The cloud MCP server acts as a gateway that adds auth, CORS, and SSE
// streaming on top of the existing REST API.
func createMemoryService() *memoryServiceShim {
	apiKey := os.Getenv("HYSTERSIS_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("MCP_API_KEY")
	}

	baseURL := os.Getenv("MEMORY_API_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8081"
	}

	return &memoryServiceShim{
		apiKey:  apiKey,
		baseURL: strings.TrimRight(baseURL, "/"),
	}
}

// memoryServiceShim proxies to the memory API server. The cloud MCP server
// acts as a gateway that adds auth, CORS, and SSE streaming on top of the
// existing REST API.
type memoryServiceShim struct {
	apiKey  string
	baseURL string
}

func (s *memoryServiceShim) BaseURL() string {
	return s.baseURL
}

func (s *memoryServiceShim) APIKey() string {
	return s.apiKey
}

func init() {
	// Set defaults from environment
	if p := os.Getenv("PORT"); p != "" {
		_ = flag.Set("port", p)
	}
}

// Compile-time interface checks.
var (
	_ mcp.CloudService = (*memoryServiceShim)(nil)
	_ fmt.Stringer     = (*memoryServiceShim)(nil)
)

func (s *memoryServiceShim) String() string {
	return fmt.Sprintf("MemoryServiceShim(baseURL=%s)", s.baseURL)
}
