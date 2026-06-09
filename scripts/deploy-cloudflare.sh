#!/bin/bash
# Deploy landing, docs, and dashboard to the same Cloudflare account.
# Requires: CLOUDFLARE_API_TOKEN, CLOUDFLARE_ACCOUNT_ID
#
# Usage:
#   bash scripts/deploy-cloudflare.sh              # deploy all
#   bash scripts/deploy-cloudflare.sh landing      # landing only
#   bash scripts/deploy-cloudflare.sh docs         # docs only
#   bash scripts/deploy-cloudflare.sh dashboard    # dashboard only
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TARGET="${1:-all}"

if [ -z "${CLOUDFLARE_API_TOKEN:-}" ] || [ -z "${CLOUDFLARE_ACCOUNT_ID:-}" ]; then
  echo "error: set CLOUDFLARE_API_TOKEN and CLOUDFLARE_ACCOUNT_ID" >&2
  exit 1
fi

deploy_landing() {
  echo "==> Building landing (hystersis.com + blogs.hystersis.com)"
  cd "$ROOT/landing"
  export VITE_SANITY_PROJECT_ID="${VITE_SANITY_PROJECT_ID:-yhvdqwt4}"
  export VITE_DASHBOARD_URL="${VITE_DASHBOARD_URL:-https://app.hystersis.com}"
  export VITE_API_URL="${VITE_API_URL:-https://api.hystersis.com}"
  npm ci
  npm run build

  if grep -R "http://localhost:3000" "$ROOT/landing/dist" >/dev/null 2>&1; then
    echo "error: landing production build contains localhost dashboard URL" >&2
    exit 1
  fi

  cd "$ROOT"
  bash scripts/build-docs.sh
  test -f landing/dist/install.sh

  echo "==> Deploying worker: agent-memory"
  echo "    Domains: hystersis.com, www.hystersis.com, blogs.hystersis.com"
  CLOUDFLARE_ACCOUNT_ID="${CLOUDFLARE_ACCOUNT_ID}" npx wrangler deploy
}

deploy_docs() {
  echo "==> Building docs (docs.hystersis.com)"
  cd "$ROOT"
  bash scripts/build-docs.sh

  echo "==> Deploying worker: hystersis-docs"
  echo "    Domain: docs.hystersis.com"
  cd "$ROOT/docs"
  CLOUDFLARE_ACCOUNT_ID="${CLOUDFLARE_ACCOUNT_ID}" npx wrangler deploy --config wrangler.jsonc
}

deploy_dashboard() {
  echo "==> Building dashboard (app.hystersis.com)"
  cd "$ROOT/dashboard"
  npm ci --legacy-peer-deps
  rm -rf .next .open-next
  export NEXT_PUBLIC_API_URL="${NEXT_PUBLIC_API_URL:-https://api.hystersis.com}"
  # Fix build-time BetterAuthError when secret is missing
  export BETTER_AUTH_SECRET="${BETTER_AUTH_SECRET:-build-time-secret-placeholder}"
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
  docs) deploy_docs ;;
  dashboard) deploy_dashboard ;;
  all)
    deploy_landing
    deploy_docs
    deploy_dashboard
    ;;
  *)
    echo "usage: $0 [all|landing|docs|dashboard]" >&2
    exit 1
    ;;
esac

echo "==> Deploy complete"
echo "    https://hystersis.com"
echo "    https://blogs.hystersis.com"
echo "    https://docs.hystersis.com"
echo "    https://app.hystersis.com"
echo ""
echo "==> Verifying domains..."
bash "$ROOT/scripts/verify-domains.sh" || true
