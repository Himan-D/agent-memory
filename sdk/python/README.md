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

```bash
export AGENT_MEMORY_API_KEY="your-key"
export AGENT_MEMORY_BASE_URL="https://api.agentmemory.io"
```

```python
# Now you can omit credentials
client = AgentMemory()  # Uses env vars automatically
```

## Full Example

```python
from agentmemory import AgentMemory

# Initialize
client = AgentMemory(
    base_url="https://api.agentmemory.io",
    api_key="am_xxxxxxxxxxxxx"
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

# 4. Search past similar issues
similar = client.search("permission denied dashboard", limit=5)
print(f"Found {len(similar)} similar issues")

# 5. Get context for next response
context = client.get_context(session["id"])
# Use context in your LLM prompt
```

## Documentation

- [API Documentation](./docs/openapi.yaml)
- [Quick Start Guide](../../QUICKSTART.md)
- [Use Cases](../../docs/use-cases.md)

## License

MIT