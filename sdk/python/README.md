# Agent Memory Python SDK

<p align="center">
  <a href="https://pypi.org/project/agentmemory/">
    <img src="https://img.shields.io/pypi/v/agentmemory" alt="PyPI">
  </a>
  <a href="https://pypi.org/project/agentmemory/">
    <img src="https://img.shields.io/pypi/pyversions/agentmemory" alt="PyPI - Python Version">
  </a>
</p>

Give your AI agents permanent memory with graph relationships and semantic search.

## Why Use Agent Memory SDK?

- **Simple** - One-line installation, intuitive API
- **Powerful** - Combine conversation history with knowledge graphs
- **Production-ready** - Type hints, error handling, timeouts
- **ProMem Extraction** - 97%+ accuracy memory compression (PROPRIETARY)
- **Spreading Activation** - +23% better multi-hop reasoning (PROPRIETARY)

## Installation

```bash
pip install agentmemory
```

Or with extra dependencies:

```bash
pip install agentmemory[async]  # For async support (coming soon)
```

## Quick Start

```python
from agentmemory import AgentMemory

# Connect to your agent memory server
client = AgentMemory(
    base_url="https://api.yourserver.com",  # or http://localhost:8080
    api_key="your-api-key"  # or set AGENT_MEMORY_API_KEY env var
)

# Create a conversation session
session = client.create_session(agent_id="my-assistant")

# Add messages
client.add_message(session["id"], "user", "I love machine learning!")
client.add_message(session["id"], "assistant", "That's great! What type?")

# Later, search semantically
results = client.search("deep learning")
# Returns: [{"score": 0.92, "entity": {...}}, ...]
```

---

# NEW: Compression Engine

Hystersis includes a **proprietary compression engine** that outperforms Mem0:

| Metric | Hystersis | Mem0 |
|--------|-----------|------|
| Accuracy Retention | **97%+** | 91% |
| Token Reduction | **80-85%** | 80% |
| Multi-hop Reasoning | **+23%** | baseline |

## Compression Control

```python
from agentmemory import CompressionMode

# Set compression mode
client.set_compression_mode(CompressionMode.EXTRACT)   # 97%+ accuracy
# Options: EXTRACT, BALANCED, AGGRESSIVE

# Get compression statistics
stats = client.get_compression_stats()
print(stats)
# {
#     "accuracy_retention": 0.973,
#     "token_reduction": 0.84,
#     "total_tokens_saved": 1500000,
#     "extractions_performed": 450,
#     "spreading_activations": 230,
#     "avg_latency_ms": 187,
#     "p95_latency_ms": 245
# }
```

### Compression Modes

| Mode | Accuracy | Reduction | Use Case |
|------|----------|-----------|----------|
| EXTRACT | 97%+ | 80-85% | Maximum accuracy (default) |
| BALANCED | 95%+ | 85-90% | General use |
| AGGRESSIVE | 92%+ | 90-93% | Cost optimization |

## Tiered Memory

```python
from agentmemory import TierPolicy

# Configure memory tier policy
client.set_tier_policy(TierPolicy.CONSERVATIVE)  # 30-day hot storage
# Options: AGGRESSIVE (1 day), BALANCED (7 days), CONSERVATIVE (30 days)
```

## Enhanced Search

```python
from agentmemory import SearchMode

# Use Spreading Activation for complex queries
results = client.search_enhanced(
    "complex multi-hop query that requires reasoning across memories",
    mode=SearchMode.SPREADING  # Uses graph-based retrieval
)

# Options:
# - SPREADING: Graph propagation (best for multi-hop)
# - VECTOR: Standard similarity (fast)
# - HYBRID: Combine both

# Or use standard search
results = client.search("simple semantic query")
```

### Why Spreading Activation?

Standard vector search only finds memories with similar embeddings. Spreading Activation:
1. Starts with vector similarity to get initial nodes
2. Propagates activation through the knowledge graph
3. Finds related memories even without surface similarity
4. **+23% improvement** on multi-hop reasoning tasks

## Configurable LLM Providers

```python
# Configure which LLM powers compression
# Fast model for simple extraction, Verify model for complex verification
client.configure_llm(
    extraction_provider="openai",       # Fast: GPT-4o-mini, Groq
    extraction_model="gpt-4o-mini",
    verification_provider="anthropic", # Verify: Claude
    verification_model="claude-3-5-sonnet"
)

# Or use environment variables:
# AGENT_MEMORY_EXTRACTION_PROVIDER=openai
# AGENT_MEMORY_EXTRACTION_MODEL=gpt-4o-mini
# AGENT_MEMORY_VERIFICATION_PROVIDER=anthropic
# AGENT_MEMORY_VERIFICATION_MODEL=claude-3-5-sonnet
```

---

## Features

