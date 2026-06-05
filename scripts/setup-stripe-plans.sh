#!/usr/bin/env bash
# Create Hystersis Pro and Team Stripe products + recurring prices.
# Usage:
#   STRIPE_SECRET_KEY=sk_test_... ./scripts/setup-stripe-plans.sh
# Or export STRIPE_SECRET_KEY first, then run the script.

set -euo pipefail

if [[ -z "${STRIPE_SECRET_KEY:-}" ]]; then
  echo "Error: STRIPE_SECRET_KEY is required"
  echo "Get it from https://dashboard.stripe.com/apikeys"
  exit 1
fi

API="https://api.stripe.com/v1"
auth=(-u "${STRIPE_SECRET_KEY}:")

create_product() {
  local name="$1" desc="$2" tier="$3"
  curl -s "${auth[@]}" "$API/products" \
    -d "name=$name" \
    -d "description=$desc" \
    -d "metadata[tier]=$tier" \
    | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('id','')) if 'id' in d else (_ for _ in ()).throw(Exception(d.get('error',{}).get('message','product create failed')))"
}

create_price() {
  local product="$1" amount="$2" tier="$3"
  curl -s "${auth[@]}" "$API/prices" \
    -d "product=$product" \
    -d "unit_amount=$amount" \
    -d "currency=usd" \
    -d "recurring[interval]=month" \
    -d "metadata[tier]=$tier" \
    | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('id','')) if 'id' in d else (_ for _ in ()).throw(Exception(d.get('error',{}).get('message','price create failed')))"
}

echo "Creating Stripe products and prices..."

PRO_PRODUCT=$(create_product "Hystersis Pro" "Production AI memory — 50k memories, 10 agents, compression engine" "pro")
PRO_PRICE=$(create_price "$PRO_PRODUCT" 2900 "pro")

TEAM_PRODUCT=$(create_product "Hystersis Team" "Team collaboration — 200k memories, 50 agents, audit logs" "team")
TEAM_PRICE=$(create_price "$TEAM_PRODUCT" 9900 "team")

echo ""
echo "✅ Stripe plans created successfully"
echo ""
echo "Add these to your API server environment:"
echo ""
echo "STRIPE_PRO_PRICE_ID=$PRO_PRICE"
echo "STRIPE_TEAM_PRICE_ID=$TEAM_PRICE"
echo ""
echo "Webhook endpoint (register in Stripe Dashboard → Developers → Webhooks):"
echo "  URL: https://api.hystersis.ai/stripe/webhook"
echo "  Events: checkout.session.completed, customer.subscription.deleted, invoice.payment_succeeded, invoice.payment_failed"
echo ""
echo "Dashboard: https://dashboard.stripe.com/products"
