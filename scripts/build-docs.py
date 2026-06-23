import os
import sys
import shutil
import zipfile
import subprocess
from pathlib import Path

root_dir = Path(__file__).resolve().parent.parent
docs_dir = root_dir / "docs"
out_dir = root_dir / "landing" / "dist" / "docs"
export_zip = docs_dir / "export.zip"

print("==> Cleaning output directory...")
if out_dir.exists():
    shutil.rmtree(out_dir)
out_dir.mkdir(parents=True, exist_ok=True)

print("==> Exporting Mintlify docs...")
subprocess.run(["npx", "mintlify", "export", "--output", str(export_zip)], cwd=docs_dir, shell=True, check=True)

print(f"==> Extracting to {out_dir}...")
with zipfile.ZipFile(export_zip, 'r') as zip_ref:
    zip_ref.extractall(out_dir)
export_zip.unlink()

print("==> Generating llms.txt...")
pages = []
for p in docs_dir.rglob("*"):
    if p.is_file() and p.suffix in {".md", ".mdx"} and "node_modules" not in p.parts:
        rel = p.relative_to(docs_dir)
        route = rel.with_suffix("")
        if route.name == "index":
            route_str = "/"
        else:
            route_str = "/" + str(route).replace("\\", "/")
        
        # Extract title from first H1
        title = ""
        try:
            with open(p, "r", encoding="utf-8") as f:
                for line in f:
                    if line.startswith("# "):
                        title = line.replace("# ", "").strip()
                        break
        except Exception:
            pass
        if not title:
            title = route_str
        pages.append((title, route_str))

pages.sort(key=lambda x: x[1])

llms_content = ["# Hystersis Documentation", "", "Base URL: https://docs.hystersis.com", "", "Available documentation pages:"]
for title, route in pages:
    llms_content.append(f"- {title}: {route}")

with open(out_dir / "llms.txt", "w", encoding="utf-8") as f:
    f.write("\n".join(llms_content) + "\n")

# Copy openapi spec
openapi = docs_dir / "openapi.json"
swagger = root_dir / "cmd" / "server" / "swagger.json"
if openapi.exists():
    swagger.parent.mkdir(parents=True, exist_ok=True)
    shutil.copy2(openapi, swagger)

print("==> Docs build complete")
