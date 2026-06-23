#!/usr/bin/env bash
# Build Mintlify docs static export for Cloudflare deployment.
set -e

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
python3 "$ROOT/scripts/build-docs.py" || python "$ROOT/scripts/build-docs.py"
