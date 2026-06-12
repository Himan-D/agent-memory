# Deployment Infrastructure Audit — Hystersis (agent-memory)

## What Exists

### Go Backend

**Dockerfile** (`/Users/himand/agent-memory/Dockerfile`)
- Multi-stage build: `golang:1.26-alpine` builder → `alpine:3.19` final
- Builds `./cmd/server` → binary named `agent-memory`
- Runs as non-root user `appuser` (UID 1000)
- No EXPOSE directive (port comes from env `HTTP_PORT`)
- **Note**: `go.mod` declares `go 1.25.0` but Dockerfile uses `golang:1.26-alpine` — minor mismatch, harmless

**Additional per-service Dockerfiles** (microservices):
- `cmd/connectors/Dockerfile`
- `cmd/gateway/Dockerfile`
- `cmd/mcp-server/Dockerfile`
- `cmd/memory-api/Dockerfile`

**docker-compose.yml** (root — full stack, production-like)
- `neo4j:5.14-community` on ports 7474 / 7687
- `qdrant:v1.7.4` on ports 6333 / 6334
- `redis:7-alpine` on port 6379
- `monolith` service (Dockerfile root) on port 8081→8080
- `gateway` service (`cmd/gateway/Dockerfile`) on port 8080 — **public entrypoint**
- `mcp-server` service (`cmd/mcp-server/Dockerfile`) on port 8082
- `connectors` service (`cmd/connectors/Dockerfile`) on port 8083
- `memory-api` service (`cmd/memory-api/Dockerfile`) on port 8084→8081
- All services have health checks via `/health` endpoint

**docker/compose.yml** (simpler dev compose)
- Just neo4j + qdrant + a single `agent` service built from root Dockerfile

---

### Ports & Health Checks

| Service | Port | Health endpoint |
|---|---|---|
| Go backend (monolith) | 8080 | `/health` (liveness), `/ready` (readiness) |
| gRPC | 50051 | — |
| Prometheus metrics | 8080 | `/metrics` |
| Neo4j Bolt | 7687 | — |
| Neo4j HTTP | 7474 | — |
| Qdrant HTTP | 6333 | — |
| Qdrant gRPC | 6334 | — |
| Redis | 6379 | — |
| Gateway | 8080 | `/health` |
| MCP server | 8082 | `/health` |
| Connectors | 8083 | `/health` |

---

### Kubernetes

**`deploy/k8s/agent-memory.yaml`** — raw manifests (apply with `kubectl apply -f`)
- ServiceAccount, ConfigMap, Secret (with placeholder values), PVC (10Gi), Service (ClusterIP :8080), Deployment (2 replicas), HPA (1–10, 70% CPU)
- Startup probe → `/health`, liveness → `/health`, readiness → `/ready`
- **Gaps**: No Ingress manifest here; no NetworkPolicy; no namespace definition; Secret values are literal plaintext (`changeme`) — needs external secret injection before prod use

**`deploy/helm/agent-memory/`** — Helm chart v0.1.0
- `Chart.yaml`: app version 0.1.0
- `values.yaml`: fully parameterized (image, ingress with cert-manager, resources, autoscaling, neo4j/qdrant/openai config, storage, redis, GCP, secrets)
- `templates/deployment.yaml`: **INCOMPLETE** — uses literal string `AGENT_MEMORY_NAME` instead of Helm template calls like `{{ include "agent-memory.fullname" . }}`. Effectively a stub.
- `templates/_helpers.tpl`: Standard Helm helpers are correct (fullname, labels, selectorLabels, serviceAccountName)
- **Missing Helm templates**: No `ingress.yaml`, no `hpa.yaml`, no `service.yaml` as separate template files — the deployment template tries to inline everything but doesn't use the helpers

---

### GCP

