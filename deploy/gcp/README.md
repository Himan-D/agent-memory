# GCP / Firebase Deployment (Scaled Backend)

The Hystersis Go API runs on **Cloud Run** for auto-scaling (1–20+ instances). **Firebase Hosting** optionally fronts the API with global CDN + custom domains.

## Architecture

```
                    ┌─────────────────────┐
  api.hystersis.ai  │  Firebase Hosting   │ (optional CDN + SSL)
        or          │  rewrites → Cloud Run│
  *.run.app         └──────────┬──────────┘
                               │
                    ┌──────────▼──────────┐
                    │   Cloud Run         │
                    │   hystersis (Go)    │
                    │   min=1 max=20      │
                    └──────────┬──────────┘
           ┌───────────────────┼───────────────────┐
           ▼                   ▼                   ▼
    Neo4j Aura / VM      Qdrant Cloud / VM    Memorystore Redis
           │                   │                   │
           └───────────────────┴───────────────────┘
                    Secret Manager (credentials)
```

## Quick Deploy (recommended)

### Project: `avian-infusion-491311-b3`

Pre-configured files:
- `.firebaserc` — Firebase project
- `deploy/gcp/project.env` — `GCP_PROJECT_ID` and region
- `scripts/deploy-avian.sh` — one-command deploy wrapper

```bash
gcloud auth login
gcloud auth application-default login
source deploy/gcp/project.env
./scripts/deploy-avian.sh
```

### 1. Prerequisites

- GCP project with billing enabled (`avian-infusion-491311-b3`)
- [gcloud CLI](https://cloud.google.com/sdk/docs/install) authenticated
- Managed services OR Terraform-provisioned VMs:
  - Neo4j (Aura or self-hosted)
  - Qdrant (Cloud or self-hosted)
  - Redis (Memorystore or Upstash)

### 2. Configure secrets

```bash
export GCP_PROJECT_ID=your-project-id

export NEO4J_URI=bolt://your-neo4j:7687
export NEO4J_PASSWORD=your-password
export QDRANT_URL=https://your-qdrant:6333
export QDRANT_API_KEY=your-key          # optional
export LLM_API_KEY=sk-...
export REDIS_URL=redis://host:6379      # optional
export JWT_SECRET=$(openssl rand -hex 32)

# Stripe (from scripts/setup-stripe-plans.sh output)
export STRIPE_SECRET_KEY=sk_live_...
export STRIPE_WEBHOOK_SECRET=whsec_...
export STRIPE_PRO_PRICE_ID=price_...
export STRIPE_TEAM_PRICE_ID=price_...
```

### 3. Deploy

```bash
chmod +x scripts/*.sh deploy/gcp/*.sh
./scripts/deploy-gcp.sh
```

With Firebase Hosting proxy:

```bash
cp .firebaserc.example .firebaserc   # set your project ID
npm install -g firebase-tools
./scripts/deploy-gcp.sh --firebase
```

### 4. Custom domain

**Option A — Cloud Run (direct, lowest latency for API)**

```bash
gcloud run domain-mappings create \
  --service=hystersis \
  --domain=api.hystersis.ai \
  --region=us-central1
```

Add the DNS records shown by the command (usually CNAME → `ghs.googlehosted.com`).

**Option B — Firebase Hosting**

1. Firebase Console → Hosting → Add custom domain → `api.hystersis.ai`
2. Firebase automatically provisions SSL and proxies to Cloud Run via `firebase.json` rewrites.

### 5. Update Stripe webhook

If the API URL changed, update the webhook endpoint in [Stripe Dashboard](https://dashboard.stripe.com/webhooks):

```
https://api.hystersis.ai/stripe/webhook
```

## Scaling knobs

Edit substitutions in `deploy/gcp/cloudbuild.yaml` or pass at build time:

| Variable | Default | Purpose |
|----------|---------|---------|
| `_MIN_INSTANCES` | 1 | Keep warm instances (avoid cold starts) |
| `_MAX_INSTANCES` | 20 | Peak scale ceiling |
| `_MEMORY` | 2Gi | Per-instance memory |
| `_CPU` | 2 | vCPUs per instance |
| `_CONCURRENCY` | 100 | Requests per instance |

```bash
gcloud builds submit --config deploy/gcp/cloudbuild.yaml \
  --substitutions=_MIN_INSTANCES=2,_MAX_INSTANCES=50,_MEMORY=4Gi
```

## CI/CD (GitHub Actions)

Workflow: `.github/workflows/deploy-api.yml`

Required repository secrets:

| Secret | Description |
|--------|-------------|
| `GCP_PROJECT_ID` | GCP project ID |
| `GCP_SERVICE_ACCOUNT` | Deploy SA email |
| `GCP_WORKLOAD_IDENTITY_PROVIDER` | WIF provider for OIDC |

Trigger manually via **Actions → Deploy API (Cloud Run) → Run workflow**, or push to `main`/`master` when server code changes.

## Option 2: Full Infrastructure (Terraform)

Provisions VPC, Cloud Run, Neo4j VM, Qdrant VM, Redis, Secret Manager.

```bash
cd terraform
cp terraform.tfvars.example terraform.tfvars   # fill in project_id, llm_api_key, stripe_* etc.
gcloud auth application-default login
terraform init -backend-config=backend.hcl
terraform plan
terraform apply
```

Terraform fixes included:
- `NEO4J_URI` uses `bolt://IP:7687` (not bare IP)
- Cloud Run domain mapping + CNAME for custom domains
- Optional Stripe secrets via Secret Manager

## Container image

```bash
# Cloud Build (CI)
gcloud builds submit --config deploy/gcp/cloudbuild.yaml

# Manual
docker build -t us-central1-docker.pkg.dev/$PROJECT_ID/hystersis/api:latest -f docker/Dockerfile .
docker push us-central1-docker.pkg.dev/$PROJECT_ID/hystersis/api:latest
```

## Health checks

Cloud Run uses `/health` (liveness) and `/ready` (readiness). Verify after deploy:

```bash
curl -s "$(gcloud run services describe hystersis --region=us-central1 --format='value(status.url)')/health"
```

## Migrating from VM (api.hystersis.ai)

1. Deploy to Cloud Run (steps above)
2. Smoke-test on `*.run.app` URL
3. Point `api.hystersis.ai` DNS to Cloud Run or Firebase Hosting
4. Update Stripe webhook if URL changed
5. Decommission old VM/PM2 once stable
