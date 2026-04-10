# Claude Code Agent Rules

## Project: Agent Memory System

A competitive memory system for AI agents featuring LLM-powered inference, vector storage, and knowledge graphs.

## Architecture

### Core Components
- **Memory Service** (`internal/memory/service.go`): Core memory operations
- **Memory Processor** (`internal/memory/processor.go`): LLM-based memory inference
- **LLM Templates** (`internal/memory/templates.go`): Prompt templates for extraction, entity detection, conflict resolution
- **Vector Providers** (`internal/vector/providers.go`): Multi-provider vector storage (Qdrant, Pinecone, etc.)
- **API Server** (`cmd/server/api.go`): REST endpoints

### Configuration
All config via `internal/config/config.go`:
- LLMConfig: provider, api_key, model, base_url
- MemoryConfig: processing_enabled, auto_extract_facts
- Vector providers: Pinecone, Qdrant, etc.

## Implementation Guidelines

### Adding New Features
1. Follow existing patterns in the codebase
2. Use proper error wrapping: `fmt.Errorf("feature: operation: %w", err)`
3. Add tests for new functionality
4. Update this file if adding new conventions

### Memory Processing Flow
1. Content received → MemoryProcessor.ExtractFacts() via LLM
2. Facts → determine importance score
3. Entities → extract and link
4. Conflicts → ResolveConflict template if needed

### Vector Providers
- Full implementations: Qdrant, Pinecone
- Stubs: Weaviate, Chroma, Milvus, Elastic, Vespa, Redis, Mongo, Pgvector
- Interface in `provider.go`, implementations in `providers.go`

## Testing
```bash
go test ./...      # Run all tests
go build ./...     # Verify build
```

## Git Workflow
- Commit message format: `feat: description`, `fix: description`, `docs: description`
- Push after successful build and tests
