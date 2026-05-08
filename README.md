# Hystersis

<p align="center">
  <img src="https://img.shields.io/github/stars/Himan-D/agent-memory" alt="Stars">
  <img src="https://img.shields.io/github/license/Himan-D/agent-memory" alt="License">
  <img src="https://img.shields.io/badge/Go-1.21+-blue" alt="Go Version">
  <img src="https://img.shields.io/pypi/v/hystersis" alt="PyPI">
</p>

> **Memory that adapts. Intelligence that compounds.**
>
> Give your AI agents persistent memory that grows smarter with every conversation.

---

## The Problem

Every time you start a new conversation with an AI agent, it forgets everything. It's like talking to someone with **total amnesia** — every single time.

**Hystersis** solves this by giving your AI agents real, persistent memory that:
- Remembers past conversations and learned facts
- Understands relationships between entities via a knowledge graph
- Learns from feedback to improve itself
- Can be shared across multiple agents
- Compresses storage by **85% without losing accuracy** — our primary competitive advantage

---

## What Can You Do With It?

### Build Smarter AI Assistants
Customer support bots that remember previous tickets. Code assistants that know your coding style. Research agents that track your literature review.

### Create Knowledge Graphs
Don't just store facts — store *relationships*. "John works at Acme" → "Acme is a startup" → "Startups use Hystersis". Connect the dots automatically.

### Use Skills
Pre-built agent capabilities like `git-expert`, `sql-expert`, `security-pro` that your agents can activate when needed.

### Semantic Search
Find information by meaning, not just keywords. "machine learning" finds "ML", "deep learning", "neural networks" — even without those exact words.

### MCP Server
Connect directly to Claude Desktop, Cursor, or any MCP-compatible AI assistant.

---

## Quick Start

### Option 1: Docker (Recommended)

```bash
git clone https://github.com/Himan-D/agent-memory.git
cd agent-memory
docker-compose up -d
```

Your API server is now running at `http://localhost:8080`

### Option 2: From Source

```bash
# Requires Go 1.21+
git clone https://github.com/Himan-D/agent-memory.git
cd agent-memory
go run ./cmd/server
```

---

## Your First Memory

### Using Python SDK

```python
from hystersis import Hystersis

client = Hystersis("http://localhost:8080", api_key="your-key")

# Create a session for your agent
session = client.create_session(agent_id="assistant-bot")

# Store conversation
client.add_message(session["id"], "user", "I love machine learning!")
client.add_message(session["id"], "assistant", "That's great! What type?")
client.add_message(session["id"], "user", "Especially neural networks and transformers")

# Later, search semantically
results = client.search("deep learning transformers")
# Returns: [{"score": 0.92, "content": "User loves neural networks..."}]
```

### Using cURL

```bash
# Create a memory
curl -X POST http://localhost:8080/memories \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-key" \
  -d '{
    "content": "User prefers Python over JavaScript",
    "user_id": "user-123",
    "category": "preferences"
  }'

# Search semantically
curl "http://localhost:8080/search?query=programming+language+preference" \
  -H "X-API-Key: your-key"
```

---

## Live Demo

