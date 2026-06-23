#!/usr/bin/env bash
# Build Mintlify docs static export for Cloudflare deployment.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DOCS_DIR="$ROOT/docs"
OUT_DIR="$ROOT/landing/dist/docs"
EXPORT_ZIP="/tmp/hystersis-docs-export.zip"

echo "==> Exporting Mintlify docs..."
cd "$DOCS_DIR"
npx --cache "$ROOT/.npm-cache" mintlify@latest export --output "$EXPORT_ZIP"

echo "==> Extracting to $OUT_DIR..."
rm -rf "$OUT_DIR"
mkdir -p "$OUT_DIR"
unzip -q -o "$EXPORT_ZIP" -d "$OUT_DIR"

{
  echo "# Hystersis Documentation"
  echo
  echo "Base URL: https://docs.hystersis.com"
  echo
  echo "Available documentation pages:"
  find "$DOCS_DIR" -type f \( -name '*.md' -o -name '*.mdx' \) \
    ! -path '*/node_modules/*' \
    | sort \
    | while IFS= read -r page; do
      rel="${page#"$DOCS_DIR"/}"
      route="${rel%.*}"
      if [ "$route" = "index" ]; then
        route="/"
      else
        route="/$route"
      fi
      title="$(awk '/^# / { sub(/^# /, ""); print; exit }' "$page")"
      if [ -z "$title" ]; then
        title="$route"
      fi
      echo "- $title: $route"
    done
} > "$OUT_DIR/llms.txt"

# Sync OpenAPI spec used by Mintlify into backend embed path
if [ -f "$DOCS_DIR/openapi.json" ]; then
  cp "$DOCS_DIR/openapi.json" "$ROOT/cmd/server/swagger.json"
fi

echo "==> Docs build complete ($(du -sh "$OUT_DIR" | cut -f1))"
