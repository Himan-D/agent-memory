#!/bin/bash
# Agent Memory System - Quick Install
# Usage: curl -fsSL https://raw.githubusercontent.com/agent-memory/agent-memory/main/scripts/install.sh | bash
#
# Or with custom options:
#   VERSION=v0.1.0 INSTALL_DIR=$HOME/.myapp curl -fsSL https://... | bash

set -e

VERSION="${VERSION:-latest}"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.agent-memory}"

# Quick install to /usr/local/bin if possible, else $HOME/bin
if [ -w /usr/local/bin ]; then
    TARGET_DIR="/usr/local/bin"
else
    TARGET_DIR="$HOME/bin"
fi

# Check for docker
if ! command -v docker &> /dev/null; then
    echo "Error: Docker is required but not installed."
    echo "Install from: https://docs.docker.com/get-docker"
    exit 1
fi

# Create install dir
mkdir -p "$INSTALL_DIR"
mkdir -p "$INSTALL_DIR/docker"

# Download docker-compose
cat > "$INSTALL_DIR/docker/docker-compose.yml" << 'EOF'
version: "3.9"
services:
  neo4j:
    image: neo4j:5.23-community
    ports:
      - "7474:7474"
      - "7687:7687"
    environment:
      NEO4J_AUTH: neo4j/password
    volumes:
      - neo4j_data:/data
  qdrant:
    image: qdrant/qdrant:v1.7.4
    ports:
      - "6333:6333"
      - "6334:6334"
    volumes:
      - qdrant_data:/qdrant/storage
volumes:
  neo4j_data:
  qdrant_data:
EOF

# Create config
cat > "$INSTALL_DIR/.env" << 'EOF'
NEO4J_URI=bolt://localhost:7687
NEO4J_USER=neo4j
NEO4J_PASSWORD=password
QDRANT_URL=localhost:6334
HTTP_PORT=:8080
EOF

# Check if Go is available to build
if command -v go &> /dev/null; then
    echo "Building from source..."
    cd "$INSTALL_DIR"
    git clone --depth 1 https://github.com/agent-memory/agent-memory /tmp/agent-memory 2>/dev/null || true
    cd /tmp/agent-memory
    CGO_ENABLED=0 go build -o "$TARGET_DIR/agent-memory" ./cmd/server 2>/dev/null || {
        echo "Build failed. Please install manually."
        exit 1
    }
    echo "Built: $TARGET_DIR/agent-memory"
else
    # Just download compose file
    echo "Go not found. Docker-only mode."
    echo "You can run: cd $INSTALL_DIR/docker && docker compose up -d"
fi

echo ""
echo "============================================"
echo "Agent Memory Installed!"
echo "============================================"
echo ""
echo "1. Start databases:"
echo "   cd $INSTALL_DIR/docker && docker compose up -d"
echo ""
echo "2. Start agent-memory:"
echo "   cd $INSTALL_DIR && $TARGET_DIR/agent-memory"
echo ""
echo "3. API available at: http://localhost:8080"
echo ""
