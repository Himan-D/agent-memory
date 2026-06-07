#!/usr/bin/env bash
# Rewrite Mintlify root-relative asset URLs to /docs/* for hystersis.com/docs hosting.
set -euo pipefail

DOCS_DIR="${1:?docs output directory required}"

python3 - "$DOCS_DIR" <<'PY'
import sys
from pathlib import Path

docs_dir = Path(sys.argv[1])
prefixes = [
    "/_next/",
    "/logo/",
    "/favicons/",
    "/images/",
    "/icons/",
    "/favicon.svg",
    "/sitemap.xml",
    "/llms.txt",
    "/public/",
]

def rewrite(text: str) -> str:
    for prefix in prefixes:
        for quote in ('"', "'"):
            old = f"={quote}{prefix}"
            new = f"={quote}/docs{prefix}"
            text = text.replace(old, new)
    return text.replace("/docs/docs/", "/docs/")

count = 0
for path in docs_dir.rglob("*"):
    if path.suffix.lower() not in {".html", ".js", ".json"}:
        continue
    original = path.read_text(encoding="utf-8")
    updated = rewrite(original)
    if updated != original:
        path.write_text(updated, encoding="utf-8")
        count += 1

print(f"==> Rewrote asset paths in {count} files under {docs_dir}")
PY
