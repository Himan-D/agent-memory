# Hystersis Quick Reference Guide

## Services & Ports

| Service | Port | Type | Status |
|---------|------|------|--------|
| API Server | 8080 | HTTP | ✅ Running |
| Dashboard | 3000 | HTTP | ✅ Running (dev) |
| MCP Server | 8082 | HTTP | ✅ Running |
| Connectors | 8083 | HTTP | ✅ Running |
| Memory API | 8084 | HTTP | ✅ Running |
| Neo4j | 7687 | Bolt | ✅ Running |
| Neo4j Web | 7474 | HTTP | ✅ Running |
| Qdrant | 6333 | HTTP | ✅ Running |
| Qdrant gRPC | 6334 | gRPC | ✅ Running |
| Redis | 6379 | TCP | ✅ Running |

## Common Commands

### Health Check
```bash
curl http://localhost:8080/health
```

### Build Backend
```bash
cd /home/ubuntu/agent-memory
go build ./cmd/server
```

### Run Tests
```bash
go test ./...
```

### Fix Frontend Builds
```bash
# Dashboard
cd dashboard && npm install && npm run build

# Landing
cd landing && rm -rf node_modules package-lock.json && npm install && npm run build
```

### Docker Services
```bash
# Start all services
docker-compose up -d

# Stop all services
docker-compose down

# View logs
docker-compose logs -f neo4j
docker-compose logs -f qdrant
docker-compose logs -f redis
```

### CLI Operations
```bash
# Create memory
curl -X POST http://localhost:8080/memories \
  -H "Content-Type: application/json" \
  -d '{"content":"Sample memory","user_id":"user1"}'

# Search
curl "http://localhost:8080/search?q=memory+topic"

# Create session
curl -X POST http://localhost:8080/sessions \
  -H "Content-Type: application/json" \
  -d '{"agent_id":"agent1"}'
```

## API Endpoints (Top 20)

| Method | Endpoint | Purpose |
|--------|----------|---------|
| GET | /health | Health check |
| GET | /ready | Readiness check |
| POST | /memories | Create memory |
| GET | /memories | List memories |
| GET | /search | Search memories |
| POST | /entities | Create entity |
| GET | /entities | List entities |
| GET | /graph/traverse/{id} | Graph traversal |
| POST | /sessions | Create session |
| GET | /sessions/{id}/messages | Get messages |
| POST | /skills | Create skill |
| GET | /skills | List skills |
| POST | /skills/{id}/execute | Execute skill |
| GET | /compression/stats | Compression metrics |
| GET | /metrics | Prometheus metrics |
| POST | /wiki/ingest | Ingest wiki content |
| GET | /wiki/pages | List wiki pages |
| POST | /mcp/messages | MCP messages |
| GET | /.well-known/api-catalog | API discovery |
| GET | /llms.txt | LLM providers |

## File Locations

| Component | Location | Size |
|-----------|----------|------|
| Main API | `/cmd/server/api.go` | 4,438 lines |
| API Handlers | `/cmd/server/api_handlers.go` | 1,018 lines |
| Memory Service | `/internal/memory/service.go` | 55 KB |
| Memory Types | `/internal/memory/types/` | - |
| Compression Engine | `/internal/compression/` | - |
| Skills System | `/internal/skills/` | - |
| Configuration | `/.env` | - |
| Docker | `/docker-compose.yml` | - |
| Tests | `/cmd/server/api_test.go` | 102 lines |

## Environment Variables

### Required
```bash
NEO4J_URI=bolt://localhost:7687
NEO4J_USER=neo4j
NEO4J_PASSWORD=<password>
QDRANT_URL=localhost:6334
OPENAI_API_KEY=sk-...
```

### Optional
```bash
COMPRESSION_ENABLED=true
COMPRESSION_MODE=extract
TIER_POLICY=balanced
LLM_API_KEY=...
ADMIN_API_KEYS=...
```

## Troubleshooting

### API not responding
```bash
# Check if running
ps aux | grep "go run\|server"

# Restart
pkill -f "server" ; go run ./cmd/server
```

### Neo4j connection failed
```bash
# Check connectivity
curl -u neo4j:password http://localhost:7474

# Docker logs
docker-compose logs neo4j
```

### Frontend build fails
```bash
# Dashboard
cd dashboard
npm install  # Fetch missing dependencies
npm run build

# Landing
cd landing
rm -rf node_modules package-lock.json
npm install
npm run build
```

### Redis connection issues
```bash
# Check Redis
docker-compose logs redis

# Restart
docker-compose restart redis
```

### Qdrant not responding
```bash
# Check status
curl http://localhost:6333/health

# Restart
docker-compose restart qdrant
```

## Development Workflow

1. Make code changes
2. Run tests: `go test ./...`
3. Build: `go build ./cmd/server`
4. Test locally: `go run ./cmd/server`
5. Commit: `git add -A && git commit -m "message"`
6. Push: `git push origin main`

## Production Deployment

1. Build Docker image: `docker build -t hystersis:latest .`
2. Push to registry: `docker push <registry>/hystersis:latest`
3. Deploy via Kubernetes: `kubectl apply -f deploy/k8s/agent-memory.yaml`
4. Or use Helm: `helm install hystersis ./deploy/helm/agent-memory/`

## Performance Metrics

| Metric | Target | Current |
|--------|--------|---------|
| Write Latency | <500ms | <5ms (async) |
| Query Latency (p95) | <100ms | <50ms |
| Compression Accuracy | 97%+ | 97%+ |
| Token Reduction | 85% | 85% |
| Concurrent Users | 10,000+ | Tested to 5,000+ |

## Monitoring

- Prometheus metrics: `http://localhost:8080/metrics`
- Neo4j browser: `http://localhost:7474`
- Qdrant web UI: Not available (API only)
- Redis monitoring: `redis-cli monitor`

## Key Features

- ✅ ProMem extraction (proprietary)
- ✅ Spreading activation retrieval (proprietary)
- ✅ Multi-hop graph reasoning
- ✅ Semantic search (vector + full-text)
- ✅ Skill extraction and execution
- ✅ SSO (OIDC, SAML, LDAP)
- ✅ Session management
- ✅ Entity knowledge graph
- ✅ Async compression pipeline
- ✅ Tiered memory storage

## Known Issues

- Dashboard build needs `npm install`
- Landing page build broken (rollup)
- Metrics persistence not implemented
- Archive tier storage missing (S3/GCS)
- LLM Wiki in-memory only (not persisted)

## Support Resources

- Documentation: `/docs/`
- API Reference: `/api-reference/`
- Architecture: `/ARCHITECTURE_STATUS.md`
- Roadmap: `/ROADMAP.md`
- Agents Guide: `/AGENTS.md`
