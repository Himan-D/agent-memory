# Claude Code Agent Rules

## Project: Hystersis (Agent Memory System)

A competitive memory system for AI agents featuring LLM-powered inference, vector storage, and knowledge graphs. Primary differentiator vs Mem0 is the **proprietary compression engine**.

## Architecture

### Core Components
- **Memory Service** (`internal/memory/service.go`): Core memory operations
- **Memory Processor** (`internal/memory/processor.go`): LLM-based memory inference
- **LLM Templates** (`internal/memory/templates.go`): Prompt templates for extraction, entity detection, conflict resolution
- **Vector Providers** (`internal/vector/providers.go`): Multi-provider vector storage (Qdrant, Pinecone, etc.)
- **API Server** (`cmd/server/api.go`): REST endpoints
- **Compression Engine** (`internal/compression/`): PROPRIETARY - Primary competitive advantage

### Compression Engine (PROPRIETARY - CRITICAL)
This is the core differentiator. Implementation must follow these rules:

1. **What is PROPRIETARY** (NOT open source):
   - ProMem extraction algorithm (`internal/compression/extractor/`)
   - Spreading activation retrieval (`internal/compression/retrieval/`)
   - LLM routing logic (`internal/compression/llm/router.go`)
   - Hyperparameter tuning (decay=0.85, threshold=0.1, hops=3)

2. **What is OPEN SOURCE**:
   - Basic compression.go (summarization)
   - Vector similarity search
   - Neo4j storage
   - API endpoints

3. **Feature Priority**:
   - P1: ProMem Extraction (97%+ accuracy)
   - P1: Spreading Activation (+23% multi-hop)
   - P2: Async Pipeline (<5ms write impact)
   - P2: Tiered Memory (Working→Hot→Cold→Archive)
   - P3: Observability

### Configuration
All config via `internal/config/config.go`:
- LLMConfig: provider, api_key, model, base_url
- MemoryConfig: processing_enabled, auto_extract_facts
- Vector providers: Pinecone, Qdrant, etc.
- **NEW**: CompressionConfig: fast_provider, verify_provider, complexity_threshold

## Implementation Guidelines

### Adding New Features
1. Follow existing patterns in the codebase
2. Use proper error wrapping: `fmt.Errorf("feature: operation: %w", err)`
3. Add tests for new functionality
4. Update AGENTS.md if adding new conventions
5. **For compression features**: Ensure PROPRIETARY boundary is maintained

### Memory Processing Flow
1. Content received → MemoryProcessor.ExtractFacts() via LLM
2. Facts → determine importance score
3. Entities → extract and link
4. Conflicts → ResolveConflict template if needed

### Compression Flow (NEW)
1. Memory received → Route via LLMRouter (fast or verify path)
2. ProMem extraction → Self-questioning + verification
3. Spreading activation → Graph propagation + ranking
4. Async pipeline → Non-blocking background processing
5. Tiered storage → Working→Hot→Cold→Archive

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
- **IMPORTANT**: Do NOT commit proprietary compression code in public repos

## Environment Variables (Compression)
```bash
COMPRESSION_ENABLED=true
COMPRESSION_LLM_FAST_PROVIDER=openai
COMPRESSION_LLM_FAST_MODEL=gpt-4o-mini
COMPRESSION_LLM_VERIFY_PROVIDER=anthropic
COMPRESSION_LLM_VERIFY_MODEL=claude-3-5-sonnet
COMPRESSION_COMPLEXITY_THRESHOLD=0.6
COMPRESSION_MODE=extract
TIER_POLICY=balanced
REDIS_URL=redis://localhost:6379
```

## Key References
- ProMem: arXiv:2601.04463
- Synapse: arXiv:2601.02744
- Target: 97%+ accuracy, 80-85% token reduction, +23% multi-hop