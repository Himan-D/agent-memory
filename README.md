# Hystersis

> Persistent memory infrastructure for AI agents. Remember more, forget less.

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

## Why Hystersis?

AI agents forget everything after each conversation. **Hystersis** gives your agents persistent memory that compounds over time:

- **Memory That Adapts** - Intelligence that grows with every interaction
- **Knowledge Graphs** - Understand relationships between entities  
- **Semantic Search** - Find similar information using vector embeddings
- **Multi-Agent Support** - Shared memory across agent teams
- **Procedural Memory** - Extract and reuse skills from interactions
- **Enterprise Ready** - SSO, audit logs, compliance

## Architecture

```
┌─────────────┐     ┌─────────────────┐     ┌────────────┐
│   AI Agent  │────▶│    Hystersis    │────▶│   Neo4j    │
└─────────────┘     │   (This Server)  │     │  (Graph)   │
                    └─────────────────┘     └────────────┘
                             │
                             │
                    ┌─────────▼─────────┐
                    │      Qdrant       │
                    │    (Vectors)      │
                    └──────────────────┘
```

**How it works:**

1. **Store** - Agent stores messages, entities, and relationships
2. **Embed** - Content is embedded using OpenAI (or custom provider)
3. **Search** - Query using natural language, get semantically similar results
4. **Graph** - Traverse relationships to find connected information
5. **Feedback** - Signals help improve future searches

## Quick Start

```bash
# Install CLI
curl -fsSL https://raw.githubusercontent.com/Himan-D/agent-memory/main/install.sh | bash

# Or with custom options
VERSION=v0.1.0 INSTALL_DIR=$HOME/.hystersis curl -fsSL ... | bash
```

### Python SDK

```python
from agentmemory import AgentMemory

# Connect to your Hystersis server
client = AgentMemory("https://api.yourserver.com", api_key="your-key")

# Create a session for your agent
session = client.create_session(agent_id="assistant-bot")

# Add conversation messages
client.add_message(session["id"], "user", "I love machine learning!")
client.add_message(session["id"], "assistant", "That's great! What type?")
client.add_message(session["id"], "user", "Especially neural networks and transformers")

# Store a semantic memory
memory = client.create_memory(
    content="User is interested in machine learning and AI",
    user_id="user-123",
    category="preferences"
)

# Later, search semantically
results = client.semantic_search("deep learning transformers")
# Returns: [{"score": 0.92, "content": "User is interested in...", "id": "..."}]
```

## Features

### Memory Types
- **Conversational** - Session-based message history
- **Semantic** - Vector-based similarity search with reranking  
- **Knowledge Graph** - Entities with typed relationships (Neo4j)
- **Procedural** - Extract and reuse skills from agent interactions

### Core Operations
- **CRUD** - Create, read, update, delete memories
- **Batch Operations** - Process up to 1000 memories at once
- **Filtering** - AND/OR/NOT operators, wildcards, date ranges
- **Categories** - Organize memories by custom categories
- **TTL/Expiration** - Automatic cleanup with expiration dates

### Self-Improving
- **Feedback Loop** - Positive/Negative feedback on memories
- **Reranking** - Improved search precision with reranking
- **History/Audit** - Track all modifications to memories
- **Conflict Resolution** - Automatic deduplication

### Agent Framework Integrations
- **LangChain** - Memory component, retriever, vector store
- **LlamaIndex** - Reader, index, query engine
- **CrewAI** - Shared memory for multi-agent crews
- **MCP Server** - Model Context Protocol for AI assistants

### Production Ready
- REST API with authentication
- Prometheus metrics & health checks
- Rate limiting (100 req/min per key)
- Structured JSON logging
- Graceful shutdown
- Connection pooling

## Pricing

| Tier | Price | Seats | Agents | Key Features |
|------|-------|-------|--------|--------------|
| **Self-Hosted** | Free | 1 | Unlimited | Full source, self-hosted |
| **Pro** | $29/seat/mo | 5 | Unlimited | Skills extraction, priority support |
| **Team** | $99/seat/mo | 20 | Unlimited | Collaboration, audit logs, analytics |
| **Enterprise** | Custom | Unlimited | Unlimited | SSO, SLA, compliance |

## API Endpoints

### Memory CRUD
| Endpoint | Description |
|----------|-------------|
| `POST /memories` | Create a new memory |
| `GET /memories` | List memories with filters |
| `GET /memories/{id}` | Get a specific memory |
| `PUT /memories/{id}` | Update a memory |
| `DELETE /memories/{id}` | Delete a memory |

### Batch Operations
| Endpoint | Description |
|----------|-------------|
| `POST /memories/batch` | Create up to 1000 memories |
| `PUT /memories/batch-update` | Batch update/archive/delete |

### Search
| Endpoint | Description |
|----------|-------------|
| `GET /search` | Semantic search |
| `POST /search` | Search with filters |
| `POST /search/advanced` | Advanced filter logic |

### Knowledge Graph
| Endpoint | Description |
|----------|-------------|
| `POST /entities` | Create entity |
| `GET /entities/{id}` | Get entity |
| `POST /relations` | Create relationship |
| `GET /graph/traverse/{id}` | Traverse graph |

## Use Cases

### 1. Customer Support Bot
Remembers past tickets, customer history, and resolution patterns.

### 2. Code Assistant
Remembers codebase context, similar issues, coding patterns.

### 3. Research Agent
Maintains literature graph, finds related papers, tracks findings.

### 4. Multi-Agent Teams
Shared organizational memory across agent teams with pub/sub sync.

## Integrations

### MCP (Model Context Protocol)

Hystersis supports the MCP standard for connecting to AI assistants.

```bash
# Run as MCP server
SERVER_MODE=mcp-stdio ./hystersis
```

**Available MCP Tools:**
- `add_memory` / `search_memories` / `get_memories`
- `create_entity` / `create_relation` / `get_context`
- `create_session` / `add_message`
- `add_feedback`

### LangChain

```python
from agentmemory.integrations.langchain import AgentMemoryMemory

memory = AgentMemoryMemory(
    session_id="user-123",
    memory_type="user",
    api_key="your-key",
    base_url="http://localhost:8080"
)
```

## Configuration

### Environment Variables

```bash
# Neo4j
NEO4J_URI=bolt://localhost:7687
NEO4J_USER=neo4j
NEO4J_PASSWORD=your-password

# Qdrant
QDRANT_URL=http://localhost:6333
QDRANT_API_KEY=your-key

# OpenAI (optional, for embeddings)
OPENAI_API_KEY=sk-...

# Server
HTTP_PORT=:8080

# Auth
AUTH_ENABLED=true
API_KEYS=key1:tenant1,key2:tenant2
```

## Tech Stack

- **Server**: Go 1.26+
- **Graph DB**: Neo4j
- **Vector DB**: Qdrant
- **Embeddings**: OpenAI (configurable to any provider)
- **SDK**: Python 3.9+
- **Protocol**: MCP (Model Context Protocol)

## License

MIT - See [LICENSE](./LICENSE)

---

<p align="center">
  <strong>Memory that adapts. Intelligence that compounds.</strong>
</p>
