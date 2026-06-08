#!/usr/bin/env bash
# Validate Cloudflare deploy credentials before any production deploy starts.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ACCOUNT_ID="${CLOUDFLARE_ACCOUNT_ID:-}"
TOKEN="${CLOUDFLARE_API_TOKEN:-}"

if [ -z "$TOKEN" ]; then
  echo "::error::CLOUDFLARE_API_TOKEN is missing." >&2
  exit 1
fi

if [ -z "$ACCOUNT_ID" ]; then
  echo "::error::CLOUDFLARE_ACCOUNT_ID is missing." >&2
  exit 1
fi

api_get() {
  local path="$1"
  local out="$2"
  curl -sS -o "$out" -w "%{http_code}" \
    "https://api.cloudflare.com/client/v4${path}" \
    -H "Authorization: Bearer ${TOKEN}"
}

require_200() {
  local label="$1"
  local path="$2"
  local out="/tmp/hystersis-cf-preflight-${label// /-}.json"
  local status

  status="$(api_get "$path" "$out")"
  if [ "$status" != "200" ]; then
    echo "::error::Cloudflare token failed ${label} preflight (HTTP ${status})." >&2
    echo "Required token permissions:" >&2
    echo "  Account: Workers Scripts:Edit" >&2
    echo "  Account: Workers Routes:Edit" >&2
    echo "  Account: Account Settings:Read" >&2
    echo "  Zone hystersis.com: Workers Routes:Edit" >&2
    echo "  Zone hystersis.com: DNS:Edit when provisioning custom domains" >&2
    cat "$out" >&2 || true
    exit 1
  fi
}

require_worker_name() {
  local config="$1"
  local expected="$2"
  if ! grep -q "\"name\"[[:space:]]*:[[:space:]]*\"${expected}\"" "$ROOT/$config"; then
    echo "::error::$config must deploy worker name '${expected}'." >&2
    exit 1
  fi
}

require_worker_name "wrangler.jsonc" "agent-memory"
require_worker_name "docs/wrangler.jsonc" "hystersis-docs"
require_worker_name "dashboard/wrangler.jsonc" "hystersis-app"

require_200 "account workers services" "/accounts/${ACCOUNT_ID}/workers/services"
require_200 "agent-memory worker" "/accounts/${ACCOUNT_ID}/workers/services/agent-memory"
require_200 "hystersis-docs worker" "/accounts/${ACCOUNT_ID}/workers/services/hystersis-docs"
require_200 "hystersis-app worker" "/accounts/${ACCOUNT_ID}/workers/services/hystersis-app"

echo "Cloudflare deploy preflight passed."
