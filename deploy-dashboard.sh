#!/bin/bash

# Deploy fresh build of dashboard to production
# Run this on production server

set -e

echo "=== Hystersis Dashboard Deployment ==="
echo "Date: $(date)"
echo ""

# Navigate to dashboard directory
cd /home/ubuntu/agent-memory/dashboard

# Stop current server
echo "Stopping current server..."
pkill -f "npm run start" || true
pkill -f "next start" || true
pkill -f "node.*next" || true
sleep 3

# Clear stale build cache
echo "Clearing .next cache..."
rm -rf .next

# Install dependencies (if needed)
echo "Installing dependencies..."
npm install --silent

# Build fresh
echo "Building fresh Next.js application..."
npm run build

# Start production server
echo "Starting production server..."
nohup npm run start > /tmp/dashboard.log 2>&1 &
echo "Server started with PID: $!"
echo ""
echo "=== Deployment Complete ==="
echo "Waiting for server to start..."
sleep 10

# Verify server is running
if curl -s http://localhost:3000/demo > /dev/null; then
    echo "✓ Demo page is accessible at http://localhost:3000/demo"
else
    echo "✗ Demo page is not accessible"
    echo "Check logs: tail -f /tmp/dashboard.log"
fi
echo ""
echo "=== Deployment Notes ==="
echo "Demo page: http://localhost:3000/demo (no auth required)"
echo "Dashboard: http://localhost:3000/ (requires authentication)"
echo "Logs: tail -f /tmp/dashboard.log"
