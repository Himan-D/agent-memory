#!/usr/bin/env bash
# Product readiness gate: unit tests, builds, live stores, real (non-mock) benchmark slice.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
FAIL=0

ok() { echo "  OK  $*"; }
bad() { echo "  FAIL $*"; FAIL=1; }

echo "== Unit / package tests =="
if go test ./internal/tenant/ ./internal/fs/vfs/ ./internal/storage/ ./internal/evaluation/ ./cmd/server/ -count=1 -short; then
  ok "core packages"
else
  bad "core packages"
fi

echo "== Builds =="
for t in ./cmd/server ./cmd/cli ./cmd/agentfs ./cmd/benchmark; do
  if go build -o /dev/null "$t"; then ok "build $t"; else bad "build $t"; fi
done

echo "== Live stores =="
if curl -sf http://localhost:7474 >/dev/null; then ok "Neo4j :7474"; else bad "Neo4j down (docker compose up -d neo4j)"; fi
if curl -sf http://localhost:6333/readyz >/dev/null; then ok "Qdrant :6333"; else bad "Qdrant down (docker compose up -d qdrant)"; fi

echo "== Real dataset present =="
if [[ -f data/benchmarks/locomo/dataset.json ]]; then
  python3 - <<'PY'
import json
d=json.load(open("data/benchmarks/locomo/dataset.json"))
n,q=len(d.get("memories",[])),len(d.get("questions",[]))
print(f"  OK  locomo memories={n} questions={q} source={d.get('source','?')}")
if n < 100:
  print("  WARN locomo looks like a tiny fixture; run scripts/convert_locomo.py for official data")
PY
else
  bad "missing data/benchmarks/locomo/dataset.json — run python3 scripts/convert_locomo.py"
fi

echo "== Live real-data benchmark (LIMIT=20, no mock) =="
if curl -sf http://localhost:7474 >/dev/null && curl -sf http://localhost:6333/readyz >/dev/null; then
  mkdir -p docs/benchmarks/results
  OUT="docs/benchmarks/results/readiness-locomo-live.json"
  if go run ./cmd/benchmark --suite retrieval --dataset locomo --mode hybrid --parallel 1 --limit 20 --output "$OUT"; then
    python3 - <<PY
import json
d=json.load(open("$OUT"))
print("  OK  live run dataset=%s hit@10=%.3f mrr=%.3f publishable=%s method=%s ingested=%s" % (
  d.get("dataset"), d.get("hit_at_10") or 0, d.get("mrr") or 0, d.get("publishable"), d.get("score_method"), d.get("memories_ingested")))
if d.get("ingest_errors",0)>0 or d.get("search_errors",0)>0:
  raise SystemExit("errors in live run")
PY
    ok "live non-mock locomo slice"
  else
    bad "live benchmark failed"
  fi
else
  echo "  SKIP live benchmark (stores not up)"
fi

echo
if [[ $FAIL -eq 0 ]]; then
  echo "READINESS: PASS (unit + builds + real dataset path; live slice if stores up)"
  exit 0
else
  echo "READINESS: FAIL"
  exit 1
fi
