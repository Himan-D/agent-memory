#!/bin/bash
set -e

VERSION="${VERSION:-latest}"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.agent-memory}"
BIN_DIR="${BIN_DIR:-$HOME/bin}"
REPO_URL="${REPO_URL:-https://github.com/agent-memory/agent-memory}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# Detect OS
detect_os() {
    if [[ "$OSTYPE" == "darwin"* ]]; then
        echo "darwin"
    elif [[ "$OSTYPE" == "linux"* ]]; then
        echo "linux"
    else
        echo "unknown"
    fi
}

# Detect architecture
detect_arch() {
    case $(uname -m) in
        x86_64) echo "amd64" ;;
        aarch64|arm64) echo "arm64" ;;
        *) echo "amd64" ;;
    esac
}

# Check dependencies
check_deps() {
    local missing=()
    
    if ! command -v docker &> /dev/null; then
        missing+=("docker")
    fi
    
    if [ ${#missing[@]} -gt 0 ]; then
        log_error "Missing dependencies: ${missing[*]}"
        echo "Please install: brew install ${missing[*]} (macOS) or apt-get install ${missing[*]} (Linux)"
        exit 1
    fi
}

# Download binary
download_binary() {
    local os=$1
    local arch=$2
    local version=$3
    
    log_info "Downloading agent-memory for $os-$arch..."
    
    # Try GitHub releases first
    local download_url="https://github.com/agent-memory/agent-memory/releases/${version}/download/agent-memory-${os}-${arch}.tar.gz"
    
    if curl -fsSL "$download_url" -o /tmp/agent-memory.tar.gz 2>/dev/null; then
        tar -xzf /tmp/agent-memory.tar.gz -C "$INSTALL_DIR"
        rm -f /tmp/agent-memory.tar.gz
        return 0
    fi
    
    return 1
}

# Build from source
build_from_source() {
    log_info "Building from source..."
    
    if ! command -v go &> /dev/null; then
        log_error "Go not found. Please install Go 1.26+"
        exit 1
    fi
    
    mkdir -p "$INSTALL_DIR"
    
    # Clone or use existing repo
    if [ -d "/tmp/agent-memory" ]; then
        rm -rf /tmp/agent-memory
    fi
    
    git clone --depth 1 "$REPO_URL" /tmp/agent-memory 2>/dev/null || {
        log_error "Failed to clone repo. Please install manually."
        exit 1
    }
    
    cd /tmp/agent-memory
    CGO_ENABLED=0 GOOS=$(detect_os) GOARCH=$(detect_arch) go build -o "$INSTALL_DIR/agent-memory" ./cmd/server
    
    log_info "Built successfully!"
}

# Setup docker compose
setup_docker() {
    log_info "Setting up Docker services..."
    
    mkdir -p "$INSTALL_DIR/docker"
    
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

    log_info "Docker compose file created at $INSTALL_DIR/docker/docker-compose.yml"
}

# Create config
create_config() {
    cat > "$INSTALL_DIR/.env" << EOF
# Agent Memory Configuration
# Copy this to your project or modify as needed

# Neo4j (required)
NEO4J_URI=bolt://localhost:7687
NEO4J_USER=neo4j
NEO4J_PASSWORD=password

# Qdrant (required)
QDRANT_URL=localhost:6334

# OpenAI (optional - for embeddings)
# OPENAI_API_KEY=sk-...

# Server
HTTP_PORT=:8080

# Authentication (optional)
# AUTH_ENABLED=true
# API_KEYS=your-key:tenant
EOF

    log_info "Config file created at $INSTALL_DIR/.env"
}

# Create systemd service
create_service() {
    if command -v systemctl &> /dev/null && [ "$1" = "systemd" ]; then
        cat > "$HOME/.config/systemd/user/agent-memory.service" << EOF
[Unit]
Description=Agent Memory System
After=network.target

[Service]
Type=simple
WorkingDirectory=$INSTALL_DIR
ExecStart=$INSTALL_DIR/agent-memory
EnvironmentFile=$INSTALL_DIR/.env
Restart=on-failure

[Install]
WantedBy=default.target
EOF
        
        systemctl --user daemon-reload
        systemctl --user enable agent-memory
        
        log_info "Systemd service created. Run: systemctl --user start agent-memory"
    fi
}

# Main installation
main() {
    echo ""
    echo "============================================"
    echo "  Agent Memory System Installer"
    echo "============================================"
    echo ""
    
    local os=$(detect_os)
    local arch=$(detect_arch)
    
    log_info "Detected: $os-$arch"
    
    # Check dependencies
    check_deps
    
    # Create directories
    mkdir -p "$INSTALL_DIR"
    mkdir -p "$BIN_DIR"
    
    # Try download first, then build
    if ! download_binary "$os" "$arch" "$VERSION" 2>/dev/null; then
        log_warn "Pre-built binary not available. Building from source..."
        build_from_source
    fi
    
    # Setup docker
    setup_docker
    
    # Create config
    create_config
    
    # Add to PATH
    if [[ ":$PATH:" != *":$BIN_DIR:"* ]]; then
        echo "export PATH=\"\$PATH:$BIN_DIR\"" >> "$HOME/.bashrc"
        log_info "Added $BIN_DIR to PATH"
    fi
    
    echo ""
    echo "============================================"
    echo -e "${GREEN}Installation Complete!${NC}"
    echo "============================================"
    echo ""
    echo "Next steps:"
    echo "  1. Start Docker services:"
    echo "     cd $INSTALL_DIR/docker && docker compose up -d"
    echo ""
    echo "  2. Start agent-memory:"
    echo "     cd $INSTALL_DIR && ./agent-memory"
    echo ""
    echo "  3. Or use the quick start:"
    echo "     source $INSTALL_DIR/.env"
    echo ""
}

main "$@"
