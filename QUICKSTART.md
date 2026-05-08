# Hystersis — Quick Start

## One-Line Install

```bash
curl -fsSL https://raw.githubusercontent.com/Himan-D/agent-memory/main/install.sh | bash
```

## Manual Install

```bash
# 1. Clone the repo
git clone https://github.com/Himan-D/agent-memory.git
cd agent-memory

# 2. Start databases (Neo4j + Qdrant + Redis)
docker compose -f docker/compose.yml up -d

# 3. Build and run
go run ./cmd/server
```

Or with Docker:

```bash
docker build -t hystersis:latest -f docker/Dockerfile .

docker run -d --name hystersis \
  -p 8080:8080 \
  -e NEO4J_URI=bolt://host.docker.internal:7687 \
  -e NEO4J_USER=neo4j \
  -e NEO4J_PASSWORD=password \
  -e QDRANT_URL=host.docker.internal:6334 \
  -e REDIS_URL=redis://host.docker.internal:6379 \
  hystersis:latest
```

---

## Verify It's Running

```bash
curl http://localhost:8080/health
# {"status":"ok","neo4j":"healthy","qdrant":"healthy"}
```

---

## API Key Setup

```bash
# Create your first API key
curl -X POST http://localhost:8080/admin/api-keys \
  -H "Content-Type: application/json" \
  -H "X-API-Key: admin-key" \
  -d '{"label": "my-app", "scopes": ["read", "write"], "expires_in_hours": 8760}'

# List keys
curl http://localhost:8080/admin/api-keys \
  -H "X-API-Key: admin-key"

# Delete a key
curl -X DELETE http://localhost:8080/admin/api-keys/key_1 \
  -H "X-API-Key: admin-key"
```

**API key scopes:** `read`, `write`, `admin`

---

## Core Operations (cURL)

### Store a Memory

```bash
curl -X POST http://localhost:8080/memories \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-key" \
  -d '{
    "content": "User prefers dark mode and Python",
    "user_id": "user-123",
    "category": "preferences",
    "tags": ["ui", "coding"],
    "importance": "high"
  }'
```

### Semantic Search

```bash
curl "http://localhost:8080/search?q=coding+preferences&limit=10&threshold=0.5" \
  -H "X-API-Key: your-key"
```

### Spreading Activation Search (Enhanced)

```bash
# +23% multi-hop accuracy via graph propagation
curl "http://localhost:8080/search/enhanced?query=coding+preferences&mode=spreading" \
  -H "X-API-Key: your-key"
```

### Session Management

```bash
# Create session
curl -X POST http://localhost:8080/sessions \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-key" \
  -d '{"agent_id": "my-agent"}'

# Add message
curl -X POST "http://localhost:8080/sessions/{session_id}/messages" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-key" \
  -d '{"role": "user", "content": "Hello!"}'

# Get messages
curl "http://localhost:8080/sessions/{session_id}/messages" \
  -H "X-API-Key: your-key"
```

### Knowledge Graph

```bash
# Create entity
curl -X POST http://localhost:8080/entities \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-key" \
  -d '{"name": "Machine Learning", "type": "Concept"}'

# Create relationship
curl -X POST http://localhost:8080/relations \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-key" \
  -d '{"from_id": "entity1", "to_id": "entity2", "type": "RELATED_TO"}'

# Traverse graph (3 hops)
curl "http://localhost:8080/graph/traverse/{entity_id}?depth=3" \
  -H "X-API-Key: your-key"

# Run Cypher query
curl -X POST http://localhost:8080/graph/query \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-key" \
  -d '{"cypher": "MATCH (e:Entity) RETURN e LIMIT 5"}'
```

---

## Compression Engine

### Check Compression Stats

```bash
curl http://localhost:8080/compression/stats \
  -H "X-API-Key: your-key"
# {
#   "accuracy_retention": 0.973,
#   "token_reduction": 0.84,
#   "total_tokens_saved": 1500000,
#   "extractions_performed": 450,
#   "avg_latency_ms": 187
# }
```

### Set Compression Mode

```bash
# Modes: "extract" (default), "balanced", "aggressive"
curl -X PUT http://localhost:8080/compression/mode \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-key" \
  -d '{"mode": "extract"}'
```

### Set Memory Tier Policy

```bash
# Policies: "aggressive" (1-day hot), "balanced" (7-day, default), "conservative" (30-day)
curl -X PUT http://localhost:8080/tier/policy \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-key" \
  -d '{"policy": "balanced"}'
```

---

## Python SDK

```bash
pip install hystersis
```

```python
from hystersis import Hystersis

client = Hystersis(
    api_key="your-key",
    base_url="http://localhost:8080"
)

# Store memory (automatically compressed via ProMem)
memory = client.create_memory(
    content="User loves neural networks and transformers",
    user_id="user-123",
    category="interests",
    importance="high"
)

# Semantic search
results = client.search(
    query="machine learning interests",
    user_id="user-123",
    limit=10
)

# Session
session = client.create_session(agent_id="assistant-bot")
client.add_message(session["id"], "user", "What do I like?")

# Compression stats
stats = client.get_compression_stats()
print(f"Token savings: {stats['token_reduction']:.0%}")
```

---

## Node.js SDK

```bash
npm install @hystersis/sdk
```

```javascript
const { Hystersis } = require('@hystersis/sdk');

const client = new Hystersis({
  baseUrl: 'http://localhost:8080',
  apiKey: 'your-key'
});

// Store memory
const memory = await client.memories.create({
  content: 'User prefers TypeScript over JavaScript',
  userId: 'user-123',
  tags: ['preferences'],
  importance: 'high'
});

// Search
const results = await client.search({
  query: 'programming preferences',
  userId: 'user-123'
});
```

---

## MCP (Claude Desktop / Cursor)

Add to your `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "hystersis": {
      "command": "/path/to/hystersis",
      "env": {
        "SERVER_MODE": "mcp-stdio",
        "NEO4J_URI": "bolt://localhost:7687",
        "NEO4J_PASSWORD": "your-password",
        "QDRANT_URL": "http://localhost:6333",
        "REDIS_URL": "redis://localhost:6379"
      }
    }
  }
}
```

Run the MCP server:

```bash
SERVER_MODE=mcp-stdio ./hystersis
```

---

## Environment Variables

```bash
# Authentication
AUTH_ENABLED=true
API_KEYS=key1:tenant1,key2          # key or key:tenant format

# Databases
NEO4J_URI=bolt://localhost:7687
NEO4J_USER=neo4j
NEO4J_PASSWORD=password
QDRANT_URL=localhost:6334
REDIS_URL=redis://localhost:6379

# LLM (for memory processing)
LLM_PROVIDER=openai
LLM_API_KEY=sk-...
LLM_MODEL=gpt-4o

# Compression Engine
COMPRESSION_ENABLED=true
COMPRESSION_LLM_FAST_PROVIDER=openai
COMPRESSION_LLM_FAST_MODEL=gpt-4o-mini
COMPRESSION_LLM_VERIFY_PROVIDER=anthropic
COMPRESSION_LLM_VERIFY_MODEL=claude-3-5-sonnet
COMPRESSION_COMPLEXITY_THRESHOLD=0.6
COMPRESSION_MODE=extract
TIER_POLICY=balanced

# Server
HTTP_PORT=:8080
```

---

## Monitoring

```bash
curl http://localhost:8080/health    # Health check
curl http://localhost:8080/ready     # Readiness probe
curl http://localhost:8080/metrics   # Prometheus metrics (no auth)
```
