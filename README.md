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
- **Multi-Level Memory** - Conversation, session, user, and organizational memory
- **Self-Improving** - Feedback loops to improve memory quality over time
- **Multi-Agent Support** - Separate memory for different agents/tenants

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
5. **Feedback** - User signals help improve future searches

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

# Store a semantic memory
memory = client.create_memory(
    content="User is interested in machine learning and AI",
    user_id="user-123",
    category="preferences"
)

# Add feedback to improve future searches
client.add_feedback(memory["id"], "positive")

# Later, search semantically
results = client.search("deep learning transformers")
# Returns: [{"score": 0.92, "content": "User is interested in...", "id": "..."}]
```

## Key Features

### Memory Types
- **Conversational** - Session-based message history
- **Semantic** - Vector-based similarity search with reranking
- **Knowledge Graph** - Entities with typed relationships (Neo4j)
- **Multi-level** - User, Organization, Agent, and Session memory

### Memory Operations
- **CRUD** - Create, read, update, delete memories
- **Batch Operations** - Process up to 1000 memories at once
- **Filtering** - AND/OR/NOT operators, wildcards, date ranges
- **Categories** - Organize memories by custom categories
- **Immutable Memories** - Flag memories that can't be modified
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

## API Endpoints

### Memory CRUD
| Endpoint | Description |
|----------|-------------|
| `POST /memories` | Create a new memory |
| `GET /memories` | List memories with filters |
| `GET /memories/{id}` | Get a specific memory |
| `PUT /memories/{id}` | Update a memory |
| `DELETE /memories/{id}` | Delete a memory |
| `GET /memories/{id}/history` | Get memory modification history |

### Batch Operations
| Endpoint | Description |
|----------|-------------|
| `POST /memories/batch` | Create up to 1000 memories |
| `PUT /memories/batch-update` | Batch update/archive/delete |
| `DELETE /memories/bulk-delete` | Bulk delete by filter |

### Search
| Endpoint | Description |
|----------|-------------|
| `GET /search` | Semantic search |
| `POST /search` | Search with filters |
| `POST /search/advanced` | Advanced filter logic |

### Feedback
| Endpoint | Description |
|----------|-------------|
| `POST /feedback` | Add feedback to a memory |
| `GET /feedback/memories` | Get memories by feedback |

### Sessions & Context
| Endpoint | Description |
|----------|-------------|
| `POST /sessions` | Create a session |
| `POST /sessions/{id}/messages` | Add a message |
| `GET /sessions/{id}/messages` | Get conversation history |
| `GET /sessions/{id}/context` | Get context for LLM |

### Knowledge Graph
| Endpoint | Description |
|----------|-------------|
| `POST /entities` | Create entity |
| `GET /entities/{id}` | Get entity |
| `POST /relations` | Create relationship |
| `GET /entities/{id}/relations` | Get entity relations |
| `GET /graph/traverse/{id}` | Traverse graph |
| `POST /graph/query` | Raw Cypher (admin) |

### Admin
| Endpoint | Description |
|----------|-------------|
| `POST /admin/cleanup` | Cleanup expired memories |
| `POST /admin/sync` | Sync entities to vector store |
| `POST /admin/api-keys` | Create API key |
| `GET /admin/api-keys` | List API keys |

## Use Cases

### 1. Customer Support Bot
Remembers past tickets, customer history, and resolution patterns. Feedback helps prioritize important information.

### 2. Code Assistant
Remembers codebase context, similar issues, coding patterns. Search finds relevant past solutions.

### 3. Research Agent
Maintains literature graph, finds related papers, tracks findings. Multi-level memory for user/org data.

### 4. Personal Assistant
Remember conversations, preferences, important dates. Feedback refines what gets remembered.

### 5. Multi-Agent Teams
Shared organizational memory across agents. CrewAI and LangChain integrations enable crew-wide context.

## Integrations

### MCP (Model Context Protocol)

Agent Memory supports the MCP standard for connecting to AI assistants like Claude Desktop, Cursor, Windsurf, and VS Code.

```bash
# Run as MCP server
SERVER_MODE=mcp-stdio ./agent-memory
```

**Available MCP Tools:**
- `add_memory` - Save text/conversations
- `search_memories` - Semantic search with filters
- `get_memories` - List with pagination
- `get_memory` - Get by ID
- `update_memory` - Overwrite by ID
- `delete_memory` - Delete by ID
- `delete_all_memories` - Bulk delete
- `add_feedback` - Provide feedback
- `get_memory_history` - Get modification history
- `create_entity` / `create_relation` - Knowledge graph
- `create_session` / `add_message` / `get_context` - Sessions

See [MCP.md](./MCP.md) for detailed setup instructions.

### LangChain

```python
from agentmemory.integrations.langchain import AgentMemoryMemory

memory = AgentMemoryMemory(
    session_id="user-123",
    memory_type="user",
    api_key="your-key",
    base_url="http://localhost:8080"
)

# Use with LangChain conversation chain
conversation = ConversationChain(
    llm=llm,
    memory=memory,
    verbose=True
)
```

### LlamaIndex

```python
from agentmemory.integrations.llamaindex import AgentMemoryIndex

index = AgentMemoryIndex(
    user_id="user-123",
    base_url="http://localhost:8080"
)

# Query the index
retriever = index.as_retriever()
nodes = retriever.retrieve("What did I learn about AI?")
```

### OpenClaw

Use Agent Memory with [OpenClaw](https://openclaw.ai) for a personal AI assistant with persistent memory.

See [openclaw/README.md](./openclaw/README.md) for setup instructions.

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
- Batch operations (up to 1000 memories)
- Reranking for improved precision

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
SERVER_MODE=http  # or mcp-stdio
HTTP_PORT=:8080

# Auth
AUTH_ENABLED=false
API_KEYS=key1:tenant1,key2:tenant2
```

### Python SDK Configuration

```python
client = AgentMemory(
    base_url="http://localhost:8080",
    api_key="your-api-key",
    timeout=30,
)
```

## Tech Stack

- **Server**: Go 1.26
- **Graph DB**: Neo4j
- **Vector DB**: Qdrant
- **Embeddings**: OpenAI (configurable)
- **SDK**: Python 3.9+
- **Protocol**: MCP (Model Context Protocol)

## License

MIT - See [LICENSE](./LICENSE)

---

<p align="center">
  <strong>Give your agents memory. Build smarter products.</strong>
</p>
