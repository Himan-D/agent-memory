#!/usr/bin/env bash
# Track A smoke: unit tests + MCP stdio + optional live API + CLI doctor.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

PASS=0
FAIL=0
ok()   { echo "  ok  $*"; PASS=$((PASS+1)); }
fail() { echo "  FAIL $*"; FAIL=$((FAIL+1)); }

echo "== Track A smoke =="

# ── Build ─────────────────────────────────────────────────────────────────────
echo "> Build binaries"
if go build -o /tmp/hystersis-cli ./cmd/cli && \
   go build -o /tmp/hystersis-mcp ./cmd/mcp-server; then
  ok "go build cli + mcp"
else
  fail "go build"
  exit 1
fi

# ── MCP stdio ─────────────────────────────────────────────────────────────────
echo "> MCP stdio initialize + tools/list"
OUT=$(printf '%s\n' \
  '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"smoke","version":"0"}}}' \
  '{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}' \
  | /tmp/hystersis-mcp --stdio --memory-api http://127.0.0.1:9 --api-key smoke 2>/dev/null || true)
if echo "$OUT" | grep -q '"name":"hystersis-memory"' && echo "$OUT" | grep -q '"name":"add_memory"'; then
  ok "MCP stdio tools/list"
else
  fail "MCP stdio tools/list"
fi

# ── CLI mcp print ─────────────────────────────────────────────────────────────
echo "> CLI mcp print"
if /tmp/hystersis-cli mcp print --command /tmp/hystersis-mcp 2>/dev/null | grep -q mcpServers; then
  ok "mcp print"
else
  fail "mcp print"
fi

# ── Python unit tests ─────────────────────────────────────────────────────────
echo "> Python SDK unit tests"
if command -v python3 >/dev/null 2>&1; then
  VENV="${TMPDIR:-/tmp}/hys-sdk-smoke-venv"
  if [ ! -x "$VENV/bin/pytest" ]; then
    python3 -m venv "$VENV"
    "$VENV/bin/pip" install -q -e "sdk/python[dev]" >/dev/null
  fi
  if (cd sdk/python && "$VENV/bin/pytest" tests/test_hystersis.py -q); then
    ok "python unit tests"
  else
    fail "python unit tests"
  fi
else
  fail "python3 missing"
fi

# ── Live API (real or ephemeral mock) ─────────────────────────────────────────
echo "> Live SDK smoke"
LIVE_URL="${HYSTERSIS_API_URL:-}"
LIVE_KEY="${HYSTERSIS_API_KEY:-}"
MOCK_PID=""

cleanup_mock() {
  if [ -n "${MOCK_PID:-}" ] && kill -0 "$MOCK_PID" 2>/dev/null; then
    kill "$MOCK_PID" 2>/dev/null || true
  fi
}
trap cleanup_mock EXIT

use_mock=false
if [ -n "$LIVE_URL" ] && [ -n "$LIVE_KEY" ]; then
  # Probe real API — must return JSON health
  if curl -fsS --max-time 3 "$LIVE_URL/health" 2>/dev/null | grep -q '{'; then
    ok "using real API $LIVE_URL"
  else
    echo "  .. real API health not JSON; starting mock"
    use_mock=true
  fi
else
  use_mock=true
fi

if [ "$use_mock" = true ]; then
  MOCK_PORT=18765
  LIVE_URL="http://127.0.0.1:$MOCK_PORT"
  LIVE_KEY="smoke-key"
  python3 - <<'PY' &
import json, uuid
from http.server import BaseHTTPRequestHandler, HTTPServer

class H(BaseHTTPRequestHandler):
    def log_message(self, *a): pass
    def _json(self, code, obj):
        b = json.dumps(obj).encode()
        self.send_response(code)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(b)))
        self.end_headers()
        self.wfile.write(b)
    def do_GET(self):
        path = self.path.split("?", 1)[0]
        if path.startswith("/health"):
            return self._json(200, {"status": "ok"})
        if path.startswith("/search"):
            return self._json(200, [{"score": 0.9, "text": "smoke"}])
        if path.startswith("/memories"):
            return self._json(200, {"memories": []})
        if path.startswith("/sessions"):
            return self._json(200, {"messages": []})
        self._json(404, {"error": "not found", "path": path})
    def do_POST(self):
        n = int(self.headers.get("Content-Length") or 0)
        _ = self.rfile.read(n) if n else b""
        path = self.path.split("?", 1)[0]
        if path.startswith("/memories"):
            return self._json(200, {"id": str(uuid.uuid4()), "content": "ok"})
        if path.startswith("/search"):
            return self._json(200, [{"score": 0.9, "text": "smoke"}])
        if path.startswith("/sessions"):
            return self._json(200, {"id": str(uuid.uuid4()), "agent_id": "smoke-agent"})
        if "/messages" in path:
            return self._json(200, {"status": "ok"})
        self._json(404, {"error": "not found", "path": path})

HTTPServer(("127.0.0.1", 18765), H).serve_forever()
PY
  MOCK_PID=$!
  sleep 0.4
  ok "mock API on $LIVE_URL"
fi

export HYSTERSIS_API_URL="$LIVE_URL"
export HYSTERSIS_API_KEY="$LIVE_KEY"
VENV="${TMPDIR:-/tmp}/hys-sdk-smoke-venv"
if (cd sdk/python && "$VENV/bin/pytest" tests/test_smoke_live.py -m live -o addopts= -q); then
  ok "python live smoke"
else
  fail "python live smoke"
fi

# ── Doctor with fixed health semantics ────────────────────────────────────────
echo "> MCP doctor"
# Global flags must precede the subcommand for urfave/cli
/tmp/hystersis-cli --url "$LIVE_URL" --api-key "$LIVE_KEY" mcp doctor 2>&1 | sed 's/^/  /' || true
ok "mcp doctor ran"

echo ""
echo "== Results: $PASS passed, $FAIL failed =="
if [ "$FAIL" -gt 0 ]; then
  exit 1
fi
exit 0
