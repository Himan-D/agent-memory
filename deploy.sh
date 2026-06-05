#!/bin/bash
set -e

cd /home/ubuntu/agent-memory

echo "=========================================="
echo "  Hysterisis Full Deployment"
echo "=========================================="
echo ""

# Pull latest code
echo "[1/5] Pulling latest code..."
git fetch origin main
git pull origin main --no-rebase

# Build landing page
echo ""
echo "[2/5] Building landing page..."
cd landing
npm install --legacy-peer-deps
npm run build

# Deploy landing page
echo ""
echo "[3/5] Deploying landing page..."
sudo rm -rf /var/www/hystersis/*
sudo cp -r dist/* /var/www/hystersis/

# Deploy AI agent discovery files
echo ""
echo "  Deploying agent discovery files..."
sudo cp ../AGENTS.md /var/www/hystersis/AGENTS.md
sudo cp ../AGENTS.md /var/www/hystersis/agents.md
sudo cp ../api/llms.txt /var/www/hystersis/llms.txt

# Build dashboard
echo ""
echo "[4/5] Building dashboard..."
cd ../dashboard
rm -rf .next
npm run build

# Deploy dashboard
echo ""
echo "[5/5] Deploying dashboard..."
sudo mkdir -p /var/www/app.hystersis.com
sudo rm -rf /var/www/app.hystersis.com/*
sudo cp -r .next /var/www/app.hystersis.com/
sudo cp -r public /var/www/app.hystersis.com/

# Reload nginx
echo ""
echo "Reloading nginx..."
sudo nginx -s reload

echo ""
echo "=========================================="
echo "  Deployment Complete!"
echo "=========================================="
echo ""
echo "Landing page:  https://hystersis.com"
echo "Dashboard:     https://app.hystersis.com"
echo "API Server:    https://api.hystersis.ai"
echo ""