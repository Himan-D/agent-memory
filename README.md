# Agent Memory System

> Give your AI agents permanent, semantic memory with graph relationships

<p align="center">
  <a href="https://github.com/Himan-D/agent-memory/actions">
    <img src="https://img.shields.io/github/actions/workflow/status/Himan-D/agent-memory/test.yml?branch=main" alt="Tests">
  </a>
  <a href="https://goreportcard.com/report/github.com/Himan-D/agent-memory">
    <img src="https://goreportcard.com/badge/github.com/Himan-D/agent-memory" alt="Go Report">
  </a>
  <a href="https://github.com/Himan-D/agent-memory/blob/main/LICENSE">
    <img src="https://img.shields.io/github/license/Himan-D/agent-memory" alt="License">
  </a>
  <a href="https://pypi.org/project/agentmemory/">
    <img src="https://img.shields.io/pypi/v/agentmemory" alt="PyPI">
  </a>
</p>

## Why Agent Memory?

Current AI agents forget everything after each conversation. **Agent Memory** gives your agents:

- **Persistent Context** - Remember past conversations across sessions
- **Knowledge Graphs** - Understand relationships between entities
- **Semantic Search** - Find similar information using vector embeddings
- **Multi-agent Support** - Separate memory for different agents/tenants

## How It Works

```
┌─────────────┐     ┌─────────────────┐     ┌────────────┐
│   AI Agent  │────▶│   Agent Memory   │────▶│   Neo4j    │
└─────────────┘     │   (This Server)  │     │  (Graph)   │
                   └─────────────────┘     └────────────┘
                           │
                           │
                   ┌───────▼───────┐
                   │   Qdrant     │
                   │  (Vectors)   │
                   └──────────────┘
```

1. **Store** - Agent stores messages, entities, and relationships
2. **Embed** - Content is embedded using OpenAI (or custom)
3. **Search** - Query using natural language, get semantically similar results
4. **Graph** - Traverse relationships to find connected information

## Quick Start

```bash
# One-line install
curl -fsSL https://raw.githubusercontent.com/Himan-D/agent-memory/main/install.sh | bash

# Or with custom options
VERSION=v0.1.0 INSTALL_DIR=$HOME/.agent-memory curl -fsSL ... | bash
```

### Python (Recommended)

```python
from agentmemory import AgentMemory

# Connect to your agent memory server
client = AgentMemory("https://api.yourserver.com", api_key="your-key")

# Create a session for your agent
session = client.create_session(agent_id="assistant-bot")

# Add conversation messages
client.add_message(session["id"], "user", "I love machine learning!")
client.add_message(session["id"], "assistant", "That's great! What type?")
client.add_message(session["id"], "user", "Especially neural networks and transformers")

# Later, search semantically
results = client.semantic_search("deep learning transformers")
# Returns: [{"score": 0.92, "content": "I love machine learning!...", "id": "..."}]
```

## Key Features

### Memory Types
- **Conversational** - Session-based message history
- **Knowledge Graph** - Entities with typed relationships
- **Semantic** - Vector-based similarity search

### Production Ready
- REST API with authentication
- Prometheus metrics
- Health checks for all dependencies
- Rate limiting (100 req/min per key)
- Structured JSON logging
- Graceful shutdown

### Deployment Options
- **Docker** - Single command deployment
- **Kubernetes** - Production-grade manifests
- **Helm** - One-click cluster installation

## API Endpoints

| Endpoint | Description |
|----------|-------------|
| `POST /sessions` | Create a new agent session |
| `POST /sessions/{id}/messages` | Add a message |
| `GET /sessions/{id}/messages` | Get conversation history |
| `POST /entities` | Create a knowledge graph entity |
| `GET /entities/{id}` | Get entity with relationships |
| `POST /relations` | Connect entities with relationships |
| `GET /search` | Semantic vector search |
| `POST /graph/query` | Raw Cypher query (admin only) |

## Use Cases

### 1. Customer Support Bot
Remembers past tickets, customer history, and resolution patterns.

### 2. Code Assistant
Remembers codebase context, similar issues, and coding patterns.

### 3. Research Agent
Maintains literature graph, finds related papers, tracks findings.

### 4. Personal Assistant
Remember conversations, preferences, and important dates.

## Security

- API key authentication (configurable)
- Admin keys for advanced operations
- Rate limiting prevents abuse
- Input validation prevents injection
- Tenant isolation for multi-tenant deployments
- No default secrets (must be provided)

## Performance

- Connection pooling (50 connections)
- Batch message buffering
- Vector search (sub-100ms)
- Graph queries optimized with indexes

## Pricing

### Self-Hosted (Free)
- Full source code
- Docker, K8s, Helm charts
- Community support

### Cloud (Coming Soon)
- Managed hosting
- Auto-scaling
- 99.9% SLA
- Priority support
- Advanced analytics

**Contact**: For enterprise pricing, email `hello@agentmemory.io`

## Documentation

- [Quick Start Guide](./QUICKSTART.md)
- [API Reference](./docs/openapi.yaml)
- [Python SDK Docs](./sdk/python/README.md)
- [Deployment Guide](./deploy/)

## Tech Stack

- **Server**: Go 1.26
- **Graph DB**: Neo4j
- **Vector DB**: Qdrant
- **Embeddings**: OpenAI (configurable)
- **SDK**: Python 3.9+

## License

MIT - See [LICENSE](./LICENSE)

---

<p align="center">
  <strong>Give your agents memory. Build smarter products.</strong>
</p>