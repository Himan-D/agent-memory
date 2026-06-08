#!/usr/bin/env bash
# Verify DNS and HTTP for all Hystersis production domains.
set -euo pipefail

check_dns() {
  local host="$1"
  local result
  result=$(dig +short "$host" A 2>/dev/null | head -1)
  if [ -z "$result" ]; then
    result=$(dig +short "$host" CNAME 2>/dev/null | head -1)
  fi
  if [ -z "$result" ]; then
    echo "DNS_FAIL $host (no A/CNAME record)"
    return 1
  fi
  echo "DNS_OK   $host → $result"
  return 0
}

check_http() {
  local url="$1"
  local expect="${2:-200}"
  local code
  code=$(curl -sS -o /dev/null -w "%{http_code}" --max-time 15 "$url" 2>/dev/null || echo "000")

  if ! [[ "$code" =~ ^[0-9]{3}$ ]]; then
    echo "HTTP_FAIL $url → unreachable"
    return 1
  fi

  if [ "$expect" = "any" ] && [ "$code" -lt 500 ]; then
    echo "HTTP_OK  $url → $code"
    return 0
  fi

  if [ "$code" = "$expect" ]; then
    echo "HTTP_OK  $url → $code"
    return 0
  fi

  echo "HTTP_FAIL $url → $code (expected $expect)"
  return 1
}

failures=0

echo "==> DNS checks"
for host in hystersis.com www.hystersis.com app.hystersis.com blogs.hystersis.com api.hystersis.com; do
  check_dns "$host" || failures=$((failures + 1))
done

echo ""
echo "==> HTTP checks"
check_http "https://hystersis.com/" "any" || failures=$((failures + 1))

# Apex must serve landing SPA, not dashboard (Next.js redirects to /auth/signin)
if headers=$(curl -sS -I --max-time 15 "https://hystersis.com/" 2>/dev/null); then
  if echo "$headers" | grep -qi '^location:.*auth/signin'; then
    echo "HTTP_FAIL https://hystersis.com/ → dashboard on apex (expected landing)"
    echo "         Fix: restore wrangler.jsonc name to agent-memory and redeploy landing worker"
    failures=$((failures + 1))
  elif body=$(curl -sS --max-time 15 "https://hystersis.com/" 2>/dev/null); then
    if echo "$body" | grep -q 'Hystersis Dashboard'; then
      echo "HTTP_FAIL https://hystersis.com/ → dashboard HTML on apex (expected landing)"
      failures=$((failures + 1))
    fi
  fi
fi
check_http "https://hystersis.com/docs" "any" || failures=$((failures + 1))
check_http "https://hystersis.com/blog" "any" || failures=$((failures + 1))
check_http "https://blogs.hystersis.com/" "any" || failures=$((failures + 1))
check_http "https://app.hystersis.com/auth/signin" "any" || failures=$((failures + 1))
check_http "https://api.hystersis.com/health" "200" || failures=$((failures + 1))

echo ""
if [ "$failures" -gt 0 ]; then
  echo "::error::$failures domain/endpoint check(s) failed."
  echo ""
  echo "Common fixes:"
  echo "  1. Add CLOUDFLARE_API_TOKEN + CLOUDFLARE_ACCOUNT_ID to GitHub secrets"
  echo "  2. Run: bash scripts/deploy-cloudflare.sh all"
  echo "  3. Or trigger Workers Builds for agent-memory + hystersis-app in Cloudflare dashboard"
  echo "  4. Ensure custom domains in wrangler.jsonc are deployed (provisions DNS automatically)"
  echo "  5. /blog HTTP 500 = stale worker — merge workers/site.js SPA fix and redeploy"
  echo "  6. Apex redirects to /auth/signin = worker name collision — wrangler.jsonc must be agent-memory, not hystersis-app"
  exit 1
fi

echo "All domain checks passed."
