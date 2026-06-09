#!/usr/bin/env bash
# Cloudflare Workers Builds pipeline for agent-memory (landing + docs ONLY).
# Dashboard (agent-memorydash) must deploy separately — same worker name causes apex to serve Next.js.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DEPLOY_VERSION="$(cat "$ROOT/landing/DEPLOY_VERSION" 2>/dev/null || echo "unknown")"

echo "==> Cloudflare build version: ${DEPLOY_VERSION}"
export VITE_SANITY_PROJECT_ID="${VITE_SANITY_PROJECT_ID:-yhvdqwt4}"
export VITE_DASHBOARD_URL="${VITE_DASHBOARD_URL:-https://app.hystersis.com}"
export VITE_API_URL="${VITE_API_URL:-https://api.hystersis.com}"

echo "==> Building landing (hystersis.com + blogs.hystersis.com)"
cd "$ROOT/landing"
npm ci
npm run build

if grep -R "http://localhost:3000" "$ROOT/landing/dist" >/dev/null 2>&1; then
  echo "error: landing production build contains localhost dashboard URL" >&2
  exit 1
fi

if [[ "${SKIP_DOCS_BUILD:-0}" == "1" ]]; then
  echo "==> Skipping Mintlify docs build (SKIP_DOCS_BUILD=1)"
else
  echo "==> Building Mintlify docs"
  bash "$ROOT/scripts/build-docs.sh"
fi

echo "==> Landing + docs build complete"
echo "    Workers Builds will deploy agent-memory worker with custom domains:"
echo "    - hystersis.com"
echo "    - www.hystersis.com"
echo "    - blogs.hystersis.com"
echo ""
echo "    Dashboard (agent-memorydash → app.hystersis.com) deploys separately."
echo "    Do NOT deploy dashboard from this build — both used same name and overwrote landing."

echo "==> Cloudflare build complete"
