#!/usr/bin/env bash
# Called from Cloud Build to deploy Cloud Run with optional Secret Manager bindings.

set -euo pipefail

IMAGE="$1"
REGION="${_REGION:-us-central1}"
SERVICE="${_SERVICE:-hystersis}"
PROJECT="${PROJECT_ID:?}"

ARGS=(
  run deploy "$SERVICE"
  --image="$IMAGE"
  --region="$REGION"
  --platform=managed
  --project="$PROJECT"
  --port=8080
  --memory="${_MEMORY:-2Gi}"
  --cpu="${_CPU:-2}"
  --min-instances="${_MIN_INSTANCES:-1}"
  --max-instances="${_MAX_INSTANCES:-20}"
  --concurrency="${_CONCURRENCY:-100}"
  --timeout=300
  --cpu-boost
  --allow-unauthenticated
  --set-env-vars="ENVIRONMENT=production,HTTP_PORT=:8080,COMPRESSION_ENABLED=true,COMPRESSION_MODE=extract,MULTI_SIGNAL_ENABLED=true,AUTH_ENABLED=true,ALLOWED_ORIGINS=${_ALLOWED_ORIGINS:-https://app.hystersis.com,https://hystersis.com,https://hystersis.ai}"
)

# Map env var -> secret id (hystersis-*)
declare -A SECRET_MAP=(
  [NEO4J_URI]=neo4j-uri
  [NEO4J_PASSWORD]=neo4j-password
  [QDRANT_URL]=qdrant-url
  [QDRANT_API_KEY]=qdrant-api-key
  [LLM_API_KEY]=llm-api-key
  [REDIS_URL]=redis-url
  [JWT_SECRET]=jwt-secret
  [STRIPE_SECRET_KEY]=stripe-secret-key
  [STRIPE_WEBHOOK_SECRET]=stripe-webhook-secret
  [STRIPE_PRO_PRICE_ID]=stripe-pro-price-id
  [STRIPE_TEAM_PRICE_ID]=stripe-team-price-id
  [SENTRY_DSN]=sentry-dsn
)

for env_name in "${!SECRET_MAP[@]}"; do
  secret_id="hystersis-${SECRET_MAP[$env_name]}"
  if gcloud secrets describe "$secret_id" --project="$PROJECT" >/dev/null 2>&1; then
    ARGS+=(--set-secrets="${env_name}=${secret_id}:latest")
  fi
done

if [[ -n "${_VPC_CONNECTOR:-}" ]]; then
  ARGS+=(--vpc-connector="${_VPC_CONNECTOR}" --vpc-egress=private-ranges-only)
fi

gcloud "${ARGS[@]}"

gcloud run services describe "$SERVICE" \
  --region="$REGION" \
  --project="$PROJECT" \
  --format='value(status.url)'
