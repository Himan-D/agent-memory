# ADR-001: Modular Monolith Over Microservices

**Status**: Accepted  
**Date**: 2026-06-07  
**Deciders**: Engineering  

## Context

Hystersis needs to ship fast while supporting memory, compression, skills, wiki, SSO, analytics, and webhooks. A full microservices split would multiply operational overhead before product-market fit is proven.

## Options Considered

| Option | Complexity | Cost | Scalability | Dev Speed |
|--------|-----------|------|-------------|-----------|
| A. Single monolith (`cmd/server`) | Low | Low | Vertical + replicas | Fastest |
| B. Service mesh (10+ services) | Very high | High | Excellent | Slow |
| C. Modular monolith + optional sidecars | Medium | Medium | Good | Fast |

## Decision

**Option C: Modular monolith** with optional standalone services (`gateway`, `mcp-server`, `connectors`, `memory-api`) that can be deployed independently when needed.

Core logic lives in `internal/` packages with clear boundaries. The monolith is the default deployment unit.

## Consequences

**Positive**
- Single codebase, shared types, simpler debugging
- Optional extraction path for MCP and connectors
- Matches current Docker Compose and Cloud Run deploys

**Negative**
- Monolith binary grows (~4700 lines in `api.go`)
- All features share the same deploy cycle unless split later

## Follow-up

Extract compression observability and archive tier to separate workers when load warrants it (see ADR-003).
