# GitHub Copilot Repository Instructions

This repository is Hystersis, a Go-first memory platform for AI agents with dashboard, landing page, SDK, deployment, and documentation surfaces.

## Multi-Agent Coordination

Codex may run in parallel with Cursor Cloud Agents. Read `.github/MULTI_AGENT_COORDINATION.md` before editing:

- `wrangler.jsonc`, `workers/site.js`, `scripts/build-*.sh`, `scripts/deploy-*.sh`, `scripts/verify-domains.sh`

Do **not** rename the root worker to `hystersis-app` — that collides with the dashboard worker and breaks `hystersis.com`.

## Default Workflow

- Work from `master` through PR branches named `cursor/<task>-6161`.
- Keep diffs focused and production-grade.
- Read `AGENTS.md` and `.github/AGENTS.md` before changing code.
- Do not push directly to `master`.
- Do not commit secrets, `.env` files, binaries, `node_modules`, `.open-next`, generated docs output, or local data directories.
- Leave unrelated untracked files untouched.

## Required Verification

For Go changes:

```bash
go build ./...
go test ./cmd/... ./internal/...
```

The integration package under `tests/` requires Neo4j and Qdrant. Use it only when those services are available:

```bash
docker compose up -d neo4j qdrant
go test ./tests
```

For dashboard changes:

```bash
npm --prefix dashboard ci --legacy-peer-deps
npm --prefix dashboard run build
```

For landing/docs changes:

```bash
npm --prefix landing ci
npm --prefix landing run build
bash scripts/build-docs.sh
```

## Engineering Rules

- Prefer existing package boundaries and local helper APIs over new abstractions.
- Use `safeHTTPError()` in API handlers instead of exposing raw errors.
- Use `GetMemoriesByIDs()`, `BatchCreateMemories()`, and `BatchDeleteMemories()` for bulk memory operations.
- Keep proprietary compression code in `internal/compression/**`; do not move or expose it as open source code.
- Add tests proportional to risk and blast radius.
- Update docs or ADRs for architecture-level decisions.

## Product Priorities

1. Reliability of compression, retrieval, and tiered memory.
2. Durable wiki persistence and vector-backed wiki search.
3. Security hardening: audit coverage, RBAC, skill sharing policy enforcement.
4. Observability: metrics correctness, traces, dashboards, alerts.
5. Deployment smoke tests for Docker, Kubernetes, and Terraform paths.

## PR Expectations

- Use Conventional Commits titles such as `fix: correct compression p95 metric`.
- Include a concise summary and verification commands.
- Keep generated artifacts out of the diff.
- Let CI and auto-merge handle merge to `master`.

