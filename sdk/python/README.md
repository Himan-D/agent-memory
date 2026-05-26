# Hystersis Python SDK

<p align="center">
  <a href="https://pypi.org/project/hystersis/">
    <img src="https://img.shields.io/pypi/v/hystersis" alt="PyPI">
  </a>
  <a href="https://pypi.org/project/hystersis/">
    <img src="https://img.shields.io/pypi/pyversions/hystersis" alt="PyPI - Python Version">
  </a>
</p>

Persistent memory infrastructure for AI agents.
Give your agents memory that adapts and compounds over time.

## Why Use Hystersis SDK?

- **Simple** - One-line installation, intuitive API
- **Powerful** - Combine conversation history with knowledge graphs
- **Production-ready** - Type hints, error handling, timeouts, retries
- **Async + Sync** - Full async support with `AsyncHystersis`, sync wrapper with `Hystersis`
- **ProMem Extraction** - 97%+ accuracy memory compression (PROPRIETARY)
- **Spreading Activation** - +23% better multi-hop reasoning (PROPRIETARY)

## Installation

```bash
pip install hystersis
```

With integrations:

```bash
pip install hystersis[integrations]
```

## Quick Start

### Sync Client

```python
from hystersis import Hystersis

client = Hystersis(
    base_url="https://api.hystersis.ai",
    api_key="your-api-key"
)

# Check health
print(client.health())

# Create a session
session = client.create_session(agent_id="my-assistant")

# Add messages
client.add_message(session["id"], "user", "I love machine learning!")
client.add_message(session["id"], "assistant", "That's great! What type?")

# Search semantically
results = client.search("deep learning")

client.close()
```

### Async Client

```python
import asyncio
from hystersis import AsyncHystersis

async def main():
    async with AsyncHystersis(
        base_url="https://api.hystersis.ai",
        api_key="your-api-key"
    ) as client:
        session = await client.create_session(agent_id="my-assistant")
        memory = await client.create_memory(
            content="User likes ML",
            user_id="user-123"
        )
        results = await client.memories_search("deep learning")

asyncio.run(main())
```

---

## Compression Engine

Hystersis includes a **proprietary compression engine**:

| Metric | Hystersis | Mem0 |
|--------|-----------|------|
| Accuracy Retention | **97%+** | 91% |
| Token Reduction | **80-85%** | 80% |
| Multi-hop Reasoning | **+23%** | baseline |

### Compression Control

```python
from hystersis import Hystersis, CompressionMode

client = Hystersis(api_key="your-key")

# Set compression mode
client.compression_set_mode(CompressionMode.EXTRACT)
# Options: EXTRACT, BALANCED, AGGRESSIVE

# Get compression statistics
stats = client.compression_get_stats()
print(stats)
```

### Compression Modes

| Mode | Accuracy | Reduction | Use Case |
|------|----------|-----------|----------|
| EXTRACT | 97%+ | 80-85% | Maximum accuracy (default) |
| BALANCED | 95%+ | 85-90% | General use |
| AGGRESSIVE | 92%+ | 90-93% | Cost optimization |

## Tiered Memory

```python
from hystersis import TierPolicy

client.tier_set_policy(TierPolicy.CONSERVATIVE)
# Options: AGGRESSIVE (1 day), BALANCED (7 days), CONSERVATIVE (30 days)
```

## Enhanced Search

```python
from hystersis import SearchMode

# Spreading Activation for complex queries
results = client.search_enhanced(
    "complex multi-hop query",
    mode=SearchMode.SPREADING
)

# Options: SPREADING (graph), VECTOR (fast), HYBRID (both)
```

---

## API Reference

### Initialization

```python
from hystersis import Hystersis, AsyncHystersis

# Sync
client = Hystersis(
    base_url="https://api.hystersis.ai",
    api_key="your-key",
)

# Async
async with AsyncHystersis(base_url="...", api_key="...") as client:
    ...
```

Or use environment variables:

```bash
export HYSTERSIS_API_KEY="your-key"
```

```python
client = Hystersis()  # Uses env var automatically
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
session = client.create_session(agent_id="support-bot", metadata={"user": "123"})
client.add_message(session["id"], "user", "Hello!")
messages = client.get_messages(session["id"])
```

### Memories

```python
memory = client.create_memory(content="User likes Python", user_id="user-123")
memories = client.memories_list(user_id="user-123", limit=50)
results = client.memories_search("python programming", limit=10)
```

### Entities & Knowledge Graph

```python
entity = client.entities_create(name="Python", entity_type="Language")
client.relations_create(from_id=entity_a["id"], to_id=entity_b["id"], relation_type="RELATED_TO")
relations = client.entities_get_relations(entity["id"])
```

### Skills

```python
skill = client.skills_create(name="debugger", trigger="code error", action="analyze stack trace")
suggestions = client.skills_suggest(trigger="code error", context="python traceback")
```

### Error Handling

```python
from hystersis import (
    HystersisError,
    AuthenticationError,
    NotFoundError,
    ValidationError,
    RateLimitError,
)

try:
    client.create_session(agent_id="my-agent")
except AuthenticationError:
    print("Invalid API key")
except RateLimitError:
    print("Too many requests")
```

## Integrations

```bash
pip install hystersis[integrations]
```

```python
# LangChain
from hystersis.integrations.langchain import HystersisMemory

# LlamaIndex
from hystersis.integrations.llamaindex import HystersisReader

# CrewAI
from hystersis.integrations.crewai import CrewMemory

# LangGraph
from hystersis.integrations.langgraph import HystersisChecker

# AutoGen
from hystersis.integrations.autogen import AutoGenMemory
```

## Full Example

```python
from hystersis import Hystersis, CompressionMode

with Hystersis(base_url="https://api.hystersis.ai", api_key="your-key") as client:
    # Create session
    session = client.create_session(agent_id="support-bot")

    # Store conversation
    client.add_message(session["id"], "user", "I can't access my dashboard")
    client.add_message(session["id"], "assistant", "I'll help you troubleshoot")

    # Create knowledge graph entity
    issue = client.entities_create(
        name="dashboard-access-issue",
        entity_type="Issue",
        properties={"status": "open"}
    )

    # Semantic search
    results = client.memories_search("dashboard access problems")

    # Enhanced search with spreading activation
    similar = client.search_enhanced(
        "permission denied dashboards",
        mode="spreading"
    )

    # Compression stats
    stats = client.compression_get_stats()
    print(f"Token reduction: {stats['token_reduction']*100}%")
```

## Environment Variables

| Variable | Description |
|----------|-------------|
| `HYSTERSIS_API_KEY` | Default API key |
| `AGENT_MEMORY_API_KEY` | Alias for API key (backward compat) |

## License

MIT
