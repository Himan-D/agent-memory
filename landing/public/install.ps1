<#
.SYNOPSIS
    Hystersis — One-line install for Windows.

.DESCRIPTION
    Installs: CLI binary, Python SDK, Node.js SDK, Skills CLI, Docker services

    Usage:
        irm https://hystersis.com/install.ps1 | iex

    Options:
        -Minimal      CLI binary only (no SDKs, no Docker)
        -CliOnly      CLI binary + Docker services (no SDKs)
        -NoDocker     CLI binary + SDKs (no Docker services)
        -NoPython     Skip Python SDK
        -NoNode       Skip Node.js SDK & Skills CLI

    Environment variables:
        $env:VERSION   Release version (default: latest)
        $env:BIN_DIR   Install location (default: $env:LOCALAPPDATA\hystersis)

    Requires:
        - Windows 10 1803+ (for tar.exe) or Windows 11
        - PowerShell 5.1+ (ships with Windows 10/11)
#>

[CmdletBinding()]
param(
    [switch]$Minimal,
    [switch]$CliOnly,
    [switch]$NoDocker,
    [switch]$NoPython,
    [switch]$NoNode
)

$ErrorActionPreference = 'Stop'

# ── Configuration ────────────────────────────────────────────────────────────────
$REPO_URL = if ($env:REPO_URL) { $env:REPO_URL } else { 'https://github.com/Himan-D/agent-memory' }
$VERSION  = if ($env:VERSION)  { $env:VERSION }  else { 'latest' }
$BIN_DIR  = if ($env:BIN_DIR)  { $env:BIN_DIR }  else { Join-Path $env:LOCALAPPDATA 'hystersis' }
$INSTALL_DIR = $BIN_DIR

$INSTALL_PYTHON = -not $Minimal -and -not $NoPython
$INSTALL_NODE   = -not $Minimal -and -not $NoNode
$INSTALL_DOCKER = -not $Minimal -and -not $CliOnly -and -not $NoDocker

$BIN_NAME = 'hystersis.exe'

# ── Output helpers ───────────────────────────────────────────────────────────────
function Step($msg) { Write-Host ''; Write-Host "  > $msg" -ForegroundColor Cyan }
function Info($msg) { Write-Host "    $msg" -ForegroundColor Gray }
function Warn($msg) { Write-Host "    ! $msg" -ForegroundColor Yellow }

# ── Architecture detection ───────────────────────────────────────────────────────
$procArch = $env:PROCESSOR_ARCHITECTURE
$ARCH = switch ($procArch) {
    'AMD64' { 'amd64' }
    'ARM64' { 'arm64' }
    'x86'   { throw '32-bit Windows is not supported.' }
    default { throw "Unsupported architecture: $procArch" }
}

$OS = 'windows'

# ── Banner ───────────────────────────────────────────────────────────────────────
Write-Host ''
Write-Host '  Hystersis' -ForegroundColor White
Write-Host '  Memory that adapts. Intelligence that compounds.'
Write-Host '  https://hystersis.com'
Write-Host ''

# Ensure directories exist
New-Item -ItemType Directory -Path $BIN_DIR -Force | Out-Null

