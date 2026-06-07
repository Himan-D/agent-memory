# Hystersis Architecture

## Product Scope

Hystersis is a persistent memory platform for AI agents. It stores conversation, semantic, episodic, procedural, and profile memory; retrieves context through vector, keyword, graph, and hybrid search; and exposes the system through a Go API server, SDKs, CLIs, MCP, and web dashboards.

The current repository is a multi-surface product rather than a greenfield monorepo. The production architecture keeps the existing Go backend as the control plane and memory engine, with separate frontend and SDK packages around it.

## System Design

```text
Clients and Agents
  |-- REST API / SDKs / CLI / MCP / Dashboard
  v
Go API Server
  |-- Auth, RBAC, rate limits, audit logs
  |-- Memory service
  |-- Skills service
  |-- Wiki service
  |-- Compression pipeline
  |-- Metrics and alerts
  v
Storage and Retrieval
  |-- Neo4j: graph memory, entities, relationships, audit, metrics snapshots
  |-- Qdrant: vector memory and semantic retrieval
  |-- Redis: sessions, hot-tier memory, cache
  |-- Object storage: archive tier and exportable artifacts
  v
Observability and Delivery
  |-- Prometheus metrics
  |-- OpenTelemetry tracing
  |-- Grafana dashboards and alerts
  |-- Docker, Kubernetes, Terraform, Helm, Cloudflare/GCP deployment assets
```

## Service Boundaries

- `cmd/server`: HTTP API, auth middleware, routing, static API artifacts, wiki handlers, sessions, SSE, and operational endpoints.
- `internal/memory`: core memory service, storage interfaces, graph/vector adapters, sessions, chunking, search strategies, provenance, self-improvement, and tier routing.
- `internal/compression`: proprietary extraction, retrieval, LLM routing, async compression pipeline, and compression algorithms.
- `internal/skills`: file-backed and graph-backed procedural memory skills.
- `internal/metrics`, `internal/telemetry`, `internal/alerts`: metrics snapshots, Prometheus-facing collectors, tracing, and alerting.
- `dashboard`, `landing`, `docs`: operator UI, public site, and documentation surfaces.
- `sdk/python`, `sdk/nodejs`, `skills-npm`: external developer interfaces.
- `deploy`, `docker`, `terraform`, `.github`: delivery, infrastructure, and CI automation.

## Data Flow

1. A client sends memory, session, wiki, skill, or search requests through REST, SDK, CLI, or MCP.
2. API middleware authenticates the caller, applies RBAC and rate limits, and emits audit context.
3. The memory service extracts facts, chunks content, scores importance, links entities, and stores graph/vector records.
4. The compression pipeline may asynchronously extract verified facts, compress tokens, and update tier placement.
5. Retrieval routes the query through vector, keyword, graph, spreading activation, or hybrid search, then optionally re-ranks and distills context.
6. Responses include memory content, source attribution, relevance signals, and operational metadata where applicable.
7. Metrics, traces, audit events, and alerts are emitted throughout the flow.

## API Surface

Core API groups:

- Memories, entities, sessions, search, and enhanced search.
- Compression stats and mode controls.
- Tier policy controls.
- Skills CRUD, suggestion, synthesis, execution, review, and chains.
- Wiki ingest, query, lint, page/source CRUD, stats, index, and logs.
- Admin, auth, SSO, audit, analytics, webhooks, billing, and alerts.

The source of truth for generated API shape is `docs/openapi.json` and `cmd/server/swagger.json`.

## Event Flows

- `memory.created`: extraction, embedding, graph linkage, compression enqueue, metrics update, optional webhook.
- `memory.searched`: retrieval route selection, vector/graph/keyword lookup, rerank, source attribution, metrics update.
- `skill.reviewed`: approval or rejection, audit event, optional synthesis eligibility update.
- `compression.completed`: token savings, latency, retention score, tier routing, metrics snapshot.
- `wiki.ingested`: page generation, source tracking, lint pass, index update.

## Agent Flows

The platform supports modular agent responsibilities:

- Planner agent: breaks user goals into execution steps and memory queries.
- Research agent: searches long-term memory, wiki pages, and external connectors.
- Execution agent: performs tool or workflow actions.
- Memory agent: writes, reconciles, compresses, and retrieves memories.
- Evaluation agent: scores retrieval quality and answer faithfulness.
- Critic agent: reviews outputs for policy, correctness, and missing context.

Agents share memory through stable APIs and should not bypass memory service boundaries.

## Security Model

- API keys and sessions are authenticated before route handling.
- RBAC middleware gates sensitive routes by tenant, group, and role permission.
- SSO providers include OIDC, SAML, LDAP, and social auth integrations.
- Rate limiting protects API and login surfaces.
- Audit logs record high-risk operations and administrative changes.
- Secrets are loaded from environment or deployment secret managers, never committed.
- Sensitive error details must use `safeHTTPError()` in API handlers.
- CI runs build, tests, vet, and secret scanning; production deployments should add dependency and container scanning gates.

## Scaling Strategy

- Keep writes low-latency by treating compression and consolidation as asynchronous jobs.
- Scale stateless API replicas horizontally behind a load balancer.
- Use Redis for session and hot-tier cache locality.
- Scale Neo4j and Qdrant independently based on graph and vector load.
- Keep archive storage object-backed for cold and exportable memory artifacts.
- Use Prometheus/Grafana and OpenTelemetry to detect retrieval latency, compression p95 drift, queue backlog, cache misses, and storage saturation.

## Roadmap Milestones

1. Architecture and governance: canonical docs, ADRs, API reference consistency, CI signal quality.
2. Observability correctness: reliable compression metrics, persisted snapshots, alerts, tracing coverage.
3. Retrieval quality: BM25 signal, hybrid routing benchmarks, source attribution regression tests.
4. Wiki persistence: durable page/source/log storage and vector-backed page search.
5. Security hardening: complete audit coverage for skills, policy checks for skill sharing, route-level permission review.
6. Deployment hardening: reproducible smoke tests for Docker, Kubernetes, Terraform, dashboards, and SDK compatibility.

