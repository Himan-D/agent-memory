# ADR-002: Cloudflare Workers for Frontend Edge Deployment

**Status**: Accepted  
**Date**: 2026-06-07  
**Deciders**: Engineering, DevOps  

## Context

Hystersis serves a marketing site, Mintlify docs, and a Next.js dashboard. We need global low-latency delivery, simple SSL, and minimal ops for frontends while the API runs on traditional compute.

## Options Considered

| Option | Complexity | Cost | Scalability | Security |
|--------|-----------|------|-------------|----------|
| A. Vercel (dashboard) + Netlify (landing) | Low | Medium | Auto | Good |
| B. Self-hosted K8s ingress | High | High | Manual | Full control |
| C. Cloudflare Workers + OpenNext | Medium | Low | Global edge | WAF + DDoS included |

## Decision

**Option C: Cloudflare Workers** for all public frontends.

- `agent-memory` worker → landing + docs (`hystersis.com`)
- `hystersis-app` worker → dashboard via OpenNext (`app.hystersis.com`)
- Unified build: `scripts/build-cloudflare.sh` deploys all three artifacts

API remains on `api.hystersis.com` (Cloud Run / self-hosted).

## Consequences

**Positive**
- Single Cloudflare account, unified DNS
- Workers Builds auto-deploy on push (no GitHub secrets required)
- Dashboard piggybacks on landing build via `deploy-dashboard-builds.sh`

**Negative**
- OpenNext Cloudflare adapter adds build complexity
- Dashboard deploy depends on landing Workers Builds watch paths
- GitHub Actions deploy requires `CLOUDFLARE_API_TOKEN` as fallback

## Verification

Post-deploy smoke tests in `scripts/verify-production.sh`:
- Dashboard JS bundle must not contain `demo@hystersis`
- Docs CSS must return HTTP 200 at `/docs/_next/static/chunks/*.css`
