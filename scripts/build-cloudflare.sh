#!/usr/bin/env bash
# Full Cloudflare Workers Builds pipeline for agent-memory (landing + docs + dashboard).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

echo "==> Building landing (hystersis.com)"
cd "$ROOT/landing"
npm ci
npm run build

echo "==> Building Mintlify docs"
bash "$ROOT/scripts/build-docs.sh"

echo "==> Deploying dashboard (app.hystersis.com)"
if bash "$ROOT/scripts/deploy-dashboard-builds.sh"; then
  echo "==> Dashboard deploy succeeded"
else
  echo "::warning:: Dashboard deploy failed — landing and docs will still deploy"
fi

echo "==> Cloudflare build complete"
