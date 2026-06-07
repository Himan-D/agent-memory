#!/usr/bin/env bash
# Cloudflare Workers Builds pipeline for agent-memory (landing + docs ONLY).
# Dashboard (hystersis-app) must deploy separately — same worker name causes apex to serve Next.js.
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
echo ""
echo "    Dashboard (hystersis-app → app.hystersis.com) deploys separately."
echo "    Do NOT deploy dashboard from this build — both used name hystersis-app and overwrote landing."

echo "==> Cloudflare build complete"
