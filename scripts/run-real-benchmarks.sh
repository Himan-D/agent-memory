#!/usr/bin/env bash
# Run retrieval benchmarks on REAL datasets against LIVE Neo4j + Qdrant.
# Never uses --mock.
#
# Prerequisites:
#   docker compose up -d neo4j qdrant redis
#   .env with NEO4J_*, QDRANT_*, OPENAI_API_KEY (embeddings)
#   LLM judge (for publishable=true):
#     export EVALUATOR_API_KEY=sk-...   # or OPENAI_API_KEY
#     export EVALUATOR_PROVIDER=openai
#     export EVALUATOR_MODEL=gpt-4o-mini
#
# Optional tuning (no silent defaults beyond flag zero-values):
#   BENCHMARK_PARALLEL, BENCHMARK_LIMIT, BENCHMARK_SEARCH_LIMIT,
#   BENCHMARK_CONTEXT_TOPK, BENCHMARK_RRF_K, BENCHMARK_CANDIDATE_LIMIT,
#   BENCHMARK_MAX_TOKENS
#
# Usage:
#   ./scripts/run-real-benchmarks.sh              # all built-in datasets
#   ./scripts/run-real-benchmarks.sh locomo       # one dataset
#   LIMIT=50 BENCHMARK_PARALLEL=8 ./scripts/run-real-benchmarks.sh locomo
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

# Load .env without printing secrets
if [[ -f .env ]]; then
  set -a
  # shellcheck disable=SC1091
  source .env
  set +a
fi

DATASET="${1:-all}"
LIMIT="${LIMIT:-${BENCHMARK_LIMIT:-0}}"
MODE="${MODE:-hybrid}"
OUT_DIR="${OUT_DIR:-docs/benchmarks/results}"
# Competitive defaults aimed at Mem0-comparable retrieval depth + reader QA.
PARALLEL="${BENCHMARK_PARALLEL:-${PARALLEL:-4}}"
SEARCH_LIMIT="${BENCHMARK_SEARCH_LIMIT:-40}"
CONTEXT_TOPK="${BENCHMARK_CONTEXT_TOPK:-15}"
RRF_K="${BENCHMARK_RRF_K:-60}"
CANDIDATE_LIMIT="${BENCHMARK_CANDIDATE_LIMIT:-40}"
MAX_TOKENS="${BENCHMARK_MAX_TOKENS:-256}"

export EVALUATOR_PROVIDER="${EVALUATOR_PROVIDER:-openai}"
export EVALUATOR_MODEL="${EVALUATOR_MODEL:-gpt-4o-mini}"
if [[ -z "${EVALUATOR_API_KEY:-}" && -n "${OPENAI_API_KEY:-}" ]]; then
  export EVALUATOR_API_KEY="$OPENAI_API_KEY"
fi
mkdir -p "$OUT_DIR" data/benchmarks/raw

if [[ -z "${EVALUATOR_API_KEY:-}${OPENAI_API_KEY:-}" ]]; then
  echo "WARN: no EVALUATOR_API_KEY/OPENAI_API_KEY — will score with token_f1 (not publishable)" >&2
else
  echo "LLM judge enabled: provider=${EVALUATOR_PROVIDER} model=${EVALUATOR_MODEL}"
fi

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

export BENCHMARK_DATASET_PATH="${BENCHMARK_DATASET_PATH:-}"

# Prefer a prebuilt binary when present (faster than go run for multi-dataset).
BENCH_BIN="${BENCH_BIN:-}"
if [[ -z "$BENCH_BIN" ]]; then
  if [[ -x /tmp/hyst-benchmark ]]; then
    BENCH_BIN=/tmp/hyst-benchmark
  else
    go build -o /tmp/hyst-benchmark ./cmd/benchmark
    BENCH_BIN=/tmp/hyst-benchmark
  fi
fi

ts="$(date -u +%Y%m%dT%H%M%SZ)"
run_one() {
  local name="$1"
  local out="$OUT_DIR/${name}-live-${ts}.json"
  echo "==> running REAL dataset=$name mode=$MODE limit=${LIMIT:-0} parallel=$PARALLEL (no mock)"
  local args=(
    --suite retrieval
    --dataset "$name"
    --mode "$MODE"
    --parallel "$PARALLEL"
    --output "$out"
  )
  if [[ -n "${LIMIT}" && "${LIMIT}" != "0" ]]; then
    args+=(--limit "$LIMIT")
  fi
  if [[ -n "${SEARCH_LIMIT}" ]]; then
    args+=(--search-limit "$SEARCH_LIMIT")
  fi
  if [[ -n "${CONTEXT_TOPK}" ]]; then
    args+=(--context-topk "$CONTEXT_TOPK")
  fi
  if [[ -n "${RRF_K}" ]]; then
    args+=(--rrf-k "$RRF_K")
  fi
  if [[ -n "${CANDIDATE_LIMIT}" ]]; then
    args+=(--candidate-limit "$CANDIDATE_LIMIT")
  fi
  if [[ -n "${MAX_TOKENS}" ]]; then
    args+=(--max-tokens "$MAX_TOKENS")
  fi
  "$BENCH_BIN" "${args[@]}"
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
ls -la "$OUT_DIR"/*-live-${ts}.json 2>/dev/null || ls -la "$OUT_DIR" | tail -20
