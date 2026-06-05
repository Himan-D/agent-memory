#!/usr/bin/env bash
# Deploy Hystersis API to GCP project avian-infusion-491311-b3
#
# Prerequisites:
#   gcloud auth login
#   gcloud auth application-default login
#   Export secrets (see deploy/gcp/README.md)
#
# Usage:
#   ./scripts/deploy-avian.sh              # Cloud Run only
#   ./scripts/deploy-avian.sh --firebase   # Cloud Run + Firebase Hosting

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck source=/dev/null
source "$ROOT/deploy/gcp/project.env"

exec "$ROOT/scripts/deploy-gcp.sh" "$@"
