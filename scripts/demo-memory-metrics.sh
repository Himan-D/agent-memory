#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

API_URL="${API_URL:-http://127.0.0.1:8080}"
API_KEY="${HYSTERSIS_API_KEY:-demo-key}"
ADMIN_API_KEY="${ADMIN_API_KEY:-}"

for command_name in curl jq; do
  if ! command -v "$command_name" >/dev/null 2>&1; then
    echo "ERROR: $command_name is required. Install it and rerun this script." >&2
    exit 1
  fi
done

percent() {
  jq -nr --argjson value "${1:-0}" '$value * 100 | tostring'
}

echo "============================================================"
echo " Hystersis Memory: Retrieval, Compression & Operations Demo"
echo "============================================================"
echo "API: $API_URL"
echo

echo "[OPERATIONAL] Service health"
health="$(curl -fsS "$API_URL/health")" || {
  echo "ERROR: GET $API_URL/health failed. Start the server and its stores first." >&2
  exit 1
}
echo "$health" | jq .
if [[ "$(jq -r '.status // empty' <<<"$health")" != "healthy" ]]; then
  echo "ERROR: /health did not report status=healthy." >&2
  exit 1
fi
ready="$(curl -fsS "$API_URL/ready")" || {
  echo "ERROR: GET $API_URL/ready failed." >&2
  exit 1
}
echo "$ready" | jq .
echo

echo "[LIVE DEMO FIXTURE] Three-memory paraphrase retrieval"
run_id="demo-metrics-$(date -u +%Y%m%dT%H%M%SZ)-$$"
user_id="$run_id-user"
declare -a memory_ids queries

facts=(
  "Project Lantern's emergency rollback owner is Amara, and rollback begins when replication lag exceeds 45 seconds. Unique marker: ${run_id}-lantern."
  "The sealed recovery workbook is stored in vault folder Juniper-29; reviewers must open it only through the controlled virtual desktop. Unique marker: ${run_id}-vault."
  "The Mercury launch review is scheduled for 17 November 2026 at 14:30 UTC, and Dev is the final go-no-go approver. Unique marker: ${run_id}-mercury."
)
queries=(
  "${run_id}-lantern Who is responsible for reversing the deployment, and what delay triggers that action?"
  "${run_id}-vault Where should an auditor find the disaster-recovery spreadsheet, and how may it be opened?"
  "${run_id}-mercury When is the release decision meeting, and who makes the final decision?"
)

for index in 0 1 2; do
  payload="$(jq -n \
    --arg content "${facts[$index]}" \
    --arg user_id "$user_id" \
    --arg agent_id "$run_id-agent" \
    --arg run_id "$run_id" \
    '{content:$content,user_id:$user_id,agent_id:$agent_id,category:"demo-metric",metadata:{demo_run:$run_id}}')"
  created="$(curl -fsS -X POST "$API_URL/memories" \
    -H "X-API-Key: $API_KEY" -H 'Content-Type: application/json' \
    --data "$payload")" || {
    echo "ERROR: could not create live demo fixture $((index + 1))." >&2
    exit 1
  }
  memory_id="$(jq -r '.id // .memory.id // empty' <<<"$created")"
  if [[ -z "$memory_id" ]]; then
    echo "ERROR: create response did not contain a memory ID: $created" >&2
    exit 1
  fi
  memory_ids[$index]="$memory_id"
  echo "created fixture $((index + 1)): $memory_id"
done

top1_hits=0
top3_hits=0
positive_hop_results=0
latencies_json='[]'
for index in 0 1 2; do
  search_output="$(curl -sS -G "$API_URL/search/enhanced" \
    -H "X-API-Key: $API_KEY" \
    --data-urlencode 'mode=spreading' \
    --data-urlencode "query=${queries[$index]}" \
    --data-urlencode "user_id=$user_id" \
    -w $'\n%{http_code}\n%{time_total}')"
  latency_seconds="${search_output##*$'\n'}"
  without_latency="${search_output%$'\n'*}"
  http_code="${without_latency##*$'\n'}"
  search_json="${without_latency%$'\n'*}"
  if [[ "$http_code" != "200" ]]; then
    echo "ERROR: retrieval query $((index + 1)) returned HTTP $http_code: $search_json" >&2
    exit 1
  fi

  expected_id="${memory_ids[$index]}"
  rank1_id="$(jq -r '(.results // .data.results // [])[0] | (.id // .memory_id // empty)' <<<"$search_json")"
  if [[ "$rank1_id" == "$expected_id" ]]; then
    top1_hits=$((top1_hits + 1))
  fi
  if jq -e --arg id "$expected_id" '[(.results // .data.results // [])[:3][] | (.id // .memory_id)] | index($id) != null' <<<"$search_json" >/dev/null; then
    top3_hits=$((top3_hits + 1))
  fi
  query_hops="$(jq '[.results // .data.results // [] | .[] | select((.hops // .activation_hops // 0) > 0)] | length' <<<"$search_json")"
  positive_hop_results=$((positive_hop_results + query_hops))
  latency_ms="$(jq -nr --arg seconds "$latency_seconds" '$seconds | tonumber * 1000')"
  latencies_json="$(jq -c --argjson value "$latency_ms" '. + [$value]' <<<"$latencies_json")"
  echo "query $((index + 1)): expected=$expected_id rank1=${rank1_id:-none} latency=$(jq -nr --argjson n "$latency_ms" '$n|round')ms"
