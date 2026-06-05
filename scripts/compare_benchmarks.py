#!/usr/bin/env python3
"""Compare benchmark results against a baseline.

Usage: python3 scripts/compare_benchmarks.py <baseline.json> <latest.json>

Exits with code 0 if all metrics are within tolerance, 1 if any regress beyond threshold.
"""

import json
import sys
import os

TOLERANCE = {
    "overall_score": -0.05,    # Allow 5% drop in score
    "single_hop_score": -0.05,
    "multi_hop_score": -0.05,
    "latency_p50_ms": 0.20,    # Allow 20% increase in latency
    "latency_p95_ms": 0.20,
}


def load_results(path):
    with open(path) as f:
        return json.load(f)


def compare(baseline, latest):
    failures = []
    baseline_map = {r["dataset"]: r for r in baseline}
    latest_map = {r["dataset"]: r for r in latest}

    all_ok = True
    for dataset, bl in sorted(baseline_map.items()):
        if dataset not in latest_map:
            print(f"⚠️  {dataset}: missing from latest results")
            continue
        lt = latest_map[dataset]

        for metric, tol in TOLERANCE.items():
            bl_val = bl.get(metric, 0)
            lt_val = lt.get(metric, 0)
            if bl_val == 0:
                continue  # Baseline not established yet

            if metric.endswith("score"):
                # Higher is better — regress if below (bl_val + tol * bl_val)
                threshold = bl_val * (1 + tol)
                ok = lt_val >= threshold
                status = "✅" if ok else "❌"
                if not ok:
                    all_ok = False
                print(f"  {status} {dataset}.{metric}: {lt_val:.3f} vs baseline {bl_val:.3f} (threshold: {threshold:.3f})")
            else:
                # Lower is better (latency) — regress if above (bl_val * (1 + tol))
                threshold = bl_val * (1 + tol)
                ok = lt_val <= threshold
                status = "✅" if ok else "❌"
                if not ok:
                    all_ok = False
                print(f"  {status} {dataset}.{metric}: {lt_val:.1f} vs baseline {bl_val:.1f} (threshold: {threshold:.1f})")

    return all_ok


def main():
    if len(sys.argv) < 3:
        print("Usage: compare_benchmarks.py <baseline.json> <latest.json>")
        sys.exit(1)

    baseline_path = sys.argv[1]
    latest_path = sys.argv[2]

    if not os.path.exists(baseline_path):
        print(f"Baseline not found at {baseline_path}, skipping comparison")
        sys.exit(0)

    baseline = load_results(baseline_path)
    latest = load_results(latest_path)

    print(f"Comparing benchmarks:")
    print(f"  Baseline: {baseline_path} ({len(baseline)} dataset(s))")
    print(f"  Latest:   {latest_path} ({len(latest)} dataset(s))")

    all_ok = compare(baseline, latest)
    if all_ok:
        print("\n✅ All metrics within tolerance")
    else:
        print("\n❌ Some metrics outside tolerance")
    sys.exit(0 if all_ok else 1)


if __name__ == "__main__":
    main()
