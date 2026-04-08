# Agent Memory System

A Go-based memory backend for AI agents combining Neo4j (graph database) and Qdrant (vector database).

## Features

- **Graph Storage**: Neo4j for entity relationships and knowledge graphs
- **Vector Search**: Qdrant for semantic similarity search
- **REST API**: Full HTTP API with authentication
- **Multi-tenant**: Built-in tenant isolation
- **Python SDK**: Official Python client library
- **Production-ready**: Prometheus metrics, health checks, graceful shutdown
- **Deploy anywhere**: Docker, Kubernetes, Helm

## Quick Install

```bash
curl -fsSL https://raw.githubusercontent.com/Himan-D/agent-memory/main/install.sh | bash
```

## Quick Start

```bash
# Start databases
docker compose -f docker/compose.yml up -d

# Run the server
docker run -d -p 8080:8080 \
  -e NEO4J_URI=bolt://host.docker.internal:7687 \
  -e NEO4J_USER=neo4j \
  -e NEO4J_PASSWORD=password \
  -e QDRANT_URL=host.docker.internal:6334 \
  agent-memory:latest
```

## API Usage

```bash
# Health check
curl http://localhost:8080/health

# Create session (requires API key)
curl -X POST http://localhost:8080/sessions \
  -H "X-API-Key: your-key" \
  -d '{"agent_id": "my-agent"}'

# Add message
curl -X POST http://localhost:8080/sessions/{session_id}/messages \
  -H "X-API-Key: your-key" \
  -d '{"role": "user", "content": "Hello!"}'

# Semantic search
curl "http://localhost:8080/search?q=your+query" \
  -H "X-API-Key: your-key"
```

## Python SDK

```bash
pip install agentmemory
```

```python
from agentmemory import AgentMemory

client = AgentMemory("http://localhost:8080", api_key="your-key")
session = client.create_session(agent_id="my-agent")
client.add_message(session["id"], "user", "Hello!")

results = client.semantic_search("machine learning")
```

## Deployment

- **Docker**: `docker/`
- **Kubernetes**: `deploy/k8s/`
- **Helm**: `deploy/helm/`

## Documentation

See [QUICKSTART.md](./QUICKSTART.md) for complete documentation.

## License

MIT
