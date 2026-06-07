# Hystersis Blog Studio (Sanity)

Sanity CMS for the Hystersis landing blog at [hystersis.com/blog](https://hystersis.com/blog).

**Project:** `yhvdqwt4` · **Dataset:** `production`

## Sanity MCP (Cursor)

The repo includes the [Sanity MCP server](https://www.sanity.io/docs/ai/mcp-server) in `.cursor/mcp.json`. After merging:

1. Open **Cursor → MCP Settings** (or run `View: Open MCP Settings`)
2. Authenticate **Sanity** via OAuth when prompted
3. Ask the agent to manage blog content, run GROQ queries, or deploy schemas

**Token auth (optional):** Copy `.cursor/mcp.json.example` and set `SANITY_API_TOKEN` to a token with Editor + deploy permissions from [sanity.io/manage/project/yhvdqwt4/api](https://www.sanity.io/manage/project/yhvdqwt4/api).

**CLI configure:**

```bash
npx sanity@latest mcp configure
```

## Local Studio

```bash
cd landing/sanity
npm ci
SANITY_STUDIO_PROJECT_ID=yhvdqwt4 npm run dev
```

Open http://localhost:3333

## Deploy Schema

```bash
export SANITY_AUTH_TOKEN=<token-with-deploy-permissions>
export SANITY_STUDIO_PROJECT_ID=yhvdqwt4
npx sanity schema deploy
```

## Agent Skills

Sanity best-practice skills are installed under `.agents/skills/` via:

```bash
npx skills add sanity-io/agent-toolkit
```
