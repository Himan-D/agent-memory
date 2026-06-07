# Hystersis Agent API Reference

This document provides a machine-readable reference for AI agents to discover and interact with the Hystersis memory platform.

## Overview

Hystersis is a persistent memory infrastructure for AI agents. It provides:
- **Memory Storage**: Create, search, update, and delete memories with semantic search
- **Knowledge Graph**: Entity-relationship storage with Neo4j-backed graph traversal
- **Skills System**: Procedural memory with LLM-powered extraction, suggestion, and execution
- **Compression Engine**: Proprietary ProMem extraction and spreading activation retrieval
- **Document Extraction**: PDF, HTML, image, audio, and text file processing
- **Multi-Agent Sync**: Real-time memory sharing between agents via Redis pub/sub

## Authentication

All requests require an `X-API-Key` header:

```
X-API-Key: your-api-key-here
```

Create keys: `POST /api-keys` or `POST /admin/api-keys`

## Content Types

All request/response bodies are JSON: `Content-Type: application/json`

File uploads use multipart form data: `Content-Type: multipart/form-data`

## Quick Start Examples

### Python SDK

```python
from hystersis import Hystersis

client = Hystersis(base_url="https://api.hystersis.com", api_key="your-key")

# Add a memory
memory = client.memories.add(
    content="User prefers dark mode and concise responses",
    agent_id="my-agent",
    metadata={"source": "chat", "confidence": 0.95}
)

# Search memories
results = client.search("user preferences", limit=10, threshold=0.7)

# Enhanced search with spreading activation
results = client.search_enhanced("user preferences", mode="spreading")

# Hybrid search (semantic + keyword)
results = client.search_hybrid(query="user preferences", semantic_weight=0.7, keyword_weight=0.3)

# Create an entity
entity = client.entities.create(
    name="UserService",
    entity_type="Class",
    properties={"file": "user.py", "language": "python"}
)

# Create a relation
client.relations.create(
    from_id="entity-1",
    to_id="entity-2",
    relation_type="DEPENDS_ON"
)

# Extract skills from content
skills = client.skills.extract(content="When the server returns 429, wait exponential backoff...")
```

### Node.js SDK

```javascript
import { Hystersis } from 'hystersis';

const client = new Hystersis({ baseUrl: 'https://api.hystersis.com', apiKey: 'your-key' });

// Add a memory
const memory = await client.memories.add({
  content: 'User prefers dark mode and concise responses',
  agentId: 'my-agent'
});

// Search
const results = await client.search('user preferences');

// Enhanced search
const results = await client.searchEnhanced('user preferences', { mode: 'spreading' });

// Extract from documents
const doc = await client.documents.extract(file, 'application/pdf');
```

### REST API

```bash
# Create API key
curl -X POST https://api.hystersis.com/api-keys \
  -H "Content-Type: application/json" \
  -d '{"name": "my-app", "tenant_id": "default"}'

# Add memory
curl -X POST https://api.hystersis.com/memories \
  -H "X-API-Key: your-key" \
  -H "Content-Type: application/json" \
  -d '{"content": "User prefers dark mode", "agent_id": "my-agent"}'

# Search
curl "https://api.hystersis.com/search?query=user+preferences&limit=10" \
  -H "X-API-Key: your-key"

# Hybrid search
curl -X POST https://api.hystersis.com/search/hybrid \
  -H "X-API-Key: your-key" \
  -H "Content-Type: application/json" \
  -d '{"query": "user preferences", "semantic_weight": 0.7, "keyword_weight": 0.3}'

# Extract from PDF
curl -X POST https://api.hystersis.com/documents/extract \
  -H "X-API-Key: your-key" \
  -F "file=@document.pdf"

# Spreading activation search
curl "https://api.hystersis.com/search/enhanced?query=user+preferences&mode=spreading" \
  -H "X-API-Key: your-key"
```

### MCP Server (for Claude Code, Cursor, etc.)

```bash
npx hystersis-mcp --api-key YOUR_KEY
```

Available tools: add_memory, search_memories, get_memories, create_entity, create_relation, create_skill, suggest_skills, execute_skill, create_agent, create_agent_group, share_memory_to_group, add_feedback, list_agents, and more.

## Key Data Models

### Memory

```json
{
  "id": "mem_abc123",
  "content": "User prefers dark mode",
  "agent_id": "my-agent",
  "importance": "high",
  "metadata": {"source": "chat"},
  "created_at": "2025-01-15T10:00:00Z",
  "updated_at": "2025-01-15T10:00:00Z"
}
```

### Entity

```json
{
  "id": "ent_abc123",
  "name": "UserService",
  "type": "Class",
  "properties": {"file": "user.py"},
  "created_at": "2025-01-15T10:00:00Z"
}
```

### Skill

```json
{
  "id": "skill_abc123",
  "name": "Error Recovery",
  "domain": "engineering",
  "trigger": "server returns 429 or 5xx",
  "action": "implement exponential backoff and retry",
  "confidence": 0.92,
  "usage_count": 15
}
```

### Search Results

```json
{
  "results": [
    {
      "id": "mem_abc123",
      "content": "User prefers dark mode and concise responses",
      "score": 0.94,
      "source": "vector",
      "importance": "high"
    }
  ],
  "count": 1
}
```

### Compression Stats

```json
{
  "accuracy_retention": 0.973,
  "token_reduction": 0.84,
  "total_tokens_saved": 1500000,
  "extractions_performed": 450,
  "spreading_activations": 230,
  "avg_latency_ms": 187,
  "p95_latency_ms": 245
}
```

## Compression Modes

| Mode | Accuracy | Reduction | Use Case |
|------|----------|-----------|----------|
| extract | 97%+ | 80-85% | Default - highest accuracy |
| balanced | 95%+ | 85-90% | Production - good balance |
| aggressive | 92%+ | 90-93% | High-volume - max compression |

## Tier Policies

| Policy | Hot Retention | Use Case |
|--------|--------------|----------|
| aggressive | 1 day | High-volume, ephemeral |
| balanced | 7 days | Default production |
| conservative | 30 days | Compliance, long-term |

## Search Modes

| Mode | Method | Description |
|------|--------|-------------|
| vector | GET /search | Standard vector similarity |
| spreading | GET /search/enhanced?mode=spreading | Graph propagation + vector |
| hybrid | GET /search/enhanced?mode=hybrid | Spreading + vector merged |
| POST hybrid | POST /search/hybrid | Semantic + keyword weighted |

## Supported Document Types

| Type | MIME | Notes |
|------|------|-------|
| Text | text/plain, text/markdown | Direct extraction |
| HTML | text/html | Strips scripts/styles, extracts metadata |
| PDF | application/pdf | Stream-based extraction |
| Images | image/png, image/jpeg, image/gif, image/webp | Metadata extraction (OCR via Whisper API) |
| Audio | audio/wav, audio/mp3, audio/mpeg, audio/ogg, audio/flac | Metadata extraction (transcription via Whisper API) |

## Error Responses

All errors follow this format:

```json
{
  "error": "invalid request",
  "message": "query required",
  "status": 400
}
```

Common status codes: 400 (bad request), 401 (unauthorized), 404 (not found), 429 (rate limited), 500 (internal error)

## Rate Limits

Default: 100 requests/minute per API key. Rate limit headers included in responses:
- `X-RateLimit-Limit`: Request limit per minute
- `X-RateLimit-Remaining`: Remaining requests
- `X-RateLimit-Reset`: Seconds until limit resets