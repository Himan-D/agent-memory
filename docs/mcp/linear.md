MCP Linear Executor

Overview

The linear MCP endpoint executes a linear sequence of tool calls (no branching) against the local MCP toolset. Useful for scripting agent behavior, bootstrapping memories, and testing tool chains reproducibly.

Endpoint

POST /mcp/linear
Content-Type: application/json

Request body:
{
  "calls": [
    {"tool": "addMemory", "params": {"content": "User likes cats", "userId": "user-1"}},
    {"tool": "recall", "params": {"query": "likes cats", "limit": 3}}
  ]
}

Response:
{
  "results": [
    {"index": 0, "tool": "addMemory", "success": true, "result": {"content": [...] }},
    {"index": 1, "tool": "recall", "success": true, "result": {...}}
  ]
}

Behavior

- Calls execute sequentially using the ToolHandler. Each call is independent; errors are recorded per-call but do not abort the whole sequence.
- Context cancellation will stop further execution and return partial results.
- Tenant resolution respects X-Tenant-ID header or Authorization: Bearer <token> header.

Security

- The endpoint is intended for local use. Use reverse proxies and auth in production.
- Validate payload sizes before exposing to untrusted networks.

CLI (agent harness)

Agent REPL provides an /mcp command to manage the local MCP server and invoke sequences:

- /mcp start [port]  — start local MCP server (default :8081 or MCP_PORT env)
- /mcp stop           — stop MCP server
- /mcp status         — show running status
- /mcp invoke <json|file> — invoke linear sequence (file path or inline JSON)

Examples

# Example JSON file (docs/examples/mcp_example.json)
{
  "calls": [
    {"tool": "addMemory", "params": {"content": "User likes hiking", "userId": "user-42"}},
    {"tool": "recall", "params": {"query": "likes hiking", "limit": 5}}
  ]
}

# Invoke from agent REPL
/mcp start :8082
/mcp invoke docs/examples/mcp_example.json

# Invoke via HTTP client
curl -X POST http://localhost:8082/mcp/linear -H "Content-Type: application/json" --data @docs/examples/mcp_example.json

Notes

- For programmatic use, prefer the internal/mcp client: mcp.NewClient("http://localhost:8081").ExecuteLinear(ctx, calls)
- See internal/mcp for implementation details and unit tests.
