# MCP Server Integration

Hystersis includes a Model Context Protocol (MCP) server so Cursor, Claude Desktop, and other MCP clients can use persistent memory.

## One-click setup (recommended)

After installing Hystersis and creating an API key:

```bash
# Point CLI at your API (cloud or local)
hystersis init --url https://api.hystersis.com --api-key <your-key>

# Write Cursor + Claude Desktop config
hystersis mcp setup --target all

# Verify binary, key, and API
hystersis mcp doctor
```

Then **restart Cursor / Claude Desktop**.

### What this installs

| Mode | Binary | Needs local DB? | Use when |
|------|--------|-----------------|----------|
| **proxy** (default) | `hystersis-mcp --stdio` | No | Cloud API or any remote/local HTTP API |
| **local** | `hystersis-server` + `SERVER_MODE=mcp-stdio` | Yes (Neo4j, Qdrant, Redis) | Fully offline stack |

```bash
# Print config without writing
hystersis mcp print

# Cursor only / Claude only
hystersis mcp setup --target cursor
hystersis mcp setup --target claude

# Overwrite existing entry
hystersis mcp setup --force
```

## Manual config

### Cursor (`~/.cursor/mcp.json` or project `.cursor/mcp.json`)

```json
{
  "mcpServers": {
    "hystersis": {
      "command": "hystersis-mcp",
      "args": [
        "--stdio",
        "--memory-api",
        "https://api.hystersis.com",
        "--api-key",
        "your-api-key"
      ],
      "env": {
        "HYSTERSIS_API_URL": "https://api.hystersis.com",
        "HYSTERSIS_API_KEY": "your-api-key"
      }
    }
  }
}
```

### Claude Desktop

`~/Library/Application Support/Claude/claude_desktop_config.json` (macOS):

```json
{
  "mcpServers": {
    "hystersis": {
      "command": "/Users/YOU/.local/bin/hystersis-mcp",
      "args": ["--stdio", "--memory-api", "https://api.hystersis.com", "--api-key", "your-api-key"]
    }
  }
}
```

See also `mcp-config.example.json` in the repo root.

## Install the MCP binary

```bash
# One-line install (builds CLI + server + mcp when Go is available)
curl -fsSL https://hystersis.com/install.sh | bash

# Or from source
go build -o ~/.local/bin/hystersis-mcp ./cmd/mcp-server
```

## Run modes

### Stdio proxy (recommended for IDEs)

Talks MCP on stdin/stdout; calls the REST API with your API key:

```bash
hystersis-mcp --stdio \
  --memory-api https://api.hystersis.com \
  --api-key "$HYSTERSIS_API_KEY"
```

Env alternatives: `HYSTERSIS_API_URL`, `HYSTERSIS_API_KEY`, `MCP_API_KEY`, `MCP_STDIO=1`.

### Local full stack (stdio)

Runs the full memory engine inside the process (no separate HTTP API):

```bash
SERVER_MODE=mcp-stdio hystersis-server
```

Requires Neo4j, Qdrant, Redis, and LLM keys.

### HTTP MCP sidecar

```bash
hystersis-mcp --port 8082 --memory-api http://localhost:8080
```

Endpoints: `/mcp`, `/sse`, `/message`, `/health`.

## Available tools (proxy)

Core tools include:

| Tool | Description |
|------|-------------|
| `add_memory` | Store a memory |
| `recall` / `search` | Semantic search |
| `get_memories` | List recent memories |
| `get_memory` / `update_memory` / `delete_memory` | CRUD |
| `create_session` / `get_context` | Session context |
| `list_entities` / `add_entity` / `create_relation` | Knowledge graph |
| `create_skill` / `list_skills` | Procedural skills |
| `who_am_i` | Auth / key prefix |

Full local stdio mode (`cmd/server` MCP) exposes an expanded tool set including agents, groups, and more.

## Discovery

`GET /.well-known/mcp/server-card.json` on the API describes the preferred transport.

## Troubleshooting

```bash
hystersis mcp doctor
hystersis health
```

- **Binary not found** — re-run the installer or `go build -o ~/.local/bin/hystersis-mcp ./cmd/mcp-server`
- **401 / unauthorized** — set API key via `hystersis init` or `--api-key`
- **Tools not showing** — restart the IDE after `mcp setup`
- **Local mode fails** — ensure Docker services are up (`docker compose up -d`)

## Python SDK smoke (optional)

Against a live API:

```bash
export HYSTERSIS_API_URL=https://api.hystersis.com
export HYSTERSIS_API_KEY=your-key
cd sdk/python && pip install -e ".[dev]" && pytest -m live -q
```
