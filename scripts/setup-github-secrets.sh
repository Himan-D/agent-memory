#!/usr/bin/env bash
# Set Cloudflare deploy secrets in GitHub Actions (run locally with gh auth login).
#
# Usage:
#   export CLOUDFLARE_API_TOKEN='your-token'
#   export CLOUDFLARE_ACCOUNT_ID='c50d52c51722d57e2c06c3eab5510dc3'
#   bash scripts/setup-github-secrets.sh
#
# Token must have: Account.Workers Scripts.Edit, Account.Workers Routes.Edit,
#                  Zone.DNS.Edit (for custom domains), Account.Account Settings.Read
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

echo "==> Verifying Cloudflare token can access Workers API..."
status=$(curl -sS -o /tmp/cf-workers-check.json -w "%{http_code}" \
  "https://api.cloudflare.com/client/v4/accounts/${CLOUDFLARE_ACCOUNT_ID}/workers/scripts" \
  -H "Authorization: Bearer ${CLOUDFLARE_API_TOKEN}")

if [ "$status" != "200" ]; then
  echo "::error::Token cannot list Workers scripts (HTTP ${status})."
  echo "Create a new token: Cloudflare Dashboard → My Profile → API Tokens"
  echo "  Use template: 'Edit Cloudflare Workers'"
  echo "  Account: Trinetra AI Solutions (${CLOUDFLARE_ACCOUNT_ID})"
  cat /tmp/cf-workers-check.json 2>/dev/null || true
  exit 1
fi

echo "==> Token OK — setting GitHub secrets on ${REPO}"
gh secret set CLOUDFLARE_API_TOKEN --body "$CLOUDFLARE_API_TOKEN" --repo "$REPO"
gh secret set CLOUDFLARE_ACCOUNT_ID --body "$CLOUDFLARE_ACCOUNT_ID" --repo "$REPO"

echo "==> Secrets set. Trigger deploy:"
echo "  gh workflow run deploy-cloudflare.yml --repo $REPO -f target=all"
echo "  Or: GitHub → Actions → Deploy Cloudflare (All) → Run workflow → all"
