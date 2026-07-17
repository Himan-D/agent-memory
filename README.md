# Hystersis

<p align="center">
  <img src="https://img.shields.io/github/stars/Himan-D/agent-memory" alt="Stars">
  <img src="https://img.shields.io/github/license/Himan-D/agent-memory" alt="License">
  <img src="https://img.shields.io/badge/Go-1.21+-blue" alt="Go Version">
  <img src="https://img.shields.io/pypi/v/hystersis" alt="PyPI">
</p>

> **Memory that adapts. Intelligence that compounds.**
>
> Persistent memory infrastructure for AI agents — graph + vectors, skills, MCP, and a production dashboard.

---

## What is Hystersis?

Hystersis gives AI agents **long-term memory** they can search, link, and improve over time:

- **Semantic + hybrid search** over stored facts and conversations  
- **Knowledge graph** (Neo4j) for entities and relationships  
- **Vector store** (Qdrant) for similarity retrieval  
- **Skills & chains** for procedural memory  
- **MCP** so Cursor / Claude Desktop can use memory as tools  
- **Dashboard** for operators: memories, webhooks, audit, billing, live SSE  

Repo: [github.com/Himan-D/agent-memory](https://github.com/Himan-D/agent-memory)  
Docs: [hystersis.com/docs](https://hystersis.com/docs) · Site: [hystersis.com](https://hystersis.com)

---

## Quick start

### One-line install

```bash
curl -fsSL https://hystersis.com/install.sh | bash
```

Options:

```bash
curl -fsSL https://hystersis.com/install.sh | bash -s -- --minimal    # CLI only
curl -fsSL https://hystersis.com/install.sh | bash -s -- --cli-only   # CLI + Docker deps
curl -fsSL https://hystersis.com/install.sh | bash -s -- --no-docker  # CLI + SDKs, no Docker
```

Installs (when available): `hystersis` CLI, `hystersis-server`, `hystersis-agent`, `hystersis-mcp`, Python/Node SDKs, Skills CLI, and local Neo4j/Qdrant/Redis compose files.

### Point the CLI at an API

```bash
hystersis init --url https://api.hystersis.com --api-key <your-key>
# or local
hystersis init --url http://localhost:8080 --api-key <your-key>
hystersis health
hystersis memories add --agent-id default --content "First memory"
```

### One-click MCP (Cursor / Claude Desktop)

```bash
hystersis mcp setup --target all
hystersis mcp doctor
# restart Cursor / Claude Desktop
```

Proxy mode talks MCP over stdio and calls your REST API with an API key (no local Neo4j required):

```bash
hystersis-mcp --stdio \
  --memory-api https://api.hystersis.com \
  --api-key "$HYSTERSIS_API_KEY"
```

Full offline stack (local DBs):

```bash
SERVER_MODE=mcp-stdio hystersis-server
```

Details: [`MCP.md`](./MCP.md) · example config: [`mcp-config.example.json`](./mcp-config.example.json)

### Docker / from source

```bash
git clone https://github.com/Himan-D/agent-memory.git
cd agent-memory
docker compose up -d          # Neo4j, Qdrant, Redis (if compose present)
go run ./cmd/server           # API on :8080
```

```bash
pip install hystersis
# or
npm install -g @hystersis/sdk
```

---

## Your first memory

### Python

```python
from hystersis import Hystersis

client = Hystersis(base_url="http://localhost:8080", api_key="your-key")

session = client.create_session(agent_id="assistant-bot")
client.add_message(session["id"], "user", "I love machine learning!")
client.create_memory(content="User prefers Python", user_id="user-123")

results = client.search("programming language preference")
client.close()
```

### cURL

```bash
curl -X POST http://localhost:8080/memories \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-key" \
  -d '{"content":"User prefers Python","user_id":"user-123","category":"preferences"}'

curl "http://localhost:8080/search?query=programming+preference" \
  -H "X-API-Key: your-key"
```

### Live smoke (SDK + MCP)

```bash
# Unit + MCP stdio + live SDK (uses mock API if HYSTERSIS_* not set)
bash scripts/smoke-track-a.sh

# Against a real API
export HYSTERSIS_API_URL=https://api.hystersis.com
export HYSTERSIS_API_KEY=your-key
cd sdk/python && pytest -m live -o addopts= -q
```

---

## Product surfaces

| Surface | Role |
|---------|------|
| **Go API** (`cmd/server`) | Auth, RBAC, memories, search, skills, wiki, billing, SSE `/events` |
| **MCP** (`cmd/mcp-server`, stdio) | IDE tools → REST API |
| **CLI** (`cmd/cli`) | `init`, `health`, `mcp setup/print/doctor`, CRUD helpers |
| **Dashboard** (`dashboard/`) | Operator UI: memories, webhooks, audit, billing, live activity |
| **Landing / docs** (`landing/`, `docs/`) | Marketing site + Mintlify docs |
| **SDKs** | Python `hystersis`, Node `@hystersis/sdk` |

### Dashboard highlights

- Memories, entities, sessions, skills, **chains (step editor)**, groups, projects, documents  
- **Webhooks** — events, deliveries, dead-letter queue, health, PATCH updates  
- **Audit trail** with filters + export  
- **Billing** tiers aligned to quotas (`free` / `pro` / `team` / `enterprise`)  
- **Live SSE** feed + connection indicator (⌘K search, breadcrumbs, offline banner)  
- API proxy with **SSRF allowlist**, rate-limit header forwarding, PATCH support  

### Webhook events (examples)

`memory.created|updated|deleted|archived` · `entity.*` · `session.created|ended` ·  
`skill.executed` · `search.performed` · `agent.connected|disconnected` · `alert.triggered` · `webhook.delivery`

Delivery logs + DLQ persist to disk (`data/webhook_state.json`) and optionally Neo4j.

---

## Architecture

```text
Clients / Agents / IDEs
  REST · Python/Node SDKs · CLI · MCP (stdio) · Dashboard
                    │
              Go API server
    auth · RBAC · rate limits · audit · webhooks · SSE
                    │
     Memory service · Skills · Sources/Wiki · Compression
                    │
     Neo4j (graph) · Qdrant (vectors) · Redis (hot) · object storage
```

### Memory flow (simplified)

1. Write → validate → optional quota check  
2. Entity extraction → Neo4j + embeddings → Qdrant  
3. Optional async compression / consolidation  
4. Search → hybrid / enhanced (spreading activation) → optional rerank  
5. Feedback → importance / self-improvement signals  
6. Webhooks + SSE notify subscribers  

See [`docs/architecture.md`](./docs/architecture.md) for design notes and roadmap.

---

## Integrations

### MCP tools (proxy)

Includes: `add_memory`, `recall` / `search`, `get_memories`, `get_memory`,  
`update_memory`, `delete_memory`, `create_session`, `get_context`,  
`list_entities`, `add_entity`, `create_relation`, `list_skills`, `who_am_i`, …

### Frameworks

| Framework | Python | Node |
|-----------|--------|------|
| LangChain / LangGraph | ✅ | ✅ |
| LlamaIndex | ✅ | ✅ |
| CrewAI / AutoGen | ✅ | ✅ |
| OpenAI Agents / Pydantic AI | ✅ | — |
| Google ADK / Agno | ✅ | partial |

### Core API map

| Area | Endpoints (sample) |
|------|---------------------|
| Memories | `POST/GET /memories`, `PUT/DELETE /memories/{id}` |
| Search | `GET/POST /search`, `POST /search/hybrid`, `GET /search/enhanced` |
| V3 compat | `POST /v3/memories/add`, `search`, list |
| Sources | `POST /sources/ingest`, `upload`, list/delete |
| Graph | `POST /entities`, `POST /relations` |
| Skills / chains | CRUD + execute + executions |
| Webhooks | CRUD, `PATCH`, `/deliveries`, `/retry`, `/dead-letter` |
| Audit | `GET /audit/events`, `/audit/export` |
| Live | `GET /events` (SSE) |
| Billing | `/billing/usage`, `/billing/subscription`, Stripe checkout |

Full reference: [docs API](https://hystersis.com/docs) · OpenAPI under `cmd/server/swagger.json`.

---

## Configuration

```bash
# Data stores
NEO4J_URI=bolt://localhost:7687
NEO4J_USER=neo4j
NEO4J_PASSWORD=password
QDRANT_URL=http://localhost:6333
REDIS_URL=redis://localhost:6379

# API
HTTP_PORT=:8080
API_BASE_URL=https://api.hystersis.com
ADMIN_API_KEYS=am_admin_...   # or bootstrap via installer

# Embeddings / LLM
OPENAI_API_KEY=sk-...
# optional dual-provider compression routing
COMPRESSION_ENABLED=true
COMPRESSION_MODE=extract
TIER_POLICY=balanced

# MCP proxy
HYSTERSIS_API_URL=https://api.hystersis.com
HYSTERSIS_API_KEY=your-key
# SERVER_MODE=mcp-stdio   # full in-process MCP on the server binary
```

CLI config file: `~/.agent-memory.json` (`base_url`, `api_key`).

---

## Repository layout

```text
cmd/
  server/           # HTTP API + SSE + MCP-stdio mode
  mcp-server/       # Thin MCP proxy (stdio/HTTP) → REST
  cli/              # hystersis CLI
  agent/            # Interactive agent REPL
internal/
  memory/           # Core service, Neo4j, Qdrant, search, sessions
  compression/      # Proprietary extraction / retrieval pipeline
  webhook/          # Webhooks, deliveries, DLQ
  skills/ audit/ stripe/ alerts/ ...
dashboard/          # Next.js operator UI
landing/            # Marketing site + install scripts
sdk/python/         # PyPI package
sdk/nodejs/         # npm package
docs/               # Mintlify documentation
scripts/smoke-track-a.sh
```

---

## Development

```bash
# Backend
go build ./...
go test ./internal/webhook/ ./internal/memory/ -count=1
go run ./cmd/server

# CLI + MCP
go build -o hystersis ./cmd/cli
go build -o hystersis-mcp ./cmd/mcp-server
hystersis mcp doctor

# Dashboard
cd dashboard && npm install && npm run dev

# Python SDK
cd sdk/python && pip install -e ".[dev]"
pytest -q                    # unit (live smokes skipped by default)
pytest -m live -o addopts=   # needs HYSTERSIS_API_URL + HYSTERSIS_API_KEY

# Track A smoke (build + MCP stdio + unit + live)
bash scripts/smoke-track-a.sh
```

Conventions and agent rules: [`AGENTS.md`](./AGENTS.md).  
Honest competitive status vs Mem0: [`docs/features/mem0-v3-parity.mdx`](./docs/features/mem0-v3-parity.mdx).

### Benchmarks

Measured numbers require a live store + evaluator LLM. Prefer the local runner:

```bash
go run ./cmd/benchmark --mock --suite retrieval --dataset locomo   # plumbing only
# Live judged runs: configure LLM + stores, then publish under docs/benchmarks/
```

Do not treat target/marketing tables as verified production results until scored reports are committed.

---

## Pricing (hosted)

| Tier | Guide | Quotas (enforced when billing is wired) |
|------|--------|----------------------------------------|
| **Self-hosted** | Free | Unlimited (your infra) |
| **Free** | $0 | ~1k memories, 10k searches, 2 agents |
| **Pro** | $29/mo | ~50k memories, 100k searches, 10 agents |
| **Team** | $99/mo | Higher limits, webhooks + collaboration |
| **Enterprise** | Custom | Unlimited + SSO / SLA |

---

## Security notes

- Never commit API keys, SSH keys, or `.env` files  
- Proxy only allows allowlisted path prefixes on `NEXT_PUBLIC_API_URL`  
- Prefer `X-API-Key` / session Bearer; rotate keys regularly  
- Webhook secrets are signed (`X-AgentMemory-Signature`)  

---

## Resources

| Resource | Link |
|----------|------|
| Documentation | https://hystersis.com/docs |
| Demo | https://hystersis.com/demo |
| Discord | https://discord.gg/Q7bfvqKG |
| PyPI | https://pypi.org/project/hystersis/ |
| npm skills | https://www.npmjs.com/package/@hystersis/skills |

---

## Contributing

1. Fork and branch from `master` (or work on a feature branch).  
2. `go build ./...` and relevant tests before commit.  
3. Conventional commits: `feat:`, `fix:`, `docs:`, `chore:`.  
4. Open a PR with summary + test plan.

---

<p align="center">
  <strong>Give your AI agents memory. Watch them get smarter.</strong>
</p>
