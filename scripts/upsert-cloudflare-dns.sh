#!/usr/bin/env bash
# Upsert proxied Cloudflare CNAME records for Worker custom-domain hostnames.
set -euo pipefail

ZONE_NAME="${ZONE_NAME:-hystersis.com}"
DNS_TARGET="${DNS_TARGET:-hystersis.com}"
DNS_HOSTS="${DNS_HOSTS:-api.hystersis.com}"

if [ -z "${CLOUDFLARE_API_TOKEN:-}" ]; then
  echo "error: set CLOUDFLARE_API_TOKEN with Zone DNS Edit permission" >&2
  exit 1
fi

api() {
  curl -fsS \
    -H "Authorization: Bearer ${CLOUDFLARE_API_TOKEN}" \
    -H "Content-Type: application/json" \
    "$@"
}

zone_json="$(api "https://api.cloudflare.com/client/v4/zones?name=${ZONE_NAME}")"
zone_id="$(ZONE_JSON="$zone_json" node -e '
const data = JSON.parse(process.env.ZONE_JSON);
const zone = data.result && data.result[0];
if (!zone) process.exit(1);
process.stdout.write(zone.id);
')"

for host in $DNS_HOSTS; do
  echo "==> Upserting proxied CNAME: ${host} -> ${DNS_TARGET}"
  record_json="$(api "https://api.cloudflare.com/client/v4/zones/${zone_id}/dns_records?name=${host}")"
  record_id="$(RECORD_JSON="$record_json" node -e '
const data = JSON.parse(process.env.RECORD_JSON);
const record = data.result && data.result[0];
process.stdout.write(record ? record.id : "");
')"
  payload="$(HOST="$host" TARGET="$DNS_TARGET" node -e '
console.log(JSON.stringify({
  type: "CNAME",
  name: process.env.HOST,
  content: process.env.TARGET,
  ttl: 1,
  proxied: true
}));
')"

  if [ -n "$record_id" ]; then
    api -X PUT \
      --data "$payload" \
      "https://api.cloudflare.com/client/v4/zones/${zone_id}/dns_records/${record_id}" >/dev/null
  else
    api -X POST \
      --data "$payload" \
      "https://api.cloudflare.com/client/v4/zones/${zone_id}/dns_records" >/dev/null
  fi
done

echo "==> DNS upsert complete"