done

top1_pct="$(jq -nr --argjson n "$top1_hits" '$n / 3 * 100')"
hit3_pct="$(jq -nr --argjson n "$top3_hits" '$n / 3 * 100')"
avg_latency="$(jq 'add / length' <<<"$latencies_json")"
p95_latency="$(jq 'sort | .[((length * 0.95 | ceil) - 1)]' <<<"$latencies_json")"
printf 'Top-1 accuracy: %.1f%% (%d/3)\n' "$top1_pct" "$top1_hits"
printf 'Hit@3: %.1f%% (%d/3)\n' "$hit3_pct" "$top3_hits"
printf 'Average latency: %.1f ms\n' "$avg_latency"
printf 'P95 latency: %.1f ms\n' "$p95_latency"
if (( positive_hop_results > 0 )); then
  echo "Results reporting hops > 0: $positive_hop_results"
else
  echo "Results reporting hops > 0: 0 — graph propagation not demonstrated"
fi
echo "Fixtures retained for inspection (user_id=$user_id)."
echo

compression_samples="$(jq -n --arg run "$run_id" '{
  samples: [
    ("During the " + $run + " resilience review, the operations team documented that every production deployment requires a named rollback owner, a tested restoration procedure, replicated audit evidence, and explicit approval before traffic is shifted. The same policy is repeated for regional launches so responders can act without reconstructing context during an incident."),
    ("Customer preference memory for " + $run + " records communication windows, accessibility needs, approved data regions, escalation contacts, and the reasoning behind each constraint. Preserving named entities, dates, thresholds, and causal links matters more than preserving repeated connective prose or duplicated reminders."),
    ("The architecture report for " + $run + " explains that semantic retrieval gathers candidate memories, graph relationships may add connected evidence, ranking orders the candidates, and an answer model synthesizes the supplied context. It repeats these stages with operational examples to provide enough text for meaningful byte-compression measurement.")
  ],
  algorithms:["radix","smart_radix","smart_hybrid","real_best","gzip"],
  iterations:3,
  warmup:1,
  min_retention:0.9,
  include_examples:false
}')"

benchmark_ok=false
run_admin_benchmark="${RUN_COMPRESSION_BENCHMARK:-0}"
if [[ "${1:-}" == "--full" ]]; then
  run_admin_benchmark=1
