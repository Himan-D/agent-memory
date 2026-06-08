---
title: GCP Cloud Run API
description: Deploy the Hystersis Go backend to Cloud Run behind the Cloudflare API Worker
---

# GCP Cloud Run API Deployment

This is the production path for the Go backend behind `https://api.hystersis.com`.

## Architecture

- Cloudflare owns public DNS and routes `api.hystersis.com` to the root Worker.
- The root Worker delegates API traffic to `hystersis-api`.
- `hystersis-api` proxies to Cloud Run when `BACKEND_URL` is configured.
- Cloud Run runs the Go monolith from `Dockerfile`.
- Neo4j, Qdrant, and Redis are external managed services or private GCP services.

## Required Secrets

Create these in GCP Secret Manager before deployment:

- `NEO4J_URI`
- `NEO4J_PASSWORD`
- `QDRANT_URL`
- `QDRANT_API_KEY`
- `REDIS_URL`
- `LLM_API_KEY`
- `JWT_SECRET`

## Deploy

```bash
gcloud auth login
gcloud config set project <project-id>
PROJECT_ID=<project-id> REGION=us-central1 bash scripts/deploy-gcp-cloud-run.sh
```

Then copy the printed Cloud Run URL into Cloudflare:

```bash
printf '%s' 'https://<cloud-run-service-url>' \
  | npx wrangler secret put BACKEND_URL --config wrangler-api.jsonc

npx wrangler deploy --config wrangler-api.jsonc
npx wrangler deploy
```

## Verify

```bash
curl -fsSL https://api.hystersis.com/health
curl -fsSL https://api.hystersis.com/ready
curl -fsSL https://api.hystersis.com/openapi.json
```

`/health` should show `"backend_configured": true` once the Worker secret is set.
