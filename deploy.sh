#!/bin/bash
set -e

cd /home/ubuntu/agent-memory

echo "Pulling latest code..."
git fetch origin main
git pull origin main --no-rebase

echo "Building landing page..."
cd landing
npm install --legacy-peer-deps
npm run build

echo "Deploying to web root..."
sudo rm -rf /var/www/hystersis/*
sudo cp -r dist/* /var/www/hystersis/

echo "Reloading nginx..."
sudo nginx -s reload

echo "Deployment complete!"
