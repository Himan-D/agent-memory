#!/usr/bin/env bash
# One-command deploy: Artifact Registry + Cloud Build + Cloud Run (+ optional Firebase Hosting).
#
# Usage:
#   export GCP_PROJECT_ID=your-project
#   ./scripts/deploy-gcp.sh
#
# Options:
#   --secrets-only   Only run setup-gcp-secrets.sh
#   --firebase       Also deploy Firebase Hosting rewrites to Cloud Run
#   --region REGION  Override region (default: us-central1)

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

PROJECT="${GCP_PROJECT_ID:-${GOOGLE_CLOUD_PROJECT:-}}"
REGION="${GCP_REGION:-us-central1}"
FIREBASE=false
SECRETS_ONLY=false

while [[ $# -gt 0 ]]; do
  case "$1" in
    --firebase) FIREBASE=true; shift ;;
    --secrets-only) SECRETS_ONLY=true; shift ;;
    --region) REGION="$2"; shift 2 ;;
    *) echo "Unknown option: $1"; exit 1 ;;
  esac
done

if [[ -z "$PROJECT" ]]; then
  echo "Error: set GCP_PROJECT_ID"
  exit 1
fi

if ! command -v gcloud >/dev/null 2>&1; then
  echo "Error: gcloud CLI not found. Install: https://cloud.google.com/sdk/docs/install"
  exit 1
fi

gcloud config set project "$PROJECT" --quiet

echo "==> Enabling required APIs..."
gcloud services enable \
  run.googleapis.com \
  cloudbuild.googleapis.com \
  artifactregistry.googleapis.com \
  secretmanager.googleapis.com \
  firebase.googleapis.com \
  --quiet

echo "==> Ensuring Artifact Registry repository..."
if ! gcloud artifacts repositories describe hystersis \
  --location="$REGION" --project="$PROJECT" >/dev/null 2>&1; then
  gcloud artifacts repositories create hystersis \
    --repository-format=docker \
    --location="$REGION" \
    --description="Hystersis API containers" \
    --quiet
fi

echo "==> Configuring secrets..."
bash "$ROOT/scripts/setup-gcp-secrets.sh"

if $SECRETS_ONLY; then
  echo "Secrets updated. Exiting (--secrets-only)."
  exit 0
fi

echo "==> Building and deploying via Cloud Build..."
gcloud builds submit \
  --config deploy/gcp/cloudbuild.yaml \
  --substitutions="_REGION=${REGION}" \
  --project="$PROJECT"

SERVICE_URL=$(gcloud run services describe hystersis \
  --region="$REGION" \
  --project="$PROJECT" \
  --format='value(status.url)')

echo ""
echo "API deployed: $SERVICE_URL"
echo "Health:       ${SERVICE_URL}/health"

if $FIREBASE; then
  if ! command -v firebase >/dev/null 2>&1; then
    echo "Warning: firebase CLI not installed; skip hosting deploy"
    exit 0
  fi
  if [[ ! -f .firebaserc ]]; then
    echo "Copy .firebaserc.example to .firebaserc and set your project ID"
    exit 1
  fi
  echo "==> Deploying Firebase Hosting (proxies to Cloud Run)..."
  firebase deploy --only hosting --project "$PROJECT"
  echo "Firebase Hosting live. Map custom domain in Firebase Console → Hosting."
fi

echo ""
echo "Next steps:"
echo "  1. Map api.hystersis.ai → Cloud Run (or Firebase Hosting)"
echo "  2. Update Stripe webhook URL if the API URL changed"
echo "  3. curl ${SERVICE_URL}/health"
