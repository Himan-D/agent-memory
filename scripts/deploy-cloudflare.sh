#!/bin/bash
# Deploy landing + dashboard to the same Cloudflare account.
# Requires: CLOUDFLARE_API_TOKEN, CLOUDFLARE_ACCOUNT_ID
#
# Usage:
#   bash scripts/deploy-cloudflare.sh              # deploy both
#   bash scripts/deploy-cloudflare.sh landing      # landing only
#   bash scripts/deploy-cloudflare.sh dashboard    # dashboard only
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TARGET="${1:-all}"

if [ -z "${CLOUDFLARE_API_TOKEN:-}" ] || [ -z "${CLOUDFLARE_ACCOUNT_ID:-}" ]; then
  echo "error: set CLOUDFLARE_API_TOKEN and CLOUDFLARE_ACCOUNT_ID" >&2
  exit 1
fi

deploy_landing() {
  echo "==> Building landing (hystersis.com)"
  cd "$ROOT/landing"
  npm ci
  npm run build
  cd "$ROOT"
  bash scripts/build-docs.sh
  test -f landing/dist/install.sh

  echo "==> Deploying worker: agent-memory → hystersis.com"
  npx wrangler deploy
}

deploy_dashboard() {
  echo "==> Building dashboard (app.hystersis.com)"
  cd "$ROOT/dashboard"
  npm ci --legacy-peer-deps
  rm -rf .next .open-next
  export NEXT_PUBLIC_API_URL="${NEXT_PUBLIC_API_URL:-https://api.hystersis.com}"
  npm run deploy

  if [ -n "${BETTER_AUTH_SECRET:-}" ]; then
    echo "==> Syncing BETTER_AUTH_SECRET"
    printf '%s' "$BETTER_AUTH_SECRET" | npx wrangler secret put BETTER_AUTH_SECRET
  fi
  if [ -n "${BETTER_AUTH_API_KEY:-}" ]; then
    echo "==> Syncing BETTER_AUTH_API_KEY"
    printf '%s' "$BETTER_AUTH_API_KEY" | npx wrangler secret put BETTER_AUTH_API_KEY
  fi
  if [ -n "${ADMIN_API_KEY:-}" ]; then
    echo "==> Syncing ADMIN_API_KEY"
    printf '%s' "$ADMIN_API_KEY" | npx wrangler secret put ADMIN_API_KEY
  fi
}

case "$TARGET" in
  landing)  deploy_landing ;;
  dashboard) deploy_dashboard ;;
  all)
    deploy_landing
    deploy_dashboard
    ;;
  *)
    echo "usage: $0 [all|landing|dashboard]" >&2
    exit 1
    ;;
esac

echo "==> Deploy complete"
echo "    https://hystersis.com"
echo "    https://app.hystersis.com"