### Conversation Memory
- Create sessions for different agents/users
- Store message history with roles (user/assistant/system/tool)
- Retrieve full context for continuing conversations

### Knowledge Graph
- Create entities with types and properties
- Connect entities with typed relationships
- Query relationships to understand connections

### Semantic Search
- Natural language search over all memories
- Vector-based similarity scoring
- Configurable threshold and limit

## API Reference

### Initialization

```python
client = AgentMemory(
    base_url="http://localhost:8080",  # Default
    api_key="your-key",                # Or use AGENT_MEMORY_API_KEY env
    timeout=30                         # Request timeout in seconds
)
```

Compression can also be configured at init:

```python
from agentmemory import AgentMemory, CompressionMode, TierPolicy

client = AgentMemory(
    base_url="http://localhost:8080",
    api_key="your-key",
    compression_mode=CompressionMode.EXTRACT,   # 97%+ accuracy
    tier_policy=TierPolicy.BALANCED,            # 7-day hot
)
```

### Sessions

```python
# Create session
session = client.create_session(
    agent_id="support-bot",
    metadata={"customer_id": "CUST-123"}  # Optional metadata
)

# Add messages
client.add_message(session["id"], "user", "Hello!")
client.add_message(session["id"], "assistant", "Hi! How can I help?")

# Get history
messages = client.get_messages(session["id"], limit=50)
```

### Entities

```python
# Create entity
entity = client.create_entity(
    name="auth-service",
    type="Service",
    properties={"port": 8080, "language": "python"}
)

# Get entity
entity = client.get_entity(entity["id"])

# Get relationships
relations = client.get_entity_relations(entity["id"])
```

### Relations

```python
# Create relationship (types are limited for security)
client.create_relation(
    from_id="entity-a-id",
    to_id="entity-b-id",
    rel_type="KNOWS"  # Or HAS, RELATED_TO, USES, etc.
)
```

### Search

```python
# Semantic search
results = client.search(
    query="machine learning transformers",
    limit=10,
    threshold=0.5
)

# Results contain score and entity info
for r in results:
    print(f"Score: {r['score']:.2f}")
```

## Error Handling

```python
from agentmemory import (
    AgentMemory,
    AuthenticationError,
    NotFoundError,
    ValidationError,
    RateLimitError
)

try:
    session = client.create_session(agent_id="my-agent")
except AuthenticationError:
    print("Invalid API key")
except ValidationError as e:
    print(f"Invalid input: {e}")
except RateLimitError:
    print("Too many requests - wait a bit")
```

## Environment Variables

- `AGENT_MEMORY_API_KEY` - Default API key for all clients
- `AGENT_MEMORY_BASE_URL` - Default base URL
- `AGENT_MEMORY_COMPRESSION_MODE` - Default compression mode
- `AGENT_MEMORY_TIER_POLICY` - Default tier policy
- `AGENT_MEMORY_EXTRACTION_PROVIDER` - LLM provider for extraction
- `AGENT_MEMORY_VERIFICATION_PROVIDER` - LLM provider for verification

```bash
export AGENT_MEMORY_API_KEY="your-key"
export AGENT_MEMORY_BASE_URL="https://api.agentmemory.io"
export AGENT_MEMORY_COMPRESSION_MODE=extract
export AGENT_MEMORY_TIER_POLICY=balanced
```

```python
# Now you can omit credentials
client = AgentMemory()  # Uses env vars automatically
```

## Full Example

```python
from agentmemory import AgentMemory, CompressionMode

# Initialize with compression enabled
client = AgentMemory(
    base_url="https://api.agentmemory.io",
    api_key="am_xxxxxxxxxxxxx",
    compression_mode=CompressionMode.EXTRACT
)

# 1. Create conversation session
session = client.create_session(
    agent_id="customer-support-bot",
    metadata={"customer_id": "CUST-456", "tier": "premium"}
)

# 2. Store conversation
client.add_message(session["id"], "user", "I can't access my dashboard")
client.add_message(session["id"], "assistant", "I'll help you troubleshoot")
client.add_message(session["id"], "user", "It says permission denied")

# 3. Create knowledge graph entry for this issue
issue = client.create_entity(
    name="dashboard-permission-issue",
    type="Issue",
    properties={"customer": "CUST-466", "status": "resolved"}
)

# 4. Use Spreading Activation for complex queries
similar = client.search_enhanced(
    "permission denied dashboards for premium customers",
    mode="spreading"
)
print(f"Found {len(similar)} similar issues via graph search")

# 5. Check compression stats
stats = client.get_compression_stats()
print(f"Token reduction: {stats['token_reduction']*100}%")
print(f"Accuracy retention: {stats['accuracy_retention']*100}%")
```

## Documentation

- [API Documentation](./docs/openapi.yaml)
- [Quick Start Guide](../../QUICKSTART.md)
- [Use Cases](../../docs/use-cases.md)

## License

MIT