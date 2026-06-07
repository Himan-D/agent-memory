#!/usr/bin/env bash
# Verify production dashboard and docs after deploy.
set -euo pipefail

verify_dashboard_signin() {
  local attempt chunk js
  for attempt in $(seq 1 15); do
    local page
    page=$(curl -fsSL "https://app.hystersis.com/auth/signin" 2>/dev/null || echo "")

    if [ -z "$page" ]; then
      echo "Attempt $attempt: could not fetch sign-in page"
      sleep 30
      continue
    fi

    chunk=$(echo "$page" | grep -oE '/_next/static/chunks/app/auth/signin/page-[a-zA-Z0-9_-]+\.js' | head -1 || true)
    if [ -z "$chunk" ]; then
      echo "Attempt $attempt: sign-in JS chunk not found in HTML"
      sleep 30
      continue
    fi

    js=$(curl -fsSL "https://app.hystersis.com${chunk}" 2>/dev/null || echo "")
    if [ -z "$js" ]; then
      echo "Attempt $attempt: could not fetch ${chunk}"
      sleep 30
      continue
    fi

    if echo "$js" | grep -q 'demo@hystersis'; then
      echo "Attempt $attempt: bundle still contains demo credentials"
      sleep 30
      continue
    fi

    if echo "$js" | grep -q 'you@company.com'; then
      echo "Dashboard sign-in bundle is current (no demo credentials)."
      return 0
    fi

    echo "Attempt $attempt: bundle missing expected new UI markers"
    sleep 30
  done

  echo "::error::Dashboard sign-in still stale after 7.5 minutes."
  return 1
}

verify_docs_css() {
  local html css code
  html=$(curl -fsSL "https://hystersis.com/docs" 2>/dev/null || echo "")

  if [ -z "$html" ]; then
    echo "::error::Could not fetch https://hystersis.com/docs"
    return 1
  fi

  css=$(echo "$html" | grep -oE '/docs/_next/static/chunks/[a-f0-9]+\.css' | head -1 || true)
  if [ -z "$css" ]; then
    css=$(echo "$html" | grep -oE '/_next/static/chunks/[a-f0-9]+\.css' | head -1 || true)
    if [ -n "$css" ]; then
      css="/docs${css}"
    fi
  fi

  if [ -z "$css" ]; then
    echo "::error::No docs CSS path found in /docs HTML"
    return 1
  fi

  code=$(curl -sS -o /dev/null -w "%{http_code}" "https://hystersis.com${css}" || echo "000")
  if [ "$code" != "200" ]; then
    echo "::error::Docs CSS ${css} returned HTTP ${code}"
    return 1
  fi

  echo "Docs CSS OK: ${css} → HTTP 200"
  return 0
}

case "${1:-all}" in
  dashboard) verify_dashboard_signin ;;
  docs) verify_docs_css ;;
  all)
    verify_dashboard_signin
    verify_docs_css
    ;;
  *)
    echo "usage: $0 [dashboard|docs|all]" >&2
    exit 1
    ;;
esac
