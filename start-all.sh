#!/bin/bash

echo "=== Starting Hystersis ==="

# Kill existing processes properly
pkill -9 -f "agent-backend" 2>/dev/null
pkill -9 -f "next dev" 2>/dev/null
killall -9 -f "next" 2>/dev/null
sleep 3

# Make sure port is free
if netstat -tuln 2>/dev/null | grep -q ":8080 "; then
    echo "Port 8080 still in use, waiting..."
    sleep 2
fi

# Export API key
export LLM_API_KEY="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"

cd /home/ubuntu/agent-memory
echo "Building backend..."
/usr/local/go/bin/go build -o /tmp/agent-backend ./cmd/server

echo "Starting backend..."
nohup /tmp/agent-backend > /tmp/backend.log 2>&1 &

# Start frontend
cd /home/ubuntu/agent-memory/dashboard
echo "Starting frontend..."
npm run dev > /tmp/frontend.log 2>&1 &

sleep 5

echo ""
echo "=== Services Started ==="
echo "Backend:  http://localhost:8080"
echo "Frontend: http://localhost:3000"
echo "Playground: https://hysteris.ai/playground"
echo ""
echo "Check server with: curl http://localhost:8080/compression/stats"