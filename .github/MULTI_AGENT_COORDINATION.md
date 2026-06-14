# Multi-Agent Coordination (Cursor + Codex + Copilot)

Multiple agents may work on this repo in parallel. Follow these rules to avoid conflicts.

## Active ownership (check before editing)

| Area | Owner / PR | Status | Do NOT |
|------|------------|--------|--------|
| Landing worker name + apex routing | **PR #95** `cursor/fix-landing-worker-name-6161` | OPEN | Rename root `wrangler.jsonc` to `hystersis-app` |
| Landing SPA routing | Merged #88 | Done | Revert `workers/site.js` SPA fallback |
| Domain `.ai` → `.com` migration | Merged #94 | Done | Re-introduce `*.hystersis.ai` in prod URLs |
| Cloudflare MCP config | Merged #92 | Done | Duplicate `.cursor/mcp.json` changes |
| SDK/API smoke tests | `cursor/deployment-sdk-api-verify-6161` | In progress | Touch `wrangler.jsonc` or deploy scripts |
| Sanity CMS | PR #84 draft | Draft | — |

## Critical deploy files — coordinate first

These files cause production outages when two agents edit concurrently:

```
wrangler.jsonc              → landing worker (name MUST be agent-memory)
dashboard/wrangler.jsonc    → dashboard worker (name MUST be agent-memorydash)
scripts/build-cloudflare.sh
scripts/deploy-cloudflare.sh
scripts/deploy-dashboard-builds.sh
scripts/verify-domains.sh
workers/site.js
landing/DEPLOY_VERSION
```

**Rule:** Only one agent per deploy-fix PR. If you need these files, check open PRs first:

```bash
gh pr list --state open
git fetch origin && git diff origin/master...origin/<branch> -- wrangler.jsonc scripts/
```

## Worker naming (DO NOT CHANGE)

| Worker `name` | Config | Serves |
|---------------|--------|--------|
| `agent-memory` | root `wrangler.jsonc` | hystersis.com, www, blogs |
| `agent-memorydash` | `dashboard/wrangler.jsonc` | app.hystersis.com |

Using the same name for both **overwrites landing with dashboard** (apex redirects to `/auth/signin`).

## Branch conventions

| Agent | Branch prefix | Notes |
|-------|---------------|-------|
| Cursor Cloud Agent | `cursor/<task>-6161` | Auto-merge when CI passes |
| Codex / Copilot | `cursor/<task>-6161` or bot branches | Same rules |
| OpenAI repo bot | `cursor/openai-repo-bot-6161` | Scheduled; avoid same files |
| Dependabot | `dependabot/*` | Deps only — do not bundle with deploy fixes |

## Before you push

1. `git fetch origin master`
2. `git rebase origin/master` (or merge if rebase fails)
3. Check open PRs for overlapping paths
4. Run scoped build (don't need full monorepo if touching one surface)
5. One PR = one concern (deploy fix ≠ SDK tests ≠ docs migration)

## After a deploy fix merges

1. Wait for **Deploy Cloudflare (All)** to finish on `master`
2. Confirm the workflow preflight passed Cloudflare Worker edit permissions
3. Run `bash scripts/verify-domains.sh`
4. Confirm apex: `curl -sI https://hystersis.com/` → **200**, not 307 to `/auth/signin`
5. Append learnings to `.jules/bolt.md` if you hit a new pitfall

## Conflict resolution priority

When `sync-with-master.yml` auto-resolves deploy files, **prefer the branch that fixes production**:

1. Worker name correctness (`agent-memory` vs `agent-memorydash`)
2. SPA routing in `workers/site.js`
3. Verify scripts
4. Comment-only / DEPLOY_VERSION bumps

If unsure, add label `do-not-merge` and leave a PR comment tagging the other agent's work.
