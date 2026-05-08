# MCP Server Integration

Hystersis includes a Model Context Protocol (MCP) server so AI assistants can use persistent memory through the MCP standard.

## What is MCP?

The [Model Context Protocol](https://modelcontextprotocol.io) is a standard for connecting AI models to external tools and services. It provides a standardized way for AI assistants to list available tools, call them with arguments, and receive structured responses.

## Running the MCP Server

### Stdio Mode (Recommended)

```bash
SERVER_MODE=mcp-stdio ./hystersis
```

This runs an MCP server over stdin/stdout using JSON-RPC 2.0.

### Connect to Claude Desktop

Add to `~/Library/Application Support/Claude/claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "hystersis": {
      "command": "/path/to/hystersis",
      "env": {
        "SERVER_MODE": "mcp-stdio",
        "NEO4J_URI": "bolt://localhost:7687",
        "NEO4J_PASSWORD": "your-password",
        "QDRANT_URL": "http://localhost:6333",
        "REDIS_URL": "redis://localhost:6379",
        "LLM_API_KEY": "sk-..."
      }
    }
  }
}
```

### Connect to Cursor

Add to Cursor settings → MCP Servers:

```json
{
  "hystersis": {
    "command": "/path/to/hystersis",
    "env": {
      "SERVER_MODE": "mcp-stdio",
      "NEO4J_URI": "bolt://localhost:7687",
      "NEO4J_PASSWORD": "your-password",
      "QDRANT_URL": "http://localhost:6333"
    }
  }
}
```

---

## Available Tools

### Session Management

#### `create_session`
Create a new agent session for storing conversation context.

```json
{
  "name": "create_session",
  "arguments": {
    "agent_id": "my-assistant",
    "metadata": {"user": "john", "channel": "web"}
  }
}
```

#### `add_message`
Add a message to an existing session.

```json
{
  "name": "add_message",
  "arguments": {
    "session_id": "session-123",
    "role": "user",
    "content": "I love machine learning!"
  }
}
```

**Roles:** `user`, `assistant`, `system`, `tool`

#### `get_messages`
Retrieve messages from a session.

```json
{
  "name": "get_messages",
  "arguments": {
    "session_id": "session-123",
    "limit": 50
  }
}
```

---

### Memory Operations

#### `add_memory`
Store a new memory with automatic ProMem compression.

```json
{
  "name": "add_memory",
  "arguments": {
    "content": "User prefers dark mode and uses Python daily",
    "user_id": "user-123",
    "category": "preferences",
    "importance": "high",
    "tags": ["ui", "coding"]
  }
}
```

#### `get_memories`
List memories for a user with optional filtering.

```json
{
  "name": "get_memories",
  "arguments": {
    "user_id": "user-123",
    "category": "preferences",
    "limit": 20
  }
}
```

#### `search_memories`
Semantic search across stored memories.

```json
{
  "name": "search_memories",
  "arguments": {
    "query": "what are this user's coding preferences?",
    "user_id": "user-123",
    "limit": 10,
    "threshold": 0.5
  }
}
```

#### `add_feedback`
Add feedback to improve memory quality and future search relevance.

```json
{
  "name": "add_feedback",
  "arguments": {
    "memory_id": "mem-abc123",
    "type": "positive",
    "comment": "This is accurate"
  }
}
```

**Types:** `positive`, `negative`, `very_negative`

---

### Knowledge Graph

#### `create_entity`
Create a knowledge graph entity.

```json
{
  "name": "create_entity",
  "arguments": {
    "name": "Transformer",
    "entity_type": "Architecture",
    "properties": {"year": 2017, "paper": "Attention Is All You Need"}
  }
}
```

#### `get_entity`
Retrieve an entity and its relationships.

```json
{
  "name": "get_entity",
  "arguments": {
    "entity_id": "entity-456"
  }
}
```

#### `create_relation`
Create a typed relationship between entities.

```json
{
  "name": "create_relation",
  "arguments": {
    "from_id": "entity-1",
    "to_id": "entity-2",
    "relation_type": "USES",
    "weight": 0.95
  }
}
```

#### `get_context`
Get formatted LLM context from a session.

```json
{
  "name": "get_context",
  "arguments": {
    "session_id": "session-123",
    "limit": 10
  }
}
```

---

### Compression Engine

#### `compress_memory`
Manually trigger ProMem compression on a memory.

```json
{
  "name": "compress_memory",
  "arguments": {
    "memory_id": "mem-abc123",
    "mode": "extract"
  }
}
```

**Modes:** `extract` (default), `balanced`, `aggressive`

#### `get_compression_stats`
Get compression performance metrics.

```json
{
  "name": "get_compression_stats",
  "arguments": {}
}
```

Returns accuracy retention, token reduction percentage, total tokens saved, average latency.

#### `set_compression_mode`
Change the compression mode for new memories.

```json
{
  "name": "set_compression_mode",
  "arguments": {
    "mode": "extract"
  }
}
```

---

### Skills

#### `execute_skill`
Execute a named skill with context.

```json
{
  "name": "execute_skill",
  "arguments": {
    "skill_id": "sql-expert",
    "context": {
      "query": "SELECT * FROM users",
      "database": "postgres"
    }
  }
}
```

#### `search_skills`
Search for available skills by trigger or domain.

```json
{
  "name": "search_skills",
  "arguments": {
    "trigger": "optimize query",
    "domain": "database",
    "limit": 5
  }
}
```

---

### System

#### `health_check`
Check if the memory service is healthy.

```json
{
  "name": "health_check",
  "arguments": {}
}
```

Returns status of Neo4j, Qdrant, Redis, and compression engine.

---

## Protocol Details

| Property | Value |
|----------|-------|
| Protocol Version | `2024-11-05` |
| Transport | Stdio (JSON-RPC 2.0) |
| Encoding | UTF-8 JSON |

---

## Example Session

```
$ SERVER_MODE=mcp-stdio ./hystersis

→ {"jsonrpc":"2.0","method":"initialize","params":{},"id":1}
← {"jsonrpc":"2.0","result":{"protocolVersion":"2024-11-05","capabilities":{"tools":{}},"serverInfo":{"name":"hystersis","version":"1.0.0"}},"id":1}

→ {"jsonrpc":"2.0","method":"tools/list","params":{},"id":2}
← {"jsonrpc":"2.0","result":{"tools":[{"name":"create_session",...},{"name":"add_memory",...},...]},"id":2}

→ {"jsonrpc":"2.0","method":"tools/call","params":{"name":"health_check","arguments":{}},"id":3}
← {"jsonrpc":"2.0","result":{"content":[{"type":"text","text":"{\"neo4j\":\"healthy\",\"qdrant\":\"healthy\",\"redis\":\"healthy\",\"compression\":\"active\"}"}]},"id":3}

→ {"jsonrpc":"2.0","method":"tools/call","params":{"name":"get_compression_stats","arguments":{}},"id":4}
← {"jsonrpc":"2.0","result":{"content":[{"type":"text","text":"{\"accuracy_retention\":0.973,\"token_reduction\":0.84}"}]},"id":4}
```
