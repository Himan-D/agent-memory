#!/usr/bin/env bash
# Set Cloudflare deploy secrets in GitHub Actions (run locally with gh auth login).
#
# Usage:
#   export CLOUDFLARE_API_TOKEN='your-token'
#   export CLOUDFLARE_ACCOUNT_ID='c50d52c51722d57e2c06c3eab5510dc3'
#   bash scripts/setup-github-secrets.sh
#
# Token must pass scripts/preflight-cloudflare-token.sh:
# Account Workers Scripts Edit, Workers Routes Edit, Account Settings Read,
# and zone hystersis.com Workers Routes Edit/DNS Edit for custom domains.
set -euo pipefail

REPO="${GITHUB_REPO:-Himan-D/agent-memory}"

if ! command -v gh >/dev/null 2>&1; then
  echo "error: install GitHub CLI (gh) and run: gh auth login" >&2
  exit 1
fi

if [ -z "${CLOUDFLARE_API_TOKEN:-}" ]; then
  echo "error: set CLOUDFLARE_API_TOKEN" >&2
  exit 1
fi

if [ -z "${CLOUDFLARE_ACCOUNT_ID:-}" ]; then
  echo "error: set CLOUDFLARE_ACCOUNT_ID" >&2
  exit 1
fi

echo "==> Verifying Cloudflare token can deploy Workers..."
bash "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/preflight-cloudflare-token.sh"

echo "==> Token OK — setting GitHub secrets on ${REPO}"
gh secret set CLOUDFLARE_API_TOKEN --body "$CLOUDFLARE_API_TOKEN" --repo "$REPO"
gh secret set CLOUDFLARE_ACCOUNT_ID --body "$CLOUDFLARE_ACCOUNT_ID" --repo "$REPO"

echo "==> Secrets set. Trigger deploy:"
echo "  gh workflow run deploy-cloudflare.yml --repo $REPO -f target=all"
echo "  Or: GitHub → Actions → Deploy Cloudflare (All) → Run workflow → all"
