# GitHub Agent Rules

Instructions for Cloud Agents, Cursor, and other automation working on this repository.

## Branch & PR conventions

| Rule | Value |
|------|-------|
| Base branch | `master` |
| Agent branches | `cursor/<descriptive-name>-6161` |
| PR title | Conventional Commits: `feat:`, `fix:`, `chore:`, `docs:`, `ci:` |
| Merge method | Squash merge (automated) |
| Draft PRs | `false` unless explicitly requested |

## Agent workflow (every task)

1. Create branch: `git checkout -b cursor/<task>-6161`
2. Implement minimal focused diff
3. **Build before commit**: `go build ./...`
4. **Test**: `go test -short ./...` (+ package-specific tests)
5. Commit: `git add -A && git commit -m "feat: description"`
6. Push: `git push -u origin cursor/<task>-6161`
7. Open PR to `master` — auto-labeled `agent` + `automerge`
8. Wait for **CI Success** — auto-merge squash + branch delete

## Auto-merge policy

PRs are **automatically squash-merged** when:

- Branch starts with `cursor/` (Cloud Agents), OR
- Author is `dependabot[bot]`, OR
- Label `automerge` is present

Blocked when:

- Label `do-not-merge` or `merge-conflict` is present
- **CI Success** check fails
- Agent guard detects secrets/binaries in diff

## Merge conflicts

`sync-with-master.yml` runs on every agent PR:

1. Merges `master` into the PR branch when behind
2. **Auto-resolves** conflicts in `.github/workflows/`, `workers/`, `landing/`, `docs/`, `wrangler.jsonc` (prefers PR branch)
3. Labels `merge-conflict` if manual resolution needed — Cloud Agent should fix and push
4. After sync succeeds, **auto-merge** proceeds when CI passes

## Multi-agent coordination (Cursor + Codex)

Codex, Copilot, and Cursor Cloud Agents may run **in parallel**. Before editing deploy or worker files, read **[MULTI_AGENT_COORDINATION.md](MULTI_AGENT_COORDINATION.md)**.

Quick rules:

- Root `wrangler.jsonc` → worker name **`agent-memory`** (never `agent-memorydash`)
- `dashboard/wrangler.jsonc` → worker name **`agent-memorydash`**
- One agent per deploy-fix PR; check `gh pr list --state open` for overlaps
- Rebase on `master` before push: `git fetch origin && git rebase origin/master`

## Jules / agent learnings

Machine learnings from automated agents live in `.jules/bolt.md`. Append new patterns there after fixing performance or merge issues.

## CI gates (required)

| Check | Scope |
|-------|-------|
| CI Success | Aggregator — must pass for merge |
| Go Backend | `go build`, `go test -short`, lint, vet |
| Dashboard | lint + build |
| Landing Page | build + install.sh verify |
| Mintlify Docs | validate + export |
| Node.js SDK | typecheck + build |
| Python SDK | ruff + pytest |
| Security | gosec + secret scan |
| Agent Guard | PR conventions, no forbidden files |

Path filters skip unrelated jobs on PRs for speed.

## Gemini Review Requirement

Gemini Code Assist reviews are handled by the GitHub app, not by a GitHub
Actions workflow and not by a `GEMINI_API_KEY` secret. Configure behavior in
`.gemini/config.yaml` and review rules in `.gemini/styleguide.md`.

Every non-draft PR should receive a Gemini Code Assist review automatically
after the app is installed for this repository. To manually invoke it, comment:

```text
/gemini review
```

## Never commit

- `node_modules/`, `dist/`, `.open-next/`, binaries (`agent`, `server`, `cli`)
- SSH keys, API keys, `.env` files
- Generated `landing/dist/docs/` (built in CI)

## Deploy after merge to master

| Target | Trigger | Workflow |
|--------|---------|----------|
| Landing + blogs | push `landing/**`, `wrangler.jsonc`, `workers/**` | `deploy-cloudflare.yml` |
| Docs subdomain | push `docs/**`, docs build scripts | `deploy-cloudflare.yml` |
| Dashboard | push `dashboard/**`, dashboard config | `deploy-cloudflare.yml` |
| Docker image | push `master` | `ci.yml` docker job |

Requires secrets: `CLOUDFLARE_API_TOKEN`, `CLOUDFLARE_ACCOUNT_ID`. Deploys fail fast if `scripts/preflight-cloudflare-token.sh` cannot validate Worker edit permissions.

## Efficient GitHub usage

- **One PR per branch** — do not stack unrelated changes
- **Path-scoped commits** — landing, docs, and backend in separate PRs when possible
- **Rebase on master** before push if branch is stale
- **Delete branch** after merge (automated)
- Use `workflow_dispatch` on `bootstrap-labels.yml` once to create repo labels

## One-time repo setup

See [.github/REPOSITORY_SETUP.md](REPOSITORY_SETUP.md) for branch protection and auto-merge settings.
