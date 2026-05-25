# OpenClaw Integration

Agent Memory can be integrated with [OpenClaw](https://openclaw.ai) to provide persistent memory for your personal AI assistant.

## Quick Start

### 1. Start Agent Memory Server

```bash
# Start the HTTP API server
./agent-memory

# Or run in MCP stdio mode
SERVER_MODE=mcp-stdio ./agent-memory
```

### 2. Configure OpenClaw

Add to your OpenClaw configuration:

```json
{
  "skills": {
    "agent-memory": {
      "enabled": true,
      "config": {
        "apiUrl": "http://localhost:8080",
        "apiKey": "your-api-key"
      }
    }
  }
}
```

### 3. MCP Server Mode

For OpenClaw's MCP integration, run agent-memory in MCP mode:

```bash
SERVER_MODE=mcp-stdio ./agent-memory
```

This starts a JSON-RPC server over stdio that implements the MCP protocol.

## Available Tools

When integrated with OpenClaw, you can use these tools:

### Memory Tools

- `create_session` - Create a new agent session
- `add_message` - Add a message to a session
- `get_messages` - Retrieve session messages
- `semantic_search` - Search for similar content

### Knowledge Graph Tools

- `create_entity` - Create an entity (Person, Concept, etc.)
- `get_entity` - Get entity with relationships
- `create_relation` - Connect two entities

### System Tools

- `health_check` - Check service status

## Example Usage

Once configured, you can say things like:

- "Remember that I prefer dark mode"
- "What did we discuss about machine learning last time?"
- "Create a knowledge graph entry for my project"
- "Search for similar issues from past conversations"

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `SERVER_MODE` | Run mode: `http` or `mcp-stdio` | `http` |
| `NEO4J_URI` | Neo4j connection URI | `bolt://localhost:7687` |
| `NEO4J_PASSWORD` | Neo4j password | - |
| `QDRANT_URL` | Qdrant URL | `http://localhost:6333` |
| `HTTP_PORT` | HTTP server port | `:8080` |
