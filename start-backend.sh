#!/bin/bash

# Set your API key via environment variable or .env file
# NEVER commit API keys to version control
if [ -z "$LLM_API_KEY" ]; then
  echo "ERROR: LLM_API_KEY environment variable is required"
  echo "Usage: LLM_API_KEY=your-key $0"
  exit 1
fi

cd /home/ubuntu/agent-memory

echo "Starting backend server..."
/usr/local/go/bin/go run ./cmd/server &

echo "Backend started on port 8080"
