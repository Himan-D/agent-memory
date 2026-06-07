# Gemini Code Assist Review Guide

Review pull requests for correctness, security, deployment safety, and regression risk.

## Repository Rules

- Root `wrangler.jsonc` must keep Worker name `agent-memory`.
- `dashboard/wrangler.jsonc` must keep Worker name `hystersis-app`.
- Do not approve production URL changes that reintroduce `hystersis.ai`; production uses `hystersis.com`.
- Do not approve secrets, credentials, private keys, `.env` files, generated builds, binaries, `node_modules`, `landing/dist`, or `dashboard/.open-next`.
- Treat Cloudflare deploy changes as high risk when they touch Worker names, custom domains, route patterns, or shared deploy scripts.
- Landing, dashboard, docs, API, and SDK changes should stay path-scoped unless the PR clearly explains cross-surface coupling.
- API handlers should avoid exposing raw errors; use the repo's safe error handling conventions.

## Review Output

- Lead with blocking issues and security/deployment risks.
- Include file paths and line references when possible.
- Call out missing tests or missing deployment verification.
- If no issues are found, say that clearly and mention remaining risk.