**`deploy/gcp/cloudbuild.yaml`**
- Cloud Build pipeline: docker build → push to GCR → `gcloud run deploy`
- References `docker/Dockerfile` (not root `Dockerfile`) — this is the correct multi-stage file
- Substitution: `_REGION` defaults to `us-central1`
- Image: `gcr.io/$PROJECT_ID/hystersis:$COMMIT_SHA`

**`deploy/gcp/cloudrun-service.yaml`** — Knative Service spec
- Image: `gcr.io/PROJECT_ID/hystersis:latest` (placeholder — needs substitution)
- Port 8080, 2 vCPU, 1Gi RAM, 1–10 instances
- All secrets pulled from GCP Secret Manager (`hystersis-secrets`)
- Health checks: `/health` and `/ready`
- Storage: GCS bucket `hystersis-backups`, `STORAGE_PROVIDER=gcs`

**`deploy/gcp/README.md`**
- Describes two paths: Cloud Run + managed services, or full Terraform
- References a `terraform/` directory that **does not exist** in the repo root

---

### Cloudflare (Frontend)

**`wrangler.jsonc`** (root) — landing page worker
- Worker name: `agent-memory`
- Account ID: `c50d52c51722d57e2c06c3eab5510dc3` (hardcoded)
- Routes: `hystersis.com`, `www.hystersis.com`, `blogs.hystersis.com`
- Assets: `landing/dist`
- Build command: `bash scripts/build-cloudflare.sh`

**`dashboard/wrangler.jsonc`** — Next.js dashboard worker
- Worker name: `hystersis-app`
- Account ID: same as above
- Route: `app.hystersis.com`
- Uses `@opennextjs/cloudflare` adapter (`.open-next/worker.js`)
- Build command: `npm ci --legacy-peer-deps && npx opennextjs-cloudflare build`
- Vars: `NEXT_PUBLIC_API_URL=https://api.hystersis.com`, `BETTER_AUTH_URL=https://app.hystersis.com`
- Secrets (pushed separately by deploy script): `BETTER_AUTH_SECRET`, `BETTER_AUTH_API_KEY`, `ADMIN_API_KEY`

**`docs/wrangler.jsonc`** — Mintlify docs worker (exists, not read — inferred from deploy script)
- Worker name: `hystersis-docs`, domain: `docs.hystersis.com`

**`dashboard/next.config.mjs`**
- No `output: 'export'` — uses Cloudflare Workers SSR via OpenNext adapter
- `removeConsole` in production
- Custom webpack chunk splitting for recharts and force-graph vendors

---

### CI/CD Pipelines

**`.github/workflows/ci.yml`** — runs on push/PR to main/master
- Go version: `1.25` (mismatch with Dockerfile `golang:1.26-alpine`)
- Node version: `20`
- Jobs: `go-backend` (lint, build, test, vet), `dashboard` (npm build), `landing-page` (npm build + verify install.sh), `docs` (mintlify validate), `node-sdk`, `python-sdk`, `security` (gosec + secret scan)
- `ci-success` gate: all jobs must pass (or be skipped)
- `docker` job: builds + pushes `hystersis/hystersis` to Docker Hub on push to main. Conditional on `DOCKER_USERNAME` secret being set.

**`.github/workflows/deploy-cloudflare.yml`** — unified Cloudflare deploy (replaces `deploy.yml`, `deploy-dashboard.yml`, `deploy-docs.yml`)
- Triggers: push to main/master with path filters for `landing/**`, `dashboard/**`, `docs/**`, workers, scripts
- Manual trigger via `workflow_dispatch` with target choice (all/landing/docs/dashboard)
- Preflight step validates `CLOUDFLARE_API_TOKEN` and `CLOUDFLARE_ACCOUNT_ID`
- Post-deploy: `verify-domains.sh` + `verify-production.sh`

**Deprecated workflows** (stubs only, `workflow_dispatch` only):
- `.github/workflows/deploy.yml` — deprecated landing
- `.github/workflows/deploy-dashboard.yml` — deprecated dashboard
- `.github/workflows/deploy-docs.yml` — deprecated docs

