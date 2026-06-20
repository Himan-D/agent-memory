#!/bin/bash
# Hystersis — One-line install
#   curl -fsSL https://hystersis.com/install.sh | bash
#   curl -fsSL https://hystersis.com/install | bash
#
# Installs: CLI binary, Python SDK, Node.js SDK, Skills CLI, Docker services
#
# Options:
#   --minimal      CLI binary only (no SDKs, no Docker)
#   --cli-only     CLI binary + Docker services (no SDKs)
#   --no-docker    CLI binary + SDKs (no Docker services)
#   --no-python    Skip Python SDK
#   --no-node      Skip Node.js SDK & Skills CLI

set -e

VERSION="${VERSION:-latest}"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.hystersis}"
BIN_DIR="${BIN_DIR:-$HOME/.local/bin}"
REPO_URL="https://github.com/Himan-D/agent-memory"

INSTALL_PYTHON=true
INSTALL_NODE=true
INSTALL_DOCKER=true

for arg in "$@"; do
  case "$arg" in
    --minimal)    INSTALL_PYTHON=false; INSTALL_NODE=false ;;
    --cli-only)   INSTALL_PYTHON=false; INSTALL_NODE=false; INSTALL_DOCKER=false ;;
    --no-docker)  INSTALL_DOCKER=false ;;
    --no-python)  INSTALL_PYTHON=false ;;
    --no-node)    INSTALL_NODE=false ;;
  esac
done

GREEN='\033[0;32m'; YELLOW='\033[1;33m'; CYAN='\033[0;36m'; BOLD='\033[1m'; NC='\033[0m'
info()  { echo -e "  ${GREEN}ok${NC} $1"; }
warn()  { echo -e "  ${YELLOW}..${NC} $1"; }
step()  { echo -e "\n  ${CYAN}> $1${NC}"; }

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
  x86_64|amd64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
esac

echo ""
echo "  Hystersis"
echo "  Memory that adapts. Intelligence that compounds."
echo "  https://hystersis.com"
echo ""

mkdir -p "$INSTALL_DIR" "$BIN_DIR"
export PATH="$BIN_DIR:$PATH"

# ── CLI Binary ──────────────────────────────────────────────────────────────────
step "Installing CLI..."

CLI_BIN="$BIN_DIR/hystersis"
DOWNLOAD_URL="$REPO_URL/releases/$VERSION/download/hystersis-${OS}-${ARCH}.tar.gz"
BUILT=false

if command -v go >/dev/null 2>&1 && [ "$VERSION" = "latest" ]; then
  info "Building from source..."
  TMPDIR=$(mktemp -d)
  BUILDDIR="$TMPDIR/build"
  mkdir -p "$BUILDDIR"
  if git clone --depth 1 "$REPO_URL" "$BUILDDIR" >/dev/null 2>&1; then
    (cd "$BUILDDIR" && CGO_ENABLED=0 go build -o "$CLI_BIN" ./cmd/cli)
    (cd "$BUILDDIR" && CGO_ENABLED=0 go build -o "$BIN_DIR/hystersis-server" ./cmd/server)
    (cd "$BUILDDIR" && CGO_ENABLED=0 go build -o "$BIN_DIR/hystersis-agent" ./cmd/agent)
    rm -rf "$TMPDIR"
    if [ -f "$CLI_BIN" ]; then
      info "CLI binary: $CLI_BIN"
      info "Server binary: $BIN_DIR/hystersis-server"
      info "Agent REPL: $BIN_DIR/hystersis-agent"
      BUILT=true
    fi
  else
    rm -rf "$TMPDIR"
    warn "Could not clone repository."
  fi
fi

if [ "$BUILT" = false ]; then
  if curl -fsSL "$DOWNLOAD_URL" -o /tmp/hystersis.tar.gz >/dev/null 2>&1; then
    tar -xzf /tmp/hystersis.tar.gz -C "$BIN_DIR"
    rm -f /tmp/hystersis.tar.gz
    info "Downloaded CLI to $BIN_DIR/hystersis"
  else
    warn "No pre-built binary for ${OS}-${ARCH}. Install Go: https://go.dev/dl"
  fi
fi

# ── Docker Services ──────────────────────────────────────────────────────────────
if [ "$INSTALL_DOCKER" = true ]; then
  if command -v docker >/dev/null 2>&1; then
    step "Setting up Docker services..."
    mkdir -p "$INSTALL_DIR"

    if [ ! -f "$INSTALL_DIR/docker-compose.yml" ]; then
      cat > "$INSTALL_DIR/docker-compose.yml" << 'DOCKER'
services:
  neo4j:
    image: neo4j:5.23-community
    ports:
      - "7474:7474"
      - "7687:7687"
    environment:
      NEO4J_AUTH: neo4j/password
      NEO4J_PLUGINS: '["apoc"]'
    volumes:
      - neo4j_data:/data
    healthcheck:
      test: ["CMD-SHELL", "cypher-shell -u neo4j -p password 'RETURN 1'"]
      interval: 10s
      timeout: 5s
      retries: 5
  qdrant:
    image: qdrant/qdrant:v1.7.4
    ports:
      - "6333:6333"
      - "6334:6334"
    volumes:
      - qdrant_data:/qdrant/storage
  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 5s
      timeout: 3s
      retries: 5
volumes:
  neo4j_data:
  qdrant_data:
