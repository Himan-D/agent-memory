#!/bin/bash
# Deploy dashboard worker during Cloudflare Workers Builds.
# Workers Builds injects CF credentials automatically — no API token needed.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

echo "==> Building dashboard (hystersis-app → app.hystersis.com)"
cd "$ROOT/dashboard"
npm ci --legacy-peer-deps
rm -rf .next .open-next
export NEXT_PUBLIC_API_URL="${NEXT_PUBLIC_API_URL:-https://api.hystersis.com}"
npx opennextjs-cloudflare build
npx opennextjs-cloudflare deploy

echo "==> Dashboard deployed to https://app.hystersis.com"
