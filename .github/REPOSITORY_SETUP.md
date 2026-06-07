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

## 4. Cloudflare auto-deploy (landing + dashboard)

Both workers target account `c50d52c51722d57e2c06c3eab5510dc3`:

| Worker | Domain | Config | Auto-deploy |
|--------|--------|--------|-------------|
| `agent-memory` | hystersis.com, blogs.hystersis.com | `wrangler.jsonc` | ✅ Workers Builds (connected) |
| `hystersis-app` | app.hystersis.com | `dashboard/wrangler.jsonc` | ⚠️ Via root Workers Builds OR GH Actions token |

### How dashboard auto-deploys (no GitHub token needed)

Root `wrangler.jsonc` build command runs `scripts/deploy-dashboard-builds.sh` after the landing build. Cloudflare Workers Builds for `agent-memory` already has deploy credentials, so the dashboard worker (`hystersis-app`) is built and deployed on every landing Workers Builds run.

**Required one-time Cloudflare setting** for `agent-memory` worker:

**Settings → Builds → Build watch paths → Include:**
- `landing/**`
- `dashboard/**`
- `wrangler.jsonc`
- `scripts/deploy-dashboard-builds.sh`

Without `dashboard/**` in watch paths, dashboard-only commits will not trigger a redeploy.

GitHub Actions (`Deploy Cloudflare (All)`) can also deploy when `CLOUDFLARE_API_TOKEN` is set in repo secrets.

### Option A: Cloudflare Workers Builds for dashboard (recommended — matches landing)

Connect the dashboard worker to Git so it auto-deploys on every `master` push, same as landing:

1. Open [Workers & Pages](https://dash.cloudflare.com/) → select **`hystersis-app`**
2. Go to **Settings → Builds → Connect**
3. Connect repo: `Himan-D/agent-memory`, branch: `master`
4. Set **Root directory**: `dashboard`
5. Set build settings:

| Setting | Value |
|---------|-------|
| Build command | `npm ci --legacy-peer-deps && npx opennextjs-cloudflare build` |
| Deploy command | `npx opennextjs-cloudflare deploy` |
| Build watch paths (include) | `dashboard/**` |

6. Add **Worker secrets** (Settings → Variables & Secrets):
   - `BETTER_AUTH_SECRET`
   - `BETTER_AUTH_API_KEY`
   - `ADMIN_API_KEY`

7. Custom domain `app.hystersis.com` is already in `dashboard/wrangler.jsonc`

8. Push to `master` — Cloudflare builds and deploys automatically

### Option B: GitHub Actions (backup or alternative)

**Settings → Secrets → Actions**

| Secret | Purpose |
|--------|---------|
| `CLOUDFLARE_API_TOKEN` | **Required for GH Actions deploy** — use "Edit Cloudflare Workers" template |
| `CLOUDFLARE_ACCOUNT_ID` | `c50d52c51722d57e2c06c3eab5510dc3` |
| `BETTER_AUTH_SECRET` | Dashboard worker JWT signing (or legacy `NEXTAUTH_SECRET`) |
| `BETTER_AUTH_API_KEY` | Better Auth infra/dash plugin |
| `ADMIN_API_KEY` | Dashboard API proxy admin key |

Then run **Actions → Deploy Cloudflare (All) → Run workflow**.

Generate local credentials: `bash scripts/generate-tokens.sh`

### Manual deploy (local)

```bash
# Dashboard only
cd dashboard && npm run deploy

# Both workers (requires CLOUDFLARE_API_TOKEN + CLOUDFLARE_ACCOUNT_ID)
npm run deploy:all
```

## 5. Other secrets (optional)

| Secret | Purpose |
|--------|---------|
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
# Full domain check (DNS + HTTP)
bash scripts/verify-domains.sh

# Content checks
bash scripts/verify-production.sh all

# Individual endpoints
curl -sS -o /dev/null -w "%{http_code}" https://hystersis.com/
curl -sS -o /dev/null -w "%{http_code}" https://blogs.hystersis.com/
curl -sS -o /dev/null -w "%{http_code}" https://app.hystersis.com/auth/signin
dig +short app.hystersis.com
dig +short blogs.hystersis.com
```

### Troubleshooting: deployment not working

| Symptom | Cause | Fix |
|---------|-------|-----|
| `app.hystersis.com` DNS fails | Dashboard worker never deployed | Add `CLOUDFLARE_API_TOKEN` to GitHub secrets, run **Deploy Cloudflare (All)** workflow |
| `blogs.hystersis.com` DNS fails | Custom domain not provisioned | `wrangler deploy` from root — Workers Builds must run after `wrangler.jsonc` change |
| GH Actions "skipped" deploy | No `CLOUDFLARE_API_TOKEN` secret | Add token with "Edit Cloudflare Workers" permission |
| Dashboard still has demo credentials | `hystersis-app` not redeployed | Run workflow with target `dashboard` or `cd dashboard && npm run deploy` |
| Cloudflare MCP shows needsAuth in cloud agent | OAuth is local to Cursor Desktop | Use GitHub secrets + workflow, or deploy from local `wrangler` |

**Required GitHub secret for reliable deploys:**

```
CLOUDFLARE_API_TOKEN  →  Edit Cloudflare Workers template
CLOUDFLARE_ACCOUNT_ID →  c50d52c51722d57e2c06c3eab5510dc3
```