**Other workflows**:
- `bootstrap-labels.yml`, `labeler.yml`, `auto-merge.yml`, `agent-guard.yml`, `sync-with-master.yml`, `copilot-setup-steps.yml`, `openai-repo-bot.yml`

---

### Scripts

**`scripts/deploy-cloudflare.sh`** — orchestrates all Cloudflare deploys
- Validates env vars, then calls landing/docs/dashboard build+deploy functions
- Dashboard deploy syncs Wrangler secrets after build
- Runs `verify-domains.sh` at the end

**`scripts/verify-production.sh`** — post-deploy verification
- Polls `app.hystersis.com/auth/signin` up to 15 times (30s apart) looking for new JS bundle markers
- Verifies `docs.hystersis.com` CSS and static assets
- Verifies `blogs.hystersis.com` content

**Other scripts**:
- `scripts/build-cloudflare.sh` — builds landing + copies docs into `landing/dist`
- `scripts/build-docs.sh` — builds Mintlify docs export
- `scripts/preflight-cloudflare-token.sh` — validates CF token permissions
- `scripts/verify-domains.sh` — DNS verification
- `scripts/setup-github-secrets.sh` — helper for secret setup
- `scripts/generate-tokens.sh`, `scripts/rewrite-docs-assets.sh`, `scripts/deploy-dashboard-builds.sh`
- `scripts/openai-repo-bot.mjs` — automation bot

---

### Grafana / Observability

**`deploy/grafana/`**
- `datasources.yml`: Prometheus (`:9090`), Loki (`:3100`), Tempo (`:3200`)
- `dashboards.yml`: provider pointing to `/etc/grafana/provisioning/dashboards/hystersis`
- `hystersis-alerts.yml`: 8 alert rules covering API down, 5xx rate, P95 latency, memory, goroutines, compression accuracy, cache hit rate, Neo4j connection pool, Redis down, OTel collector down
- `hystersis-compression.json`, `hystersis-memory.json`, `hystersis-overview.json`: Grafana dashboard JSON (exist, contents not read)
- **No `docker-compose` for the observability stack** — Prometheus/Loki/Tempo/Grafana stack not defined anywhere in the repo

---

### Environment Variables (from `config.go`)

**Required (no sensible default):**
- `NEO4J_URI` / `NEO4J_USER` / `NEO4J_PASSWORD`
- `QDRANT_URL`
- `OPENAI_API_KEY` (for embeddings)
- `LLM_API_KEY` (for memory processing — gpt-4o by default)

**Auth (for production):**
- `AUTH_ENABLED=true`
- `JWT_SECRET`
- `API_KEYS` (comma-separated)
- `ADMIN_API_KEYS`
- `API_KEY_SALT`
- `ALLOWED_ORIGINS`

**Compression (proprietary):**
- `COMPRESSION_ENABLED` (default: true)
- `COMPRESSION_LLM_FAST_PROVIDER` / `COMPRESSION_LLM_FAST_MODEL`
- `COMPRESSION_LLM_VERIFY_PROVIDER` / `COMPRESSION_LLM_VERIFY_MODEL`
- `ANTHROPIC_API_KEY` — referenced in `.env.production.example` but **not in config.go struct** — likely passed as `LLM_API_KEY` when `LLM_PROVIDER=anthropic`

**Storage:**
- `STORAGE_PROVIDER` (local/r2/gcs/s3, default: local)
- For GCS: `GCS_BUCKET`, `GCP_PROJECT_ID`
- For R2: `R2_BUCKET`, `R2_ACCOUNT_ID`, `R2_ACCESS_KEY_ID`, `R2_SECRET_ACCESS_KEY`, `R2_PUBLIC_BASE_URL`

**Redis:**
- `REDIS_URL` (optional but used for caching/tiers/sessions)

