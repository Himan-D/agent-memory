#!/bin/bash

export LLM_API_KEY="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"

cd /home/ubuntu/agent-memory

echo "Starting backend server..."
/usr/local/go/bin/go run ./cmd/server &

echo "Backend started on port 8080"
