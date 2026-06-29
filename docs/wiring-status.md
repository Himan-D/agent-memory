# Benchmark Wiring Status (Cognee-Inspired Packages)

This document tracks the production readiness of the Cognee-inspired packages
wired into `cmd/benchmark` via the `-enable-wiring` master flag. Last
updated: 2026-06-26.

## TL;DR

- **Production-ready:** the underlying packages (`internal/session`,
  `internal/retrieval`, `internal/memory/rollback`,
  `internal/memory/improve`, `internal/cogni`) are additive and do not
  change any existing behavior.
- **Experimental:** the benchmark wiring flags exercise these packages
  end-to-end, but the metrics they report are NOT yet better than the
  baseline and should not be relied on for production tuning.

## Master Gate

All wiring flags require `-enable-wiring` to take effect. Without it, the
flags are silently ignored (a warning is printed to stderr). This prevents
accidental enablement in production.

```bash
# Baseline (default) - no wiring, no behavior change.
go run ./cmd/benchmark -dataset longmemeval -limit 10

# Explicit opt-in for any wiring flag.
go run ./cmd/benchmark -dataset longmemeval -limit 10 \
    -enable-wiring \
    -distill -distill-top-k 50 \
    -use-base-retriever \
    -rollback-on-error \
    -improve -improve-build-global -improve-sync-cache
```

## Flag Status

| Flag | Status | Notes |
|------|--------|-------|
| `-enable-wiring` | Production-ready | Master gate, no behavior when off |
| `-distill` | Experimental | 54 s wall-clock for 15% acceptance in observed run; production guard added (`MinDistillationTurns=30`) so it skips on small inputs |
| `-use-base-retriever` | Experimental | After P0 fix (OrgID + Rerank), should match baseline hit@k; needs verification |
| `-rollback-on-error` | Production-ready | Synthetic probe, no real-world effect; safe to enable for diagnostics |
| `-improve` | Experimental | All 6 stages are noops; reports orchestration timings only |
| `-improve-build-global` | Experimental | Controls noop stage, no real effect |
| `-improve-sync-cache` | Experimental | Controls noop stage, no real effect |
| `-distill-top-k` | Production-ready | Cap, no behavior when distill is off |

## Known Production Blockers

### P0: Resolved in this iteration

1. **Master gate missing** → added `-enable-wiring` master flag with warning
   on misuse.
2. **BaseRetriever hit@1 regression (0.9 → 0.0)** → fixed
   `serviceSearcherAdapter` to pass `OrgID: "benchmark"`, `Rerank: true`,
   `Mode: ""` matching the existing `serviceAdapter.Search`. Verify with a
   re-run.

### P0: Out of scope for wiring

- **Baseline `latency_p95_ms = 6001`** — not caused by the wiring; the
  existing search path already has this issue. Investigate separately.

### P1: Resolved in this iteration

3. **Distillation wasted 54 s on small inputs** → added `MinDistillationTurns
   = 30` guard; `-distill` returns early with `skipped: true, skip_reason:
   "..."` when below threshold or no LLM is configured.

### P2: Not resolved

4. **Distillation acceptance rate is 3/20 (15%)** — the curator prompt is
   too narrow ("Extract 0-3 durable, entity-anchored lessons") and the
   synthetic question "Memory N context" gives the LLM no signal. Needs a
   real curator prompt + real per-memory questions.
5. **Writer accepts everything** — `AcceptAllWriter()` permits duplicates
   and trivia. Production needs a novelty-checking writer that searches
   for similar existing lessons before accepting.
6. **Improvement pipeline stages are noops** — `feedback_weights`,
   `persist_sessions`, `distill_sessions`, `memify_enrichment`,
   `global_context_index`, `sync_to_cache` all return zero items by
   design. Production wiring needs Neo4j/Redis-backed implementations.

## Underlying Package Status

| Package | Production-ready? | Notes |
|---------|-------------------|-------|
| `internal/resilience` | Yes | Pure utility, no I/O, fully tested |
| `internal/memory/chunker.go` | Yes | Pure function, deterministic IDs |
| `internal/memory/engine_handle.go` | Yes | Handle pattern, hybrid write detection |
| `internal/memory/rollback` | Yes | In-memory ledger + deleter callback, fully tested |
| `internal/llm/gateway.go` | Yes | Wraps existing provider, adds memory context + JSON extraction |
| `internal/session/` | Partial | Types, gate, batcher, manager are production-ready; distiller needs a real writer + Neo4j store |
| `internal/retrieval/retriever.go` | Yes | `BaseRetriever` interface + types |
| `internal/retrieval/adapters.go` | Partial | Vector/MultiSignal/Graph adapters work but graph adapter is a thin wrapper around semantic search; full spreading activation is still in `internal/compression/retrieval/` |
| `internal/pipeline/` | Yes | Task composition framework, fully tested |
| `internal/memory/improve/` | Partial | Pipeline orchestration is production-ready; stages are noops |
| `internal/cogni/handlers.go` | Yes | Opt-in HTTP handlers, fully tested, gated by Deps |

## Rollout Plan

1. **Ship current state** with the 4 P0/P1 fixes applied. The wiring is
   opt-in, does not change baseline behavior, and adds no runtime cost when
   `-enable-wiring` is not set.
2. **Re-run benchmark** with `-enable-wiring -use-base-retriever
   -distill-top-k 50 -rollback-on-error -improve` and verify:
   - `base_retriever.hit_at_1_rate` matches baseline (~0.9)
   - `distillation.skipped == false` (input has 50 turns ≥ 30 threshold)
   - `rollback.failed_on_deleter == 1` (probe works)
   - `improvement.stages_run` lists all 6 stages
3. **Future work** (separate PRs):
   - Real curator prompt + per-memory questions for distillation
   - Novelty-checking writer using graph search
   - Neo4j/Redis-backed improvement stages
   - LLM reranker wired into the BaseRetriever path
   - Investigation of baseline p95 = 6 s latency