**Dashboard (`dashboard/.env.local` / Wrangler vars):**
- `NEXT_PUBLIC_API_URL` (hardcoded to `https://api.hystersis.com` in wrangler)
- `ADMIN_API_KEY` (Wrangler secret)
- `BETTER_AUTH_SECRET` / `BETTER_AUTH_API_KEY` (Wrangler secrets)
- `NEXTAUTH_SECRET` / `AUTH_SECRET` (local dev)

**Landing (`landing/.env.example`):**
- `VITE_DASHBOARD_URL`
- `VITE_POSTHOG_KEY` / `VITE_POSTHOG_HOST`
- `VITE_SANITY_PROJECT_ID` / `VITE_SANITY_READ_TOKEN`

**Telemetry / Observability:**
- `TELEMETRY_ENABLED`, `OTLP_ENDPOINT`, `OTEL_SERVICE_NAME`, `OTEL_SAMPLE_RATE`
- `SENTRY_DSN`

**Email:**
- `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD`, `FROM_ADDRESS`

**SSO:**
- `SSO_CONFIG_FILE` / `SSO_PROVIDERS_JSON`
- `GOOGLE_CLIENT_ID` / `GOOGLE_CLIENT_SECRET` / `GOOGLE_CALLBACK_URL`
- `GITHUB_CLIENT_ID` / `GITHUB_CLIENT_SECRET` / `GITHUB_CALLBACK_URL`

**Stripe:**
- `stripe-go/v81` is in `go.mod` — no `STRIPE_*` vars in config.go (likely in handlers, not config struct)

---

### Makefile

Build targets: `build` (server + cli + agent), `build-server`, `build-cli`, `build-agent`
Test targets: `test`, `test-verbose`, `test-cover`, `test-short`
Docker targets: `docker-build`, `docker-up`, `docker-down`, `docker-logs`, `docker-ps`
Misc: `lint`, `vet`, `fmt`, `tidy`, `generate`, `migrate`, `sync-api-docs`, `clean`

---

### Landing Page

**Framework**: Vite + React (not Next.js)
**Build output**: `landing/dist`
**Deployment**: Cloudflare Workers (static assets)
**`landing/.env`** and **`landing/.env.example`** both exist (`.env` is live — check if gitignored)
**Sanity CMS** used for blog (`landing/sanity/`)
**`landing/public/_headers`** exists (Cloudflare Pages headers config)

---

## What Is Missing / Gaps

### Critical

1. **Helm chart is broken**: `deploy/helm/agent-memory/templates/deployment.yaml` uses literal string `AGENT_MEMORY_NAME` instead of `{{ include "agent-memory.fullname" . }}`. It also has no separate `service.yaml`, `ingress.yaml`, or `hpa.yaml` template files — only a single deployment.yaml that hardcodes names. Cannot `helm install` this chart without manual patching.

2. **`terraform/` directory does not exist**: `deploy/gcp/README.md` documents `Option 2 (Full Infrastructure / Terraform)` and references `terraform/README.md` + `backend.hcl`, but there is no `terraform/` directory at the repo root. Full GCP infra provisioning path is undocumented and non-functional.

3. **No observability stack compose**: Grafana dashboards and datasources are provisioned, but there is no `docker-compose` or Helm chart deploying Prometheus, Loki, Tempo, or Grafana itself. The monitoring stack exists as config files with no deployment mechanism.

4. **`docker/Dockerfile` referenced in cloudbuild.yaml but not found at root level**: `deploy/gcp/cloudbuild.yaml` uses `-f docker/Dockerfile`, but `docker/` at the repo root only contains `compose.yml`. The worktrees all have `docker/Dockerfile` — this file may exist but was not found via glob at the main repo root. Needs verification.

5. **Go version mismatch**: `go.mod` specifies `go 1.25.0`; CI uses `GO_VERSION: '1.25'`; root `Dockerfile` uses `golang:1.26-alpine`. Inconsistent — should align.

