#!/bin/bash
# Clean start for Hystersis - FIXED

WORK_DIR="/home/ubuntu/agent-memory"

# Load env vars from .env file
if [ -f "$WORK_DIR/.env" ]; then
  set -a
  source "$WORK_DIR/.env"
  set +a
else
  echo "ERROR: .env file not found at $WORK_DIR/.env"
  echo "Create .env with LLM_API_KEY and OPENAI_API_KEY"
  exit 1
fi

if [ -z "$LLM_API_KEY" ]; then
  echo "ERROR: LLM_API_KEY is required. Set it in .env"
  exit 1
fi

# Kill everything
pkill -9 -f "agent-backend" 2>/dev/null || true
pkill -9 -f "next" 2>/dev/null || true
sleep 2

if ss -tlnp 2>/dev/null | grep -q ":8080 "; then
    fuser -k 8080/tcp 2>/dev/null || true
    sleep 2
fi

echo "Building backend..."
cd "$WORK_DIR"
/usr/local/go/bin/go build -o /tmp/agent-backend ./cmd/server

# Start backend using env vars from .env
echo "Starting backend..."
/tmp/agent-backend > /tmp/backend.log 2>&1 &

cd "$WORK_DIR/dashboard"
npm run dev > /tmp/frontend.log 2>&1 &

sleep 5

echo "Testing..."
curl -s http://localhost:8080/compression/stats | head -c 50 || echo "Failed"

echo "Done!"