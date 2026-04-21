# Agent Memory System
# 
# One-line install:
#   curl -fsSL https://agentmemory.io/install | bash
#
# Or specify version:
#   VERSION=v0.1.0 curl -fsSL https://agentmemory.io/install | bash

VERSION="${VERSION:-latest}"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.agent-memory}"

set -e

echo "Installing Agent Memory System v$VERSION..."

# Check Docker
if ! command -v docker &> /dev/null; then
    echo "Error: Docker is required"
    exit 1
fi

# Create directories
mkdir -p "$INSTALL_DIR/docker"

# Write docker-compose
cat > "$INSTALL_DIR/docker/docker-compose.yml" << 'COMPOSE'
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
COMPOSE

# Write .env
cat > "$INSTALL_DIR/.env" << 'ENV'
NEO4J_URI=bolt://localhost:7687
NEO4J_USER=neo4j
NEO4J_PASSWORD=password
QDRANT_URL=localhost:6334
HTTP_PORT=:8080
ENV

# Build or download binary
BIN_DIR="$HOME/bin"
[ -w /usr/local/bin ] && BIN_DIR="/usr/local/bin"

if command -v go &> /dev/null; then
    echo "Building agent-memory..."
    cd /tmp
    rm -rf agent-memory 2>/dev/null || true
    git clone --depth 1 https://github.com/agent-memory/agent-memory 2>/dev/null
    cd agent-memory
    CGO_ENABLED=0 go build -o "$BIN_DIR/agent-memory" ./cmd/server
else
    echo "Go not found. Using docker-compose only."
fi

echo ""
echo "============================================"
echo "Installed to: $INSTALL_DIR"
echo "============================================"
echo ""
echo "Run these commands:"
echo ""
echo "  # Start databases"
echo "  cd $INSTALL_DIR/docker && docker compose up -d"
echo ""
echo "  # Start agent-memory"
echo "  $BIN_DIR/agent-memory"
echo ""
echo "  # Or with custom config"
echo "  cd $INSTALL_DIR && source .env && $BIN_DIR/agent-memory"
echo ""
