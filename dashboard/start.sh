#!/bin/bash

# Dashboard startup script with proper environment
cd /home/ubuntu/agent-memory/dashboard

# Ensure environment variables are loaded
export NEXTAUTH_URL=https://app.hystersis.com
export NEXTAUTH_SECRET=your-nextauth-secret-here
export NEXT_PUBLIC_API_URL=http://localhost:8080
export ADMIN_API_KEY=admin-1234567890123456789012345678901234567890123456789012345678901234
export NEXT_PUBLIC_AMPLITUDE_API_KEY=5a684520b5dcd448c4fd3874a8a9b663

# Start dashboard
npm run dev -- --hostname 0.0.0.0