fi
if [[ -n "$ADMIN_API_KEY" && "$run_admin_benchmark" == "1" ]]; then
  if ! benchmark_output="$(curl -sS -X POST "$API_URL/compression/benchmarks/run" \
    --max-time "${COMPRESSION_BENCHMARK_TIMEOUT:-300}" \
    -H "X-API-Key: $ADMIN_API_KEY" -H 'Content-Type: application/json' \
    --data "$compression_samples" -w $'\n%{http_code}')"; then
    benchmark_output=$'{}\n000'
  fi
  benchmark_code="${benchmark_output##*$'\n'}"
  benchmark_json="${benchmark_output%$'\n'*}"
  if [[ "$benchmark_code" == "200" ]] && jq -e '(.algorithms // .data.algorithms) | type == "array"' <<<"$benchmark_json" >/dev/null 2>&1; then
    benchmark_ok=true
    echo "[LEXICAL/BYTE BENCHMARK] Custom corpus, 3 samples × 3 measured iterations"
    jq -r '
      def pct: ((. // 0) * 100);
      "Evaluator: \(.evaluator // .data.evaluator // \"lexical-retention+byte-size\")",
      ((.algorithms // .data.algorithms // [])[] |
        "\(.name // .algorithm // \"unknown\"): avg reduction=\((.avg_reduction | pct) | tostring)% | retained size=\((100 - (.avg_reduction | pct)) | tostring)% | lexical retention=\((.avg_retention | pct) | tostring)% | p95=\(.p95_latency_ms // 0)ms | errors=\(.error_count // .errors // 0)")
    ' <<<"$benchmark_json"
    echo
  else
    echo "[LEXICAL/BYTE BENCHMARK] Admin benchmark unavailable (HTTP $benchmark_code); using fallback."
  fi
fi

if [[ "$benchmark_ok" != true ]]; then
  if [[ -z "$ADMIN_API_KEY" ]]; then
    echo "[LEXICAL/BYTE BENCHMARK] ADMIN_API_KEY is not set; using playground fallback."
  fi
  fallback_text="This single demo sample describes a production migration with repeated operational context. The release owner validates database replication, the security reviewer confirms access controls, and the incident commander verifies the rollback checklist before approving traffic. The release owner validates database replication again after the canary, while the incident commander keeps the rollback checklist available. Exact names, deadlines, numerical thresholds, and causal relationships must survive compression even when repeated prose is removed. This text is intentionally longer than a short note so each compression mode has enough material to process during the live demonstration."
  fallback_payload="$(jq -n --arg text "$fallback_text" '{text:$text,user_id:"demo-metrics",modes:["extraction","relational","radix","hybrid"],show_entities:false,show_facts:false}')"
  fallback_output="$(curl -sS -X POST "$API_URL/playground/compress" \
    --max-time 120 \
    -H "X-API-Key: $API_KEY" -H 'Content-Type: application/json' \
    --data "$fallback_payload" -w $'\n%{http_code}')"
  fallback_code="${fallback_output##*$'\n'}"
  fallback_json="${fallback_output%$'\n'*}"
  if [[ "$fallback_code" == "200" ]]; then
    jq -r '
      "Best mode: \(.best_mode // .data.best_mode // \"none\")",
      ((.results // .data.results // {}) | to_entries[] |
        "\(.key): reduction=\(.value.reduction_percent // 0)% | retained size=\(100 - (.value.reduction_percent // 0))% | token savings=\(.value.token_savings // 0) | latency=\(.value.latency_ms // 0)ms | fallback=\(.value.fallback // false)")
    ' <<<"$fallback_json"
    echo "These are single-sample demo values, not a publishable benchmark."
  else
    echo "Compression fallback failed with HTTP $fallback_code: $fallback_json" >&2
  fi
  echo
fi

echo "[OPERATIONAL] Compression collector snapshot"
if stats="$(curl -fsS "$API_URL/compression/stats" -H "X-API-Key: $API_KEY")"; then
  jq -r '
    "extractions_performed=\(.extractions_performed // 0)",
    "spreading_activations=\(.spreading_activations // 0)",
    "total_tokens_saved=\(.total_tokens_saved // 0)",
    "average_latency_ms=\(.avg_latency_ms // 0)",
    "p95_latency_ms=\(.p95_latency_ms // 0)",
    "compression_errors=\(.compression_errors // 0)"
  ' <<<"$stats"
  jq -r '
    (.compression_ratio_by_mode // .mode_stats // {}) as $m |
    if ($m | type) == "object" then
      $m | to_entries[] |
      "mode=\(.key) observed_avg_byte_reduction=\((.value.avg_reduction // .value.average_reduction // .value.reduction // .value.avg_ratio // .value // 0) * 100)%"
    elif ($m | type) == "array" then
      $m[] | "mode=\(.mode // .name // \"unknown\") observed_avg_byte_reduction=\((.avg_reduction // .average_reduction // .reduction // .avg_ratio // 0) * 100)%"
    else empty end
  ' <<<"$stats"
  accuracy_retention="$(jq -r '.accuracy_retention // 0' <<<"$stats")"
  token_reduction="$(jq -r '.token_reduction // 0' <<<"$stats")"
  if jq -e --argjson n "$accuracy_retention" '$n != 0' >/dev/null; then
    echo "operational collector accuracy_retention=$(percent "$accuracy_retention")%"
  fi
  if jq -e --argjson n "$token_reduction" '$n != 0' >/dev/null; then
    echo "operational collector token_reduction=$(percent "$token_reduction")%"
  fi
else
  echo "Compression stats endpoint unavailable."
fi
echo

prior_file="$ROOT/docs/benchmarks/results/locomo-live-latest.json"
if [[ "${1:-}" == "--full" ]]; then
  echo "[PREVIOUS REAL DATASET] Running a new limited LoCoMo benchmark first..."
  LIMIT="${LIMIT:-20}" BENCHMARK_PARALLEL="${BENCHMARK_PARALLEL:-4}" \
    "$ROOT/scripts/run-real-benchmarks.sh" locomo
fi
if [[ -f "$prior_file" ]]; then
  echo "[PREVIOUS REAL DATASET] $prior_file (file result, not the current live fixture run)"
  jq -r '
    "scorer=\(.score_method // \"unknown\") publishable=\(.publishable // false) questions=\(.questions_answered // .scored_questions // .total_questions // 0)",
    "overall_score=\(.overall_score // 0) multi_hop_score=\(.multi_hop_score // 0) memory_hit_rate=\(.memory_hit_rate // 0)",
    "Hit@1=\(.hit_at_1 // 0) Hit@3=\(.hit_at_3 // 0) Hit@10=\(.hit_at_10 // 0) MRR=\(.mrr // 0)",
    "p50_latency_ms=\(.latency_p50_ms // 0) p95_latency_ms=\(.latency_p95_ms // 0)"
  ' "$prior_file"
else
  echo "[PREVIOUS REAL DATASET] No saved LoCoMo result found. Use --full to generate one."
fi
echo

echo "What to say"
echo "- Top-1: how often the expected memory was the first result."
echo "- Hit@3: how often the expected memory appeared in the first three results."
echo "- Reduction: percentage of original bytes removed by compression."
echo "- Retained size: percentage of original bytes remaining after compression."
echo "- Lexical retention: overlap of important source terms; it is not semantic accuracy."
