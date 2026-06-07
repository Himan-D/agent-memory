"use client";

import { useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Check, CreditCard, Loader2, ExternalLink } from "lucide-react";
import { useSession } from "next-auth/react";
import { billingApi, type BillingPlan, type BillingSubscription } from "@/lib/api";

const FALLBACK_PLANS: BillingPlan[] = [
  {
    id: "free",
    name: "Self-Hosted",
    price_per_seat: 0,
  },
  {
    id: "pro",
    name: "Pro",
    price_per_seat: 29,
  },
  {
    id: "team",
    name: "Team",
    price_per_seat: 99,
  },
  {
    id: "enterprise",
    name: "Enterprise",
    price_per_seat: 0,
  },
];

const PLAN_FEATURES: Record<string, string[]> = {
  free: ["1,000 memories", "10,000 searches", "2 agents", "Self-hosted deployment"],
  pro: ["50,000 memories", "100,000 searches", "10 agents", "Compression engine", "Skill chains"],
  team: ["200,000 memories", "500,000 searches", "50 agents", "Audit logging", "Advanced analytics"],
  enterprise: ["Unlimited usage", "SSO/SAML/OIDC/LDAP", "Dedicated support", "Custom SLA"],
};

function formatTierName(tier: string): string {
  const names: Record<string, string> = {
    free: "Free / Self-Hosted",
    pro: "Pro",
    team: "Team",
    enterprise: "Enterprise",
  };
  return names[tier] || tier;
}

export default function BillingPage() {
  const { data: session } = useSession();
  const [loading, setLoading] = useState<string | null>(null);
  const [plans, setPlans] = useState<BillingPlan[]>(FALLBACK_PLANS);
  const [subscription, setSubscription] = useState<BillingSubscription | null>(null);
  const [stripeEnabled, setStripeEnabled] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const load = async () => {
      try {
        const [plansRes, subRes] = await Promise.all([
          billingApi.getPlans(),
          billingApi.getSubscription(),
        ]);
        if (plansRes.plans?.length) {
          setPlans(plansRes.plans);
        }
        setStripeEnabled(plansRes.stripe_enabled);
        setSubscription(subRes);
      } catch (err) {
        console.error("Failed to load billing data:", err);
      }
    };
    load();
  }, []);

  const currentTier = subscription?.tier || "free";

  const handleCheckout = async (planId: string) => {
    if (planId === "free") return;
    if (planId === "enterprise") {
      window.open("https://calendly.com/hystersis-support/30min", "_blank");
      return;
    }

    setLoading(planId);
    setError(null);
    try {
      const origin = window.location.origin;
      const data = await billingApi.createCheckout({
        plan: planId,
        seats: 1,
        success_url: `${origin}/billing?success=true`,
        cancel_url: `${origin}/billing?canceled=true`,
        email: session?.user?.email || undefined,
      });
      if (data.url) {
        window.location.href = data.url;
      } else {
        setError("Checkout URL not returned. Stripe may not be configured.");
      }
    } catch (err) {
      console.error("Checkout error:", err);
      setError("Unable to start checkout. Ensure Stripe is configured on the API server.");
    } finally {
      setLoading(null);
    }
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">Billing & Plans</h1>
        <p className="text-muted-foreground">
          Manage your subscription and billing information
        </p>
      </div>

      {error && (
        <div className="rounded-md border border-destructive/30 bg-destructive/10 p-4 text-sm text-destructive">
          {error}
        </div>
      )}

      <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-4">
        {plans.map((plan) => {
          const features = PLAN_FEATURES[plan.id] || [];
          const isCurrent = currentTier === plan.id;
          const priceLabel =
            plan.price_per_seat === 0
              ? plan.id === "enterprise"
                ? "Custom"
                : "Free"
              : `$${plan.price_per_seat}`;
          const canCheckout = plan.id === "pro" || plan.id === "team";

          return (
            <Card
              key={plan.id}
              className={plan.id === "pro" ? "border-primary shadow-lg" : ""}
            >
              <CardHeader>
                <div className="flex items-center justify-between">
                  <CardTitle>{plan.name}</CardTitle>
                  {isCurrent && (
                    <span className="rounded-full bg-primary/10 px-2 py-1 text-xs font-medium text-primary">
                      Current
                    </span>
                  )}
                </div>
                <CardDescription>
                  {plan.id === "free" && "For individuals and self-hosted deployments"}
                  {plan.id === "pro" && "For production AI applications"}
                  {plan.id === "team" && "For teams with collaboration needs"}
                  {plan.id === "enterprise" && "For organizations with advanced requirements"}
                </CardDescription>
                <div className="mt-2">
                  <span className="text-4xl font-bold">{priceLabel}</span>
                  {plan.price_per_seat > 0 && (
                    <span className="text-muted-foreground">/seat/month</span>
                  )}
                </div>
              </CardHeader>
              <CardContent>
                <ul className="space-y-2">
                  {features.map((feature) => (
                    <li key={feature} className="flex items-center gap-2">
                      <Check className="h-4 w-4 text-green-500" />
                      <span className="text-sm">{feature}</span>
                    </li>
                  ))}
                </ul>
                <Button
                  className="mt-6 w-full"
                  variant={plan.id === "pro" ? "default" : "outline"}
                  onClick={() => handleCheckout(plan.id)}
                  disabled={loading !== null || isCurrent || (canCheckout && !stripeEnabled)}
                >
                  {loading === plan.id ? (
                    <>
                      <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                      Processing...
                    </>
                  ) : isCurrent ? (
                    "Current Plan"
                  ) : plan.id === "enterprise" ? (
                    <>
                      <ExternalLink className="mr-2 h-4 w-4" />
                      Contact Sales
                    </>
                  ) : plan.id === "free" ? (
                    "Included"
                  ) : (
                    <>
                      <CreditCard className="mr-2 h-4 w-4" />
                      Upgrade
                    </>
                  )}
                </Button>
              </CardContent>
            </Card>
          );
        })}
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Current Subscription</CardTitle>
          <CardDescription>Your plan and usage this billing period</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid gap-4 md:grid-cols-2">
            <div>
              <p className="text-sm text-muted-foreground">Plan</p>
              <p className="text-lg font-medium">{formatTierName(currentTier)}</p>
            </div>
            <div>
              <p className="text-sm text-muted-foreground">Status</p>
              <p className="text-lg font-medium capitalize">
                {subscription?.status || "active"}
              </p>
            </div>
            <div>
              <p className="text-sm text-muted-foreground">Memories used</p>
              <p className="text-lg font-medium">
                {subscription?.memory_count ?? 0}
                {subscription?.max_memories >= 0
                  ? ` / ${subscription.max_memories}`
                  : " / unlimited"}
              </p>
            </div>
            <div>
              <p className="text-sm text-muted-foreground">Searches used</p>
              <p className="text-lg font-medium">
                {subscription?.search_count ?? 0}
                {subscription?.max_searches >= 0
                  ? ` / ${subscription.max_searches}`
                  : " / unlimited"}
              </p>
            </div>
          </div>
          {!stripeEnabled && (
            <p className="mt-4 text-sm text-muted-foreground">
              Stripe checkout is not configured on the API server. Set STRIPE_SECRET_KEY,
              STRIPE_WEBHOOK_SECRET, and price IDs to enable upgrades.
            </p>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
