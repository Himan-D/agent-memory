# ADR-002: Observability-First Hardening

## Status

Accepted

## Context

Compression and retrieval quality are core differentiators. Production operators need reliable metrics before they can safely tune compression modes, tier policy, or retrieval routing. A latency percentile computed from insertion order can misreport p95 and produce bad alerting decisions.

## Options Considered

### Option A: Add More Dashboards First

- Complexity: low
- Cost: low
- Scalability: medium
- Security: neutral
- Development speed: fast
- Risk: dashboards can amplify incorrect metric values

### Option B: Fix Metric Correctness And Add Focused Tests

- Complexity: low
- Cost: low
- Scalability: high
- Security: neutral
- Development speed: fast
- Risk: low

### Option C: Replace Metrics With A Full Histogram-Based System Immediately

- Complexity: medium
- Cost: medium
- Scalability: high
- Security: neutral
- Development speed: slower
- Risk: medium migration risk

## Decision

Choose Option B for the current milestone. Correct p95 computation using sorted samples and add tests that cover unordered latency input. A later ADR can move latency reporting to Prometheus histograms if production traffic requires more efficient percentile calculation.

## Consequences

- Compression p95 and average metrics become trustworthy for alerts and dashboards.
- The implementation remains simple and compatible with the existing metrics collector.
- Very large latency windows still require a histogram or streaming quantile implementation in a future milestone.

