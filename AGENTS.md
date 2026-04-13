# Agent Memory - Developer Guide

## Build & Test Commands

```bash
go build ./...     # Build all packages
go test ./...      # Run all tests
go run ./cmd/server # Run API server
go run ./cmd/agent  # Run CLI agent
```

## Project Structure

```
cmd/
  server/api.go     # REST API server (~2600 lines)
  agent/            # CLI agent harness
  cli/              # CLI commands
internal/
  memory/service.go  # Core memory service (~2500 lines)
  memory/neo4j/      # Neo4j graph implementation (~3500 lines)
  memory/types/      # Type definitions
  skills/            # Skills registry system
  sso/               # SSO providers (OIDC, SAML, LDAP)
  reranker/          # Reranking (Cohere + LLM)
  audit/             # Audit logging middleware
  analytics/         # Dashboard analytics
  webhook/           # Webhook system
  config/            # Configuration
```

## Architecture

### Memory Flow
1. Content → `MemoryProcessor.ExtractFacts()` via LLM
2. Facts → importance score
3. Entities → extract and link in graph
4. Conflicts → `ResolveConflict` template

### Key Interfaces
- `GraphStore` (`internal/memory/store.go`): Neo4j operations
- `VectorStore` (`internal/memory/store.go`): Qdrant operations
- `Provider` (`internal/llm/provider.go`): LLM completions

## Critical Conventions

### Error Handling
- Use `safeHTTPError()` not `http.Error(w, err.Error(), ...)` in API handlers
- Error wrapping: `fmt.Errorf("feature: operation: %w", err)`

### Rate Limiter
- Has cleanup goroutine - call `Stop()` on shutdown
- Located at `cmd/server/api.go`

### Neo4j Timeouts
- Configurable via `NEO4J_QUERY_TIMEOUT` env var (default 60s)
- Use `c.queryTimeout()` helper method

### Batch Operations
- Use `GetMemoriesByIDs()` instead of loops
- Use `BatchCreateMemories()` / `BatchDeleteMemories()` for bulk

## SSO Providers

| Provider | Status | File |
|---------|--------|------|
| OIDC | ✅ Full | `internal/sso/oidc.go` |
| SAML | ✅ Full | `internal/sso/saml.go` |
| LDAP | ✅ Full | `internal/sso/ldap.go` |

## Features vs Mem0

Agent-memory exceeds Mem0 Pro/Enterprise:

| Feature | Mem0 | Agent-Memory |
|---------|------|--------------|
| Graph Memory | Pro | ✅ |
| Analytics | Pro | ✅ Advanced |
| SSO | Enterprise | ✅ OIDC+SAML+LDAP |
| Audit Logs | Enterprise | ✅ Full |
| Memory Versioning | ❌ | ✅ |
| Skill Chains | ❌ | ✅ |
| 85% Compression | 80% | ✅ |
| LDAP SSO | ❌ | ✅ |

## Adding New Features

### New LLM Template
Add template in `internal/memory/templates.go`

### New Memory Processor Method
Add method in `internal/memory/processor.go`

### New Vector Provider
1. Add `ProviderType` in `internal/vector/provider.go`
2. Add implementation in `internal/vector/providers.go`

### New API Endpoint
Add handler in `cmd/server/api.go` with `safeHTTPError()`

## Environment Variables

```bash
# Neo4j
NEO4J_URI=bolt://localhost:7687
NEO4J_USER=neo4j
NEO4J_PASSWORD=secret
NEO4J_QUERY_TIMEOUT=60

# Qdrant
QDRANT_URL=http://localhost:6333

# LLM
LLM_PROVIDER=openai
LLM_API_KEY=sk-...
```

## Stubs to Complete

- None - all major features implemented

## CLI Agent Commands

Run with `go run ./cmd/agent`:

```
/help           - Show help
/agents        - List agents
/init [prompt] - Initialize memory
/remember [txt] - Save memory
/memory        - View context
/search [query]- Search memories
/compact       - Compact context
/skills        - List skills
/exit          - Quit
```
