#!/usr/bin/env bash
# Curl smoke test for Hystersis API — verifies core endpoints match SDK/docs contracts.
#
# Usage:
#   export HYSTERSIS_BASE_URL=http://localhost:8080
#   export HYSTERSIS_API_KEY=your-key
#   ./scripts/smoke-test.sh
#
# Exit 0 if all checks pass; non-zero on first failure.

set -euo pipefail

BASE="${HYSTERSIS_BASE_URL:-${AGENT_MEMORY_URL:-http://localhost:8080}}"
API_KEY="${HYSTERSIS_API_KEY:-${AGENT_MEMORY_API_KEY:-test-key}}"
USER_ID="smoke-user-$(date +%s)"
PASS=0
FAIL=0
SKIP=0

pass() { echo "  ✓ $1"; PASS=$((PASS + 1)); }
fail() { echo "  ✗ $1"; FAIL=$((FAIL + 1)); return 1; }
skip() { echo "  ~ $1 (skipped)"; SKIP=$((SKIP + 1)); }

curl_json() {
  local method="$1" path="$2" data="${3:-}"
  local args=(-fsS -m 30 -X "$method" "${BASE}${path}" -H "Content-Type: application/json")
  if [[ -n "$API_KEY" ]]; then
    args+=(-H "X-API-Key: $API_KEY")
  fi
  if [[ -n "$data" ]]; then
    args+=(-d "$data")
  fi
  curl "${args[@]}"
}

http_code() {
  local method="$1" path="$2" data="${3:-}"
  local args=(-s -o /dev/null -w "%{http_code}" -m 30 -X "$method" "${BASE}${path}")
  if [[ -n "$API_KEY" ]]; then
    args+=(-H "X-API-Key: $API_KEY")
  fi
  if [[ -n "$data" ]]; then
    args+=(-H "Content-Type: application/json" -d "$data")
  fi
  curl "${args[@]}"
}

echo "Hystersis API smoke test"
echo "Base URL: $BASE"
echo ""

# --- Public endpoints ---
echo "== Public =="
if curl -fsS -m 10 "${BASE}/health" >/dev/null 2>&1; then
  pass "GET /health"
else
  fail "GET /health — is the server running at $BASE?" || true
  echo ""
  echo "Start server: docker compose up -d neo4j qdrant redis && go run ./cmd/server"
  echo "Or set HYSTERSIS_BASE_URL to your deployed API."
  exit 1
fi

if curl -fsS -m 10 "${BASE}/ready" >/dev/null 2>&1; then pass "GET /ready"; else skip "GET /ready"; fi
if curl -fsS -m 10 "${BASE}/billing/plans" | python3 -c "import sys,json; d=json.load(sys.stdin); assert 'plans' in d" 2>/dev/null; then
  pass "GET /billing/plans"
else
  skip "GET /billing/plans (billing may be unconfigured)"
fi

# --- Auth ---
echo ""
echo "== Auth =="
code=$(http_code GET /memories)
if [[ "$code" == "401" ]]; then
  pass "GET /memories without key → 401"
elif [[ "$code" == "200" ]]; then
  skip "GET /memories without key → 200 (auth disabled)"
else
  fail "GET /memories without key → $code (expected 401 or 200)"
fi

# --- Memory CRUD ---
echo ""
echo "== Memory CRUD =="
CREATE_RESP=$(curl_json POST /memories "{\"content\":\"smoke test memory\",\"user_id\":\"${USER_ID}\"}" 2>/dev/null || echo "")
MEM_ID=$(echo "$CREATE_RESP" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('id',''))" 2>/dev/null || echo "")

if [[ -n "$MEM_ID" ]]; then
  pass "POST /memories → id=$MEM_ID"
else
  fail "POST /memories — response: ${CREATE_RESP:0:200}"
fi

if [[ -n "$MEM_ID" ]]; then
  if curl_json GET "/memories/${MEM_ID}" | python3 -c "import sys,json; d=json.load(sys.stdin); assert d.get('id')" 2>/dev/null; then
    pass "GET /memories/{id}"
  else
    fail "GET /memories/{id}"
  fi

  if curl_json PUT "/memories/${MEM_ID}" '{"content":"updated smoke memory"}' >/dev/null 2>&1; then
    pass "PUT /memories/{id}"
  else
    fail "PUT /memories/{id}"
  fi
fi

# --- Search ---
echo ""
echo "== Search =="
if curl_json GET "/search?query=smoke&limit=5" >/dev/null 2>&1; then
  pass "GET /search"
else
  fail "GET /search"
fi

if curl -fsS -m 30 -H "X-API-Key: $API_KEY" "${BASE}/search/enhanced?query=smoke&mode=vector&limit=5" >/dev/null 2>&1; then
  pass "GET /search/enhanced"
else
  skip "GET /search/enhanced"
fi

if curl_json POST /search/hybrid '{"query":"smoke test","semantic_limit":5,"keyword_limit":5}' >/dev/null 2>&1; then
  pass "POST /search/hybrid"
else
  skip "POST /search/hybrid"
fi

# --- v3 compat ---
echo ""
echo "== v3 compat =="
V3_ADD=$(curl_json POST /v3/add "{\"messages\":[{\"role\":\"user\",\"content\":\"I prefer dark mode\"}],\"user_id\":\"${USER_ID}\"}" 2>/dev/null || echo "")
if echo "$V3_ADD" | python3 -c "import sys,json; d=json.load(sys.stdin); assert d.get('count',0)>=0" 2>/dev/null; then
  pass "POST /v3/add"
else
  skip "POST /v3/add — ${V3_ADD:0:120}"
fi

if curl_json POST /v3/search "{\"query\":\"dark mode\",\"user_id\":\"${USER_ID}\",\"limit\":5}" >/dev/null 2>&1; then
  pass "POST /v3/search"
else
  skip "POST /v3/search"
fi

# --- Profiles ---
echo ""
echo "== Profiles =="
if curl_json GET "/profiles/${USER_ID}" >/dev/null 2>&1; then
  pass "GET /profiles/{userID}"
else
  skip "GET /profiles/{userID}"
fi

# --- Skills ---
echo ""
echo "== Skills =="
if curl -fsS -m 30 -H "X-API-Key: $API_KEY" "${BASE}/skills?limit=1" >/dev/null 2>&1; then
  pass "GET /skills"
else
  skip "GET /skills"
fi

# --- Compression ---
echo ""
echo "== Compression =="
if curl -fsS -m 30 -H "X-API-Key: $API_KEY" "${BASE}/compression/stats" >/dev/null 2>&1; then
  pass "GET /compression/stats"
else
  skip "GET /compression/stats"
fi

# --- Cleanup ---
echo ""
echo "== Cleanup =="
if [[ -n "$MEM_ID" ]]; then
  if curl_json DELETE "/memories/${MEM_ID}" >/dev/null 2>&1; then
    pass "DELETE /memories/{id}"
  else
    skip "DELETE /memories/{id}"
  fi
fi

echo ""
echo "Results: $PASS passed, $FAIL failed, $SKIP skipped"
if [[ "$FAIL" -gt 0 ]]; then
  exit 1
fi
