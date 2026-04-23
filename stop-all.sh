#!/bin/bash
# Stop all Hystersis services

echo "=== Stopping Hystersis ==="

# Kill all backend processes
pkill -9 -f "agent-backend" 2>/dev/null || true
pkill -9 -f "go run ./cmd/server" 2>/dev/null || true

# Kill all frontend processes  
pkill -9 -f "next" 2>/dev/null || true

# Wait for port to free
sleep 2

# Check ports
echo ""
echo "Port 8080: $(ss -tlnp | grep 8080 || echo 'free')"
echo "Port 3000: $(ss -tlnp | grep 3000 || echo 'free')"
echo ""
echo "=== Stopped ==="