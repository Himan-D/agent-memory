package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/joho/godotenv/autoload"

	"agent-memory/internal/config"
	"agent-memory/internal/llm"
	"agent-memory/internal/compression/extractor"
)

func main() {
	_ = godotenv.Load()
	cfg := config.Load()
	
	fmt.Printf("API Key present: %v\n", len(cfg.LLM.APIKey) > 0)
	
	if cfg.LLM.APIKey == "" {
		fmt.Println("ERROR: No API key!")
		os.Exit(1)
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	
	// Create provider
	fmt.Println("Creating LLM provider...")
	t0 := time.Now()
	llmClient, err := llm.NewProvider(&llm.Config{
		Provider: llm.ProviderType(cfg.LLM.Provider),
		APIKey:   cfg.LLM.APIKey,
	})
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	fmt.Printf("Provider created in %v\n", time.Since(t0))
	
	// Create extractor
	fmt.Println("Creating extractor...")
	t1 := time.Now()
	memExtractor := extractor.NewMemoryExtractor(llmClient)
	fmt.Printf("Extractor created in %v\n", time.Since(t1))
	
	// Run extraction
	testText := "machine learning is a subset of artificial intelligence that enables computers to learn from data"
	fmt.Printf("\nExtracting: %s\n", testText)
	
	t2 := time.Now()
	result, err := memExtractor.Extract(ctx, testText)
	fmt.Printf("Extraction took: %v\n", time.Since(t2))
	
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else if result != nil {
		fmt.Printf("\nFacts extracted (%d):\n", len(result.Facts))
		for _, f := range result.Facts {
			fmt.Printf("  - %s\n", f.Fact)
		}
	}
	
	fmt.Println("\n=== Test Complete ===")
}