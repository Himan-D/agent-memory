#!/bin/bash
# Deploy dashboard worker during Cloudflare Workers Builds.
# Workers Builds injects CF credentials automatically — no API token needed.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DEPLOY_VERSION="$(cat "$ROOT/dashboard/DEPLOY_VERSION" 2>/dev/null || cat "$ROOT/landing/DEPLOY_VERSION" 2>/dev/null || echo "unknown")"

echo "==> Dashboard deploy version: ${DEPLOY_VERSION}"
echo "==> Building dashboard (agent-memorydash → app.hystersis.com)"

cd "$ROOT/dashboard"

# Workers Builds may not have legacy-peer-deps in npm config
if [ -f package-lock.json ]; then
  npm ci --legacy-peer-deps
else
  npm install --legacy-peer-deps
fi

rm -rf .next .open-next

export NEXT_PUBLIC_API_URL="${NEXT_PUBLIC_API_URL:-https://api.hystersis.com}"
export BETTER_AUTH_URL="${BETTER_AUTH_URL:-https://app.hystersis.com}"
# Fix build-time BetterAuthError when secret is missing
export BETTER_AUTH_SECRET="${BETTER_AUTH_SECRET:-build-time-secret-placeholder}"

echo "==> OpenNext build..."
npx opennextjs-cloudflare build

echo "==> Wrangler deploy agent-memorydash..."
npx opennextjs-cloudflare deploy

echo "==> Dashboard deployed to https://app.hystersis.com"
echo "    If DNS fails, run: dig +short app.hystersis.com"
echo "    Custom domain should auto-provision when zone hystersis.com is on Cloudflare."
