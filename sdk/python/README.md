# Agent Memory Python SDK

A Python client for the Agent Memory System API.

## Installation

```bash
pip install agentmemory
```

## Quick Start

```python
from agentmemory import AgentMemory

# Initialize client
memory = AgentMemory(
    api_key="your-api-key",
    base_url="http://localhost:8080"
)

# Create a session
session = memory.create_session(agent_id="my-agent")
session_id = session["id"]

# Add messages
memory.add_message(session_id, "user", "Hello!")
memory.add_message(session_id, "assistant", "Hi there!")

# Get conversation
messages = memory.get_messages(session_id)

# Create entities
memory.create_entity(name="Machine Learning", entity_type="Concept")
```

## Configuration

The SDK can be configured via environment variables:

```bash
export AGENT_MEMORY_API_KEY=your-api-key
export AGENT_MEMORY_URL=http://localhost:8080
```

## API Reference

### Sessions

- `create_session(agent_id, metadata)` - Create a new session
- `get_messages(session_id, limit)` - Get messages
- `add_message(session_id, role, content)` - Add message

### Entities

- `create_entity(name, type, properties)` - Create entity
- `get_entity(entity_id)` - Get entity
- `get_relations(entity_id, type)` - Get relations

### Relations

- `create_relation(from_id, to_id, type, metadata)` - Create relation

### Search

- `search(query, limit, threshold)` - Semantic search

### Graph

- `graph_query(cypher, params)` - Execute Cypher query
