# ADR-001: Architecture-First Productization

## Status

Accepted

## Context

Hystersis already contains a broad production surface: Go backend, memory engine, compression, retrieval, skills, wiki, dashboard, landing page, SDKs, deployment assets, and CI. The primary risk is not missing scaffolding; it is inconsistent execution across docs, metrics, tests, and product boundaries.

The autonomous build request asked for divergent branches and three implementation options per major feature. The local sandbox does not permit writing Git refs, so branch creation must happen outside this session or through a writable Git environment. The implementation strategy still follows the requested comparison-and-selection discipline.

## Options Considered

### Option A: Restructure Into A New Enterprise Monorepo

- Complexity: high
- Cost: high
- Scalability: medium
- Security: neutral
- Development speed: slow
- Risk: very high churn, likely breaks existing product surfaces

### Option B: Preserve Current Repository And Add Canonical Governance Docs

- Complexity: low
- Cost: low
- Scalability: high
- Security: positive through clearer boundaries
- Development speed: fast
- Risk: low

### Option C: Split Backend, Frontend, SDKs, And Infra Into Separate Repositories

- Complexity: high
- Cost: high
- Scalability: high for large teams
- Security: mixed due to cross-repo drift
- Development speed: slow initially
- Risk: high coordination overhead before product-market stability

## Decision

Choose Option B. Keep the existing repository structure and add canonical architecture and ADR documentation while tightening high-impact implementation gaps incrementally.

## Consequences

- Existing production assets remain intact.
- Future feature work gets a clear decision journal without blocking on a repo migration.
- Large structural changes require separate ADRs and migration plans.
- Branch-driven work remains the preferred Git workflow, but this session records that local branch creation was blocked by `.git` write permissions.

