#!/usr/bin/env bash
# Run retrieval benchmarks on REAL datasets against LIVE Neo4j + Qdrant.
# Never uses --mock.
#
# Prerequisites:
#   docker compose up -d neo4j qdrant redis
#   .env with NEO4J_*, QDRANT_*, OPENAI_API_KEY (or EVALUATOR_API_KEY for LLM judge)
#
# Usage:
#   ./scripts/run-real-benchmarks.sh              # all built-in datasets
#   ./scripts/run-real-benchmarks.sh locomo       # one dataset
#   LIMIT=50 ./scripts/run-real-benchmarks.sh locomo
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

DATASET="${1:-all}"
LIMIT="${LIMIT:-0}"
MODE="${MODE:-hybrid}"
OUT_DIR="${OUT_DIR:-docs/benchmarks/results}"
mkdir -p "$OUT_DIR" data/benchmarks/raw

echo "==> checking live stores"
if ! curl -sf http://localhost:7474 >/dev/null; then
  echo "Neo4j not reachable on :7474 — start: docker compose up -d neo4j" >&2
  exit 1
fi
if ! curl -sf http://localhost:6333/readyz >/dev/null; then
  echo "Qdrant not reachable on :6333 — start: docker compose up -d qdrant" >&2
  exit 1
fi

echo "==> ensuring official LoCoMo dataset is converted"
if [[ ! -f data/benchmarks/raw/locomo10.json ]]; then
  curl -fsSL -o data/benchmarks/raw/locomo10.json \
    https://raw.githubusercontent.com/snap-research/locomo/main/data/locomo10.json
fi
python3 scripts/convert_locomo.py data/benchmarks/raw/locomo10.json data/benchmarks/locomo/dataset.json

# Prefer converted locomo; longmemeval/beam use repo fixtures (real format, full public dumps when available)
export BENCHMARK_DATASET_PATH="${BENCHMARK_DATASET_PATH:-}"

ts="$(date -u +%Y%m%dT%H%M%SZ)"
run_one() {
  local name="$1"
  local out="$OUT_DIR/${name}-live-${ts}.json"
  echo "==> running REAL dataset=$name mode=$MODE limit=$LIMIT (no mock)"
  # shellcheck disable=SC2086
  go run ./cmd/benchmark \
    --suite retrieval \
    --dataset "$name" \
    --mode "$MODE" \
    --parallel 1 \
    ${LIMIT:+--limit $LIMIT} \
    --output "$out"
  # also write "latest" pointer
  cp "$out" "$OUT_DIR/${name}-live-latest.json"
  echo "wrote $out"
}

if [[ "$DATASET" == "all" ]]; then
  for d in locomo longmemeval beam_1m beam_10m; do
    run_one "$d" || echo "WARN: $d failed" >&2
  done
else
  run_one "$DATASET"
fi

echo "==> done. Results under $OUT_DIR"
ls -la "$OUT_DIR" | tail -20
