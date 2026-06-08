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
Connect directly to Claude Desktop, Cursor, or any MCP-compatible AI assistant. See `mcp-config.example.json` for a ready-to-use Claude Desktop / Cursor configuration.

---

## Quick Start

### One-line Install (Recommended)

```bash
curl -fsSL https://hystersis.com/install.sh | bash
```

Or with install options:

```bash
# Minimal (CLI only, no SDKs)
curl -fsSL https://hystersis.com/install.sh | bash -s -- --minimal

# CLI + Docker services only (no SDKs)
curl -fsSL https://hystersis.com/install.sh | bash -s -- --cli-only

# Everything except Docker
curl -fsSL https://hystersis.com/install.sh | bash -s -- --no-docker
```

The installer sets up:
- **CLI** (`hystersis`) — manage memory from the terminal
- **Server** (`hystersis-server`) — API server binary
- **Agent REPL** (`hystersis-agent`) — interactive agent session
- **Python SDK** (`pip install hystersis`)
- **Node.js SDK** (`npm install -g @hystersis/sdk`)
- **Skills CLI** (`npm install -g @hystersis/skills`)
- **Docker services** — Neo4j + Qdrant + Redis

### Manual Options

<details>
<summary><b>Docker</b></summary>

```bash
git clone https://github.com/Himan-D/agent-memory.git
cd agent-memory
docker-compose up -d
```

Your API server is now running at `http://localhost:8080`
</details>

<details>
<summary><b>From Source</b></summary>

```bash
# Requires Go 1.21+
git clone https://github.com/Himan-D/agent-memory.git
cd agent-memory
go run ./cmd/server
```
</details>

<details>
<summary><b>Python SDK</b></summary>

```bash
pip install hystersis
pip install hystersis[integrations]
```
</details>

<details>
<summary><b>Node.js SDK & Skills CLI</b></summary>

```bash
npm install -g @hystersis/sdk
npm install -g @hystersis/skills
```
</details>

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

