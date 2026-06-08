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

## 4. Cloudflare auto-deploy (landing + docs + dashboard)

Both workers target account `c50d52c51722d57e2c06c3eab5510dc3`:

| Worker | Domain | Config | Auto-deploy |
|--------|--------|--------|-------------|
| `agent-memory` | hystersis.com, blogs.hystersis.com | `wrangler.jsonc` | ✅ GitHub Actions |
| `hystersis-docs` | docs.hystersis.com | `docs/wrangler.jsonc` | ✅ GitHub Actions |
| `hystersis-app` | app.hystersis.com | `dashboard/wrangler.jsonc` | ✅ GitHub Actions |

### Single deploy owner

Production deploys are owned by **Actions → Deploy Cloudflare (All)**. Cloudflare Workers Builds can stay connected temporarily, but should not be treated as the source of truth.

**Never use the same worker `name` across apps**:
- landing root config: `agent-memory`
- docs config: `hystersis-docs`
- dashboard config: `hystersis-app`

**Settings → Secrets → Actions**

| Secret | Purpose |
|--------|---------|
| `CLOUDFLARE_API_TOKEN` | **Required** — must pass `scripts/preflight-cloudflare-token.sh` |
| `CLOUDFLARE_ACCOUNT_ID` | `c50d52c51722d57e2c06c3eab5510dc3` |
| `BETTER_AUTH_SECRET` | Dashboard worker JWT signing (or legacy `NEXTAUTH_SECRET`) |
| `BETTER_AUTH_API_KEY` | Better Auth infra/dash plugin |
| `ADMIN_API_KEY` | Dashboard API proxy admin key |

Then run **Actions → Deploy Cloudflare (All) → Run workflow** with target `all`, `landing`, `docs`, or `dashboard`.

Generate local credentials: `bash scripts/generate-tokens.sh`

### Manual deploy (local)

```bash
# All workers (requires CLOUDFLARE_API_TOKEN + CLOUDFLARE_ACCOUNT_ID)
npm run deploy:all

# Or a single target
bash scripts/deploy-cloudflare.sh docs
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
| `hystersis.com` redirects to `/auth/signin` | **Worker name collision** — root `wrangler.jsonc` was renamed to `hystersis-app`, same as dashboard; dashboard deploy overwrote landing | Restore root `name` to `agent-memory`, remove dashboard deploy from `build-cloudflare.sh`, redeploy landing |
| `app.hystersis.com` DNS fails | Dashboard worker never deployed | Add `CLOUDFLARE_API_TOKEN` to GitHub secrets, run **Deploy Cloudflare (All)** workflow with target `dashboard` |
| `docs.hystersis.com` returns unstyled HTML | Docs worker not routing `/docs/*` assets through `docs/worker.js` | Run **Deploy Cloudflare (All)** workflow with target `docs` |
| `blogs.hystersis.com` DNS fails | Custom domain not provisioned | Run **Deploy Cloudflare (All)** with target `landing` after confirming Cloudflare preflight passes |
| GH Actions preflight fails with Cloudflare `10000` | Token is missing Worker edit permissions | Create a new token with Workers edit permissions and rerun `scripts/setup-github-secrets.sh` |
| Dashboard still has demo credentials | `hystersis-app` not redeployed | Run workflow with target `dashboard` or `cd dashboard && npm run deploy` |
| Cloudflare MCP shows needsAuth in cloud agent | OAuth is local to Cursor Desktop | Use GitHub secrets + workflow, or deploy from local `wrangler` |

**Required GitHub secrets for reliable deploys:**

```
CLOUDFLARE_API_TOKEN  →  Edit Cloudflare Workers template (NOT read-only)
CLOUDFLARE_ACCOUNT_ID →  c50d52c51722d57e2c06c3eab5510dc3
```

Token permissions:
- Account: Workers Scripts Edit
- Account: Workers Routes Edit
- Account: Account Settings Read
- Zone `hystersis.com`: Workers Routes Edit
- Zone `hystersis.com`: DNS Edit when provisioning custom domains

Set secrets locally (Cloud Agent cannot write repo secrets):

```bash
export CLOUDFLARE_API_TOKEN='your-token-with-workers-edit'
export CLOUDFLARE_ACCOUNT_ID='c50d52c51722d57e2c06c3eab5510dc3'
bash scripts/setup-github-secrets.sh
gh workflow run deploy-cloudflare.yml --repo Himan-D/agent-memory -f target=all
```

Verify token before setting secrets — must return HTTP 200:

```bash
bash scripts/preflight-cloudflare-token.sh
```

If you get `10000 Authentication error`, recreate the token with **Workers Scripts → Edit** permission.
