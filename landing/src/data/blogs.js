export const blogs = [
  {
    slug: 'why-ai-agents-need-persistent-memory',
    title: 'Why AI Agents Need Persistent Memory',
    excerpt:
      'Stateless agents forget everything between sessions. Persistent memory is the infrastructure layer that turns one-shot chatbots into compounding intelligence.',
    image: 'https://images.unsplash.com/photo-1677442136019-21780ecad995?w=1200&h=600&fit=crop',
    category: 'News',
    date: 'Jun 1, 2026',
    readTime: '6 min read',
    tags: ['AI Agents', 'Memory', 'Infrastructure'],
    content: `
# Why AI Agents Need Persistent Memory

Every new chat session starts from zero. Your agent doesn't remember yesterday's debugging session, last week's customer issue, or the preferences a user expressed three conversations ago.

That's not a UX problem — it's an **infrastructure** problem.

## The Amnesia Tax

Stateless agents pay an amnesia tax on every interaction:

- **Repeated context** — Users re-explain the same facts
- **No compounding** — Agents never get smarter about a specific user or domain
- **Broken workflows** — Multi-day tasks can't span sessions

Hystersis exists to eliminate that tax.

## Three Memory Layers

Production agent memory isn't one database — it's a stack:

### 1. Episodic Memory (Sessions)
Conversation history, tool calls, and interaction traces. Your agent recalls *what happened*.

### 2. Semantic Memory (Vectors)
Embeddings over facts and documents. Your agent finds *what's relevant* by meaning, not keywords.

### 3. Relational Memory (Graph)
Entities and relationships in Neo4j. Your agent understands *how things connect* — "John works at Acme" → "Acme uses Kubernetes."

## Compression Without Amnesia

Storing everything verbatim doesn't scale. Hystersis's proprietary compression engine extracts durable facts with 85%+ token reduction while retaining 97%+ accuracy — so memory grows without blowing context windows.

## Getting Started

\`\`\`bash
curl -fsSL https://hystersis.com/install.sh | bash
\`\`\`

Then add your first memory:

\`\`\`python
from hystersis import Hystersis

client = Hystersis(api_key="your-key")
client.add("User prefers TypeScript and uses pnpm", user_id="user-123")
\`\`\`

Persistent memory isn't optional for production agents. It's the difference between a demo and a product.
    `,
  },
  {
    slug: 'promem-compression-85-percent-token-reduction',
    title: 'ProMem Compression: 85% Token Reduction Without Losing Accuracy',
    excerpt:
      'How Hystersis proprietary ProMem extraction compresses agent memory by 85% while retaining 97%+ factual accuracy — our primary competitive advantage.',
    image: 'https://images.unsplash.com/photo-1555949963-aa79dcee981c?w=1200&h=600&fit=crop',
    category: 'Engineering',
    date: 'May 25, 2026',
    readTime: '9 min read',
    tags: ['Compression', 'ProMem', 'LLM'],
    content: `
# ProMem Compression: 85% Token Reduction Without Losing Accuracy

Context windows are finite. Agent memory is infinite. Something has to give — unless you compress intelligently.

## The Problem with Naive Summarization

Standard summarization drops facts. A support bot that "summarizes" a 50-message thread might lose the user's account ID, the exact error code, or the resolution that worked last time.

Hystersis uses **ProMem-style extraction** — proactive memory extraction with self-verification.

## How ProMem Works

1. **Self-questioning** — The system asks what a memory *means*, not just what it says
2. **Verification** — Extracted facts are validated against the source
3. **Gap detection** — Missing critical information is identified and filled
4. **Active extraction** — Key facts are pulled, not passively summarized

## Hybrid LLM Routing

Not every memory needs Claude. Simple extractions route to fast models (GPT-4o-mini); complex memories get a verify pass with high-accuracy models.

\`\`\`bash
COMPRESSION_LLM_FAST_PROVIDER=openai
COMPRESSION_LLM_FAST_MODEL=gpt-4o-mini
COMPRESSION_LLM_VERIFY_PROVIDER=anthropic
COMPRESSION_LLM_VERIFY_MODEL=claude-3-5-sonnet
\`\`\`

## Async Pipeline

Compression runs in the background — write latency impact stays under 5ms. Your agent responds immediately; memory compacts asynchronously.

## Benchmarks

| Metric | Hystersis | Baseline |
|--------|-----------|----------|
| Accuracy retention | ≥97% | ~91% |
| Token reduction | 80–85% | ~80% |
| P95 latency | <200ms | ~400ms |

Try it in the [compression playground](https://hystersis.com/demo).
    `,
  },
  {
    slug: 'spreading-activation-multi-hop-retrieval',
    title: 'Spreading Activation: Multi-Hop Memory Retrieval Beyond Vector Search',
    excerpt:
      'Pure vector search misses connected facts. Spreading activation propagates through your knowledge graph for +23% improvement on multi-hop reasoning.',
    image: 'https://images.unsplash.com/photo-1558494949-ef010cbdcc31?w=1200&h=600&fit=crop',
    category: 'Engineering',
    date: 'May 18, 2026',
    readTime: '10 min read',
    tags: ['Search', 'Graph', 'RAG'],
    content: `
# Spreading Activation: Multi-Hop Memory Retrieval

Vector search finds similar text. It doesn't find **connected** facts across hops.

"Who manages the team that owns the payment service?" requires traversing relationships — not just cosine similarity.

## Spreading Activation Algorithm

Hystersis implements graph-based retrieval inspired by episodic-semantic memory research:

1. **Query → embedding** — Initial activation from vector similarity (top-K from Qdrant)
2. **Graph propagation** — Activation spreads through Neo4j with 0.85 decay per hop
3. **Threshold collection** — Nodes above 0.1 activation are candidates
4. **Hybrid ranking** — Activation level + vector similarity combined

\`\`\`bash
GET /search/enhanced?mode=spreading&query=payment+service+owner
\`\`\`

## When to Use Each Mode

| Mode | Best for |
|------|----------|
| Vector | Direct semantic matches |
| Spreading | Multi-hop reasoning, entity chains |
| Hybrid | Production default — both signals |

## Results

Spreading activation delivers **+23% improvement** on multi-hop reasoning benchmarks vs pure vector search — the difference between retrieving a document and understanding a chain of facts.

Configure in your dashboard at [app.hystersis.com](https://app.hystersis.com) under Settings → Search.
    `,
  },
  {
    slug: 'building-memory-powered-ai-agents',
    title: 'Building Memory-Powered AI Agents from Scratch',
    excerpt:
      'A hands-on tutorial: sessions, semantic search, and knowledge graphs with the Hystersis Python SDK.',
    image: 'https://images.unsplash.com/photo-1620712943543-bcc4688e7485?w=1200&h=600&fit=crop',
    category: 'Tutorial',
    date: 'May 10, 2026',
    readTime: '8 min read',
    tags: ['Tutorial', 'Python', 'SDK'],
    content: `
# Building Memory-Powered AI Agents from Scratch

This tutorial walks through building a support agent that remembers past tickets, preferences, and resolutions.

## Setup

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

## Step 1: Create a Session

\`\`\`python
session = client.sessions.create(agent_id="support-bot", user_id="user-42")
\`\`\`

## Step 2: Store Memories

\`\`\`python
client.memories.add(
    content="User prefers email over phone for follow-ups",
    user_id="user-42",
    metadata={"source": "conversation"}
)
\`\`\`

## Step 3: Search Before Responding

\`\`\`python
context = client.search(
    query="user contact preferences",
    user_id="user-42",
    limit=5
)

for memory in context:
    print(memory.content)
\`\`\`

## Step 4: Link Entities in the Graph

\`\`\`python
client.entities.create(name="Acme Corp", type="Organization")
client.entities.create(name="user-42", type="User")
client.graph.link("user-42", "WORKS_AT", "Acme Corp")
\`\`\`

## Next Steps

- Enable [spreading activation](/blog/spreading-activation-multi-hop-retrieval) for multi-hop queries
- Configure [compression mode](https://app.hystersis.com/settings) for long-running agents
- Read the [API docs](https://hystersis.com/docs)
    `,
  },
  {
    slug: 'knowledge-graphs-for-better-rag',
    title: 'Knowledge Graphs for Better RAG Systems',
    excerpt:
      'Combine vector search with Neo4j knowledge graphs for retrieval that understands relationships, not just similarity.',
    image: 'https://images.unsplash.com/photo-1551288049-bebda4e38f71?w=1200&h=600&fit=crop',
    category: 'Engineering',
    date: 'May 3, 2026',
    readTime: '11 min read',
    tags: ['RAG', 'Neo4j', 'Graph'],
    content: `
# Knowledge Graphs for Better RAG Systems

Traditional RAG retrieves chunks. Graph-augmented RAG retrieves **connected knowledge**.

## The Limits of Chunk-Only RAG

Semantic search returns the most similar text — but similarity isn't structure. Two facts about the same entity might live in different chunks with no retrieval bridge.

## Hystersis Hybrid Approach

\`\`\`python
# Vector search for initial candidates
results = client.search(query="transformer architecture", limit=10)

# Graph expansion for related entities
graph = client.graph.query("""
    MATCH (e:Entity)-[r]->(related)
    WHERE e.name CONTAINS 'Transformer'
    RETURN related.name, type(r)
""")
\`\`\`

## Entity Extraction Pipeline

When memories are ingested, Hystersis automatically:

1. Extracts entities via LLM
2. Links them in Neo4j
3. Embeds content in Qdrant
4. Compresses facts asynchronously

## Production Tips

- Use **entity types** consistently (Person, Organization, Product)
- Set **importance scores** on high-value facts
- Enable **conflict resolution** when memories contradict

See [enhanced search docs](https://hystersis.com/docs/api-reference/search) for API details.
    `,
  },
  {
    slug: 'scaling-hystersis-to-production',
    title: 'Scaling Hystersis to Millions of Memories',
    excerpt:
      'Architecture patterns for production: tiered memory, async compression, multi-tenant isolation, and sub-200ms retrieval.',
    image: 'https://images.unsplash.com/photo-1451187580459-43490279c0fa?w=1200&h=600&fit=crop',
    category: 'Architecture',
    date: 'Apr 26, 2026',
    readTime: '12 min read',
    tags: ['Scaling', 'Architecture', 'Production'],
    content: `
# Scaling Hystersis to Millions of Memories

Hystersis is built for production from day one — not retrofitted for scale.

## Tiered Memory Architecture

| Tier | Storage | Latency | Use case |
|------|---------|---------|----------|
| Working | In-memory | <5ms | Active session context |
| Hot | Redis | <20ms | Recent memories (7-day default) |
| Cold | Neo4j + Qdrant | <100ms | Full history |
| Archive | Object storage | >1s | Compliance, long-term |

Configure policy at \`PUT /tier/policy\` — aggressive, balanced, or conservative.

## Async Compression Pipeline

Writes return immediately. Compression runs in a background worker pool (4–8 workers) so P99 write latency stays under 5ms.

## Multi-Tenant Isolation

API keys map to tenants. Every query automatically filters by tenant — no cross-customer data leakage.

\`\`\`yaml
# config.yaml
api_keys:
  prod_acme: tenant_acme
  prod_globex: tenant_globex
\`\`\`

## Performance Targets

| Metric | Target |
|--------|--------|
| Vector search P95 | <100ms |
| Graph query P95 | <50ms |
| Compression P95 | <200ms |
| Concurrent connections | 1000+ |

Deploy with [Docker Compose](https://hystersis.com/docs/deployment/docker) or [Kubernetes](https://hystersis.com/docs/deployment/kubernetes).
    `,
  },
]

export const getBlogBySlug = (slug) => blogs.find((b) => b.slug === slug)
