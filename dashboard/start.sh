#!/bin/bash

# Dashboard startup script with proper environment
cd /home/ubuntu/agent-memory/dashboard

# Ensure environment variables are loaded
export BETTER_AUTH_URL=https://app.hystersis.com
export BETTER_AUTH_SECRET=your-better-auth-secret-here
export BETTER_AUTH_API_KEY=your-better-auth-api-key-here
export NEXT_PUBLIC_API_URL=http://localhost:8080
export ADMIN_API_KEY=admin-1234567890123456789012345678901234567890123456789012345678901234
export NEXT_PUBLIC_AMPLITUDE_API_KEY=5a684520b5dcd448c4fd3874a8a9b663

# Start dashboard
npm run dev -- --hostname 0.0.0.0