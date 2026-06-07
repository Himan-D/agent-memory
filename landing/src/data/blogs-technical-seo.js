/**
 * Ten detailed technical blog posts optimized for social media SEO.
 * Each post includes seoTitle, seoDescription (≤160 chars), and keywords.
 */
export const technicalBlogs = [
  {
    slug: 'hystersis-vs-mem0-agent-memory-comparison',
    title: 'Hystersis vs Mem0: A Technical Comparison of Agent Memory Platforms',
    seoTitle: 'Hystersis vs Mem0: Agent Memory Platform Comparison (2026)',
    seoDescription:
      'Deep technical comparison of Hystersis vs Mem0: graph memory, ProMem compression, spreading activation, SSO, and production benchmarks for AI agents.',
    keywords: [
      'Hystersis vs Mem0',
      'agent memory',
      'AI memory platform',
      'graph memory',
      'vector memory',
      'Mem0 alternative',
    ],
    excerpt:
      'Mem0 popularized agent memory. Hystersis extends it with graph propagation, proprietary compression, tiered storage, and enterprise SSO — here is how they compare technically.',
    image: 'https://images.unsplash.com/photo-1633356122544-f134324a6cee?w=1200&h=630&fit=crop',
    category: 'Engineering',
    date: 'Jun 5, 2026',
    readTime: '14 min read',
    tags: ['Comparison', 'Mem0', 'Architecture'],
    featured: true,
    content: `
# Hystersis vs Mem0: A Technical Comparison

Agent memory moved from research curiosity to production requirement in 2025. Mem0 established the category; Hystersis builds on that foundation with graph-native retrieval, proprietary compression, and enterprise controls.

![Side-by-side architecture diagram of vector and graph memory systems](https://images.unsplash.com/photo-1551288049-bebda4e38f71?w=1200&h=600&fit=crop)

## Architecture Overview

| Layer | Mem0 | Hystersis |
|-------|------|-----------|
| Vector store | Pinecone, Qdrant, etc. | Qdrant (primary), multi-provider |
| Graph store | Pro tier | Neo4j (open-core) |
| Compression | ~80% summarization | ProMem extraction (85%+, 97% accuracy) |
| Multi-hop search | Graph (Pro) | Spreading activation (+23% multi-hop) |
| Tiered memory | ❌ | Working → Hot → Cold → Archive |
| SSO | Enterprise | OIDC + SAML + LDAP |

Both platforms expose REST APIs and Python SDKs. The divergence is in **how** memories are stored, compressed, and retrieved under load.

## Memory Ingestion

Mem0 uses a two-pass ADD/UPDATE extraction flow. Hystersis supports single-pass ingestion with optional LLM processing and an async compression pipeline that keeps write latency under 5ms.

\`\`\`python
from hystersis import Hystersis

client = Hystersis(api_key="your-key")

# Raw storage — skip LLM processing for latency-sensitive paths
client.add(
    "User prefers dark mode and uses Vim keybindings",
    user_id="dev-42",
    skip_processing=True
)

# Full pipeline — extraction, entity linking, async compression
client.add(
    "Acme Corp migrated payment service to Kubernetes cluster-prod-2",
    user_id="dev-42"
)
\`\`\`

Mem0's extraction is solid for conversational facts. Hystersis adds **entity graph linking** at ingest time — every memory can automatically create Person, Organization, and Product nodes in Neo4j.

## Retrieval: Vector vs Spreading Activation

Pure vector search returns semantically similar chunks. It fails on multi-hop questions like "Who owns the service that handles Acme's billing?"

Hystersis spreading activation:

1. Seeds activation from Qdrant top-K results
2. Propagates through Neo4j with 0.85 decay per hop
3. Ranks by activation + vector similarity

\`\`\`bash
curl "https://api.hystersis.com/search/enhanced?mode=spreading&query=payment+service+owner"
\`\`\`

Benchmarks show **+23% improvement** on multi-hop reasoning vs vector-only retrieval.

## Compression Engine

Context windows are finite. Both platforms compress memories — the difference is accuracy retention.

| Metric | Mem0 | Hystersis |
|--------|------|-----------|
| Token reduction | ~80% | 80–85% |
| Accuracy retention | ~91% | ≥97% |
| Processing | Synchronous | Async (<5ms write impact) |

Hystersis ProMem extraction uses self-questioning, verification, and gap detection — not naive summarization. Critical facts (account IDs, error codes, preferences) survive compression.

![Data compression visualization with token reduction metrics](https://images.unsplash.com/photo-1555949963-aa79dcee981c?w=1200&h=600&fit=crop)

## Enterprise Features

For production deployments, consider:

- **Multi-tenant isolation** — API keys map to tenants; queries auto-filter
- **Audit logs** — Full event trail for compliance
- **RBAC** — Role-based permissions on memory operations
- **SSO** — OIDC, SAML, and LDAP out of the box

Mem0 Enterprise covers SSO and audit. Hystersis includes LDAP SSO and skill chains (procedural memory) without an enterprise-only gate.

## When to Choose Each

**Choose Mem0** if you need a mature hosted service with minimal infrastructure and conversational memory is your primary use case.

**Choose Hystersis** if you need graph-native multi-hop retrieval, higher compression accuracy, self-hosted deployment, tiered memory, or LDAP SSO.

## Migration Path

\`\`\`python
# Export from Mem0, import to Hystersis
for memory in mem0_client.get_all(user_id="user-123"):
    hystersis_client.add(
        content=memory["memory"],
        user_id=memory["user_id"],
        metadata={"migrated_from": "mem0", "original_id": memory["id"]}
    )
\`\`\`

Run the [compression playground](https://hystersis.com/demo) to benchmark token reduction on your own data before migrating.
    `,
  },
  {
    slug: 'choosing-vector-database-agent-memory',
    title: 'Choosing a Vector Database for Production Agent Memory',
    seoTitle: 'Vector DB Guide for AI Agent Memory: Qdrant, Pinecone & More',
    seoDescription:
      'Technical guide to picking a vector database for agent memory: Qdrant vs Pinecone, embedding dimensions, filtering, hybrid search, and Hystersis multi-provider setup.',
    keywords: [
      'vector database',
      'agent memory',
      'Qdrant',
      'Pinecone',
      'embeddings',
      'semantic search',
    ],
    excerpt:
      'Qdrant, Pinecone, Weaviate, or pgvector? A decision framework for vector storage in production agent memory systems — latency, filtering, cost, and ops burden.',
    image: 'https://images.unsplash.com/photo-1558494949-ef010cbdcc31?w=1200&h=630&fit=crop',
    category: 'Architecture',
    date: 'Jun 4, 2026',
    readTime: '13 min read',
    tags: ['Vector DB', 'Qdrant', 'Architecture'],
    featured: true,
    content: `
# Choosing a Vector Database for Agent Memory

Every agent memory system needs vector search. The database you pick determines latency, filtering capabilities, operational cost, and how far you can scale before re-architecting.

![Server room with glowing network connections representing vector index storage](https://images.unsplash.com/photo-1451187580459-43490279c0fa?w=1200&h=600&fit=crop)

## What Agent Memory Demands from a Vector DB

Production agent workloads differ from generic RAG:

1. **High write volume** — Every conversation turn may create 1–5 memories
2. **Metadata filtering** — tenant_id, user_id, agent_id, session_id on every query
3. **Low-latency reads** — Retrieval must complete in <100ms P95
4. **Incremental updates** — Memories update and delete, not just append
5. **Multi-tenant isolation** — No cross-tenant leakage in shared indexes

## Provider Comparison

| Provider | Latency | Filtering | Self-host | Hystersis Status |
|----------|---------|-----------|-----------|------------------|
| Qdrant | Excellent | Payload filters | ✅ | Full implementation |
| Pinecone | Good | Metadata filters | ❌ SaaS | Full implementation |
| Weaviate | Good | Hybrid BM25 | ✅ | Stub |
| pgvector | Moderate | SQL WHERE | ✅ | Stub |
| Milvus | Good | Scalar fields | ✅ | Stub |

Hystersis abstracts providers behind a single interface — swap backends without changing application code.

## Why Qdrant Is the Default

For self-hosted deployments, Qdrant hits the sweet spot:

- **Payload-based filtering** maps directly to tenant/user isolation
- **HNSW indexing** with configurable ef_construct and m parameters
- **Quantization** (scalar, product) for memory-constrained clusters
- **Snapshots** for backup and disaster recovery

\`\`\`yaml
# docker-compose.yml
qdrant:
  image: qdrant/qdrant:latest
  ports:
    - "6333:6333"
  volumes:
    - qdrant_data:/qdrant/storage
\`\`\`

\`\`\`bash
QDRANT_URL=http://localhost:6333
VECTOR_PROVIDER=qdrant
\`\`\`

## Embedding Strategy

Vector quality depends on embedding model choice:

| Model | Dimensions | Best for |
|-------|------------|----------|
| text-embedding-3-small | 1536 | Cost-efficient production |
| text-embedding-3-large | 3072 | Higher accuracy, 2x storage |
| Local (e5-large) | 1024 | Air-gapped, no API cost |

Hystersis normalizes embeddings at ingest and stores the model version in metadata for re-embedding migrations.

\`\`\`python
results = client.search(
    query="user billing preferences",
    user_id="user-42",
    limit=10,
    filters={"category": "preferences"}
)
\`\`\`

![Abstract visualization of high-dimensional embedding space](https://images.unsplash.com/photo-1620712943543-bcc4688e7485?w=1200&h=600&fit=crop)

## Index Sizing and Performance

Rule of thumb for agent memory:

- **<100K vectors** — Single Qdrant node, 4GB RAM
- **100K–1M** — Enable quantization, 16GB RAM, 2 replicas
- **1M+** — Shard by tenant, separate hot/cold collections

Monitor these metrics:

- Query P95 latency (target: <50ms for top-10)
- Index build time after bulk imports
- Memory per vector (dimension × 4 bytes + payload)

## Hybrid Search: The Next Step

Pure vector search misses exact matches — order IDs, error codes, function names. Hystersis is adding BM25 keyword signals alongside vector similarity (see Mem0 v3 parity roadmap).

Until then, store critical identifiers in graph relationships where exact matching is guaranteed.

## Decision Framework

1. **SaaS, fast start** → Pinecone
2. **Self-hosted, production** → Qdrant
3. **Already on Postgres** → pgvector (acceptable under 500K vectors)
4. **Multi-provider flexibility** → Hystersis abstraction layer

Configure your provider in \`config.yaml\` or environment variables — no code changes required.
    `,
  },
  {
    slug: 'neo4j-graph-memory-patterns-for-agents',
    title: 'Neo4j Graph Memory Patterns for AI Agents',
    seoTitle: 'Neo4j Graph Memory Patterns for AI Agents | Hystersis',
    seoDescription:
      'Learn Neo4j graph memory patterns for AI agents: entity extraction, relationship types, multi-hop Cypher queries, and spreading activation retrieval with Hystersis.',
    keywords: [
      'Neo4j',
      'knowledge graph',
      'agent memory',
      'graph RAG',
      'entity extraction',
      'Cypher',
    ],
    excerpt:
      'Vector search finds similar text. Graph memory finds connected facts. Here are the Neo4j patterns Hystersis uses for entity linking, relationship traversal, and multi-hop agent reasoning.',
    image: 'https://images.unsplash.com/photo-1504639725590-34d0984388bd?w=1200&h=630&fit=crop',
    category: 'Engineering',
    date: 'Jun 3, 2026',
    readTime: '15 min read',
    tags: ['Neo4j', 'Graph', 'Patterns'],
    content: `
# Neo4j Graph Memory Patterns for AI Agents

Vectors answer "what is similar?" Graphs answer "what is connected?" Production agents need both — especially for multi-hop reasoning across entities, teams, and systems.

![Network graph with interconnected nodes representing knowledge relationships](https://images.unsplash.com/photo-1558494949-ef010cbdcc31?w=1200&h=600&fit=crop)

## Core Graph Schema

Hystersis uses a flexible entity-relationship model in Neo4j:

\`\`\`cypher
// Entity types
(:Memory {id, content, tenant_id, importance, created_at})
(:Person {name, tenant_id})
(:Organization {name, tenant_id})
(:Product {name, tenant_id})
(:Concept {name, tenant_id})

// Relationship types
(:Person)-[:WORKS_AT]->(:Organization)
(:Person)-[:PREFERS]->(:Concept)
(:Organization)-[:USES]->(:Product)
(:Memory)-[:MENTIONS]->(:Person)
(:Memory)-[:RELATED_TO]->(:Memory)
\`\`\`

Every memory ingested through the LLM pipeline automatically extracts entities and creates MENTIONS relationships.

## Pattern 1: Entity-Centric Retrieval

Start from a known entity, expand to connected memories:

\`\`\`cypher
MATCH (p:Person {name: "John Smith", tenant_id: $tenant})
      -[:WORKS_AT]->(org:Organization)
      <-[:MENTIONS]-(m:Memory)
WHERE m.tenant_id = $tenant
RETURN m.content, m.importance
ORDER BY m.importance DESC
LIMIT 10
\`\`\`

This retrieves all memories mentioning John's employer — impossible with vector search alone.

## Pattern 2: Multi-Hop Traversal

Answer "What tools does the payment team use?"

\`\`\`cypher
MATCH (team:Concept {name: "Payment Team", tenant_id: $tenant})
      <-[:MENTIONS]-(m1:Memory)
      -[:MENTIONS]->(product:Product)
RETURN DISTINCT product.name, collect(m1.content)[0..3] AS evidence
\`\`\`

Hystersis spreading activation automates this traversal with decay-weighted activation instead of explicit Cypher per query.

## Pattern 3: Conflict Detection

When new memories contradict existing ones:

\`\`\`cypher
MATCH (new:Memory {id: $new_id})-[:MENTIONS]->(entity)
MATCH (existing:Memory)-[:MENTIONS]->(entity)
WHERE existing.id <> new.id
  AND existing.tenant_id = new.tenant_id
  AND existing.category = new.category
RETURN existing.content, existing.id
\`\`\`

The LLM conflict resolution template compares candidates and either merges, supersedes, or keeps both with reduced confidence.

![Data conflict resolution workflow diagram](https://images.unsplash.com/photo-1460925895917-afdab827c52f?w=1200&h=600&fit=crop)

## Pattern 4: Temporal Chains

Track how facts evolve over time:

\`\`\`cypher
MATCH (m1:Memory)-[:SUPERSEDES*]->(m2:Memory {id: $current_id})
RETURN m1.content, m1.created_at
ORDER BY m1.created_at ASC
\`\`\`

Memory versioning lets agents understand that "User preferred Slack" (March) was superseded by "User switched to Teams" (May).

## Spreading Activation Integration

Manual Cypher queries don't scale for every user question. Spreading activation generalizes graph traversal:

1. Vector search seeds 50 initial nodes in Qdrant
2. Activation injects into matching Neo4j Memory nodes
3. Propagation follows RELATED_TO and MENTIONS edges (decay: 0.85/hop)
4. Nodes above threshold (0.1) enter the result set
5. Final ranking: 60% activation + 40% vector similarity

\`\`\`bash
GET /search/enhanced?mode=hybrid&query=who+owns+payment+service
\`\`\`

## Production Tips

- **Index tenant_id** on every node label — non-negotiable for multi-tenant
- **Cap graph depth** at 3 hops to prevent query explosions
- **Use importance scores** to prune low-value edges during propagation
- **Set NEO4J_QUERY_TIMEOUT=60** for long-running analytics queries

## Getting Started

\`\`\`bash
NEO4J_URI=bolt://localhost:7687
NEO4J_USER=neo4j
NEO4J_PASSWORD=secret
\`\`\`

\`\`\`python
client.entities.create(name="Acme Corp", type="Organization")
client.graph.link("user-42", "WORKS_AT", "Acme Corp")
\`\`\`

Graph memory transforms agents from document retrievers into systems that understand organizational structure.
    `,
  },
  {
    slug: 'async-memory-pipeline-architecture',
    title: 'Async Memory Pipeline Architecture: Sub-5ms Writes at Scale',
    seoTitle: 'Async Memory Pipeline: Sub-5ms Agent Memory Writes | Hystersis',
    seoDescription:
      'How Hystersis async memory pipeline achieves sub-5ms write latency: job queues, worker pools, ProMem compression, and non-blocking LLM extraction at scale.',
    keywords: [
      'async pipeline',
      'agent memory',
      'write latency',
      'compression pipeline',
      'worker pool',
      'queue architecture',
    ],
    excerpt:
      'LLM extraction and compression are slow. User-facing writes cannot be. Here is how Hystersis decouples ingestion from processing with an async pipeline that keeps P99 write latency under 5ms.',
    image: 'https://images.unsplash.com/photo-1555949963-aa79dcee981c?w=1200&h=630&fit=crop',
    category: 'Architecture',
    date: 'Jun 2, 2026',
    readTime: '12 min read',
    tags: ['Pipeline', 'Async', 'Performance'],
    content: `
# Async Memory Pipeline Architecture

The hardest constraint in agent memory: users expect instant responses, but memory processing (LLM extraction, embedding, compression, graph linking) takes hundreds of milliseconds to seconds.

The solution is decoupling **acknowledgment** from **processing**.

![Flow diagram of asynchronous job queue processing pipeline](https://images.unsplash.com/photo-1551288049-bebda4e38f71?w=1200&h=600&fit=crop)

## Write Path: Two Phases

### Phase 1: Fast Acknowledge (<5ms)

\`\`\`
Client POST /memories
  → Validate auth + tenant
  → Write raw memory to working tier (in-memory)
  → Enqueue compression job
  → Return 201 with memory_id
\`\`\`

The client gets an immediate response. The memory is searchable in its raw form within the working tier.

### Phase 2: Background Processing (async)

\`\`\`
Worker picks job from queue
  → LLM fact extraction (ProMem)
  → Generate embeddings
  → Write to Qdrant + Neo4j
  → Compress and update memory
  → Promote to hot/cold tier
\`\`\`

## Pipeline Components

\`\`\`go
type CompressionPipeline struct {
    jobQueue   chan CompressionJob  // buffered, 1000 capacity
    workerPool int                  // 4-8 workers
    extractor  *MemoryExtractor     // ProMem engine
    compressor *Compressor
    validator  *Validator
}
\`\`\`

| Component | Role | Latency |
|-----------|------|---------|
| Job queue | Buffer spikes, prioritize critical jobs | <1ms enqueue |
| Worker pool | Parallel processing (4–8 goroutines) | 100–500ms/job |
| LLM router | Fast path vs verify path | 50–200ms |
| Validator | Confidence check before cold storage | 10ms |

## Priority Queues

Not all memories are equal:

\`\`\`python
# Critical — user-facing preference, process immediately
client.add(content, priority=0)

# Normal — conversation transcript chunk
client.add(content, priority=2)
\`\`\`

Priority 0 jobs jump the queue. Priority 2 jobs process in FIFO order within their tier.

## Hybrid LLM Routing in the Pipeline

The pipeline's LLM router saves cost and latency:

- **Simple memories** (complexity < 0.6) → GPT-4o-mini fast extraction
- **Complex memories** (complexity ≥ 0.6) → Fast extract + Claude verification

\`\`\`bash
COMPRESSION_LLM_FAST_MODEL=gpt-4o-mini
COMPRESSION_LLM_VERIFY_MODEL=claude-3-5-sonnet
COMPRESSION_COMPLEXITY_THRESHOLD=0.6
\`\`\`

![LLM routing decision flowchart for memory compression](https://images.unsplash.com/photo-1677442136019-21780ecad995?w=1200&h=600&fit=crop)

## Failure Handling

Jobs that fail retry with exponential backoff:

1. First failure → retry in 5s
2. Second failure → retry in 30s
3. Third failure → dead letter queue + alert

Memories in the working tier remain searchable even if compression fails. The raw content is never lost.

## Observability

Monitor these pipeline metrics:

\`\`\`bash
GET /compression/stats
\`\`\`

\`\`\`json
{
  "extractions_performed": 450,
  "avg_latency_ms": 187,
  "p95_latency_ms": 245,
  "queue_depth": 12,
  "worker_utilization": 0.73
}
\`\`\`

Alert when queue depth exceeds 500 (processing falling behind) or P95 latency exceeds 500ms.

## Skip Processing for Latency-Critical Paths

\`\`\`python
client.add(
    content="URGENT: production incident — payment API returning 503",
    skip_processing=True  # raw storage, no LLM, no compression
)
\`\`\`

Use \`skip_processing=True\` for real-time tool outputs where every millisecond counts. Run compression later via batch job.

## Architecture Lesson

The async pipeline pattern applies beyond memory: any agent system that combines fast user interaction with slow intelligence processing should decouple the write path. Acknowledge immediately, enrich asynchronously.
    `,
  },
  {
    slug: 'multi-agent-memory-sync-patterns',
    title: 'Multi-Agent Memory Sync: Sharing Context Without Chaos',
    seoTitle: 'Multi-Agent Memory Sync Patterns for AI Teams | Hystersis',
    seoDescription:
      'Technical patterns for syncing memory across multiple AI agents: shared namespaces, conflict resolution, event-driven updates, and tenant isolation with Hystersis.',
    keywords: [
      'multi-agent',
      'memory sync',
      'agent collaboration',
      'shared context',
      'AI agents',
      'distributed memory',
    ],
    excerpt:
      'When three agents serve the same user, who owns the memory? Patterns for shared namespaces, write permissions, conflict resolution, and eventual consistency across agent teams.',
    image: 'https://images.unsplash.com/photo-1535378917042-10a22c95931a?w=1200&h=630&fit=crop',
    category: 'Architecture',
    date: 'May 31, 2026',
    readTime: '13 min read',
    tags: ['Multi-Agent', 'Sync', 'Collaboration'],
    content: `
# Multi-Agent Memory Sync

Single-agent memory is straightforward. Multi-agent systems — orchestrator + specialist agents, parallel tool runners, human-in-the-loop workflows — need explicit sync patterns or agents will contradict each other.

![Multiple AI agents collaborating with shared memory layer](https://images.unsplash.com/photo-1485827404703-89b55fcc595e?w=1200&h=600&fit=crop)

## The Problem

Consider a customer support stack:

- **Router agent** — classifies intent, routes to specialist
- **Billing agent** — handles payment questions
- **Technical agent** — handles bug reports

Without shared memory, the billing agent doesn't know the router already learned the user's plan tier. The technical agent re-asks for environment details the user provided five minutes ago.

## Pattern 1: Shared User Namespace

All agents for a user read/write the same memory namespace:

\`\`\`python
# All agents use the same user_id
router.add("User on Pro plan, prefers email", user_id="user-42")
billing.search("plan details", user_id="user-42")  # finds Pro plan
tech.search("environment", user_id="user-42")       # finds preferences
\`\`\`

Hystersis filters all queries by \`user_id\` + \`tenant_id\` automatically. No cross-user leakage.

## Pattern 2: Agent-Scoped Memory

Some memories belong to a specific agent:

\`\`\`python
tech_agent.add(
    "Reproduced bug in Safari 17.2, WebKit rendering issue",
    user_id="user-42",
    agent_id="tech-support-v2",
    metadata={"visibility": "agent-scoped"}
)

# Only tech agent sees agent-scoped memories by default
tech_agent.search("Safari bug", user_id="user-42", agent_id="tech-support-v2")
\`\`\`

Use agent-scoped memory for domain-specific context that would noise other agents' retrieval.

## Pattern 3: Event-Driven Sync

For real-time multi-agent systems, publish memory events:

\`\`\`python
# Agent A writes memory
memory_id = orchestrator.add("User wants refund for order #4521", user_id="user-42")

# Webhook notifies Agent B
# POST /webhooks/memory-created
# { "memory_id": "...", "user_id": "user-42", "agent_id": "orchestrator" }
\`\`\`

Hystersis webhooks fire on memory create, update, and delete. Subscribing agents invalidate local caches and re-fetch relevant context.

## Conflict Resolution

When two agents write contradictory facts:

\`\`\`
Agent A: "User prefers phone support"
Agent B: "User prefers email support"
\`\`\`

Hystersis conflict resolution pipeline:

1. Detect overlapping entities and categories
2. LLM compares both memories with timestamps
3. Resolution: supersede (newer wins), merge (both partially true), or flag for human review

\`\`\`python
# Force human review on high-stakes conflicts
client.add(
    content="User authorized $500 credit",
    metadata={"requires_review": True, "confidence": 0.6}
)
\`\`\`

![Conflict resolution flow between multiple agent memory writes](https://images.unsplash.com/photo-1555949963-aa79dcee981c?w=1200&h=600&fit=crop)

## Session Handoff

Pass context between agents in a chain:

\`\`\`python
session = client.sessions.create(agent_id="orchestrator", user_id="user-42")

# Orchestrator gathers context
context = client.search("user issue summary", user_id="user-42", limit=10)

# Hand off to specialist with session link
specialist_session = client.sessions.create(
    agent_id="billing-agent",
    user_id="user-42",
    parent_session=session.id
)
\`\`\`

Sessions create an audit trail of which agent handled which phase.

## Consistency Model

Hystersis provides **eventual consistency** across tiers:

- **Working tier** — immediate read-after-write
- **Hot tier (Redis)** — <20ms propagation
- **Cold tier (Neo4j + Qdrant)** — <100ms after async pipeline completes

Design agents to tolerate brief staleness in cold tier reads, or use \`consistency=strong\` for critical queries.

## Production Checklist

- [ ] Define shared vs agent-scoped memory policies per agent role
- [ ] Configure webhooks for cross-agent notifications
- [ ] Enable conflict resolution for high-stakes domains (billing, medical, legal)
- [ ] Set importance scores so specialist agent facts rank above router summaries
- [ ] Monitor duplicate memory rate (target: <5% of total writes)
    `,
  },
  {
    slug: 'rbac-tenant-isolation-agent-memory',
    title: 'RBAC and Tenant Isolation for Production Agent Memory',
    seoTitle: 'RBAC & Tenant Isolation for AI Agent Memory | Hystersis',
    seoDescription:
      'Implement RBAC and multi-tenant isolation for AI agent memory: API key scoping, role permissions, query filtering, and compliance patterns with Hystersis.',
    keywords: [
      'RBAC',
      'tenant isolation',
      'agent memory security',
      'multi-tenant',
      'API security',
      'compliance',
    ],
    excerpt:
      'One leaked memory across tenants is a company-ending event. How Hystersis enforces tenant isolation at the query level, RBAC on operations, and audit trails for compliance.',
    image: 'https://images.unsplash.com/photo-1563986768609-322da13575f3?w=1200&h=630&fit=crop',
    category: 'Engineering',
    date: 'May 30, 2026',
    readTime: '11 min read',
    tags: ['Security', 'RBAC', 'Multi-Tenant'],
    content: `
# RBAC and Tenant Isolation for Agent Memory

Agent memory stores user preferences, business data, and conversation history. A single cross-tenant leak is catastrophic. Security must be structural — not bolted on.

![Cybersecurity lock and shield representing data isolation](https://images.unsplash.com/photo-1550751827-4bd374c3f58b?w=1200&h=600&fit=crop)

## Tenant Isolation Architecture

Every request in Hystersis carries a tenant context derived from the API key:

\`\`\`yaml
# config.yaml
api_keys:
  prod_acme: { tenant: "tenant_acme", role: "admin" }
  prod_globex: { tenant: "tenant_globex", role: "editor" }
  read_only_demo: { tenant: "tenant_acme", role: "viewer" }
\`\`\`

The tenant ID injects into **every** database query — vector, graph, and cache:

\`\`\`cypher
// Automatic tenant filter on all Neo4j queries
MATCH (m:Memory {tenant_id: $tenant_id})
WHERE m.content CONTAINS $query
RETURN m
\`\`\`

There is no global search. No \`SELECT *\` without a tenant predicate.

## Role-Based Access Control

Hystersis RBAC defines four roles:

| Role | Read | Write | Delete | Admin |
|------|------|-------|--------|-------|
| viewer | ✅ | ❌ | ❌ | ❌ |
| editor | ✅ | ✅ | ❌ | ❌ |
| admin | ✅ | ✅ | ✅ | ❌ |
| owner | ✅ | ✅ | ✅ | ✅ |

\`\`\`go
// Middleware checks permission before handler
func requirePermission(perm Permission) mux.MiddlewareFunc {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            role := r.Context().Value("role").(roles.Role)
            if !roles.Checker.HasPermission(role, perm) {
                safeHTTPError(w, "forbidden", http.StatusForbidden)
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}
\`\`\`

## API Key Scoping

Beyond roles, scope API keys to specific operations:

\`\`\`python
# Read-only key for retrieval agents
search_client = Hystersis(api_key="read_only_key")
results = search_client.search("user preferences", user_id="user-42")  # OK
search_client.add("new memory")  # 403 Forbidden

# Write key for ingestion agents
write_client = Hystersis(api_key="write_key")
write_client.add("User prefers dark mode")  # OK
\`\`\`

## Group-Level Policies

Within a tenant, groups isolate team memories:

\`\`\`python
client.add(
    "Q3 roadmap: launch memory compression v2",
    group_id="engineering",
    metadata={"classification": "internal"}
)

# Sales agent cannot see engineering group memories
sales_agent.search("roadmap", group_id="sales")  # no results
\`\`\`

## Audit Logging

Every memory operation emits an audit event:

\`\`\`json
{
  "event": "memory.created",
  "tenant_id": "tenant_acme",
  "actor": "api_key:prod_acme",
  "role": "admin",
  "memory_id": "mem_abc123",
  "timestamp": "2026-06-05T14:30:00Z",
  "ip": "203.0.113.42"
}
\`\`\`

Audit logs support SOC 2, HIPAA, and GDPR compliance requirements. Export via \`GET /audit/logs?from=...&to=...\`.

![Audit log dashboard with security event timeline](https://images.unsplash.com/photo-1551288049-bebda4e38f71?w=1200&h=600&fit=crop)

## SSO Integration

Enterprise tenants authenticate via SSO:

| Provider | Protocol | File |
|----------|----------|------|
| Google Workspace | OIDC | \`internal/sso/oidc.go\` |
| Okta | SAML | \`internal/sso/saml.go\` |
| Active Directory | LDAP | \`internal/sso/ldap.go\` |

SSO users map to tenant + role automatically. API keys remain available for agent-to-agent communication.

## Security Checklist

- [ ] Unique API key per agent deployment (never share keys)
- [ ] Rotate keys quarterly; use \`PUT /api-keys/{id}/rotate\`
- [ ] Enable audit logging in production
- [ ] Set \`NEO4J_QUERY_TIMEOUT\` to prevent DoS via expensive graph queries
- [ ] Review RBAC roles during onboarding — default to \`viewer\`, not \`admin\`
- [ ] Test tenant isolation with cross-tenant query attempts in CI
    `,
  },
  {
    slug: 'procedural-memory-skills-for-ai-agents',
    title: 'Procedural Memory: Skills and Reusable Agent Patterns',
    seoTitle: 'Procedural Memory & Skills for AI Agents | Hystersis Guide',
    seoDescription:
      'Build procedural memory for AI agents with Hystersis skills system: trigger-action patterns, skill chains, LLM synthesis, and human-in-the-loop review workflows.',
    keywords: [
      'procedural memory',
      'AI agent skills',
      'skill chains',
      'agent patterns',
      'reusable workflows',
      'Hystersis skills',
    ],
    excerpt:
      'Declarative memory stores facts. Procedural memory stores how-to. Hystersis skills system lets agents discover, execute, and synthesize reusable trigger-action patterns across sessions.',
    image: 'https://images.unsplash.com/photo-1516321318423-f06f85e504b3?w=1200&h=630&fit=crop',
    category: 'Tutorial',
    date: 'May 29, 2026',
    readTime: '12 min read',
    tags: ['Skills', 'Procedural Memory', 'Tutorial'],
    content: `
# Procedural Memory: Skills for AI Agents

Facts tell an agent *what* is true. Skills tell an agent *what to do* when a situation arises. Procedural memory is the difference between an agent that remembers and an agent that improves.

![Workflow automation diagram showing trigger-action skill patterns](https://images.unsplash.com/photo-1454165804606-c3d57bc86b40?w=1200&h=600&fit=crop)

## Skills vs Memories

| | Declarative Memory | Procedural Memory (Skills) |
|---|---|---|
| Content | "User prefers TypeScript" | "When user asks about types, check tsconfig first" |
| Retrieval | Semantic search | Trigger matching |
| Lifecycle | Created from conversation | Extracted, reviewed, versioned |
| Reuse | Per-user context | Cross-user patterns |

## Creating Skills

\`\`\`bash
curl -X POST https://api.hystersis.com/skills \\
  -H "Authorization: Bearer $API_KEY" \\
  -d '{
    "name": "debug-react-hydration",
    "trigger": "hydration mismatch error in Next.js",
    "action": "Check for browser-only APIs in render path. Compare server HTML with client. Inspect suppressHydrationWarning usage.",
    "domain": "frontend",
    "confidence": 0.85,
    "tags": ["react", "nextjs", "debugging"]
  }'
\`\`\`

## Skill Discovery

Agents find relevant skills by trigger matching:

\`\`\`python
suggestions = client.skills.suggest(
    trigger="TypeError: Cannot read property of undefined",
    context={"framework": "React", "file": "App.tsx"},
    limit=5
)

for skill in suggestions:
    print(f"{skill.name} (confidence: {skill.confidence})")
\`\`\`

LLM-powered suggestion ranks skills by trigger similarity and domain relevance.

## Skill Extraction from Conversations

Turn successful debugging sessions into reusable skills:

\`\`\`python
skills = client.skills.extract(
    content="""
    User had CORS error calling API from localhost:3000.
    Fixed by adding localhost:3000 to allowed origins in config.yaml.
    Restarted server to apply changes.
    """
)
# Returns: { name: "fix-cors-localhost", trigger: "CORS error from localhost", ... }
\`\`\`

Extracted skills enter a **human review queue** before activation:

\`\`\`python
client.skills.review(review_id="rev_123", approved=True, notes="Verified fix works")
\`\`\`

## Skill Chains

Multi-step workflows compose skills into chains:

\`\`\`python
chain = client.chains.create(
    name="incident-response",
    trigger="production outage detected",
    steps=[
        {"skill_id": "check-status-page", "order": 1},
        {"skill_id": "notify-on-call", "order": 2, "continue_if": "severity >= high"},
        {"skill_id": "create-incident-ticket", "order": 3},
        {"skill_id": "run-diagnostic-suite", "order": 4},
    ]
)

result = client.chains.execute(
    chain_id=chain.id,
    context={"service": "payment-api", "severity": "critical"},
    timeout_ms=30000
)
\`\`\`

Each step executes via LLM with the skill's action template and accumulated context from prior steps.

![Skill chain execution flow with sequential steps](https://images.unsplash.com/photo-1551288049-bebda4e38f71?w=1200&h=600&fit=crop)

## Skill Synthesis

Merge related skills into generalized patterns:

\`\`\`python
generalized = client.skills.synthesize(
    skill_ids=["fix-cors-localhost", "fix-cors-staging", "fix-cors-preview"]
)
# Result: "fix-cors-environment" — adds origin to allowed list for any environment
\`\`\`

Synthesis reduces skill sprawl as agents accumulate domain knowledge.

## Built-in Skills

Hystersis ships 13 built-in skills: git-expert, code-review, debugger, planner, memory-manager, graph-expert, search-expert, and more. Load from \`~/.agent-memory/skills/\` or \`./.skills/\`.

## NPM SDK

\`\`\`javascript
const { addSkill, suggestSkills, executeSkill } = require('@hystersis/skills');

await addSkill({
  name: 'deploy-check',
  trigger: 'before production deploy',
  action: 'Run test suite, check migration status, verify env vars',
});

const result = await executeSkill(skillId, { context: { branch: 'main' } });
\`\`\`

Procedural memory turns one-off agent successes into institutional knowledge.
    `,
  },
  {
    slug: 'mcp-integration-agent-memory-hystersis',
    title: 'MCP Integration: Connecting Agent Memory to the Model Context Protocol',
    seoTitle: 'MCP + Agent Memory Integration Guide | Hystersis',
    seoDescription:
      'Integrate Hystersis agent memory with Model Context Protocol (MCP): memory tools, resource providers, Cursor and Claude Desktop setup, and cross-session context.',
    keywords: [
      'MCP',
      'Model Context Protocol',
      'agent memory',
      'Cursor MCP',
      'Claude Desktop',
      'AI tools',
    ],
    excerpt:
      'MCP standardizes how AI models access tools and data. Here is how to expose Hystersis memory as MCP tools and resources for Cursor, Claude Desktop, and custom agent hosts.',
    image: 'https://images.unsplash.com/photo-1518770660439-4636190af475?w=1200&h=630&fit=crop',
    category: 'Tutorial',
    date: 'May 28, 2026',
    readTime: '11 min read',
    tags: ['MCP', 'Integration', 'Tutorial'],
    content: `
# MCP Integration for Agent Memory

The Model Context Protocol (MCP) standardizes how AI applications access external tools and data. Hystersis memory maps naturally to MCP tools — giving any MCP-compatible host persistent, searchable context.

![Circuit board close-up representing protocol connections and integrations](https://images.unsplash.com/photo-1518770660439-4636190af475?w=1200&h=600&fit=crop)

## Why MCP + Memory?

Without MCP, every agent host implements its own memory integration. With MCP:

- **Cursor** remembers project context across sessions
- **Claude Desktop** recalls user preferences and past decisions
- **Custom agents** get memory via a standard protocol

One Hystersis deployment serves all hosts.

## MCP Server Architecture

\`\`\`
┌─────────────┐     MCP (stdio/SSE)     ┌──────────────────┐
│ Cursor IDE  │ ◄────────────────────► │ Hystersis MCP    │
│ Claude App  │                        │ Server           │
│ Custom Host │                        │                  │
└─────────────┘                        │  Tools:          │
                                       │  - remember      │
                                       │  - recall        │
                                       │  - search        │
                                       │  - forget        │
                                       │                  │
                                       │  Resources:      │
                                       │  - memory://...  │
                                       └────────┬─────────┘
                                                │
                                       ┌────────▼─────────┐
                                       │ Hystersis API    │
                                       │ (Neo4j + Qdrant) │
                                       └──────────────────┘
\`\`\`

## Tool Definitions

### remember

Store a new memory from the current session:

\`\`\`json
{
  "name": "remember",
  "description": "Store a fact or preference for future sessions",
  "inputSchema": {
    "type": "object",
    "properties": {
      "content": { "type": "string", "description": "The memory to store" },
      "user_id": { "type": "string" },
      "tags": { "type": "array", "items": { "type": "string" } }
    },
    "required": ["content"]
  }
}
\`\`\`

### recall / search

Retrieve relevant memories before responding:

\`\`\`json
{
  "name": "search",
  "description": "Search agent memory by semantic similarity",
  "inputSchema": {
    "type": "object",
    "properties": {
      "query": { "type": "string" },
      "limit": { "type": "number", "default": 5 }
    },
    "required": ["query"]
  }
}
\`\`\`

## Cursor Configuration

Add to \`.cursor/mcp.json\`:

\`\`\`json
{
  "mcpServers": {
    "hystersis": {
      "command": "npx",
      "args": ["-y", "@hystersis/mcp-server"],
      "env": {
        "HYSTERSIS_API_KEY": "your-api-key",
        "HYSTERSIS_BASE_URL": "https://api.hystersis.com"
      }
    }
  }
}
\`\`\`

The agent can now call \`remember\` after learning preferences and \`search\` before making recommendations.

## Claude Desktop Configuration

\`\`\`json
{
  "mcpServers": {
    "hystersis": {
      "command": "npx",
      "args": ["-y", "@hystersis/mcp-server"],
      "env": {
        "HYSTERSIS_API_KEY": "your-api-key"
      }
    }
  }
}
\`\`\`

## Resource Providers

Expose memory as readable resources:

\`\`\`
memory://user-42/preferences    → User preference memories
memory://user-42/recent         → Last 10 memories
memory://project/agent-memory   → Project-specific context
\`\`\`

Resources let the model pre-load context without explicit tool calls.

![Developer IDE with MCP tools integration panel](https://images.unsplash.com/photo-1516321318423-f06f85e504b3?w=1200&h=600&fit=crop)

## Best Practices

1. **Auto-remember** — Configure the MCP server to store key decisions after each session
2. **Pre-search** — Inject top-5 relevant memories into system prompt at session start
3. **Scope by project** — Use \`group_id\` to isolate memories per repository
4. **Tag consistently** — Tags like \`preference\`, \`architecture\`, \`bug\` improve retrieval precision
5. **Review memory growth** — Run \`GET /compression/stats\` monthly to monitor token savings

## Self-Hosted MCP Server

\`\`\`bash
npm install -g @hystersis/mcp-server
HYSTERSIS_API_KEY=your-key HYSTERSIS_BASE_URL=http://localhost:8080 hystersis-mcp
\`\`\`

Point at your self-hosted Hystersis instance for air-gapped environments.

MCP turns Hystersis from a backend service into a first-class capability for every MCP-compatible AI host.
    `,
  },
  {
    slug: 'qdrant-embedding-strategies-agent-memory',
    title: 'Qdrant Embedding Strategies for High-Accuracy Agent Memory',
    seoTitle: 'Qdrant Embedding Strategies for Agent Memory | Hystersis',
    seoDescription:
      'Optimize Qdrant embeddings for agent memory: model selection, chunking, payload design, HNSW tuning, quantization, and re-embedding migration strategies.',
    keywords: [
      'Qdrant embeddings',
      'vector search',
      'agent memory',
      'HNSW',
      'embedding model',
      'semantic search',
    ],
    excerpt:
      'Embedding quality determines retrieval accuracy. Practical Qdrant tuning for agent memory: model selection, chunk boundaries, payload filters, HNSW parameters, and when to re-embed.',
    image: 'https://images.unsplash.com/photo-1551288049-bebda4e38f71?w=1200&h=630&fit=crop',
    category: 'Engineering',
    date: 'May 27, 2026',
    readTime: '13 min read',
    tags: ['Qdrant', 'Embeddings', 'Performance'],
    content: `
# Qdrant Embedding Strategies for Agent Memory

Vector search is only as good as the embeddings behind it. For agent memory — short, structured facts rather than long documents — embedding strategy requires different thinking than traditional RAG.

![Data analytics dashboard showing vector similarity metrics](https://images.unsplash.com/photo-1460925895917-afdab827c52f?w=1200&h=600&fit=crop)

## Agent Memory vs Document RAG

| Aspect | Document RAG | Agent Memory |
|--------|-------------|--------------|
| Chunk size | 500–2000 tokens | 10–100 tokens (single facts) |
| Update frequency | Batch ingestion | Real-time per conversation |
| Metadata | Source, page number | user_id, agent_id, importance |
| Query type | "Find relevant docs" | "What does this user prefer?" |

Agent memories are short, dense, and metadata-rich. Your embedding and indexing strategy should reflect that.

## Model Selection

\`\`\`bash
# OpenAI (default)
EMBEDDING_PROVIDER=openai
EMBEDDING_MODEL=text-embedding-3-small  # 1536 dims, $0.02/1M tokens

# High accuracy
EMBEDDING_MODEL=text-embedding-3-large  # 3072 dims, $0.13/1M tokens

# Local (air-gapped)
EMBEDDING_PROVIDER=local
EMBEDDING_MODEL=e5-large-v2  # 1024 dims, no API cost
\`\`\`

For agent memory, **text-embedding-3-small** hits the accuracy/cost sweet spot. Facts are short — the model doesn't need 3072 dimensions to distinguish "prefers email" from "prefers phone."

## Payload Design

Store metadata in Qdrant payloads for filter-first retrieval:

\`\`\`json
{
  "tenant_id": "tenant_acme",
  "user_id": "user-42",
  "agent_id": "support-bot",
  "category": "preference",
  "importance": 0.85,
  "created_at": "2026-05-27T10:00:00Z",
  "embedding_model": "text-embedding-3-small",
  "compressed": true
}
\`\`\`

Always filter by \`tenant_id\` + \`user_id\` before vector search. This reduces the search space by 99%+ in multi-tenant deployments.

\`\`\`python
results = client.search(
    query="contact preferences",
    user_id="user-42",
    filters={"category": "preference"},
    limit=5
)
\`\`\`

## HNSW Tuning

Qdrant's HNSW index parameters affect recall and speed:

| Parameter | Default | Agent Memory Recommendation |
|-----------|---------|----------------------------|
| m | 16 | 16 (sufficient for fact-sized vectors) |
| ef_construct | 100 | 200 (higher recall at build time) |
| ef | — | 128 at query time (balance speed/recall) |

For collections under 1M vectors, defaults work well. Above 1M, enable scalar quantization:

\`\`\`json
{
  "quantization_config": {
    "scalar": {
      "type": "int8",
      "quantile": 0.99,
      "always_ram": true
    }
  }
}
\`\`\`

Quantization cuts memory usage by ~4x with <2% recall loss.

![Vector index performance benchmark chart](https://images.unsplash.com/photo-1551288049-bebda4e38f71?w=1200&h=600&fit=crop)

## Chunking Strategy

Agent memories should be atomic facts, not chunked documents:

**Good** (atomic):
- "User prefers TypeScript over JavaScript"
- "Acme Corp payment service runs on Kubernetes"
- "Bug reproduced in Safari 17.2, WebKit issue"

**Bad** (too large):
- "In today's session the user mentioned they prefer TypeScript, also discussed the payment service migration, and reported a Safari bug..."

ProMem extraction automatically splits compound memories into atomic facts before embedding.

## Re-Embedding Migration

When upgrading embedding models:

\`\`\`python
# Hystersis tracks embedding_model in metadata
stale = client.memories.list(
    filters={"embedding_model": "text-embedding-ada-002"}
)

for memory in stale:
    client.memories.reembed(memory.id, model="text-embedding-3-small")
\`\`\`

Run re-embedding as a background job. Memories remain searchable with old embeddings until migration completes.

## Monitoring Retrieval Quality

Track these metrics weekly:

- **MRR@5** (Mean Reciprocal Rank) — are the right memories in top 5?
- **Zero-result rate** — queries returning nothing (target: <5%)
- **Duplicate retrieval** — same fact appearing under different embeddings
- **Latency P95** — should stay under 50ms for filtered search

Use the [compression playground](https://hystersis.com/demo) to test retrieval quality with your own memory dataset before going to production.
    `,
  },
  {
    slug: 'production-agent-memory-observability',
    title: 'Production Observability for Agent Memory Systems',
    seoTitle: 'Agent Memory Observability: Metrics, Alerts & Dashboards',
    seoDescription:
      'Monitor agent memory in production: compression stats, retrieval latency, queue depth, tier distribution, audit logs, and alerting thresholds for Hystersis deployments.',
    keywords: [
      'observability',
      'agent memory monitoring',
      'production metrics',
      'compression stats',
      'latency monitoring',
      'AI infrastructure',
    ],
    excerpt:
      'You cannot fix what you cannot see. Metrics, dashboards, and alerting for agent memory: compression ratios, retrieval latency, pipeline queue depth, and tier distribution.',
    image: 'https://images.unsplash.com/photo-1551288049-bebda4e38f71?w=1200&h=630&fit=crop',
    category: 'Architecture',
    date: 'May 26, 2026',
    readTime: '12 min read',
    tags: ['Observability', 'Monitoring', 'Production'],
    featured: true,
    content: `
# Production Observability for Agent Memory

Agent memory runs silently in the background — until it doesn't. Slow retrieval, compression backlogs, and tier imbalance degrade agent quality long before they trigger errors. Observability is how you catch drift early.

![Monitoring dashboard with real-time metrics and graphs](https://images.unsplash.com/photo-1551288049-bebda4e38f71?w=1200&h=600&fit=crop)

## Key Metrics

### Compression Engine

\`\`\`bash
GET /compression/stats
\`\`\`

\`\`\`json
{
  "accuracy_retention": 0.973,
  "token_reduction": 0.84,
  "total_tokens_saved": 1500000,
  "extractions_performed": 450,
  "spreading_activations": 230,
  "avg_latency_ms": 187,
  "p95_latency_ms": 245
}
\`\`\`

| Metric | Healthy | Alert |
|--------|---------|-------|
| accuracy_retention | ≥0.95 | <0.90 |
| token_reduction | 0.80–0.90 | <0.70 |
| p95_latency_ms | <300 | >500 |
| queue_depth | <100 | >500 |

### Retrieval Performance

| Metric | Target | How to measure |
|--------|--------|----------------|
| Vector search P95 | <50ms | \`GET /search?query=...\` response time |
| Graph query P95 | <50ms | Neo4j query logs |
| Spreading activation P95 | <200ms | \`GET /search/enhanced?mode=spreading\` |
| Zero-result rate | <5% | Search logs with empty results |

### Write Path

| Metric | Target | Alert |
|--------|--------|-------|
| Write P99 | <5ms | >20ms |
| Pipeline queue depth | <100 | >500 |
| Failed jobs (1h) | 0 | >10 |
| Dead letter queue | 0 | >0 |

## Tier Distribution

Monitor memory distribution across tiers:

\`\`\`bash
GET /tier/policy
\`\`\`

\`\`\`json
{
  "policy": "balanced",
  "distribution": {
    "working": { "count": 1200, "tokens": 45000 },
    "hot": { "count": 85000, "tokens": 28000000 },
    "cold": { "count": 1200000, "tokens": 0 },
    "archive": { "count": 500000, "tokens": 0 }
  }
}
\`\`\`

Alert when:
- Working tier exceeds 4096 tokens (active session overflow)
- Hot tier exceeds 7-day retention with balanced policy
- Cold tier query P95 exceeds 100ms (index degradation)

![Tiered storage architecture with latency bands](https://images.unsplash.com/photo-1451187580459-43490279c0fa?w=1200&h=600&fit=crop)

## Dashboard Setup

Hystersis analytics dashboard at [app.hystersis.com](https://app.hystersis.com) provides:

- Memory growth over time (total count, tokens, compression ratio)
- Search latency histogram
- Top queries and zero-result queries
- Per-tenant usage breakdown
- Skill usage and extraction rates

For self-hosted deployments, export metrics to Prometheus:

\`\`\`yaml
# prometheus.yml
scrape_configs:
  - job_name: 'hystersis'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: '/metrics'
\`\`\`

## Alerting Rules

\`\`\`yaml
# Example Prometheus alerts
groups:
  - name: hystersis
    rules:
      - alert: CompressionQueueBacklog
        expr: hystersis_compression_queue_depth > 500
        for: 5m
        labels:
          severity: warning

      - alert: RetrievalLatencyHigh
        expr: histogram_quantile(0.95, hystersis_search_duration_seconds) > 0.2
        for: 10m
        labels:
          severity: critical

      - alert: CompressionAccuracyDrop
        expr: hystersis_compression_accuracy < 0.90
        for: 1h
        labels:
          severity: warning
\`\`\`

## Audit Trail Analysis

Beyond performance, audit logs reveal usage patterns:

\`\`\`bash
GET /audit/logs?event=memory.created&from=2026-06-01&to=2026-06-05
\`\`\`

Analyze:
- **Write rate per tenant** — detect runaway agents
- **Delete rate** — unusual deletion patterns may indicate compromise
- **Failed auth attempts** — brute force on API keys
- **Cross-tenant query attempts** — isolation breach attempts

## Health Check Endpoint

\`\`\`bash
curl https://api.hystersis.com/health
\`\`\`

\`\`\`json
{
  "status": "healthy",
  "neo4j": "connected",
  "qdrant": "connected",
  "redis": "connected",
  "pipeline_workers": 6,
  "pipeline_queue": 3
}
\`\`\`

Use for load balancer health checks and uptime monitoring. The [status page](https://hystersis.com/status) aggregates public health for all services.

## Operational Runbook

1. **Queue backlog** → Scale worker pool: \`COMPRESSION_WORKERS=12\`
2. **High retrieval latency** → Check Qdrant index size, enable quantization
3. **Accuracy drop** → Review recent LLM model changes, check verify provider
4. **Tier overflow** → Adjust policy: \`PUT /tier/policy {"policy": "aggressive"}\`
5. **Neo4j slow queries** → Check \`NEO4J_QUERY_TIMEOUT\`, add indexes on tenant_id

Observability transforms agent memory from a black box into a managed infrastructure service.
    `,
  },
]
