# GitHub Repository Setup (one-time)

Configure these settings in **GitHub → Settings** for best-in-class agent automation.

## 1. Enable auto-merge

**Settings → General → Pull Requests**

- [x] Allow auto-merge
- [x] Automatically delete head branches

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

## 4. Secrets (Settings → Secrets → Actions)

| Secret | Purpose |
|--------|---------|
| `CLOUDFLARE_API_TOKEN` | Deploy landing + docs |
| `CLOUDFLARE_ACCOUNT_ID` | Cloudflare account |
| `DOCKER_USERNAME` | Docker Hub (optional) |
| `DOCKER_TOKEN` | Docker Hub (optional) |
| `NPM_TOKEN` | SDK publish (optional) |
| `PYPI_API_TOKEN` | Python SDK publish (optional) |

## 5. Bootstrap labels

Run once: **Actions → Bootstrap Labels → Run workflow**

Creates: `automerge`, `do-not-merge`, `agent`, `dependencies`, path labels.

## 6. Dependabot

Already configured in `.github/dependabot.yml`. Patch/minor PRs get `automerge` label and merge when CI passes.

## 7. Merge queue (optional, GitHub Team+)

For high-traffic repos, enable merge queue on `master` with `CI Success` as required check.

## Verification

```bash
# Open agent PR — should auto-label agent + automerge
# After CI Success — should squash merge automatically

gh pr list --label automerge
gh run list --workflow=ci.yml
```
