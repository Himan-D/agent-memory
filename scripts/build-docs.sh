#!/usr/bin/env bash
# Build Mintlify docs static export for Cloudflare deployment.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DOCS_DIR="$ROOT/docs"
OUT_DIR="$ROOT/landing/dist/docs"
EXPORT_ZIP="/tmp/hystersis-docs-export.zip"

echo "==> Exporting Mintlify docs..."
cd "$DOCS_DIR"
npx mintlify@latest export --output "$EXPORT_ZIP"

echo "==> Extracting to $OUT_DIR..."
rm -rf "$OUT_DIR"
mkdir -p "$OUT_DIR"
unzip -q -o "$EXPORT_ZIP" -d "$OUT_DIR"

bash "$ROOT/scripts/rewrite-docs-assets.sh" "$OUT_DIR"

# Sync OpenAPI spec used by Mintlify into backend embed path
if [ -f "$DOCS_DIR/openapi.json" ]; then
  cp "$DOCS_DIR/openapi.json" "$ROOT/cmd/server/swagger.json"
fi

echo "==> Docs build complete ($(du -sh "$OUT_DIR" | cut -f1))"
