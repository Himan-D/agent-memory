#!/bin/bash

cd /home/ubuntu/agent-memory/dashboard

echo "Starting frontend (Next.js) on port 3000..."
npm run dev &

echo "Frontend started on port 3000"