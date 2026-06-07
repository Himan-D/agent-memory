#!/usr/bin/env bash
# Full Cloudflare Workers Builds pipeline for agent-memory (landing + docs + dashboard).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DEPLOY_VERSION="$(cat "$ROOT/landing/DEPLOY_VERSION" 2>/dev/null || echo "unknown")"

echo "==> Cloudflare build version: ${DEPLOY_VERSION}"
export VITE_SANITY_PROJECT_ID="${VITE_SANITY_PROJECT_ID:-yhvdqwt4}"

echo "==> Building landing (hystersis.com + blogs.hystersis.com)"
cd "$ROOT/landing"
npm ci
npm run build

echo "==> Building Mintlify docs"
bash "$ROOT/scripts/build-docs.sh"

echo "==> Landing + docs build complete"
echo "    Workers Builds will deploy agent-memory worker with custom domains:"
echo "    - hystersis.com"
echo "    - www.hystersis.com"
echo "    - blogs.hystersis.com"

# Dashboard deploy is optional during Workers Builds — failure must not block landing.
echo "==> Deploying dashboard (app.hystersis.com) — non-blocking"
if bash "$ROOT/scripts/deploy-dashboard-builds.sh"; then
  echo "==> Dashboard deploy succeeded"
else
  echo "::warning:: Dashboard deploy failed — landing/docs will still deploy."
  echo "::warning:: Fix: connect hystersis-app Workers Builds OR add CLOUDFLARE_API_TOKEN to GitHub secrets."
  exit 0
fi

echo "==> Cloudflare build complete"