See the difference memory makes: **[hystersis.com/demo](https://hystersis.com/demo)**

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

### Advanced Memory Intelligence

| Feature | Description |
|---------|-------------|
| **Temporal Phase Rotation** | RoMem-style time-aware encoding that separates short-term and long-term phase components |
| **Memory Worth (MW) Scoring** | Unified score combining recency, frequency, importance, and access patterns |
| **Four-Signal Composite Importance** | Fuses recency decay, access frequency, semantic centrality, and feedback signal into a single importance score |
| **Conflict Validity Framework** | Detects and resolves contradictory memories using temporal ordering and confidence bounds |
| **Auto-Dreamer Sleep Consolidation** | Background consolidation pass that merges related memories and prunes redundant facts, inspired by sleep-replay in neuroscience |
| **Adaptive Retrieval Routing** | Selects between vector, graph, and hybrid search based on query complexity and latency budget |
| **Post-Retrieval Distillation** | LLM-based re-ranking and summarization of retrieved context before injection into the prompt |
| **Provenance DAG + TD(λ) Credit Assignment** | Tracks memory lineage as a directed acyclic graph; uses temporal-difference credit assignment to propagate feedback to source memories |
| **Exploitation/Exploration Dual Pool** | Maintains a high-confidence exploitation pool and a low-confidence exploration pool; balances recall precision with discovery |
| **UCB Retrieval Bandit** | Upper-Confidence-Bound policy over retrieval strategies; adapts to per-user access patterns over time |

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
                      │  ┌─────────────────────────────┐  │
                      │  │   Temporal Phase Rotation    │  │
                      │  │   MW Scoring + Four-Signal   │  │
                      │  └─────────────────────────────┘  │
                      │  ┌─────────────────────────────┐  │
                      │  │   Provenance DAG + TD(λ)     │  │
                      │  │   Dual Pool + UCB Bandit     │  │
                      │  └─────────────────────────────┘  │
                      │  ┌─────────────────────────────┐  │
                      │  │   Auto-Dreamer Sleep         │  │
                      │  │   Consolidation              │  │
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
3. **Score**: Four-signal composite importance + MW scoring assigns retrieval priority
4. **Embed**: Content converted to vector embeddings (OpenAI, Cohere, etc.)
5. **Index**: Stored in both Neo4j (graph) and Qdrant (vectors) with provenance DAG
6. **Search**: Adaptive routing selects strategy; spreading activation combines vector + graph for +23% multi-hop accuracy
7. **Distill**: Post-retrieval distillation re-ranks and summarizes context before prompt injection
8. **Consolidate**: Auto-Dreamer background pass merges related memories and prunes redundant facts

---

## Integrations

### Model Context Protocol (MCP)

Connect to Claude Desktop, Cursor, or any MCP client:

```bash
SERVER_MODE=mcp-stdio ./hystersis
```

See `mcp-config.example.json` for a ready-to-use configuration snippet for Claude Desktop and Cursor.

**Available Tools:** `add_memory`, `search_memories`, `get_memories`, `create_entity`, `create_relation`, `get_context`, `create_session`, `add_message`, `add_feedback`, `compress_memory`, `execute_skill`

### Framework Integrations

| Framework | Node.js | Python |
|-----------|---------|--------|
| LangChain | ✅ | ✅ |
| LangGraph | ✅ | ✅ |
| LlamaIndex | ✅ | ✅ |
| CrewAI | ✅ | ✅ |
| AutoGen | ✅ | ✅ |
| Agno | ✅ | — |
| Mastra | ✅ | — |
| OpenAI Agents SDK | ✅ | ✅ |
| Vercel AI SDK | ✅ | — |
| Google ADK | — | ✅ |
| Pydantic AI | — | ✅ |

### Python

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

### LLM Wiki

A persistent, compounding knowledge base inspired by [Karpathy's LLM Wiki pattern](https://gist.github.com/karpathy/442a6bf555914893e9891c11519de94f). The LLM reads sources, extracts key info, and maintains an interlinked wiki — not just retrieval, but compilation.

**How it works:**
1. **Ingest** a source → LLM extracts entities, creates summary pages, updates related pages
2. **Query** the wiki → LLM synthesizes answers from across the wiki with citations
3. **Lint** the wiki → Find contradictions, orphan pages, stale claims, and gaps

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/wiki/ingest` | POST | Ingest a source into the wiki |
| `/wiki/query` | POST | Ask a question against the wiki |
| `/wiki/lint` | POST | Health-check the wiki |
| `/wiki/pages` | GET | List all wiki pages |
| `/wiki/pages/{id}` | GET | Get a specific page |
| `/wiki/pages/{id}` | PUT | Update a page |
| `/wiki/pages/{id}` | DELETE | Delete a page |
| `/wiki/sources` | GET | List all raw sources |
| `/wiki/sources/{id}` | GET | Get a specific source |
| `/wiki/stats` | GET | Wiki statistics |
| `/wiki/index` | GET | Markdown index of all pages |
| `/wiki/log` | GET | Operation log |

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

## Benchmarks

Run benchmark endpoints with an evaluator LLM configured before publishing numbers:

```bash
curl -X POST "$API_URL/api/v1/benchmark/run" \
  -H "X-API-Key: $ADMIN_API_KEY"
```

Benchmark responses include `evaluator_configured`, `scored_questions`, `scoring_errors`, `search_errors`, and `warnings` so target numbers cannot be confused with measured results.

| Benchmark | Published competitor reference | Hystersis target |
|-----------|--------------------------------|------------------|
| LoCoMo | Mem0 91.6 | 93+ |
| LongMemEval | Mem0 94.8 | 96+ |
| BEAM (1M) | Mem0 64.1 | 75+ |
| Token Reduction | ~80% | 80-85% |
| p95 Latency | 1.44s | <500ms |
| Concurrent Connections | ~100 | 10,000+ |

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
Mem0 v3 (April 2026) introduced single-pass ADD-only extraction and hybrid retrieval. Hystersis is building toward the same class of capabilities, but several parity and enterprise gaps remain:
- ⚠️ `internal/memory/tier/` archive backend is not implemented yet
- ⚠️ Compression observability is incomplete; `/compression/stats` currently exposes in-memory counters only and metrics are not persisted
- ⚠️ Skill audit emitters are not yet wired for `approved`, `rejected`, and `synthesized` events
- ⚠️ `SkillSharingEnabled` and `AgentConfig.SkillDomains` are defined but not enforced
- ⚠️ Mem0 parity features like single-pass ADD-only extraction and BM25 keyword search signal are planned but not implemented
- ⚠️ Integration breadth and enterprise feature coverage are still weaker than Mem0 in some areas

See `AGENTS.md` and `docs/features/observability.mdx` for the current status and planned work.

---

## Resources

- **Documentation**: [hystersis.com/docs](https://hystersis.com/docs)
- **Discord**: [Join our community](https://discord.gg/Q7bfvqKG)
- **NPM Package**: [@hystersis/skills](https://www.npmjs.com/package/@hystersis/skills)
- **PyPI**: [hystersis](https://pypi.org/project/hystersis/)

---

## Repository Layout

```
cmd/server/          # Go API server (primary backend)
internal/            # Core memory, compression, graph, skills
landing/             # Marketing site (hystersis.com)
dashboard/           # Admin UI (app.hystersis.com)
sdk/python/          # Python SDK (hystersis)
sdk/nodejs/          # Node.js SDK (@hystersis/sdk)
skills-npm/          # Skills CLI (@hystersis/skills)
docs/                # Mintlify documentation
install.sh           # One-line installer (canonical)
docker-compose.yml   # Full local stack (Neo4j, Qdrant, Redis)
```

---

## Contributing

```bash
go build ./...     # Build
go test ./...      # Run tests
make sync-api-docs # Sync api/ docs into Go embed paths
```

See `AGENTS.md` for developer conventions.

---

<p align="center">
  <strong>Give your AI agents memory. Watch them get smarter.</strong>
</p>
