//go:build linux

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"agent-memory/internal/fs/fuse"
	"agent-memory/internal/fs/vfs"
)

func main() {
	// Parse command-line flags
	mountPoint := flag.String("mount", "/mnt/agent", "Mount point for agentfs")
	apiKey := flag.String("api-key", "", "API key for memory service")
	mode := flag.String("mode", "fuse", "Mount mode: fuse, nfs, webdav")
	umount := flag.Bool("umount", false, "Unmount the filesystem")
	status := flag.Bool("status", false, "Show filesystem status")
	flag.Parse()

	if *umount {
		if err := fuse.Unmount(*mountPoint); err != nil {
			log.Fatalf("Failed to unmount: %v", err)
		}
		fmt.Printf("Successfully unmounted %s\n", *mountPoint)
		return
	}

	if *status {
		fmt.Println("Status: Not implemented yet")
		return
	}

	// Validate inputs
	if *mountPoint == "" {
		fmt.Fprintln(os.Stderr, "Mount point is required")
		flag.Usage()
		os.Exit(1)
	}

	if *apiKey == "" {
		*apiKey = os.Getenv("AGENT_MEMORY_API_KEY")
	}
	if *apiKey == "" {
		log.Fatal("API key is required (use -api-key or AGENT_MEMORY_API_KEY)")
	}

	// Create service interface (placeholder - would connect to real service)
	svc := createService(*apiKey)

	// Mount based on mode
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals for graceful unmount
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("Received shutdown signal...")
		cancel()
	}()

	switch *mode {
	case "fuse":
		log.Printf("Mounting agentfs via FUSE at %s...", *mountPoint)
		if err := fuse.Mount(*mountPoint, svc); err != nil {
			log.Fatalf("FUSE mount failed: %v", err)
		}
		fmt.Printf("agentfs mounted at %s (FUSE)\n", *mountPoint)
		fmt.Println("Press Ctrl+C to unmount...")

	case "nfs":
		fmt.Println("NFS mode not yet implemented")
		os.Exit(1)

	case "webdav":
		fmt.Println("WebDAV mode not yet implemented")
		os.Exit(1)

	default:
		fmt.Fprintf(os.Stderr, "Unknown mode: %s (use: fuse, nfs, webdav)\n", *mode)
		os.Exit(1)
	}

	// Wait for shutdown
	<-ctx.Done()
	log.Println("agentfs stopped")
}

// createService creates a service interface
// In production, this would connect to the real memory service
func createService(apiKey string) vfs.ServiceInterface {
	// Placeholder - returns nil interface
	// In production:
	// 1. Connect to Neo4j + Qdrant
	// 2. Initialize memory.Service
	// 3. Return the service
	log.Printf("Created service with API key: %s...", apiKey[:min(8, len(apiKey))])
	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
