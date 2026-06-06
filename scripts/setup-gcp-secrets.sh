#!/usr/bin/env bash
# Create or update Secret Manager secrets for Cloud Run deployment.
#
# Usage:
#   export GCP_PROJECT_ID=your-project
#   export NEO4J_URI=bolt://host:7687
#   export NEO4J_PASSWORD=...
#   export QDRANT_URL=https://...
#   export LLM_API_KEY=sk-...
#   export STRIPE_SECRET_KEY=sk_live_...
#   ./scripts/setup-gcp-secrets.sh
#
# Reads from environment; skips secrets whose value is empty.

set -euo pipefail

PROJECT="${GCP_PROJECT_ID:-${GOOGLE_CLOUD_PROJECT:-}}"
if [[ -z "$PROJECT" ]]; then
  echo "Error: set GCP_PROJECT_ID or GOOGLE_CLOUD_PROJECT"
  exit 1
fi

declare -A SECRETS=(
  [neo4j-uri]="${NEO4J_URI:-}"
  [neo4j-password]="${NEO4J_PASSWORD:-}"
  [qdrant-url]="${QDRANT_URL:-}"
  [qdrant-api-key]="${QDRANT_API_KEY:-}"
  [llm-api-key]="${LLM_API_KEY:-${OPENAI_API_KEY:-}}"
  [redis-url]="${REDIS_URL:-}"
  [jwt-secret]="${JWT_SECRET:-}"
  [stripe-secret-key]="${STRIPE_SECRET_KEY:-}"
  [stripe-webhook-secret]="${STRIPE_WEBHOOK_SECRET:-}"
  [stripe-pro-price-id]="${STRIPE_PRO_PRICE_ID:-}"
  [stripe-team-price-id]="${STRIPE_TEAM_PRICE_ID:-}"
  [sentry-dsn]="${SENTRY_DSN:-}"
)

upsert_secret() {
  local name="hystersis-$1"
  local value="$2"
  if [[ -z "$value" ]]; then
    echo "skip $name (empty)"
    return 0
  fi
  if gcloud secrets describe "$name" --project="$PROJECT" >/dev/null 2>&1; then
    echo -n "$value" | gcloud secrets versions add "$name" --project="$PROJECT" --data-file=-
    echo "updated $name"
  else
    echo -n "$value" | gcloud secrets create "$name" --project="$PROJECT" --replication-policy=automatic --data-file=-
    echo "created $name"
  fi
}

echo "Project: $PROJECT"
for key in "${!SECRETS[@]}"; do
  upsert_secret "$key" "${SECRETS[$key]}"
done

# Grant Cloud Run default SA access (adjust SA email if using custom)
PROJECT_NUMBER=$(gcloud projects describe "$PROJECT" --format='value(projectNumber)')
SA="${PROJECT_NUMBER}-compute@developer.gserviceaccount.com"
for key in "${!SECRETS[@]}"; do
  name="hystersis-$key"
  if gcloud secrets describe "$name" --project="$PROJECT" >/dev/null 2>&1; then
    gcloud secrets add-iam-policy-binding "$name" \
      --project="$PROJECT" \
      --member="serviceAccount:${SA}" \
      --role="roles/secretmanager.secretAccessor" \
      --quiet >/dev/null 2>&1 || true
  fi
done

echo "Done. Secrets ready for Cloud Run --set-secrets mapping."
