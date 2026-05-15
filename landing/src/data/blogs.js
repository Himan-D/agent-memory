export const blogs = [
  {
    slug: 'building-memory-powered-ai-agents',
    title: 'Building Memory-Powered AI Agents from Scratch',
    excerpt: 'Learn how to build AI agents that remember conversations across sessions using semantic search and knowledge graphs.',
    image: 'https://images.unsplash.com/photo-1620712943543-bcc4688e7485?w=1200&h=600&fit=crop',
    category: 'Tutorial',
    date: 'Jan 15, 2025',
    readTime: '8 min read',
    content: `
# Building Memory-Powered AI Agents from Scratch

AI agents are getting smarter every day, but there's one thing they all have in common: they forget everything after each conversation. In this tutorial, we'll build a memory-powered AI agent that actually remembers.

## Why Memory Matters

Without memory, every conversation starts from scratch. Your agent can't recall:
- Previous interactions with the same user
- Patterns in past conversations
- Important context that could help personalization

## The Solution: Agent Memory

We'll use Agent Memory to add three types of memory:

### 1. Conversational Memory
Store and retrieve conversation history:

\`\`\`python
session = client.create_session(agent_id="assistant-bot")
client.add_message(session["id"], "user", "I love machine learning!")
client.add_message(session["id"], "assistant", "That's great! What type?")
\`\`\`

### 2. Knowledge Graphs
Connect entities with relationships:

\`\`\`python
client.create_entity(name="UserPreferences", type="Preference")
client.create_relation("user-123", "UserPreferences", "HAS")
\`\`\`

### 3. Semantic Search
Find similar content using vector embeddings:

\`\`\`python
results = client.semantic_search("deep learning transformers")
# Returns semantically similar messages
\`\`\`

## Putting It All Together

Here's a complete example of a memory-powered support bot:

\`\`\`python
from agentmemory import AgentMemory

client = AgentMemory("https://api.yourserver.com", api_key="your-key")

# Create session for new conversation
session = client.create_session(agent_id="support-bot")

# Add user message
client.add_message(session["id"], "user", "I can't login to my account")

# Search for similar past issues
past_issues = client.semantic_search("login problems", limit=5)

# Use past context to provide better response
if past_issues:
    response = f"I see similar issues were resolved by resetting passwords. Would you like me to help with that?"
else:
    response = "I'll help you troubleshoot your login issue."

client.add_message(session["id"], "assistant", response)
\`\`\`

## Conclusion

Adding memory to your AI agents is straightforward with Agent Memory. Start with simple message storage and progressively add knowledge graphs and semantic search for more powerful capabilities.

## Next Steps

- Read our [API Reference](/docs)
- Check out [more use cases](/use-cases)
- Join our [community](https://github.com/Himan-D/agent-memory)
    `
  },
  {
    slug: 'knowledge-graphs-for-better-rag',
    title: 'Knowledge Graphs for Better RAG Systems',
    excerpt: 'How to enhance retrieval-augmented generation with structured knowledge graphs for more accurate AI responses.',
    image: 'https://images.unsplash.com/photo-1558494949-ef010cbdcc31?w=1200&h=600&fit=crop',
    category: 'Engineering',
    date: 'Jan 8, 2025',
    readTime: '12 min read',
    content: `
# Knowledge Graphs for Better RAG Systems

Retrieval-Augmented Generation (RAG) has become the standard way to build AI systems with external knowledge. But traditional RAG has limits. Let's see how knowledge graphs can help.

## The Problem with Traditional RAG

Traditional RAG uses semantic search to find relevant documents, but it misses:
- **Relationships** between entities
- **Structured data** like graphs and hierarchies
- **Context** that connects pieces of information

## Enter Knowledge Graphs

Knowledge graphs store information as connected nodes with typed relationships:

\`\`\`python
# Add entities
client.create_entity(name="Transformer", type="Architecture", 
    properties={"year": 2017, "paper": "Attention Is All You Need"})
client.create_entity(name="SelfAttention", type="Mechanism")
client.create_entity(name="GPT", type="Model", 
    properties={"provider": "OpenAI", "versions": ["GPT-3", "GPT-4"]})

# Connect them
client.create_relation("Transformer", "SelfAttention", "USES")
client.create_relation("GPT", "Transformer", "BASED_ON")
\`\`\`

## Hybrid RAG Approach

Combine semantic search with graph queries for better results:

\`\`\`python
# First, semantic search for relevant context
semantic_results = client.semantic_search("how does attention work in transformers")

# Then, graph traversal for related concepts
graph_results = client.graph_query("""
    MATCH (a:Architecture)-[:USES]->(m:Mechanism)
    WHERE a.name CONTAINS 'Transformer'
    RETURN m.name, m.properties
""")
\`\`\`

## Real-World Example

Build a research assistant that understands paper relationships:

\`\`\`python
# Index a research paper
paper = client.create_entity(
    name="Attention Is All You Need",
    type="Paper",
    properties={
        "authors": ["Vaswani", "Shazeer", "Parmar"],
        "year": 2017,
        "abstract": "..."
    }
)

# Add key concepts
client.create_entity(name="Transformer", type="Architecture")
client.create_entity(name="Self-Attention", type="Mechanism")
client.create_entity(name="Seq2Seq", type="Model")

# Connect concepts
client.create_relation(paper["id"], "Transformer", "INTRODUCES")
client.create_relation("Transformer", "Self-Attention", "USES")
\`\`\`

Now you can query: "What did Attention Is All You Need introduce?" and get precise answers.

## Benefits of Graph-Enhanced RAG

1. **Better context** - Understand relationships between concepts
2. **Accurate answers** - Graph paths provide precise information
3. **Explainability** - Trace answers back to source nodes
4. **Reasoning** - Graph traversal enables logical inference

## Conclusion

Adding knowledge graphs to your RAG system dramatically improves accuracy and enables new capabilities. Start simple and add graph features as needed.
    `
  },
  {
    slug: 'scaling-agent-memory-to-millions',
    title: 'Scaling Agent Memory to Millions of Users',
    excerpt: 'Architectural patterns and best practices for building production-ready agent memory systems at scale.',
    image: 'https://images.unsplash.com/photo-1451187580459-43490279c0fa?w=1200&h=600&fit=crop',
    category: 'Architecture',
    date: 'Jan 1, 2025',
    readTime: '15 min read',
    content: `
# Scaling Agent Memory to Millions of Users

Building a demo is easy. Building a system that scales to millions of users is harder. Here's what we've learned.

## Core Challenges

When scaling agent memory, you face three challenges:

1. **Data volume** - Millions of messages, entities, relationships
2. **Query latency** - Users expect sub-100ms responses
3. **Multi-tenancy** - Isolating data between customers

## Architecture Overview

\`\`\`
┌─────────────┐     ┌─────────────────┐     ┌────────────┐
│   Client    │────▶│   Agent Memory  │────▶│   Neo4j    │
└─────────────┘     │   (Go Server)   │     │  (Graph)   │
                    └─────────────────┘     └────────────┘
                            │
                            │
                    ┌───────▼───────┐
                    │   Qdrant     │
                    │  (Vectors)  │
                    └──────────────┘
\`\`\`

## Scaling Neo4j

### Connection Pooling

Neo4j handles concurrent connections with pooling:

\`\`\`go
// Configurable pool size
Neo4jMaxConnections: 50,
Neo4jMaxConnectionLifetime: 30 * time.Minute,
\`\`\`

### Query Optimization

Use indexes for fast lookups:

\`\`\`cypher
CREATE INDEX entity_type FOR (e:Entity) ON (e.type)
CREATE INDEX entity_name FOR (e:Entity) ON (e.name)
CREATE INDEX session_agent FOR (s:Session) ON (s.agent_id)
\`\`\`

## Scaling Qdrant

### Vector Indexing

Use HNSW for fast approximate nearest neighbor search:

\`\`\`python
# Qdrant automatically manages vector indexes
client.create_collection(
    name="messages",
    vector_size=1536,
    distance="Cosine"
)
\`\`\`

### Sharding

Partition data across multiple Qdrant nodes:

\`\`\`yaml
# qdrant.yaml
storage:
  snapshots_path: /snapshots
  
cluster:
  p2p:
    port: 6334
    num_shards: 4
\`\`\`

## Multi-Tenant Isolation

### API Key Mapping

Map API keys to tenants in configuration:

\`\`\`yaml
# config.yaml
api_keys:
  key_prod_abc123: tenant_acme
  key_prod_xyz789: tenant_globex
  key_staging: tenant_internal
\`\`\`

### Query Filtering

All queries automatically filter by tenant:

\`\`\`python
# Internal: tenant is added to all queries
def semantic_search(self, query, limit=10):
    cypher = """
        MATCH (s:Session)-[:HAS_MESSAGE]->(m:Message)
        WHERE s.agent_id = $agent_id
          AND s.tenant_id = $tenant_id
        RETURN m.content, m.embedding
        ORDER BY similarity($query_embedding, m.embedding)
        LIMIT $limit
    """
\`\`\`

## Performance Numbers

Here's what you can expect at scale:

| Metric | Value |
|-------|-------|
| Vector search | <100ms |
| Graph queries | <50ms |
| Message storage | <20ms |
| Concurrent connections | 50 per client |
| Throughput | 1000 req/sec |

## Best Practices

1. **Batch writes** - Buffer messages and write in batches
2. **Connection pooling** - Reuse connections efficiently
3. **Query caching** - Cache frequent queries
4. **Async processing** - Process non-critical operations async

## Conclusion

Scaling to millions requires careful architecture but is achievable. Start with proper indexing and pooling, then add sharding as needed.
    `
  },
  {
    slug: 'multi-tenant-saas-with-agent-memory',
    title: 'Building Multi-Tenant SaaS with Agent Memory',
    excerpt: 'How to build a complete multi-tenant SaaS platform using Agent Memory with proper isolation.',
    image: 'https://images.unsplash.com/photo-1460925895917-afdab827c52f?w=1200&h=600&fit=crop',
    category: 'Tutorial',
    date: 'Dec 15, 2024',
    readTime: '10 min read',
    content: `
# Building Multi-Tenant SaaS with Agent Memory

Multi-tenancy is essential for any SaaS product. Learn how to build a complete multi-tenant system with Agent Memory.

## What is Multi-Tenancy?

Multi-tenancy means serving multiple customers (tenants) from a single infrastructure while keeping their data completely isolated.

## Architecture

\`\`\`
┌─────────────────────────────────────────┐
│           Agent Memory Server          │
├─────────────────────────────────────────┤
│  API Key: key1 ──▶ Tenant: acme         │
│  API Key: key2 ──▶ Tenant: globex      │
│  API Key: key3 ─-▶ Tenant: stark      │
└─────────────────────────────────────────┘
\`\`\`

## Configuration

Set up API keys with tenant mapping:

\`\`\`yaml
# config.yaml
server:
  host: 0.0.0.0
  port: 8080

auth:
  api_keys:
    prod_abc123: acme_corp
    prod_xyz789: globex_inc
    prod_stark: stark_ind

database:
  neo4j:
    uri: bolt://localhost:7687
    username: neo4j
    password: 'NEO4J_PASSWORD'
  
  qdrant:
    url: http://localhost:6333
\`\`\`

## Usage in Python

Different API keys automatically get isolated data:

\`\`\`python
# Customer A - acme_corp
client_a = AgentMemory(
    "https://api.agentmemory.io", 
    api_key="prod_abc123"
)
session_a = client_a.create_session(agent_id="support-bot")
# Session is automatically tagged with tenant_id="acme_corp"

# Customer B - globex_inc  
client_b = AgentMemory(
    "https://api.agentmemory.io",
    api_key="prod_xyz789"
)
session_b = client_b.create_session(agent_id="support-bot")
# Session is automatically tagged with tenant_id="globex_inc"
\`\`\`

## Server Implementation

Here's how the server handles tenant isolation:

\`\`\`go
func (s *Server) createSession(w http.ResponseWriter, r *http.Request) {
    apiKey := r.Header.Get("X-API-Key")
    tenantID := s.auth.GetTenant(apiKey)
    
    req := &CreateSessionRequest{}
    json.NewDecoder(r.Body).Decode(req)
    
    // Tenant is automatically added
    session := &Session{
        ID:        generateUUID(),
        AgentID:   req.AgentID,
        TenantID:  tenantID,  // From API key
        Metadata:  req.Metadata,
        CreatedAt: time.Now(),
    }
    
    s.neo4j.CreateSession(ctx, session)
    
    w.JSON(201, session)
}
\`\`\`

## Security Considerations

1. **Never expose tenant IDs in URLs**
2. **Validate API keys on every request**
3. **Log tenant identifiers for audits**
4. **Separate backups per tenant**

## Benefits

| Benefit | Description |
|---------|-------------|
| Cost efficiency | Single infrastructure |
| Easy management | One deployment |
| Isolation | Complete data separation |
| Scalability | Add tenants without provisioning |

## Conclusion

Multi-tenancy with Agent Memory is straightforward. Your API keys map to tenants, and all operations automatically filter by tenant.
    `
  },
  {
    slug: 'real-time-conversation-summarization',
    title: 'Real-Time Conversation Summarization with Agent Memory',
    excerpt: 'Build AI assistants that automatically summarize conversations in real-time using Agent Memory.',
    image: 'https://images.unsplash.com/photo-1557804506-669a67965ba0?w=1200&h=600&fit=crop',
    category: 'Tutorial',
    date: 'Dec 8, 2024',
    readTime: '7 min read',
    content: `
# Real-Time Conversation Summarization with Agent Memory

One powerful use case: automatically summarize long conversations. Here's how to build it.

## The Problem

Support conversations can go on for hours. When the user returns days later, no one remembers what was discussed.

## The Solution

Use Agent Memory to store messages and generate summaries:

\`\`\`python
from agentmemory import AgentMemory
from openai import OpenAI

client = AgentMemory("https://api.yourserver.com", api_key="your-key")
openai = OpenAI(api_key="your-openai-key")

session = client.create_session(agent_id="support-bot")

# Add messages throughout conversation
client.add_message(session["id"], "user", "I need help with my order")
client.add_message(session["id"], "assistant", "I'd be happy to help. What's your order number?")
# ... more messages ...

# Generate summary when conversation ends
messages = client.get_messages(session["id"])

summary_prompt = f"""Summarize this conversation in 2-3 sentences:

User: {messages[0]['content']}
Assistant: {messages[1]['content']}
"""

summary = openai.chat.completions.create(
    model="gpt-4",
    messages=[{"role": "user", "content": summary_prompt}]
)

# Store the summary
client.create_entity(
    name=f"summary-{session['id']}",
    type="Summary",
    properties={
        "content": summary.choices[0].message.content,
        "session_id": session["id"]
    }
)
\`\`\`

## Auto-Summarization Trigger

Trigger summaries automatically:

\`\`\`python
import asyncio
from datetime import datetime, timedelta

async def monitor_sessions():
    while True:
        # Find active sessions older than 30 minutes
        old_sessions = client.find_sessions(
            status="active",
            older_than=datetime.now() - timedelta(minutes=30)
        )
        
        for session in old_sessions:
            messages = client.get_messages(session["id"])
            
            if len(messages) >= 10:
                # Summarize after 10+ messages
                await summarize_session(session["id"])
        
        await asyncio.sleep(300)  # Check every 5 minutes

asyncio.run(monitor_sessions())
\`\`\`

## Using Summaries

When the user returns:

\`\`\`python
def handle_new_session(user_id):
    # Find previous sessions
    past = client.semantic_search(
        f"support conversation {user_id}",
        limit=3
    )
    
    # Load context from past summaries
    context = "Previous conversation: " + past[0]["content"]
    
    return context
\`\`\`

## Benefits

- **No context loss** - Every conversation is summarized
- **Faster onboarding** - Agents know past issues immediately  
- **Analytics** - Analyze summaries for insights
- **Compliance** - Keep record of all conversations
    `
  }
]

export const getBlogBySlug = (slug) => blogs.find(b => b.slug === slug)