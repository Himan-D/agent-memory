# Agent Memory System - Quick Start

## Quick Install (One Line)

```bash
# Install agent-memory
curl -fsSL https://raw.githubusercontent.com/agent-memory/agent-memory/main/install.sh | bash

# Or with custom options
VERSION=v0.1.0 INSTALL_DIR=$HOME/.myapp curl -fsSL https://raw.githubusercontent.com/agent-memory/agent-memory/main/install.sh | bash
```

## Manual Install

```bash
# 1. Clone the repo
git clone https://github.com/agent-memory/agent-memory.git
cd agent-memory

# 2. Start databases
docker compose -f docker/compose.yml up -d

# 3. Build
docker build -t agent-memory:latest -f docker/Dockerfile .

# 4. Run
docker run -d --name agent-memory \
  -p 8080:8080 \
  -e NEO4J_URI=bolt://host.docker.internal:7687 \
  -e NEO4J_USER=neo4j \
  -e NEO4J_PASSWORD=password \
  -e QDRANT_URL=host.docker.internal:6334 \
  agent-memory:latest
```

## All-in-One curl Commands

### 1. Health Check
```bash
curl http://localhost:8080/health
```

### 2. Create Session (requires API key)
```bash
curl -X POST http://localhost:8080/sessions \
  -H "Content-Type: application/json" \
  -H "X-API-Key: test-key" \
  -d '{"agent_id": "my-agent"}'
```

### 3. Add Message
```bash
curl -X POST http://localhost:8080/sessions/{session_id}/messages \
  -H "Content-Type: application/json" \
  -H "X-API-Key: test-key" \
  -d '{"role": "user", "content": "Hello!"}'
```

### 4. Get Messages
```bash
curl http://localhost:8080/sessions/{session_id}/messages \
  -H "X-API-Key: test-key"
```

### 5. Create Entity
```bash
curl -X POST http://localhost:8080/entities \
  -H "Content-Type: application/json" \
  -H "X-API-Key: test-key" \
  -d '{"name": "Machine Learning", "type": "Concept"}'
```

### 6. Get Entity
```bash
curl http://localhost:8080/entities/{entity_id} \
  -H "X-API-Key: test-key"
```

### 7. Create Relation
```bash
curl -X POST http://localhost:8080/relations \
  -H "Content-Type: application/json" \
  -H "X-API-Key: test-key" \
  -d '{"from_id": "entity1", "to_id": "entity2", "type": "KNOWS"}'
```

### 8. Get Relations
```bash
curl http://localhost:8080/entities/{entity_id}/relations \
  -H "X-API-Key: test-key"
```

### 9. Semantic Search
```bash
curl "http://localhost:8080/search?q=query&limit=10&threshold=0.5" \
  -H "X-API-Key: test-key"
```

### 10. Graph Query (Cypher)
```bash
curl -X POST http://localhost:8080/graph/query \
  -H "Content-Type: application/json" \
  -H "X-API-Key: test-key" \
  -d '{"cypher": "MATCH (e:Entity) RETURN e LIMIT 5"}'
```

### 11. Admin - List API Keys
```bash
curl http://localhost:8080/admin/api-keys \
  -H "X-API-Key: test-key"
```

### 12. Admin - Create API Key
```bash
curl -X POST http://localhost:8080/admin/api-keys \
  -H "Content-Type: application/json" \
  -H "X-API-Key: test-key" \
  -d '{"label": "production", "expires_in_hours": 8760}'
```

### 13. Admin - Delete API Key
```bash
curl -X DELETE http://localhost:8080/admin/api-keys/key_1 \
  -H "X-API-Key: test-key"
```

### 14. Metrics (no auth required)
```bash
curl http://localhost:8080/metrics
```

### 15. Ready Check
```bash
curl http://localhost:8080/ready
```

## Environment Variables

```bash
# Authentication (optional)
AUTH_ENABLED=true              # Enable API key auth
API_KEYS=key1:tenant1,key2    # Comma-separated keys (format: key or key:tenant)

# Database
NEO4J_URI=bolt://localhost:7687
NEO4J_USER=neo4j
NEO4J_PASSWORD=password
QDRANT_URL=localhost:6334

# OpenAI (optional)
OPENAI_API_KEY=sk-...
OPENAI_MODEL=text-embedding-3-small

# Server
HTTP_PORT=:8080
```

## Docker Run

```bash
docker run -d --name agent-memory \
  -p 8080:8080 \
  -e NEO4J_URI=bolt://neo4j:7687 \
  -e NEO4J_USER=neo4j \
  -e NEO4J_PASSWORD=password \
  -e QDRANT_URL=qdrant:6334 \
  -e AUTH_ENABLED=true \
  -e API_KEYS="your-key:your-tenant" \
  agent-memory:latest
```

## Python SDK

```python
pip install agentmemory

from agentmemory import AgentMemory

memory = AgentMemory(
    api_key="your-key",
    base_url="http://localhost:8080"
)

session = memory.create_session("agent-1")
memory.add_message(session["id"], "user", "Hello!")
results = memory.search("query")
```