DOCKER
      info "Created $INSTALL_DIR/docker-compose.yml"
    fi

    cat > "$INSTALL_DIR/.env" << 'ENVFILE'
NEO4J_URI=bolt://localhost:7687
NEO4J_USER=neo4j
NEO4J_PASSWORD=password
QDRANT_URL=http://localhost:6333
REDIS_URL=redis://localhost:6379
HTTP_PORT=:8080
API_BASE_URL=https://api.hystersis.com
ENVFILE
    info "Created $INSTALL_DIR/.env"

    # Generate bootstrap credentials (salt, admin/user API keys, JWT secret)
    if command -v go >/dev/null 2>&1 && [ -f "./scripts/generate-tokens.sh" ]; then
      bash ./scripts/generate-tokens.sh --write "$INSTALL_DIR/.env" >/dev/null 2>&1 || true
    elif command -v openssl >/dev/null 2>&1; then
      rand_hex() { openssl rand -hex "$1"; }
      SALT=$(rand_hex 32)
      ADMIN_SHA=$(openssl rand 32 | openssl dgst -sha256 | awk '{print $2}')
      USER_SHA=$(openssl rand 32 | openssl dgst -sha256 | awk '{print $2}')
      JWT=$(rand_hex 32)
      cat >> "$INSTALL_DIR/.env" << TOKENS

# Bootstrap credentials — generated $(date -u +%Y-%m-%dT%H:%M:%SZ)
API_KEY_SALT=${SALT}
ADMIN_API_KEYS=am_admin_${ADMIN_SHA}
ADMIN_API_KEY=am_admin_${ADMIN_SHA}
API_KEYS=usr_${USER_SHA}:default
JWT_SECRET=${JWT}
TOKENS
    fi
    info "Generated API credentials in $INSTALL_DIR/.env"
  else
    warn "Docker not found. Install from: https://docker.com"
  fi
fi

# ── Python SDK ───────────────────────────────────────────────────────────────────
if [ "$INSTALL_PYTHON" = true ]; then
  step "Installing Python SDK..."
  if command -v pip3 >/dev/null 2>&1; then
    if pip3 install hystersis --quiet >/dev/null 2>&1; then
      info "Python SDK installed: pip install hystersis"
    else
      warn "Python SDK install failed. Try: pip install hystersis"
    fi
  elif command -v pip >/dev/null 2>&1; then
    if pip install hystersis --quiet >/dev/null 2>&1; then
      info "Python SDK installed: pip install hystersis"
    else
      warn "Python SDK install failed. Try: pip install hystersis"
    fi
  else
    warn "Python/pip not found. Install from: https://python.org"
  fi
fi

# ── Node.js SDK and Skills CLI ──────────────────────────────────────────────────
if [ "$INSTALL_NODE" = true ]; then
  step "Installing Node.js SDK and Skills CLI..."
  if command -v npm >/dev/null 2>&1; then
    if npm install -g @hystersis/sdk --quiet >/dev/null 2>&1; then
      info "Node.js SDK installed: npm install -g @hystersis/sdk"
    else
      warn "Node.js SDK install failed. Try: npm install -g @hystersis/sdk"
    fi
    if npm install -g @hystersis/skills --quiet >/dev/null 2>&1; then
      info "Skills CLI installed: npm install -g @hystersis/skills"
    else
      warn "Skills CLI install failed. Try: npm install -g @hystersis/skills"
    fi
  else
    warn "npm not found. Install from: https://nodejs.org"
  fi
fi

# ── Config ───────────────────────────────────────────────────────────────────────
step "Creating config..."
cat > "$INSTALL_DIR/config.json" << 'CONFIG'
{
  "api_base": "http://localhost:8080",
  "neo4j_uri": "bolt://localhost:7687",
  "neo4j_user": "neo4j",
  "neo4j_password": "password",
  "qdrant_url": "http://localhost:6333",
  "redis_url": "redis://localhost:6379",
  "tier_policy": "balanced",
  "compression_mode": "extract"
}
CONFIG

if [ -f "$HOME/.bashrc" ]; then
  if ! grep -q "HYSTERESIS_HOME" "$HOME/.bashrc" 2>/dev/null; then
    cat >> "$HOME/.bashrc" << 'BASHRC'

# HysterSIS - https://hystersis.com
export HYSTERESIS_HOME="$INSTALL_DIR"
export PATH="$PATH:$BIN_DIR"
BASHRC
    info "Added HysterSIS to ~/.bashrc"
  fi
fi

# ── Summary ──────────────────────────────────────────────────────────────────────
echo ""
echo "  ----------------------------------------"
echo "  Installation Complete!"
echo "  ----------------------------------------"
echo ""
echo "  Installed to: $INSTALL_DIR"
echo "  Binaries:     $BIN_DIR"
echo ""
echo "  Commands:"
echo "    hystersis           CLI - manage your memory"
echo "    hystersis-server    API server"
echo "    hystersis-agent     Interactive agent REPL"
echo "    skills              Skills CLI"
echo ""
echo "  Quick start:"
echo "    1. Start databases:"
echo "       docker compose -f $INSTALL_DIR/docker-compose.yml up -d"
echo ""
echo "    2. Start the API server:"
echo "       hystersis-server"
echo ""
echo "    3. Use the CLI:"
echo "       hystersis init"
echo "       hystersis remember 'Your first memory'"
echo ""
echo "  Docs:  https://hystersis.com/docs"
echo "  Repo:  $REPO_URL"
echo ""
