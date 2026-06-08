#!/usr/bin/env bash
# Build and deploy the Go monolith to Google Cloud Run, then print the URL
# to configure as the Cloudflare API Worker BACKEND_URL.
set -euo pipefail

PROJECT_ID="${PROJECT_ID:-$(gcloud config get-value project 2>/dev/null || true)}"
REGION="${REGION:-us-central1}"
SERVICE="${SERVICE:-hystersis-api}"
IMAGE="${IMAGE:-gcr.io/${PROJECT_ID}/${SERVICE}:$(git rev-parse --short HEAD)}"

if [ -z "${PROJECT_ID}" ]; then
  echo "error: set PROJECT_ID or run gcloud config set project <id>" >&2
  exit 1
fi

required_secrets=(
  NEO4J_URI
  NEO4J_PASSWORD
  QDRANT_URL
  QDRANT_API_KEY
  REDIS_URL
  LLM_API_KEY
  JWT_SECRET
)

for secret in "${required_secrets[@]}"; do
  if ! gcloud secrets describe "$secret" --project "$PROJECT_ID" >/dev/null 2>&1; then
    echo "error: missing GCP Secret Manager secret: $secret" >&2
    echo "       create it before deploying, for example:" >&2
    echo "       printf '%s' '<value>' | gcloud secrets create $secret --data-file=- --project $PROJECT_ID" >&2
    exit 1
  fi
done

echo "==> Building image: ${IMAGE}"
gcloud builds submit \
  --project "$PROJECT_ID" \
  --tag "$IMAGE" \
  --timeout 1200s

echo "==> Deploying Cloud Run service: ${SERVICE}"
gcloud run deploy "$SERVICE" \
  --project "$PROJECT_ID" \
  --region "$REGION" \
  --platform managed \
  --image "$IMAGE" \
  --port 8080 \
  --cpu 2 \
  --memory 2Gi \
  --min-instances 1 \
  --max-instances 10 \
  --timeout 300 \
  --concurrency 80 \
  --allow-unauthenticated \
  --set-env-vars "^|^ENVIRONMENT=production|AUTH_ENABLED=true|COMPRESSION_ENABLED=true|COMPRESSION_MODE=extract|MULTI_SIGNAL_ENABLED=true|API_BASE_URL=https://api.hystersis.com|ALLOWED_ORIGINS=https://hystersis.com,https://www.hystersis.com,https://app.hystersis.com" \
  --set-secrets "NEO4J_URI=NEO4J_URI:latest,NEO4J_PASSWORD=NEO4J_PASSWORD:latest,QDRANT_URL=QDRANT_URL:latest,QDRANT_API_KEY=QDRANT_API_KEY:latest,REDIS_URL=REDIS_URL:latest,LLM_API_KEY=LLM_API_KEY:latest,OPENAI_API_KEY=LLM_API_KEY:latest,JWT_SECRET=JWT_SECRET:latest"

url="$(gcloud run services describe "$SERVICE" --project "$PROJECT_ID" --region "$REGION" --format='value(status.url)')"
echo "==> Cloud Run URL: ${url}"
echo ""
echo "Next: configure Cloudflare API edge proxy:"
echo "  printf '%s' '${url}' | npx wrangler secret put BACKEND_URL --config wrangler-api.jsonc"
echo "  npx wrangler deploy --config wrangler-api.jsonc"
echo "  npx wrangler deploy"
