# GitHub Repository Setup (one-time)

Configure these settings in **GitHub → Settings** for best-in-class agent automation.

## 1. Enable auto-merge

**Settings → General → Pull Requests**

- [x] Allow auto-merge
- [x] Automatically delete head branches

Auto-merge runs after **CI Success** passes (workflow `Auto Merge` triggers on CI completion).
Eligible PRs: `cursor/*` branches, `automerge` label, or dependabot.

## 2. Branch protection (`master`)

**Settings → Branches → Add rule**

| Setting | Value |
|---------|-------|
| Branch name pattern | `master` |
| Require status checks | Yes |
| Required checks | `CI Success` |
| Require branches up to date | Yes |
| Require linear history | Optional (recommended) |
| Allow auto-merge | Yes |
| Restrict who can push | Optional |

## 3. Actions permissions

**Settings → Actions → General**

- Workflow permissions: **Read and write**
- Allow GitHub Actions to create and approve pull requests: **Yes**

## 4. Cloudflare (same account for landing + dashboard)

Both workers target account `c50d52c51722d57e2c06c3eab5510dc3`:

| Worker | Domain | Config |
|--------|--------|--------|
| `agent-memory` | hystersis.com | `wrangler.jsonc` (Workers Builds connected) |
| `hystersis-app` | app.hystersis.com | `dashboard/wrangler.jsonc` |

### Option A: GitHub Actions (recommended for dashboard)

**Settings → Secrets → Actions**

| Secret | Purpose |
|--------|---------|
| `CLOUDFLARE_API_TOKEN` | **Required** — use "Edit Cloudflare Workers" template |
| `CLOUDFLARE_ACCOUNT_ID` | `c50d52c51722d57e2c06c3eab5510dc3` |
| `BETTER_AUTH_SECRET` | Dashboard worker JWT signing (or legacy `NEXTAUTH_SECRET`) |
| `BETTER_AUTH_API_KEY` | Better Auth infra/dash plugin |
| `ADMIN_API_KEY` | Dashboard API proxy admin key |

Then run **Actions → Deploy Cloudflare (All) → Run workflow**.

Generate tokens: `bash scripts/generate-tokens.sh`

### Option B: Cloudflare Workers Builds (no GitHub token)

1. [Workers & Pages](https://dash.cloudflare.com/) → **Create** → **Worker** → **Connect Git**
2. Repo: `Himan-D/agent-memory`, branch: `master`
3. Root directory: `dashboard`
4. Build command: `npm ci --legacy-peer-deps && npx opennextjs-cloudflare build`
5. Deploy command: `npx opennextjs-cloudflare deploy`
6. Add worker secrets: `BETTER_AUTH_SECRET`, `BETTER_AUTH_API_KEY`, `ADMIN_API_KEY`
7. Custom domain: `app.hystersis.com` (auto-creates DNS when zone is on Cloudflare)

## 5. Other secrets (optional)
| `DOCKER_USERNAME` | Docker Hub (optional) |
| `DOCKER_TOKEN` | Docker Hub (optional) |
| `NPM_TOKEN` | SDK publish (optional) |
| `PYPI_API_TOKEN` | Python SDK publish (optional) |

## 6. Bootstrap labels

Run once: **Actions → Bootstrap Labels → Run workflow**

Creates: `automerge`, `do-not-merge`, `agent`, `dependencies`, path labels.

## 7. Dependabot

Already configured in `.github/dependabot.yml`. Patch/minor PRs get `automerge` label and merge when CI passes.

## 8. Merge queue (optional, GitHub Team+)

For high-traffic repos, enable merge queue on `master` with `CI Success` as required check.

## Verification

```bash
# Open agent PR — should auto-label agent + automerge
# After CI Success — should squash merge automatically

gh pr list --label automerge
gh run list --workflow=ci.yml
```
