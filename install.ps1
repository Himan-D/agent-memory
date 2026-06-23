<#
.SYNOPSIS
    Native Windows installer for Hysteresis CLI.

.DESCRIPTION
    Downloads the Hysteresis CLI binary from GitHub Releases, verifies its
    SHA256 checksum against the published SHA256SUMS, and adds it to the
    user's PATH.

    Usage:
        irm https://hystersis.com/install.ps1 | iex

    What this installer does:
      - Downloads the Windows CLI binary from GitHub Releases
      - Verifies the SHA256 checksum against the published SHA256SUMS
      - Places the binary in $BIN_DIR (default: %LOCALAPPDATA%\Programs\Hysteresis)
      - Adds $BIN_DIR to the user's PATH (takes effect in new terminals)
      - Runs `hystersis --version` as a smoke test

    What it does NOT do (install these separately if needed):
      - Python SDK    -> pip install hystersis
      - Node.js SDK   -> npm install -g hystersis
      - Skills CLI    -> npm install -g @hystersis/skills
      - Docker stack  -> Docker Desktop + manual docker compose up
      See https://hystersis.com/docs for details.

    Environment variables:
      REPO_URL  GitHub repo URL  (default: https://github.com/Himan-D/agent-memory)
      VERSION   Release version  (default: latest)
      BIN_DIR   Install location (default: $env:LOCALAPPDATA\Programs\Hysteresis)

    Requires:
      - Windows 10 1803+ (for built-in tar.exe) or Windows 11
      - PowerShell 5.1+ (ships with Windows 10/11)
#>

[CmdletBinding()]
param()

$ErrorActionPreference = 'Stop'

# ── Configuration ────────────────────────────────────────────────────────────────
$REPO_URL = if ($env:REPO_URL) { $env:REPO_URL } else { 'https://github.com/Himan-D/agent-memory' }
$VERSION  = if ($env:VERSION)  { $env:VERSION }  else { 'latest' }
$BIN_DIR  = if ($env:BIN_DIR)  { $env:BIN_DIR }  else { Join-Path $env:LOCALAPPDATA 'Programs\Hysteresis' }
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

# ── Pre-flight: tar.exe ──────────────────────────────────────────────────────────
if (-not (Get-Command tar -ErrorAction SilentlyContinue)) {
    throw "This installer requires 'tar.exe', which ships with Windows 10 1803+.`nOn older releases, install Git for Windows: https://git-scm.com/download/win"
}

# ── Banner ───────────────────────────────────────────────────────────────────────
Write-Host ''
Write-Host '  Hysteresis' -ForegroundColor White
Write-Host '  Memory that adapts. Intelligence that compounds.'
Write-Host '  https://hystersis.com'
Write-Host ''

# ── Resolve latest version if needed ─────────────────────────────────────────────
if ($VERSION -eq 'latest') {
    Step 'Resolving latest version...'
    try {
        [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
        $release = Invoke-RestMethod -Uri "$REPO_URL/releases/latest" -Headers @{ Accept = 'application/json' } -MaximumRetryCount 3 -RetryIntervalSec 2
        $VERSION = $release.tag_name.TrimStart('v')
        Info "Latest: $VERSION"
    } catch {
        throw "Could not resolve latest version: $($_.Exception.Message)`nSet `$env:VERSION explicitly (e.g. 0.26.0623-a1b2c3d) and re-run."
    }
}

$assetName   = "hystersis-windows-$ARCH.tar.gz"
$downloadUrl = "$REPO_URL/releases/v$VERSION/download/$assetName"
$shaUrl      = "$REPO_URL/releases/v$VERSION/download/SHA256SUMS"

# ── Install ──────────────────────────────────────────────────────────────────────
Step 'Installing CLI...'
Info "Source:   $downloadUrl"
Info "Install:  $BIN_DIR\$BIN_NAME"

if (-not (Test-Path $BIN_DIR)) {
    New-Item -ItemType Directory -Path $BIN_DIR -Force | Out-Null
}

$tempTar = Join-Path $env:TEMP "hystersis-$VERSION-$ARCH.tar.gz"
$tempDir = Join-Path $env:TEMP "hystersis-extract-$VERSION-$ARCH"

# Download
Info 'Downloading...'
try {
    [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
    Invoke-WebRequest -Uri $downloadUrl -OutFile $tempTar -UseBasicParsing -MaximumRetryCount 3 -RetryIntervalSec 2
} catch {
    $status = $null
    if ($_.Exception.Response) { $status = $_.Exception.Response.StatusCode.value__ }
    if ($status -eq 404) {
        throw "Release asset not found: $downloadUrl`nCheck that `$env:VERSION='$VERSION' exists at $REPO_URL/releases."
    }
    throw "Download failed: $($_.Exception.Message)"
}

# Verify checksum (best-effort: skip if SHA256SUMS is not published)
Info 'Verifying checksum...'
$shaFailed = $false
try {
    $shaBody = (Invoke-WebRequest -Uri $shaUrl -UseBasicParsing -MaximumRetryCount 2 -RetryIntervalSec 1).Content
    $expected = ($shaBody -split "`r?`n" | Where-Object { $_.Trim().EndsWith($assetName) } | Select-Object -First 1).Trim() -split '\s+' | Select-Object -First 1
    if (-not $expected) {
        throw "No checksum entry for $assetName in SHA256SUMS"
    }
    $actual = (Get-FileHash $tempTar -Algorithm SHA256).Hash.ToLower()
    if ($actual -ne $expected.ToLower()) {
        throw "Checksum mismatch`n  expected: $expected`n  actual:   $actual"
    }
    Info 'Checksum OK'
} catch {
    $status = $null
    if ($_.Exception.Response) { $status = $_.Exception.Response.StatusCode.value__ }
    if ($status -eq 404) {
        Info 'No SHA256SUMS published for this release (skipping verification)'
    } else {
        $shaFailed = $true
        Write-Host "    checksum error: $($_.Exception.Message)" -ForegroundColor Yellow
    }
}

if ($shaFailed) {
    throw "Refusing to install an artifact that failed checksum verification."
}

# Extract
Info 'Extracting...'
if (Test-Path $tempDir) { Remove-Item $tempDir -Recurse -Force }
New-Item -ItemType Directory -Path $tempDir -Force | Out-Null
tar -xzf $tempTar -C $tempDir

$src = Get-ChildItem -Path $tempDir -Recurse -Filter $BIN_NAME -ErrorAction SilentlyContinue | Select-Object -First 1
if (-not $src) {
    throw "Binary '$BIN_NAME' not found inside archive"
}
$dest = Join-Path $BIN_DIR $BIN_NAME
Move-Item -Path $src.FullName -Destination $dest -Force

# Cleanup
Remove-Item $tempTar, $tempDir -Recurse -Force -ErrorAction SilentlyContinue

# ── Update user PATH (applies to future sessions, not this one) ──────────────────
$userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
if ($userPath -notlike "*$BIN_DIR*") {
    [Environment]::SetEnvironmentVariable('Path', "$userPath;$BIN_DIR", 'User')
    Info "Added $BIN_DIR to user PATH (open a new terminal to take effect)"
} else {
    Info "User PATH already contains $BIN_DIR"
}

# Also expose it for *this* session so the smoke test below can run.
$env:Path = "$env:Path;$BIN_DIR"

# ── Smoke test ───────────────────────────────────────────────────────────────────
Info 'Smoke testing...'
try {
    & $dest --version | Out-Null
    Info "OK: & $dest --version"
} catch {
    throw "Installed binary did not run successfully: $($_.Exception.Message)"
}

# ── Summary ──────────────────────────────────────────────────────────────────────
Write-Host ''
Write-Host '  ----------------------------------------'
Write-Host '  Installation Complete!'
Write-Host '  ----------------------------------------'
Write-Host ''
Write-Host "  Installed: $dest" -ForegroundColor Green
Write-Host ''
Write-Host '  Next steps (open a NEW PowerShell window so PATH updates apply):'
Write-Host '    hystersis --help'
Write-Host '    hystersis init --url <api-url> --api-key <key>'
Write-Host ''
Write-Host '  Need the Python SDK, Node.js SDK, Skills CLI, or Docker stack?'
Write-Host '  See: https://hystersis.com/docs'
Write-Host ''