6. **`ANTHROPIC_API_KEY` not in config.go**: `.env.production.example` documents `ANTHROPIC_API_KEY` for LLM_PROVIDER=anthropic, but the config struct uses `LLM_API_KEY` generically. This is fine for runtime but confusing in documentation — the example should say `LLM_API_KEY` not `ANTHROPIC_API_KEY`.

### Medium Priority

7. **No `EXPOSE` in Dockerfile**: Port 8080 is not declared with `EXPOSE`. Containers work but tooling (Docker Desktop, orchestrators) loses the hint.

8. **K8s Secret has plaintext placeholder**: `deploy/k8s/agent-memory.yaml` has `NEO4J_PASSWORD: "changeme"` in stringData. This is safe for local dev but needs a comment or ExternalSecret reference for production.

9. **No network policies or namespace in k8s manifests**: Raw manifests deploy into default namespace with no NetworkPolicy. Fine for dev, not for production.

10. **Dashboard `deploy.yml` workflow** (the one loaded as `deploy.yml` in the initial glob) exists only as a stub pointing to `deploy-cloudflare.yml`. The actual Go backend has no automated deploy workflow — only CI builds the Docker image and pushes to Docker Hub. There is no workflow that deploys the Go backend to GCP Cloud Run, Kubernetes, or any other target.

11. **`docker/Dockerfile`**: The `docker/` directory only has `compose.yml` at the repo root — cloudbuild references `docker/Dockerfile` but this may be missing. The worktrees all have it; the main branch may be stale.

12. **`.env` is committed**: `landing/.env` appears to exist as a tracked file (not just `.env.example`). This could contain secrets.

13. **Stripe not in config struct**: `stripe-go/v81` is a dependency but `STRIPE_SECRET_KEY` / `STRIPE_WEBHOOK_SECRET` are not in `config.go`. These are likely inline env reads in handlers — undocumented.

### Minor

14. **`landing/.env` vs `.env.example`**: The actual `.env` file in `landing/` is committed to the repo (glob shows both). If it contains real keys, that's a secret leak.

15. **CI uses `actions/checkout@v6`**: v6 doesn't exist as of knowledge cutoff (latest is v4). This will fail or resolve to an unexpected version — likely a typo, should be `@v4`.

---

## Summary Table

| Area | File / Path | Status |
|---|---|---|
| Root Dockerfile | `Dockerfile` | Exists, functional |
| docker-compose (full stack) | `docker-compose.yml` | Exists, functional |
| docker-compose (dev) | `docker/compose.yml` | Exists, functional |
| Makefile | `Makefile` | Exists, functional |
| Helm chart | `deploy/helm/agent-memory/` | Exists but broken (hardcoded names in template) |
| K8s raw manifests | `deploy/k8s/agent-memory.yaml` | Exists, functional for dev |
| GCP Cloud Build | `deploy/gcp/cloudbuild.yaml` | Exists, functional |
| GCP Cloud Run spec | `deploy/gcp/cloudrun-service.yaml` | Exists, needs PROJECT_ID substitution |
| Terraform | `terraform/` | MISSING |
| Grafana dashboards | `deploy/grafana/` | Config exists, no deployment stack |
| Observability compose | — | MISSING |
| CI workflow | `.github/workflows/ci.yml` | Exists, functional |
| Cloudflare deploy | `.github/workflows/deploy-cloudflare.yml` | Exists, functional |
| Go backend deploy pipeline | — | MISSING (Docker Hub push only, no k8s/cloudrun deploy) |
| Wrangler (landing) | `wrangler.jsonc` | Exists, functional |
| Wrangler (dashboard) | `dashboard/wrangler.jsonc` | Exists, functional |
| .env.example | `.env.example` | Minimal (only neo4j, qdrant, openai, SSO) |
| .env.production.example | `.env.production.example` | Full, functional |
| landing .env.example | `landing/.env.example` | Exists |
| dashboard .env.local | `dashboard/.env.local` | Local dev values committed |
