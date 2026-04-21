#!/bin/bash
# Clean start for Hystersis - FIXED

WORK_DIR="/home/ubuntu/agent-memory"

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

# Start with explicit env using env command
echo "Starting backend..."
env LLM_API_KEY="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" OPENAI_API_KEY="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" /tmp/agent-backend > /tmp/backend.log 2>&1 &

cd "$WORK_DIR/dashboard"
npm run dev > /tmp/frontend.log 2>&1 &

sleep 5

echo "Testing..."
curl -s http://localhost:8080/compression/stats | head -c 50 || echo "Failed"

echo "Done!"