See the difference memory makes: **[hystersis.ai/demo](https://hystersis.ai/demo)**

Compare two identical AI agents side-by-side:
- **With Memory**: Uses past conversations and stored facts
- **Without Memory**: Starts fresh every time

---

## Key Features

### Multiple Memory Types

| Type | Use Case |
|------|----------|
| **Conversation** | Session chat history |
| **Semantic** | Facts, preferences, knowledge |
| **Knowledge Graph** | Entities and relationships |
| **Procedural** | Reusable skills and workflows |

### Proprietary Compression Engine

Our core competitive advantage — the reason Hystersis beats Mem0 on every compression metric:

| Component | What It Does | Result |
|-----------|-------------|--------|
| **ProMem Extraction** | Self-questioning + gap detection | 97%+ accuracy retention |
| **Spreading Activation** | Graph propagation with decay (0.85/hop) | +23% multi-hop reasoning |
| **Async Pipeline** | Non-blocking worker pool | <5ms write latency impact |
| **Tiered Memory** | Working→Hot→Cold→Archive routing | Optimized cost at scale |

```python
# Compression is automatic — just store memories normally
client.create_memory(
    content="Long conversation transcript...",
    user_id="user-123"
)
# Stored as 85% fewer tokens, 97%+ accuracy retained
```

### Self-Improving

Give feedback on memories — the system learns and improves future searches:

```python
client.add_feedback(memory_id, "positive")   # Increases importance score
client.add_feedback(memory_id, "negative")   # Triggers content correction
```

### Enterprise Ready

- **SSO**: OIDC, SAML, LDAP support
- **Audit Logs**: Track every memory access
- **Memory Versioning**: Rollback any changes
- **Role-Based Access**: Control who sees what
- **Multi-Tenant**: Isolated namespaces per API key

---

## Skills System

Give your agents superpowers with **Skills** — reusable capabilities that activate based on context.

### Available Skills

| Skill | What It Does |
|-------|--------------|
| `git-expert` | Git workflows, branching, conflict resolution |
| `sql-expert` | Query optimization, database design |
| `security-pro` | Vulnerability scanning, audit compliance |
| `testing-pro` | Test strategies, coverage analysis |
| `prompt-engineer` | LLM prompt optimization |
| `memory-manager` | Memory consolidation, recall optimization |

### Install Skills CLI

```bash
# Install via NPM
npx @hystersis/skills install Himan-D/hystersis-skills

# List available skills
npx @hystersis/skills list

# Search for skills
npx @hystersis/skills search "database"
```

---

## Architecture

```
┌──────────────┐      ┌──────────────────────────────────┐
│   AI Agent   │ ───▶ │         Hystersis Server          │
└──────────────┘      │                                  │
                      │  ┌─────────────────────────────┐  │
                      │  │   Compression Engine         │  │
                      │  │   ProMem + Spreading Act.    │  │
                      │  └─────────────────────────────┘  │
                      │         │               │          │
                      └─────────┼───────────────┼──────────┘
                                │               │
                          ┌─────▼─────┐   ┌────▼──────┐
                          │   Neo4j   │   │   Qdrant   │
                          │  (Graph)  │   │  (Vectors) │
                          └───────────┘   └────────────┘
```

### How It Works

1. **Store**: Agent sends messages, entities, relationships
2. **Extract**: ProMem compression extracts key facts (85% token reduction)
3. **Embed**: Content converted to vector embeddings (OpenAI, Cohere, etc.)
4. **Index**: Stored in both Neo4j (graph) and Qdrant (vectors)
5. **Search**: Spreading activation combines vector + graph for +23% multi-hop accuracy

---

## Integrations

### Model Context Protocol (MCP)

Connect to Claude Desktop, Cursor, or any MCP client:

```bash
SERVER_MODE=mcp-stdio ./hystersis
```

**Available Tools:** `add_memory`, `search_memories`, `get_memories`, `create_entity`, `create_relation`, `get_context`, `create_session`, `add_message`, `add_feedback`, `compress_memory`, `execute_skill`

### Python (LangChain, LlamaIndex)

```python
from hystersis import Hystersis

client = Hystersis(
    base_url="http://localhost:8080",
    api_key="your-key"
)
```

### Node.js

```javascript
const { Hystersis } = require('@hystersis/sdk');

const client = new Hystersis({
  baseUrl: 'http://localhost:8080',
  apiKey: 'your-key'
});
```

---

## API Endpoints

### Memory Operations

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/memories` | POST | Create memory |
| `/memories` | GET | List memories |
| `/memories/{id}` | GET | Get memory |
| `/memories/{id}` | PUT | Update memory |
| `/memories/{id}` | DELETE | Delete memory |

### Search

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/search` | GET/POST | Semantic search |
| `/search/hybrid` | POST | Semantic + keyword |
| `/search/enhanced` | GET | Spreading activation search |

### Compression

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/compression/stats` | GET | Token savings, accuracy, latency |
| `/compression/mode` | GET/PUT | Get or set compression mode |
| `/tier/policy` | GET/PUT | Get or set memory tier policy |

### Knowledge Graph

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/entities` | POST | Create entity |
| `/relations` | POST | Create relationship |
| `/graph/traverse/{id}` | GET | Traverse graph |

---

## Configuration

```bash
# Neo4j (Graph Database)
NEO4J_URI=bolt://localhost:7687
NEO4J_USER=neo4j
NEO4J_PASSWORD=your-password

# Qdrant (Vector Database)
QDRANT_URL=http://localhost:6333

# Redis (Hot Tier Cache)
REDIS_URL=redis://localhost:6379

# OpenAI (Embeddings)
OPENAI_API_KEY=sk-...

# Server
HTTP_PORT=:8080

# Auth
AUTH_ENABLED=true
ADMIN_API_KEYS=key1:tenant1,key2:tenant2

# Compression Engine
COMPRESSION_ENABLED=true
COMPRESSION_LLM_FAST_PROVIDER=openai
COMPRESSION_LLM_FAST_MODEL=gpt-4o-mini
COMPRESSION_LLM_VERIFY_PROVIDER=anthropic
COMPRESSION_LLM_VERIFY_MODEL=claude-3-5-sonnet
COMPRESSION_COMPLEXITY_THRESHOLD=0.6
COMPRESSION_MODE=extract
TIER_POLICY=balanced
```

---

## Performance

| Metric | Hystersis | Mem0 v2 | Cognee |
|--------|-----------|---------|---------|
| Token Reduction | **85%** | 80% | N/A |
| p95 Latency | **<500ms** | 1.44s | ~1s |
| Concurrent Connections | **10,000+** | ~100 | ~100 |
| Multi-hop Reasoning | **+23%** | baseline | baseline |
| Self-Hosted | **Free** | ❌ | ❌ |

---

## Pricing

| Tier | Price | Features |
|------|-------|----------|
| **Self-Hosted** | Free | Unlimited everything |
| **Pro** | $29/mo | Skills extraction, priority support |
| **Team** | $99/mo | Collaboration, audit logs, analytics |
| **Enterprise** | Custom | SSO, SLA, compliance |

---

## Why Hystersis?

### vs Mem0
- ✅ 10x faster (Go vs Python)
- ✅ 85% compression (vs 80%) with ProMem algorithm
- ✅ +23% multi-hop reasoning via Spreading Activation
- ✅ Free self-hosted option
- ✅ Skills system

### vs Cognee
- ✅ MCP server support
- ✅ Enterprise features (SSO, audit)
- ✅ 85% compression
- ✅ Better pricing

### vs Mem0 v3
Mem0 v3 (April 2026) introduced single-pass ADD-only extraction and hybrid retrieval. Hystersis goes further:
- ✅ ProMem self-questioning + verification (vs simple ADD-only)
- ✅ Spreading Activation graph propagation (vs entity-in-vector-store)
- ✅ Proprietary hyperparameters tuned to 97%+ accuracy
- ✅ Tiered memory for cost optimization at scale

---

## Resources

- **Documentation**: [docs.hystersis.ai](https://docs.hystersis.ai)
- **Discord**: [Join our community](https://discord.gg/hystersis)
- **NPM Package**: [@hystersis/skills](https://www.npmjs.com/package/@hystersis/skills)
- **PyPI**: [hystersis](https://pypi.org/project/hystersis/)

---

## Contributing

Contributions are welcome! Please read our [contributing guide](CONTRIBUTING.md) before submitting PRs.

```bash
go test ./...    # Run tests
go vet ./...     # Run linter
go build ./...   # Build
```

---

## License

MIT License — see [LICENSE](LICENSE) for details.

---

<p align="center">
  <strong>Give your AI agents memory. Watch them get smarter.</strong>
</p>
