#!/bin/bash
# Generate bootstrap credentials for Hystersis API + Dashboard.
# Usage: bash scripts/generate-tokens.sh [--write .env.path]
set -euo pipefail

WRITE_PATH=""
if [ "${1:-}" = "--write" ] && [ -n "${2:-}" ]; then
  WRITE_PATH="$2"
fi

if command -v go >/dev/null 2>&1; then
  BUNDLE=$(go run ./cmd/tokengen 2>/dev/null || true)
fi

if [ -z "${BUNDLE:-}" ]; then
  # Fallback when Go is unavailable
  rand_hex() { openssl rand -hex "$1"; }
  SALT=$(rand_hex 32)
  ADMIN_SHA=$(openssl rand 32 | openssl dgst -sha256 | awk '{print $2}')
  USER_SHA=$(openssl rand 32 | openssl dgst -sha256 | awk '{print $2}')
  SESSION=$(openssl rand -base64 32 | tr -d '\n/+=' | head -c 43)
  NEXTAUTH=$(rand_hex 32)
  JWT=$(rand_hex 32)
  ADMIN="am_admin_${ADMIN_SHA}"
  USER="usr_${USER_SHA}"
else
  SALT=$(echo "$BUNDLE" | jq -r '.api_key_salt')
  ADMIN=$(echo "$BUNDLE" | jq -r '.admin_api_key')
  USER=$(echo "$BUNDLE" | jq -r '.user_api_key')
  SESSION=$(echo "$BUNDLE" | jq -r '.session_token')
  NEXTAUTH=$(echo "$BUNDLE" | jq -r '.nextauth_secret')
  JWT=$(echo "$BUNDLE" | jq -r '.jwt_secret')
fi

OUTPUT=$(cat <<EOF
# Hystersis bootstrap credentials — generated $(date -u +%Y-%m-%dT%H:%M:%SZ)
API_KEY_SALT=${SALT}
ADMIN_API_KEYS=${ADMIN}
ADMIN_API_KEY=${ADMIN}
API_KEYS=${USER}:default
JWT_SECRET=${JWT}

# Dashboard (app.hystersis.com)
BETTER_AUTH_SECRET=${NEXTAUTH}
BETTER_AUTH_URL=https://app.hystersis.com
NEXT_PUBLIC_API_URL=https://api.hystersis.com

# Example session token (for testing only — real tokens come from POST /auth/login)
# SESSION_TOKEN=${SESSION}
EOF
)

echo "$OUTPUT"

if [ -n "$WRITE_PATH" ]; then
  mkdir -p "$(dirname "$WRITE_PATH")"
  echo "$OUTPUT" >> "$WRITE_PATH"
  echo "" >&2
  echo "Appended credentials to $WRITE_PATH" >&2
fi
