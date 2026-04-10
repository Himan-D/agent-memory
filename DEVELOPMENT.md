# Agent Memory System - Development Guide

## Overview
Agent Memory is a Go-based memory system for AI agents with LLM-powered inference, vector storage (Qdrant, Pinecone), and Neo4j knowledge graphs.

## Quick Start
```bash
go build ./...     # Build
go test ./...      # Test
go run ./cmd/server # Run server
```

## Architecture

### Core Components
- **Memory Service** (`internal/memory/service.go`): Core memory CRUD operations
- **Memory Processor** (`internal/memory/processor.go`): LLM-based memory inference
- **LLM Templates** (`internal/memory/templates.go`): Prompt templates for extraction, entity detection, conflict resolution
- **Memory Compression** (`internal/memory/compression.go`): 85% token reduction engine
- **Async Client** (`internal/memory/async.go`): High-concurrency async operations
- **Vector Providers** (`internal/vector/providers.go`): Multi-provider vector storage
- **API Server** (`cmd/server/api.go`): REST endpoints

### Configuration
All config via `internal/config/config.go`:
- LLMConfig: provider, api_key, model, base_url, retry settings
- MemoryConfig: processing_enabled, auto_extract_facts, cache settings
- CompactionConfig: interval, thresholds, archive rules

## Key Features

### Memory Types
- **Conversation Memory**: Short-term within sessions
- **Session Memory**: Multi-step flows
- **User Memory**: Long-term personalization
- **Organizational Memory**: Shared knowledge

### Memory Processing (LLM-based)
1. Content → ExtractFacts via LLM
2. Facts → Determine importance score
3. Entities → Extract and link
4. Conflicts → ResolveConflict template

### Memory Compression
- 85% token reduction (exceeds Mem0's 80%)
- Importance-weighted retention
- Key point extraction
- Hierarchical summarization

### Search Features
- Semantic search with vector embeddings
- Keyword search
- Hybrid search (semantic + keyword combined)
- Reranking support (Cohere)
- Filters with AND/OR/NOT logic

## Adding New Features

### 1. New LLM Template
Add template in `templates.go`:
```go
const templateName = `...{{.Content}}...`

func (r *PromptRenderer) RenderTemplate(content string) string {
    // implementation
}
```

### 2. New Memory Processor Method
Add method in `processor.go`:
```go
func (p *MemoryProcessor) NewMethod(ctx context.Context, ...) (*Result, error) {
    if p.llmProvider == nil {
        return nil, fmt.Errorf("llm provider required")
    }
    // use promptRenderer.RenderXxx() and p.llmProvider.Complete()
}
```

### 3. New Vector Provider
1. Add ProviderType in `provider.go`
2. Add provider struct and methods in `providers.go`
3. Add constructor and initialization in `newProvider()` switch

### 4. New API Endpoint
Add handler in `cmd/server/api.go`:
```go
func handleNewEndpoint(w http.ResponseWriter, r *http.Request) {
    var req RequestType
    json.NewDecoder(r.Body).Decode(&req)
    // process and respond
}
```

## Testing
```bash
go test -v ./internal/memory/...    # Memory package tests
go test -v ./cmd/server/...         # API tests
go test ./...                       # All tests
```

## Environment Variables
```bash
# Neo4j
NEO4J_URI=bolt://localhost:7687
NEO4J_USER=neo4j
NEO4J_PASSWORD=secret

# Qdrant
QDRANT_URL=http://localhost:6333
QDRANT_COLLECTION=agent_memory

# LLM
LLM_PROVIDER=openai
LLM_API_KEY=sk-...
LLM_MODEL=gpt-4o

# Memory Processing
MEMORY_PROCESSING_ENABLED=true
MEMORY_AUTO_EXTRACT_FACTS=true
MEMORY_CACHE_ENABLED=true

# Compaction
COMPACTION_ENABLED=true
COMPACTION_INTERVAL=24h
COMPACTION_SIMILARITY_THRESHOLD=0.92
```

## Performance Tips
1. Use async client for batch operations
2. Enable compression for large memory stores
3. Use importance levels to prioritize retention
4. Configure appropriate batch sizes for your workload

## Monitoring
- `/health` - Health check
- `/ready` - Readiness probe
- `/metrics` - Prometheus metrics
