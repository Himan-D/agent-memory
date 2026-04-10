# MCP Server Integration

Agent Memory includes a Model Context Protocol (MCP) server implementation that allows AI assistants to use persistent memory through the MCP standard.

## What is MCP?

The [Model Context Protocol](https://modelcontextprotocol.io) is a standard for connecting AI models to external tools and services. It provides a standardized way for AI assistants to:

- List available tools
- Call tools with arguments
- Receive structured responses

## Running the MCP Server

### Stdio Mode (Recommended)

```bash
# Set mode and start
SERVER_MODE=mcp-stdio ./agent-memory

# Or with environment
export SERVER_MODE=mcp-stdio
./agent-memory
```

This runs an MCP server over stdin/stdout using JSON-RPC 2.0.

### Connect to Claude Desktop

Add to your Claude Desktop config:

```json
{
  "mcpServers": {
    "agent-memory": {
      "command": "/path/to/agent-memory",
      "env": {
        "SERVER_MODE": "mcp-stdio",
        "NEO4J_URI": "bolt://localhost:7687",
        "NEO4J_PASSWORD": "your-password",
        "QDRANT_URL": "http://localhost:6333"
      }
    }
  }
}
```

## Available Tools

### Session Management

#### `create_session`
Create a new agent session for storing conversation context.

```json
{
  "name": "create_session",
  "arguments": {
    "agent_id": "my-assistant",
    "metadata": {"user": "john"}
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

### Knowledge Graph

#### `create_entity`
Create a knowledge graph entity.

```json
{
  "name": "create_entity",
  "arguments": {
    "name": "Transformer",
    "entity_type": "Architecture",
    "properties": {"year": 2017}
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
Create a relationship between entities.

```json
{
  "name": "create_relation",
  "arguments": {
    "from_id": "entity-1",
    "to_id": "entity-2",
    "relation_type": "USES"
  }
}
```

### Semantic Search

#### `semantic_search`
Search for semantically similar content.

```json
{
  "name": "semantic_search",
  "arguments": {
    "query": "deep learning transformers",
    "limit": 10,
    "threshold": 0.5
  }
}
```

### System

#### `health_check`
Check if the memory service is healthy.

```json
{
  "name": "health_check",
  "arguments": {}
}
```

## Protocol Details

- **Protocol Version**: 2024-11-05
- **Transport**: Stdio (JSON-RPC 2.0)
- **Encoding**: UTF-8 JSON

## Example Session

```bash
$ SERVER_MODE=mcp-stdio ./agent-memory
{"jsonrpc":"2.0","method":"initialize","params":{},"id":1}
{"jsonrpc":"2.0","result":{"protocolVersion":"2024-11-05","capabilities":{"tools":{}},"serverInfo":{"name":"agent-memory","version":"0.1.0"}},"id":1}
{"jsonrpc":"2.0","method":"tools/list","params":{},"id":2}
{"jsonrpc":"2.0","result":{"tools":[{"name":"create_session",...}]},"id":2}
{"jsonrpc":"2.0","method":"tools/call","params":{"name":"health_check","arguments":{}},"id":3}
{"jsonrpc":"2.0","result":{"content":[{"type":"text","text":"{\"Neo4j\":\"healthy\",\"Qdrant\":\"healthy\"}"}]},"id":3}
```