# ── Resolve latest version if needed ─────────────────────────────────────────────
if ($VERSION -eq 'latest') {
    Step 'Resolving latest version...'
    try {
        [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
        $release = Invoke-RestMethod -Uri "$REPO_URL/releases/latest" -Headers @{ Accept = 'application/json' } -MaximumRetryCount 3 -RetryIntervalSec 2
        $VERSION = $release.tag_name.TrimStart('v')
        Info "Latest: $VERSION"
    } catch {
        throw "Could not resolve latest version: $($_.Exception.Message)`nSet `$env:VERSION explicitly and re-run."
    }
}

$assetName = "hystersis-windows-$ARCH.tar.gz"
$downloadUrl = "$REPO_URL/releases/download/v$VERSION/$assetName"
$shaUrl = "$REPO_URL/releases/download/v$VERSION/SHA256SUMS"

# ── CLI Binary ──────────────────────────────────────────────────────────────────
Step 'Installing CLI...'
Info "Source:   $downloadUrl"
Info "Install:  $BIN_DIR\$BIN_NAME"

$cliBin = Join-Path $BIN_DIR $BIN_NAME
$BUILT = $false

# Check for Go and build from source if available
$goAvailable = Get-Command go -ErrorAction SilentlyContinue
if ($goAvailable -and $VERSION -eq 'latest') {
    Info "Building from source with Go..."

    # Try to clone and build
    $tempDir = Join-Path $env:TEMP "hystersis-source-$(Get-Random)"
    try {
        $gitAvailable = Get-Command git -ErrorAction SilentlyContinue
        if ($gitAvailable) {
            git clone --depth 1 $REPO_URL $tempDir 2>$null
            if (Test-Path $tempDir) {
                Push-Location $tempDir
                try {
                    $env:CGO_ENABLED = '0'
                    go build -o $cliBin ./cmd/cli
                    $serverBin = Join-Path $BIN_DIR 'hystersis-server.exe'
                    go build -o $serverBin ./cmd/server
                    $agentBin = Join-Path $BIN_DIR 'hystersis-agent.exe'
                    go build -o $agentBin ./cmd/agent

                    if (Test-Path $cliBin) {
                        Info "Built CLI: $cliBin"
                        Info "Built server: $serverBin"
                        Info "Built agent: $agentBin"
                        $BUILT = $true
                    }
                } finally {
                    Pop-Location
                }
            }
        }
    } catch {
        Warn "Build failed: $($_.Exception.Message)"
    } finally {
        if (Test-Path $tempDir) { Remove-Item $tempDir -Recurse -Force -ErrorAction SilentlyContinue }
    }
}

# Download if not built
if (-not $BUILT) {
    $tempTar = Join-Path $env:TEMP "hystersis-$VERSION-$ARCH.tar.gz"
    Info 'Downloading...'
    try {
        [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
        Invoke-WebRequest -Uri $downloadUrl -OutFile $tempTar -UseBasicParsing -MaximumRetryCount 3 -RetryIntervalSec 2
    } catch {
        $status = $null
        if ($_.Exception.Response) { $status = $_.Exception.Response.StatusCode.value__ }
        if ($status -eq 404) {
            Warn "No pre-built binary for windows-$ARCH. Install Go: https://go.dev/dl"
        } else {
            throw "Download failed: $($_.Exception.Message)"
        }
    }

    if (Test-Path $tempTar) {
        Info 'Extracting...'
        $extractDir = Join-Path $env:TEMP "hystersis-extract-$(Get-Random)"
        New-Item -ItemType Directory -Path $extractDir -Force | Out-Null
        tar -xzf $tempTar -C $extractDir

        $extractedBin = Get-ChildItem -Path $extractDir -Recurse -Filter $BIN_NAME -ErrorAction SilentlyContinue | Select-Object -First 1
        if ($extractedBin) {
            Move-Item -Path $extractedBin.FullName -Destination $cliBin -Force
            Info "Downloaded CLI to $cliBin"
        }

        Remove-Item $tempTar, $extractDir -Recurse -Force -ErrorAction SilentlyContinue
    }
}

# ── Docker Services ──────────────────────────────────────────────────────────────
if ($INSTALL_DOCKER) {
    $dockerAvailable = Get-Command docker -ErrorAction SilentlyContinue
    if ($dockerAvailable) {
        Step 'Setting up Docker services...'

        $dockerComposePath = Join-Path $INSTALL_DIR 'docker-compose.yml'
        $envPath = Join-Path $INSTALL_DIR '.env'

        if (-not (Test-Path $dockerComposePath)) {
            $dockerCompose = @"
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
"@
            Set-Content -Path $dockerComposePath -Value $dockerCompose -Encoding UTF8
            Info "Created $dockerComposePath"
        } else {
            Info "Docker compose already exists at $dockerComposePath"
        }

        # Create .env file
        $envContent = @"
NEO4J_URI=bolt://localhost:7687
NEO4J_USER=neo4j
NEO4J_PASSWORD=password
QDRANT_URL=http://localhost:6333
REDIS_URL=redis://localhost:6379
HTTP_PORT=:8080
API_BASE_URL=https://api.hystersis.com
"@
        Set-Content -Path $envPath -Value $envContent -Encoding UTF8
        Info "Created $envPath"

        # Generate bootstrap credentials if openssl available
        $opensslAvailable = Get-Command openssl -ErrorAction SilentlyContinue
        if ($opensslAvailable) {
            $salt = -join ((48..57) + (97..102) | Get-Random -Count 32 | ForEach-Object { [char]$_ })
            $adminSha = -join ((48..57) + (97..102) | Get-Random -Count 64 | ForEach-Object { [char]$_ })
            $userSha = -join ((48..57) + (97..102) | Get-Random -Count 64 | ForEach-Object { [char]$_ })
            $jwt = -join ((48..57) + (97..102) | Get-Random -Count 64 | ForEach-Object { [char]$_ })

            $tokenContent = @"

# Bootstrap credentials — generated $(Get-Date -Format 'yyyy-MM-ddTHH:mm:ssZ')
API_KEY_SALT=${salt}
ADMIN_API_KEYS=am_admin_${adminSha}
ADMIN_API_KEY=am_admin_${adminSha}
API_KEYS=usr_${userSha}:default
JWT_SECRET=${jwt}
"@
            Add-Content -Path $envPath -Value $tokenContent -Encoding UTF8
            Info "Generated API credentials in $envPath"
        }
    } else {
        Warn "Docker not found. Install from: https://docker.com"
    }
}

# ── Python SDK ───────────────────────────────────────────────────────────────────
if ($INSTALL_PYTHON) {
    Step 'Installing Python SDK...'
    $pythonBin = $null

    $python3 = Get-Command python3 -ErrorAction SilentlyContinue
    $python = Get-Command python -ErrorAction SilentlyContinue

    if ($python3) { $pythonBin = $python3.Source }
    elseif ($python) { $pythonBin = $python.Source }

    if ($pythonBin) {
        Info "Found Python: $pythonBin"
        $pipCmd = "$pythonBin -m pip"

        # Try direct install first
        $result = & cmd /c "$pipCmd install --user hystersis 2>&1"
        if ($LASTEXITCODE -eq 0) {
            Info "Python SDK installed: $pipCmd install --user hystersis"
        } else {
            # Try venv approach
            $venvPath = Join-Path $INSTALL_DIR 'python-sdk'
            $venvScripts = Join-Path $venvPath 'Scripts'
            $venvPython = Join-Path $venvScripts 'python.exe'

            Info "Trying venv approach..."
            & $pythonBin -m venv $venvPath 2>$null
            if (Test-Path $venvPython) {
                & $venvPython -m pip install --upgrade pip 2>$null | Out-Null
                & $venvPython -m pip install hystersis 2>$null | Out-Null
                Info "Python SDK installed in venv: $venvPath"
                Info "Activate with: $venvScripts\Activate.ps1"
            } else {
                Warn "Python SDK install failed. Try: python -m venv .venv; .venv\Scripts\pip install hystersis"
            }
        }
    } else {
        Warn "Python not found. Install from: https://python.org"
    }
}

# ── Node.js SDK and Skills CLI ──────────────────────────────────────────────────
if ($INSTALL_NODE) {
    Step 'Installing Node.js SDK and Skills CLI...'
    $npmAvailable = Get-Command npm -ErrorAction SilentlyContinue

    if ($npmAvailable) {
        Info "Found npm"

        # Install hystersis SDK
        $result = & npm install -g hystersis 2>&1
        if ($LASTEXITCODE -eq 0) {
            Info "Node.js SDK installed: npm install -g hystersis"
        } else {
            Warn "Node.js SDK install failed. Try: git clone $REPO_URL && cd agent-memory/sdk/nodejs && npm install && npm run build"
        }

        # Install Skills CLI
        $result = & npm install -g @hystersis/skills 2>&1
        if ($LASTEXITCODE -eq 0) {
            Info "Skills CLI installed: npm install -g @hystersis/skills"
        } else {
            Warn "Skills CLI install failed. Try: npm install -g $REPO_URL/skills-npm"
        }
    } else {
        Warn "npm not found. Install from: https://nodejs.org"
    }
}

# ── Config ───────────────────────────────────────────────────────────────────────
Step 'Creating config...'
$configPath = Join-Path $INSTALL_DIR 'config.json'
$configContent = @"
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
"@
Set-Content -Path $configPath -Value $configContent -Encoding UTF8
Info "Created $configPath"

# Find admin API key from .env
$envPath = Join-Path $INSTALL_DIR '.env'
$adminKey = $null
if (Test-Path $envPath) {
    $envContent = Get-Content $envPath -Raw
    if ($envContent -match 'ADMIN_API_KEY=(.+)') {
        $adminKey = $matches[1].Trim()
    }
}

# Create CLI config
$cliConfigPath = Join-Path $env:USERPROFILE '.agent-memory.json'
if ($adminKey) {
    $cliConfigContent = @"
{
  "base_url": "http://localhost:8080",
  "api_key": "$adminKey"
}
"@
    Set-Content -Path $cliConfigPath -Value $cliConfigContent -Encoding UTF8
    Info "Created CLI config: $cliConfigPath"
} else {
    Info "No local API key found; run 'hystersis init --api-key <key>' after starting the server"
}

# ── Update user PATH ─────────────────────────────────────────────────────────────
$userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
if ($userPath -notlike "*$BIN_DIR*") {
    [Environment]::SetEnvironmentVariable('Path', "$userPath;$BIN_DIR", 'User')
    Info "Added $BIN_DIR to user PATH (open a new terminal to take effect)"
} else {
    Info "User PATH already contains $BIN_DIR"
}

# ── Summary ──────────────────────────────────────────────────────────────────────
Write-Host ''
Write-Host '  ----------------------------------------'
Write-Host '  Installation Complete!'
Write-Host '  ----------------------------------------'
Write-Host ''
Write-Host "  Installed to: $INSTALL_DIR"
Write-Host "  Binaries:     $BIN_DIR"
Write-Host ''
Write-Host '  Commands:'
Write-Host "    hystersis           CLI - manage your memory"
Write-Host "    hystersis-server    API server"
Write-Host "    hystersis-agent     Interactive agent REPL"
Write-Host '    skills              Skills CLI'
Write-Host ''
Write-Host '  Quick start:'

if ($INSTALL_DOCKER) {
    Write-Host '    1. Start databases:'
    Write-Host "       docker compose -f $dockerComposePath up -d"
    Write-Host ''
    Write-Host '    2. Start the API server:'
    Write-Host "       hystersis-server"
    Write-Host '       hystersis health'
    Write-Host ''
    Write-Host "    3. Use the CLI:"
    Write-Host "       hystersis memories add --agent-id default --content 'Your first memory'"
} else {
    Write-Host '    1. Point the CLI at your API:'
    Write-Host '       hystersis init --url https://api.hystersis.com --api-key <your-key>'
    Write-Host ''
    Write-Host '    2. Check connectivity:'
    Write-Host '       hystersis health'
    Write-Host ''
    Write-Host "    3. Use the CLI:"
    Write-Host "       hystersis memories add --agent-id default --content 'Your first memory'"
}

Write-Host ''
Write-Host "  Docs:  https://hystersis.com/docs"
Write-Host "  Repo:  $REPO_URL"
Write-Host ''
