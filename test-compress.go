package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/pprof"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/rs/cors"

	"agent-memory/cmd/server"
	"agent-memory/internal/config"
	"agent-memory/internal/llm"
	"agent-memory/internal/compression/extractor"
)

func main() {
	_ = godotenv.Load()
	cfg := config.Load()
	
	log.Printf("LLM API Key loaded: %v\n", cfg.LLM.APIKey != "")
	
	if cfg.LLM.APIKey == "" {
		log.Fatal("LLM_API_KEY not set!")
	}
	
	// Test compression directly
	ctx := context.Background()
	
	// Create LLM client
	llmClient, err := llm.NewProvider(&llm.Config{
		Provider: llm.ProviderType(cfg.LLM.Provider),
		APIKey:   cfg.LLM.APIKey,
	})
	if err != nil {
		log.Fatalf("Failed to create LLM provider: %v", err)
	}
	log.Println("LLM Provider created")
	
	// Create extractor
	memExtractor := extractor.NewMemoryExtractor(llmClient)
	log.Println("Extractor created")
	
	// Run extraction
	testText := "machine learning is a subset of artificial intelligence"
	log.Printf("Testing extraction with: %s\n", testText)
	
	start := time.Now()
	result, err := memExtractor.Extract(ctx, testText)
	elapsed := time.Since(start)
	
	log.Printf("Extraction took: %v", elapsed)
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		log.Printf("Result: %+v", result)
	}
	
	// Also start HTTP server with pprof
	go func() {
		log.Println("Starting pprof server on :8081")
		http.ListenAndServe(":8081", nil)
	}()
	
	log.Fatal(server.Run())
}