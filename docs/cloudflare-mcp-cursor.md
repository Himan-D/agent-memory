# Cloudflare MCP For Cursor

This repository includes a Cursor project MCP config at `.cursor/mcp.json` with Cloudflare's managed remote MCP servers.

## Included Servers

- `cloudflare-api`: full Cloudflare API access through Code Mode.
- `cloudflare-docs`: current Cloudflare documentation.
- `cloudflare-workers-bindings`: Workers bindings and platform primitives.
- `cloudflare-workers-builds`: Workers Builds insight and management.
- `cloudflare-observability`: logs, analytics, and debugging.
- `cloudflare-radar`: Internet traffic insights and URL utilities.
- `cloudflare-containers`: sandbox development environments.
- `cloudflare-browser`: fetch pages, markdown conversion, and screenshots.
- `cloudflare-logpush`: Logpush job health summaries.
- `cloudflare-ai-gateway`: AI Gateway logs, prompt, and response inspection.
- `cloudflare-ai-search`: AutoRAG / AI Search document operations.
- `cloudflare-audit-logs`: audit log queries and reports.
- `cloudflare-dns-analytics`: DNS performance debugging.
- `cloudflare-dex`: Digital Experience Monitoring insights.
- `cloudflare-casb`: Cloudflare One CASB misconfiguration checks.
- `cloudflare-graphql`: Cloudflare GraphQL analytics.
- `cloudflare-agents-docs`: Cloudflare Agents SDK documentation.

## Cursor Setup

1. Open this repository in Cursor.
2. Restart Cursor so it reloads `.cursor/mcp.json`.
3. Open Cursor settings and confirm the Cloudflare MCP servers are connected.
4. The first time Cursor calls a Cloudflare account tool, authorize through Cloudflare OAuth and select the required permissions.

Cloudflare also recommends installing the Cloudflare Cursor plugin:

```text
/add-plugin cloudflare
```

The plugin adds Cloudflare skills and can register Cloudflare MCP servers globally. The project config remains useful because it makes this repository self-documenting and consistent for contributors.

## Security Rules

- Do not commit Cloudflare API tokens, OAuth tokens, account IDs, zone IDs, or `.env` files.
- Prefer OAuth authorization from Cursor for local interactive work.
- For CI automation, create narrowly scoped Cloudflare API tokens in GitHub repository secrets.
- Keep destructive Cloudflare operations behind explicit human review unless the task is narrowly scoped and reversible.

## Usage Guidance

- Use `cloudflare-docs` before relying on remembered product details.
- Use `cloudflare-api` for broad account operations such as DNS, R2, WAF, Workers, Zero Trust, and Pages.
- Use focused servers when they match the task; for example, `cloudflare-observability` for logs and `cloudflare-workers-bindings` for Workers binding design.
- Use Wrangler for local Worker development and deploy commands from the terminal.

## References

- Cloudflare Cursor setup: `https://developers.cloudflare.com/agent-setup/cursor/`
- Cloudflare MCP server catalog: `https://developers.cloudflare.com/agents/model-context-protocol/cloudflare/servers-for-cloudflare/`

