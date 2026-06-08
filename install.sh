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
SOURCE_DIR=""
SOURCE_TMP=""

INSTALL_PYTHON=true
INSTALL_NODE=true
INSTALL_DOCKER=true

for arg in "$@"; do
  case "$arg" in
    --minimal)    INSTALL_PYTHON=false; INSTALL_NODE=false; INSTALL_DOCKER=false ;;
    --cli-only)   INSTALL_PYTHON=false; INSTALL_NODE=false ;;
    --no-docker)  INSTALL_DOCKER=false ;;
    --no-python)  INSTALL_PYTHON=false ;;
    --no-node)    INSTALL_NODE=false ;;
  esac
done

GREEN='\033[0;32m'; YELLOW='\033[1;33m'; CYAN='\033[0;36m'; BOLD='\033[1m'; NC='\033[0m'
info()  { echo -e "  ${GREEN}ok${NC} $1"; }
warn()  { echo -e "  ${YELLOW}..${NC} $1"; }
step()  { echo -e "\n  ${CYAN}> $1${NC}"; }

cleanup() {
  if [ -n "$SOURCE_TMP" ] && [ -d "$SOURCE_TMP" ]; then
    rm -rf "$SOURCE_TMP"
  fi
}
trap cleanup EXIT

ensure_source_checkout() {
  if [ -n "$SOURCE_DIR" ] && [ -d "$SOURCE_DIR" ]; then
    return 0
  fi
  if ! command -v git >/dev/null 2>&1; then
    return 1
  fi
  SOURCE_TMP=$(mktemp -d)
  SOURCE_DIR="$SOURCE_TMP/agent-memory"
  git clone --depth 1 "$REPO_URL" "$SOURCE_DIR" >/dev/null 2>&1
}

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
  if ensure_source_checkout; then
    (cd "$SOURCE_DIR" && CGO_ENABLED=0 go build -o "$CLI_BIN" ./cmd/cli)
    (cd "$SOURCE_DIR" && CGO_ENABLED=0 go build -o "$BIN_DIR/hystersis-server" ./cmd/server)
    (cd "$SOURCE_DIR" && CGO_ENABLED=0 go build -o "$BIN_DIR/hystersis-agent" ./cmd/agent)
    if [ -f "$CLI_BIN" ]; then
      info "CLI binary: $CLI_BIN"
      info "Server binary: $BIN_DIR/hystersis-server"
      info "Agent REPL: $BIN_DIR/hystersis-agent"
      BUILT=true
    fi
  else
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
  PYTHON_BIN=""
  if command -v python3 >/dev/null 2>&1; then
    PYTHON_BIN="$(command -v python3)"
  elif command -v python >/dev/null 2>&1; then
    PYTHON_BIN="$(command -v python)"
  fi

  if [ -n "$PYTHON_BIN" ]; then
    if "$PYTHON_BIN" -m pip install --user hystersis --quiet >/dev/null 2>&1; then
      info "Python SDK installed: $PYTHON_BIN -m pip install --user hystersis"
    else
      PY_SDK_VENV="$INSTALL_DIR/python-sdk"
      if "$PYTHON_BIN" -m venv "$PY_SDK_VENV" >/dev/null 2>&1 && "$PY_SDK_VENV/bin/python" -m pip install --upgrade pip --quiet >/dev/null 2>&1 && "$PY_SDK_VENV/bin/python" -m pip install hystersis --quiet >/dev/null 2>&1; then
        info "Python SDK installed in venv: $PY_SDK_VENV"
        info "Use it with: source $PY_SDK_VENV/bin/activate"
      else
        warn "Python SDK install failed. Try: python3 -m venv .venv && .venv/bin/pip install hystersis"
      fi
    fi
  else
    warn "Python not found. Install from: https://python.org"
  fi
fi

# ── Node.js SDK and Skills CLI ──────────────────────────────────────────────────
if [ "$INSTALL_NODE" = true ]; then
  step "Installing Node.js SDK and Skills CLI..."
  if command -v npm >/dev/null 2>&1; then
    if npm install -g hystersis --quiet >/dev/null 2>&1; then
      info "Node.js SDK installed: npm install -g hystersis"
    elif ensure_source_checkout && (cd "$SOURCE_DIR/sdk/nodejs" && npm install --quiet >/dev/null 2>&1 && npm run build --silent >/dev/null 2>&1 && PKG_TGZ=$(npm pack --silent) && npm install -g "$PWD/$PKG_TGZ" --quiet >/dev/null 2>&1); then
      info "Node.js SDK installed from source checkout"
    else
      warn "Node.js SDK install failed. Try: git clone $REPO_URL && cd agent-memory/sdk/nodejs && npm install && npm run build"
    fi
    if npm install -g @hystersis/skills --quiet >/dev/null 2>&1; then
      info "Skills CLI installed: npm install -g @hystersis/skills"
    elif ensure_source_checkout && npm install -g "$SOURCE_DIR/skills-npm" --quiet >/dev/null 2>&1; then
      info "Skills CLI installed from source checkout"
    else
      warn "Skills CLI install failed. Try: git clone $REPO_URL && npm install -g agent-memory/skills-npm"
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

BOOTSTRAP_API_KEY=""
if [ -f "$INSTALL_DIR/.env" ]; then
  BOOTSTRAP_API_KEY=$(grep '^ADMIN_API_KEY=' "$INSTALL_DIR/.env" | tail -n 1 | cut -d= -f2-)
fi
if [ -n "$BOOTSTRAP_API_KEY" ]; then
  cat > "$HOME/.agent-memory.json" << CONFIG
{
  "base_url": "http://localhost:8080",
  "api_key": "$BOOTSTRAP_API_KEY"
}
CONFIG
  info "Created CLI config: $HOME/.agent-memory.json"
else
  warn "No local API key found; run hystersis init --api-key <key> after starting the server"
fi

add_shell_profile() {
  PROFILE="$1"
  if [ -f "$PROFILE" ] && ! grep -q "HYSTERESIS_HOME" "$PROFILE" 2>/dev/null; then
    cat >> "$PROFILE" << BASHRC

# HysterSIS - https://hystersis.com
export HYSTERESIS_HOME="$INSTALL_DIR"
export PATH="\$PATH:$BIN_DIR"
BASHRC
    info "Added HysterSIS to $PROFILE"
  fi
}
add_shell_profile "$HOME/.zshrc"
add_shell_profile "$HOME/.bashrc"

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
if [ "$INSTALL_DOCKER" = true ]; then
echo "    1. Start databases:"
echo "       docker compose -f $INSTALL_DIR/docker-compose.yml up -d"
echo ""
echo "    2. Start the API server:"
echo "       source $INSTALL_DIR/.env"
echo "       hystersis-server"
echo "       hystersis health"
echo ""
echo "    3. Use the CLI:"
echo "       hystersis memories add --agent-id default --content 'Your first memory'"
else
echo "    1. Point the CLI at your API:"
echo "       hystersis init --url https://api.hystersis.com --api-key <your-key>"
echo ""
echo "    2. Check connectivity:"
echo "       hystersis health"
echo ""
echo "    3. Use the CLI:"
echo "       hystersis memories add --agent-id default --content 'Your first memory'"
fi
echo ""
echo "  Docs:  https://hystersis.com/docs"
echo "  Repo:  $REPO_URL"
echo ""
