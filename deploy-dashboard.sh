#!/bin/bash

# Deploy fresh build of dashboard to production
# Run this on production server

set -e

echo "=== Hystersis Dashboard Deployment ==="
echo "Date: $(date)"
echo ""

# Navigate to dashboard directory
cd /home/ubuntu/agent-memory/dashboard

# Build fresh
echo "[1/3] Building fresh Next.js application..."
rm -rf .next
npm run build

# Restart via PM2 (picks up new .next build automatically)
echo ""
echo "[2/3] Restarting dashboard via PM2..."
if pm2 list | grep -q dashboard; then
  pm2 restart dashboard --update-env
  echo "✓ Restarted existing PM2 process"
else
  pm2 start npm --name "dashboard" -- start
  echo "✓ Created new PM2 process"
fi

# Verify server is running
echo ""
echo "[3/3] Verifying deployment..."
sleep 8

if curl -s http://localhost:3000/demo > /dev/null; then
    echo "✓ Demo page is accessible at http://localhost:3000/demo"
else
    echo "✗ Demo page is not accessible"
    echo "Check logs: pm2 logs dashboard"
fi
echo ""
echo "=== Deployment Complete ==="
echo "Dashboard: https://dashboard.hystersis.ai"
echo "Demo:      https://dashboard.hystersis.ai/demo"
echo "Logs:      pm2 logs dashboard"
pm2 status dashboard
