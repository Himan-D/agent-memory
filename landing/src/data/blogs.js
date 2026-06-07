import { technicalBlogs } from './blogs-technical-seo.js'

const coreBlogs = [
  {
    slug: 'why-ai-agents-need-persistent-memory',
    title: 'Why AI Agents Need Persistent Memory',
    seoTitle: 'Why AI Agents Need Persistent Memory in 2026 | Hystersis',
    seoDescription:
      'Stateless AI agents forget every session. Learn why persistent memory — episodic, semantic, and graph layers — is essential infrastructure for production agents.',
    keywords: ['AI agent memory', 'persistent memory', 'agent infrastructure', 'Hystersis', 'LLM agents'],
    excerpt:
      'Stateless agents forget everything between sessions. Persistent memory is the infrastructure layer that turns one-shot chatbots into compounding intelligence.',
    image: 'https://images.unsplash.com/photo-1677442136019-21780ecad995?w=1200&h=600&fit=crop',
    category: 'News',
    date: 'Jun 1, 2026',
    readTime: '12 min read',
    tags: ['AI Agents', 'Memory', 'Infrastructure'],
    featured: true,
    content: `
# Why AI Agents Need Persistent Memory

Every new chat session starts from zero. Your agent doesn't remember yesterday's debugging session, last week's customer issue, or the preferences a user expressed three conversations ago.

That's not a UX problem — it's an **infrastructure** problem. Production agents need memory that survives sessions, compresses intelligently, and retrieves the right facts at the right time.

![AI neural network visualization representing persistent agent memory](https://images.unsplash.com/photo-1620712943543-bcc4688e7485?w=1200&h=600&fit=crop)

## The Amnesia Tax

Stateless agents pay an amnesia tax on every interaction:

- **Repeated context** — Users re-explain the same facts, preferences, and constraints
- **No compounding** — Agents never get smarter about a specific user, team, or domain
- **Broken workflows** — Multi-day tasks, onboarding flows, and incident response can't span sessions
- **Higher token costs** — Re-sending full history on every request burns context window budget

A support bot that forgets a customer's plan tier will offer the wrong SLA. A coding agent that forgets your repo conventions will keep suggesting patterns you've already rejected. The tax compounds with every session.

## What Production Memory Actually Looks Like

Production agent memory isn't one database — it's a **stack** of complementary stores:

### 1. Episodic Memory (Sessions)

Conversation history, tool calls, and interaction traces. Your agent recalls *what happened* — the sequence of events, not just the final outcome.

\`\`\`python
session = client.sessions.create(agent_id="support-bot", user_id="user-42")
client.memories.add(
    content="User reported checkout timeout on mobile Safari",
    session_id=session.id,
    metadata={"channel": "chat", "severity": "high"}
)
\`\`\`

### 2. Semantic Memory (Vectors)

Embeddings over facts and documents. Your agent finds *what's relevant* by meaning, not keywords — "billing issue" matches "invoice didn't generate" even when the words differ.

### 3. Relational Memory (Graph)

Entities and relationships in Neo4j. Your agent understands *how things connect*: "John works at Acme" → "Acme uses Kubernetes" → "payment service runs on cluster-prod-2."

![Knowledge graph nodes connected in a network diagram](https://images.unsplash.com/photo-1558494949-ef010cbdcc31?w=1200&h=600&fit=crop)

## Memory vs. Context Window

Teams often confuse stuffing the prompt with building memory infrastructure:

| Approach | Problem |
|----------|---------|
| Dump full chat history | Context overflow, latency, cost |
| Summarize once per session | Loses facts silently |
| RAG over raw documents | Misses relationships and user-specific state |
| **Structured memory layer** | Extracts, compresses, retrieves, and links facts |

Hystersis sits in the fourth row — it's not a bigger context window, it's a system that decides **what deserves to persist** and **how to fetch it back**.

## Compression Without Amnesia

Storing everything verbatim doesn't scale. Long-running agents accumulate thousands of facts. Hystersis's proprietary compression engine extracts durable memories with **85%+ token reduction** while retaining **97%+ factual accuracy** — so memory grows without blowing context windows.

The pipeline runs asynchronously: writes return in under 5ms, compression happens in the background.

## Architecture at a Glance

\`\`\`
User message → Extract facts (LLM) → Importance score
              → Vector embed (Qdrant) → Graph link (Neo4j)
              → Async compression → Tiered storage
\`\`\`

When the agent needs context, retrieval combines vector similarity, graph propagation, and optional spreading activation for multi-hop queries.

## Getting Started in 60 Seconds

\`\`\`bash
curl -fsSL https://hystersis.com/install.sh | bash
\`\`\`

\`\`\`python
from hystersis import Hystersis

client = Hystersis(api_key="your-key")
client.memories.add(
    "User prefers TypeScript, uses pnpm, and deploys to Cloudflare Workers",
    user_id="user-123"
)

results = client.search("deployment preferences", user_id="user-123", limit=5)
for m in results:
    print(m.content)
\`\`\`

![Developer working with code on multiple monitors](https://images.unsplash.com/photo-1498050108023-c5249f4df085?w=1200&h=600&fit=crop)

## When You Need Persistent Memory

You need a memory layer if your agent:

1. Serves the same users across multiple sessions
2. Operates on long-horizon tasks (days or weeks)
3. Must respect preferences, policies, or org-specific knowledge
4. Runs tool chains where prior outcomes inform next steps

Persistent memory isn't optional for production agents. It's the difference between a demo and a product.

**Next reads:** [ProMem compression](/blog/promem-compression-85-percent-token-reduction) · [Building agents tutorial](/blog/building-memory-powered-ai-agents) · [Docs](https://hystersis.com/docs)
    `,
  },
  {
    slug: 'promem-compression-85-percent-token-reduction',
    title: 'ProMem Compression: 85% Token Reduction Without Losing Accuracy',
    seoTitle: 'ProMem Compression: 85% Token Reduction, 97% Accuracy | Hystersis',
    seoDescription:
      'How Hystersis ProMem extraction compresses agent memory 85% with 97%+ accuracy: self-questioning, verification, hybrid LLM routing, and async pipeline.',
    keywords: ['ProMem', 'memory compression', 'token reduction', 'LLM compression', 'agent memory'],
    excerpt:
      'How Hystersis proprietary ProMem extraction compresses agent memory by 85% while retaining 97%+ factual accuracy — our primary competitive advantage.',
    image: 'https://images.unsplash.com/photo-1555949963-aa79dcee981c?w=1200&h=600&fit=crop',
    category: 'Engineering',
    date: 'May 25, 2026',
    readTime: '14 min read',
    tags: ['Compression', 'ProMem', 'LLM'],
    featured: true,
    content: `
# ProMem Compression: 85% Token Reduction Without Losing Accuracy

Context windows are finite. Agent memory is infinite. Something has to give — unless you compress intelligently.

Most teams discover this when their agent's memory bill exceeds their model bill. You're paying to re-send the same facts, slightly rephrased, thousands of times per day.

![Abstract data compression and code visualization](https://images.unsplash.com/photo-1555066931-4365d14bab8c?w=1200&h=600&fit=crop)

## The Problem with Naive Summarization

Standard summarization drops facts. A support bot that "summarizes" a 50-message thread might lose:

- The user's account ID or tenant slug
- The exact error code (\`PAYMENT_TIMEOUT_429\`)
- The resolution that worked last time
- Temporal markers ("issue started after the June deploy")

Hystersis uses **ProMem-style extraction** — proactive memory extraction with self-verification, inspired by recent research on proactive memory for LLM agents.

## How ProMem Works (Four Phases)

### Phase 1: Self-Questioning

The system doesn't ask "what is this text about?" — it asks **what does this memory mean for future decisions?**

Generated questions target durable facts: preferences, constraints, outcomes, identifiers, and relationships.

### Phase 2: Answer in Context

Each question is answered strictly from the source material. No hallucinated supplements at this stage.

### Phase 3: Verification

Extracted facts are validated against the original memory using a high-accuracy model path. Contradictions and unsupported claims are rejected or flagged.

### Phase 4: Gap Detection

The system identifies missing critical information — if a support ticket mentions "payment failed" but omits the payment provider, gap detection surfaces the hole and requests supplementation.

![Server room with glowing lights representing async processing pipeline](https://images.unsplash.com/photo-1558494949-ef010cbdcc31?w=1200&h=600&fit=crop)

## Hybrid LLM Routing

Not every memory needs Claude. Simple extractions route to fast models; complex memories get a verify pass with high-accuracy models.

\`\`\`bash
COMPRESSION_LLM_FAST_PROVIDER=openai
COMPRESSION_LLM_FAST_MODEL=gpt-4o-mini
COMPRESSION_LLM_VERIFY_PROVIDER=anthropic
COMPRESSION_LLM_VERIFY_MODEL=claude-3-5-sonnet
COMPRESSION_COMPLEXITY_THRESHOLD=0.6
\`\`\`

| Path | When | Model tier |
|------|------|------------|
| Fast | Short, low-ambiguity facts | GPT-4o-mini, Groq |
| Verify | Multi-entity, policy, numeric | Claude, GPT-4o |

## Async Pipeline — Writes Stay Fast

Compression runs in a background worker pool. Your agent responds immediately; memory compacts asynchronously.

\`\`\`
POST /memories → 200 OK (<5ms write impact)
                 ↳ job queue → extract → verify → store compressed
\`\`\`

Configure worker count and priority in your deployment. Critical memories can jump the queue.

## Benchmarks vs. Baseline

| Metric | Hystersis ProMem | Naive summarize | Mem0 baseline |
|--------|------------------|-----------------|---------------|
| Accuracy retention | **≥97%** | ~85% | ~91% |
| Token reduction | **80–85%** | ~60% | ~80% |
| P95 compression latency | **<200ms** | ~400ms | ~400ms |
| Write impact | **<5ms** | blocking | blocking |

## API & Dashboard

\`\`\`bash
# Stats
GET /compression/stats

# Mode: extract | balanced | aggressive
PUT /compression/mode
{ "mode": "extract" }
\`\`\`

Try compression live in the [playground](https://hystersis.com/demo) or tune modes in [app.hystersis.com/settings](https://app.hystersis.com/settings).

![Analytics dashboard with charts showing performance metrics](https://images.unsplash.com/photo-1551288049-bebda4e38f71?w=1200&h=600&fit=crop)

## Operational Tips

1. Start with **extract** mode — highest accuracy retention
2. Move to **balanced** once you have 10k+ memories and want more aggressive savings
3. Monitor \`accuracy_retention\` and \`token_reduction\` in \`/compression/stats\`
4. Use tiered memory so hot facts stay in Redis while cold archives compress aggressively

Compression isn't about making memory smaller for its own sake. It's about **keeping the right facts** while spending fewer tokens to retrieve them.
    `,
  },
  {
    slug: 'spreading-activation-multi-hop-retrieval',
    title: 'Spreading Activation: Multi-Hop Memory Retrieval Beyond Vector Search',
    seoTitle: 'Spreading Activation: Multi-Hop Agent Memory Retrieval | Hystersis',
    seoDescription:
      'Go beyond vector search with spreading activation: graph propagation through Neo4j delivers +23% multi-hop reasoning improvement for AI agent memory.',
    keywords: ['spreading activation', 'multi-hop retrieval', 'graph search', 'vector search', 'agent memory'],
    excerpt:
      'Pure vector search misses connected facts. Spreading activation propagates through your knowledge graph for +23% improvement on multi-hop reasoning.',
    image: 'https://images.unsplash.com/photo-1558494949-ef010cbdcc31?w=1200&h=600&fit=crop',
    category: 'Engineering',
    date: 'May 18, 2026',
    readTime: '15 min read',
    tags: ['Search', 'Graph', 'RAG'],
    featured: true,
    content: `
# Spreading Activation: Multi-Hop Memory Retrieval

Vector search finds similar text. It doesn't find **connected** facts across hops.

Ask: *"Who manages the team that owns the payment service?"* — cosine similarity won't bridge User → Team → Service → Owner without explicit graph traversal.

![Network graph with interconnected nodes and edges](https://images.unsplash.com/photo-1451187580459-43490279c0fa?w=1200&h=600&fit=crop)

## The Multi-Hop Gap

Pure RAG fails on questions that require chaining:

1. Find the payment service entity
2. Traverse \`OWNED_BY\` to the platform team
3. Traverse \`MANAGED_BY\` to the engineering manager
4. Return the person + supporting context

Vector search might return a document mentioning the manager **if** those words appear together. Graph propagation returns the chain even when facts live in separate memories.

## Spreading Activation Algorithm

Hystersis implements graph-based retrieval inspired by episodic-semantic memory research:

### Step 1: Initial activation

Query → embedding → top-K vector hits from Qdrant. Each hit receives an activation score based on similarity.

### Step 2: Graph propagation

Activation spreads through Neo4j relationships with **0.85 decay per hop**. Strongly connected nodes accumulate signal; distant nodes fade.

### Step 3: Threshold collection

Nodes above **0.1 activation** become retrieval candidates.

### Step 4: Hybrid ranking

Final score blends activation level + vector similarity + optional recency boost.

\`\`\`bash
GET /search/enhanced?mode=spreading&query=payment+service+owner&limit=10
\`\`\`

\`\`\`json
{
  "results": [...],
  "mode": "spreading",
  "activation_hops": 3,
  "hop_breakdown": [12, 8, 3]
}
\`\`\`

## Hyperparameters (Tunable)

| Parameter | Default | Effect |
|-----------|---------|--------|
| \`decayFactor\` | 0.85 | Activation loss per hop |
| \`threshold\` | 0.1 | Minimum activation to include |
| \`maxHops\` | 3 | Max propagation depth |
| \`initialBudget\` | 1.0 | Starting activation per seed node |

Tune in production based on graph density. Dense org graphs may need lower decay; sparse graphs may need more hops.

![Data visualization with glowing connection lines](https://images.unsplash.com/photo-1551288049-bebda4e38f71?w=1200&h=600&fit=crop)

## When to Use Each Mode

| Mode | Best for | Example query |
|------|----------|---------------|
| **Vector** | Direct semantic match | "refund policy for EU customers" |
| **Spreading** | Multi-hop reasoning | "who approved the last infra change?" |
| **Hybrid** | Production default | Combines both signals |

## Benchmark Results

Spreading activation delivers **+23% improvement** on multi-hop reasoning benchmarks vs pure vector search — measured on entity-chain queries where answers require 2–3 relationship traversals.

The gain is largest when:

- Memories are entity-rich (people, teams, services, accounts)
- Facts are stored in separate sessions but linked in the graph
- Queries use organizational language, not document keywords

## Wiring It Up

\`\`\`python
results = client.search(
    query="payment service owner",
    mode="spreading",
    limit=10
)

for r in results:
    print(r.content, r.score, r.metadata.get("activation"))
\`\`\`

Enable in dashboard: **Settings → Search → Enhanced mode → Spreading**.

Combine with [knowledge graphs](/blog/knowledge-graphs-for-better-rag) for best results — spreading activation needs edges to propagate across.
    `,
  },
  {
    slug: 'building-memory-powered-ai-agents',
    title: 'Building Memory-Powered AI Agents from Scratch',
    seoTitle: 'Build Memory-Powered AI Agents: Python SDK Tutorial | Hystersis',
    seoDescription:
      'Hands-on tutorial: build a memory-powered AI agent with Hystersis Python SDK — sessions, semantic search, entity graphs, and production patterns.',
    keywords: ['AI agent tutorial', 'Python SDK', 'agent memory', 'Hystersis tutorial', 'build AI agent'],
    excerpt:
      'A hands-on tutorial: sessions, semantic search, and knowledge graphs with the Hystersis Python SDK.',
    image: 'https://images.unsplash.com/photo-1620712943543-bcc4688e7485?w=1200&h=600&fit=crop',
    category: 'Tutorial',
    date: 'May 10, 2026',
    readTime: '18 min read',
    tags: ['Tutorial', 'Python', 'SDK'],
    content: `
# Building Memory-Powered AI Agents from Scratch

This tutorial builds a **support agent** that remembers past tickets, contact preferences, and resolutions — across sessions, users, and handoffs.

![Python code on a laptop screen](https://images.unsplash.com/photo-1526374965328-7f61d4dc18c5?w=1200&h=600&fit=crop)

## Prerequisites

- Python 3.10+
- Hystersis API key ([get one](https://app.hystersis.com))
- 15 minutes

## Step 1: Install & Connect

\`\`\`bash
pip install hystersis
\`\`\`

\`\`\`python
from hystersis import Hystersis

client = Hystersis(
    api_key="your-api-key",
    base_url="https://api.hystersis.com"
)
\`\`\`

## Step 2: Create a Session

Sessions group episodic memories for a single interaction thread.

\`\`\`python
session = client.sessions.create(
    agent_id="support-bot",
    user_id="user-42",
    metadata={"channel": "web_chat"}
)
print(session.id)
\`\`\`

## Step 3: Store Memories

Add facts as they emerge — don't wait until conversation end.

\`\`\`python
client.memories.add(
    content="User prefers email over phone for follow-ups",
    user_id="user-42",
    session_id=session.id,
    metadata={"source": "conversation", "confidence": 0.95}
)

client.memories.add(
    content="User on Pro plan, SLA: 4-hour response",
    user_id="user-42",
    metadata={"source": "crm_sync"}
)
\`\`\`

## Step 4: Search Before Responding

Retrieve context **before** calling your LLM — not after.

\`\`\`python
def get_support_context(user_id: str, query: str) -> str:
    results = client.search(
        query=query,
        user_id=user_id,
        limit=8
    )
    return "\\n".join(f"- {m.content}" for m in results)

context = get_support_context("user-42", "contact preferences and plan tier")
print(context)
\`\`\`

![Team collaboration around a laptop](https://images.unsplash.com/photo-1522071820081-009f0129c71c?w=1200&h=600&fit=crop)

## Step 5: Link Entities in the Graph

Structured entities enable multi-hop retrieval later.

\`\`\`python
client.entities.create(name="Acme Corp", type="Organization", user_id="user-42")
client.entities.create(name="user-42", type="User", user_id="user-42")
client.entities.create(name="Pro Plan", type="Product", user_id="user-42")

client.graph.link("user-42", "WORKS_AT", "Acme Corp")
client.graph.link("user-42", "SUBSCRIBED_TO", "Pro Plan")
\`\`\`

## Step 6: Put It Together — Agent Loop

\`\`\`python
def handle_message(user_id: str, message: str) -> str:
    context = get_support_context(user_id, message)

    prompt = f"""You are a support agent. Use this context:
{context}

User message: {message}
"""

    # Call your LLM here
    response = call_llm(prompt)

    # Store new facts from the exchange
    client.memories.add(
        content=f"User asked: {message[:200]}",
        user_id=user_id,
        metadata={"type": "interaction"}
    )

    return response
\`\`\`

## Step 7: Enable Compression (Optional)

For long-running agents, enable async compression so memory stays bounded.

\`\`\`bash
curl -X PUT https://api.hystersis.com/compression/mode \\
  -H "Authorization: Bearer $API_KEY" \\
  -d '{"mode": "balanced"}'
\`\`\`

## What to Build Next

- Add [spreading activation](/blog/spreading-activation-multi-hop-retrieval) for org-chart queries
- Wire [skills](https://hystersis.com/docs/api-reference/skills) for procedural memory
- Set [tier policy](https://hystersis.com/docs) for hot/cold retention

Full API reference: [hystersis.com/docs](https://hystersis.com/docs)
    `,
  },
  {
    slug: 'knowledge-graphs-for-better-rag',
    title: 'Knowledge Graphs for Better RAG Systems',
    seoTitle: 'Knowledge Graphs for Better RAG: Neo4j + Vector Hybrid | Hystersis',
    seoDescription:
      'Combine vector search with Neo4j knowledge graphs for RAG that understands relationships. Entity extraction, graph expansion, and production tips.',
    keywords: ['knowledge graph RAG', 'Neo4j RAG', 'graph augmented retrieval', 'entity extraction', 'hybrid RAG'],
    excerpt:
      'Combine vector search with Neo4j knowledge graphs for retrieval that understands relationships, not just similarity.',
    image: 'https://images.unsplash.com/photo-1551288049-bebda4e38f71?w=1200&h=600&fit=crop',
    category: 'Engineering',
    date: 'May 3, 2026',
    readTime: '16 min read',
    tags: ['RAG', 'Neo4j', 'Graph'],
    content: `
# Knowledge Graphs for Better RAG Systems

Traditional RAG retrieves chunks. Graph-augmented RAG retrieves **connected knowledge** — entities, relationships, and the paths between them.

![Data analytics dashboard with network visualization](https://images.unsplash.com/photo-1460925895917-afdab827c52f?w=1200&h=600&fit=crop)

## The Limits of Chunk-Only RAG

Semantic search returns the most similar text — but similarity isn't structure.

Two facts about the same customer might live in different chunks with no retrieval bridge:

- "Acme upgraded to Enterprise in March"
- "Acme's primary contact is Jane Doe"

A vector query for "who is Acme's contact?" might miss the first chunk entirely. A graph query traverses \`Acme → HAS_CONTACT → Jane\` directly.

## Hystersis Hybrid Architecture

\`\`\`
Ingest → LLM entity extraction → Neo4j (graph) + Qdrant (vectors)
Query  → Vector seeds → Graph expansion → Ranked memories
\`\`\`

### Vector search for candidates

\`\`\`python
results = client.search(query="transformer architecture", limit=10)
\`\`\`

### Graph expansion for related entities

\`\`\`python
graph = client.graph.query("""
    MATCH (e:Entity)-[r]->(related)
    WHERE e.name CONTAINS 'Transformer'
    RETURN related.name, type(r), r.confidence
    ORDER BY r.confidence DESC
    LIMIT 20
""")
\`\`\`

![Database server infrastructure](https://images.unsplash.com/photo-1558494949-ef010cbdcc31?w=1200&h=600&fit=crop)

## Entity Extraction Pipeline

When memories are ingested, Hystersis automatically:

1. **Extracts entities** via LLM (Person, Organization, Product, Concept, …)
2. **Links them** in Neo4j with typed relationships
3. **Embeds content** in Qdrant for semantic fallback
4. **Compresses facts** asynchronously to control growth

You can also supply entities explicitly for high-precision domains:

\`\`\`python
client.entities.create(
    name="payment-service",
    type="Service",
    metadata={"team": "platform", "tier": "critical"}
)
\`\`\`

## Relationship Types That Matter

| Relationship | Example | Why it helps |
|--------------|---------|--------------|
| \`WORKS_AT\` | User → Org | Org-scoped retrieval |
| \`OWNS\` | Team → Service | Incident routing |
| \`DEPENDS_ON\` | Service → Service | Blast radius queries |
| \`PREFERS\` | User → Channel | Personalization |

Consistent typing is more important than exhaustive typing. Start with 5–10 relationship types and expand.

## Conflict Resolution

Memories contradict. Users change jobs. Plans upgrade.

Hystersis runs LLM-powered conflict resolution when new facts collide with stored ones — surfacing superseded memories instead of silently overwriting.

\`\`\`python
client.memories.add(
    "User prefers Slack notifications",
    user_id="user-42"
)
# Later...
client.memories.add(
    "User prefers email notifications",
    user_id="user-42"
)
# → Conflict detected, latest preference wins with audit trail
\`\`\`

## Production Tips

1. Use **entity types** consistently across agents
2. Set **importance scores** on high-value facts (policies, IDs, SLAs)
3. Enable **enhanced search** with spreading activation for multi-hop
4. Monitor graph growth — run entity deduplication quarterly

See [enhanced search docs](https://hystersis.com/docs/api-reference/search) and [Neo4j configuration](https://hystersis.com/docs/getting-started/configuration).
    `,
  },
  {
    slug: 'scaling-hystersis-to-production',
    title: 'Scaling Hystersis to Millions of Memories',
    seoTitle: 'Scale Agent Memory to Millions: Tiered Architecture | Hystersis',
    seoDescription:
      'Scale Hystersis to millions of memories: tiered storage, async compression, multi-tenant isolation, and sub-200ms retrieval at production load.',
    keywords: ['scale agent memory', 'tiered memory', 'production AI', 'multi-tenant', 'memory architecture'],
    excerpt:
      'Architecture patterns for production: tiered memory, async compression, multi-tenant isolation, and sub-200ms retrieval.',
    image: 'https://images.unsplash.com/photo-1451187580459-43490279c0fa?w=1200&h=600&fit=crop',
    category: 'Architecture',
    date: 'Apr 26, 2026',
    readTime: '17 min read',
    tags: ['Scaling', 'Architecture', 'Production'],
    content: `
# Scaling Hystersis to Millions of Memories

Hystersis is built for production from day one — not retrofitted for scale. This guide covers the architecture patterns that keep latency flat as memory count grows.

![Global network connectivity visualization](https://images.unsplash.com/photo-1451187580459-43490279c0fa?w=1200&h=600&fit=crop)

## Tiered Memory Architecture

Not every memory deserves the same storage cost or retrieval speed.

| Tier | Storage | Latency | Default retention | Use case |
|------|---------|---------|-------------------|----------|
| **Working** | In-memory | <5ms | Session | Active tool loop context |
| **Hot** | Redis | <20ms | 7 days | Recent user facts |
| **Cold** | Neo4j + Qdrant | <100ms | Unlimited | Full history |
| **Archive** | Object storage | >1s | Compliance | Legal hold, exports |

\`\`\`bash
# Policy: aggressive | balanced | conservative
PUT /tier/policy
{ "policy": "balanced" }
\`\`\`

**Balanced** (default): 7-day hot window, archive after 100 accesses or 90 days cold.

## Async Compression Pipeline

Writes must never block on LLM compression.

\`\`\`
POST /memories → persist raw → 200 OK
                 ↳ queue (priority 0=critical, 2=normal)
                 ↳ worker pool (4–8 workers)
                 ↳ ProMem extract → verify → replace with compressed
\`\`\`

P99 write latency stays **under 5ms** because compression is fully async.

![Cloud infrastructure and server racks](https://images.unsplash.com/photo-1558494949-ef010cbdcc31?w=1200&h=600&fit=crop)

## Multi-Tenant Isolation

API keys map to tenants. Every query automatically filters by tenant — no cross-customer data leakage.

\`\`\`yaml
# config.yaml
api_keys:
  prod_acme: tenant_acme
  prod_globex: tenant_globex
\`\`\`

RBAC layers on top: roles control admin operations, skill sharing, and audit access. See [RBAC docs](https://hystersis.com/docs/features/rbac).

## Retrieval at Scale

| Operation | Target P95 | Scaling lever |
|-----------|------------|---------------|
| Vector search | <100ms | Qdrant sharding, HNSW tuning |
| Graph 1-hop | <30ms | Neo4j indexes on entity name + type |
| Graph 3-hop | <80ms | Limit max hops, prune low-confidence edges |
| Spreading activation | <150ms | Seed cap (top-50), activation threshold |

### Indexing checklist

\`\`\`cypher
CREATE INDEX entity_name IF NOT EXISTS FOR (e:Entity) ON (e.name);
CREATE INDEX entity_tenant IF NOT EXISTS FOR (e:Entity) ON (e.tenant_id);
\`\`\`

## Observability

\`\`\`bash
GET /compression/stats
GET /analytics/dashboard
\`\`\`

Track: \`accuracy_retention\`, \`token_reduction\`, \`avg_latency_ms\`, \`p95_latency_ms\`, memories per tenant, graph node count.

Wire alerts when P95 exceeds SLO or compression queue depth grows beyond threshold.

## Deployment Options

| Environment | Path |
|-------------|------|
| Local dev | \`docker compose up\` |
| Single VM | [Install script](https://hystersis.com/install.sh) |
| Kubernetes | [Helm chart](https://hystersis.com/docs/deployment/kubernetes) |
| Edge (landing/dashboard) | Cloudflare Workers |

## Capacity Planning Rule of Thumb

| Memories | Qdrant | Neo4j | Redis hot |
|----------|--------|-------|-----------|
| 100K | 1 node, 4GB | 2GB heap | 512MB |
| 1M | 3-node cluster | 8GB heap | 2GB |
| 10M | Sharded collection | 16GB+ heap | 8GB cluster |

Start smaller. Tiered memory and compression keep the hot set bounded — you rarely query all 10M memories on every request.

![Team reviewing architecture diagrams on whiteboard](https://images.unsplash.com/photo-1531482615713-2afd69097998?w=1200&h=600&fit=crop)

## Next Steps

- Load test with your agent's actual query distribution
- Enable [spreading activation](/blog/spreading-activation-multi-hop-retrieval) if >30% of queries are multi-hop
- Review [security hardening](https://hystersis.com/docs/production/security) before GA
    `,
  },
]

export const blogs = [...technicalBlogs, ...coreBlogs]

export const getBlogBySlug = (slug) => blogs.find((b) => b.slug === slug)
