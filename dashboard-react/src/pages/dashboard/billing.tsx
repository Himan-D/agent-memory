
import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Check, CreditCard, Loader2 } from "lucide-react";
import { billingApi } from "@/lib/api";
import { toast } from "sonner";

const plans = [
  {
    id: "starter",
    stripePlan: "starter",
    name: "Starter",
    price: "$0",
    description: "For individuals and small projects",
    features: [
      "1,000 memories",
      "Basic search",
      "Community support",
      "Single agent",
    ],
  },
  {
    id: "pro",
    stripePlan: "pro",
    name: "Pro",
    price: "$29",
    description: "For growing teams and production use",
    features: [
      "Unlimited memories",
      "Advanced search & spreading activation",
      "Priority support",
      "Up to 10 agents",
      "Skill chains",
      "Compression engine",
    ],
    popular: true,
  },
  {
    id: "enterprise",
    stripePlan: "enterprise",
    name: "Enterprise",
    price: "$99",
    description: "For large organizations with advanced needs",
    features: [
      "Everything in Pro",
      "Unlimited agents",
      "SSO (SAML/OIDC/LDAP)",
      "Audit logs",
      "Dedicated support",
      "Custom integrations",
    ],
  },
];

const tierLabels: Record<string, string> = {
  selfHosted: "Starter",
  starter: "Starter",
  pro: "Pro",
  team: "Enterprise",
  enterprise: "Enterprise",
};

export default function BillingPage() {
  const [loading, setLoading] = useState<string | null>(null);

  const { data: subscription, isLoading: subLoading } = useQuery({
    queryKey: ["billing-subscription"],
    queryFn: () => billingApi.getSubscription(),
  });

  const { data: usage } = useQuery({
    queryKey: ["billing-usage"],
    queryFn: () => billingApi.getUsage(),
  });

  const currentTier = subscription?.tier || "selfHosted";

  const handleCheckout = async (planId: string) => {
    if (planId === "starter") {
      toast.info("You are already on the free Starter plan");
      return;
    }

    setLoading(planId);
    try {
      const data = await billingApi.createCheckout(planId);
      if (data.url) {
        window.location.href = data.url;
      } else {
        toast.error("Checkout session could not be created");
      }
    } catch (error) {
      const message = error instanceof Error ? error.message : "Checkout failed";
      toast.error(message);
    } finally {
      setLoading(null);
    }
  };

  const isCurrentPlan = (planId: string) => {
    if (planId === "starter") return currentTier === "selfHosted" || currentTier === "starter";
    if (planId === "pro") return currentTier === "pro";
    if (planId === "enterprise") return currentTier === "team" || currentTier === "enterprise";
    return false;
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">Billing & Plans</h1>
        <p className="text-muted-foreground">Manage your subscription and billing information</p>
      </div>

      <div className="grid gap-6 md:grid-cols-3">
        {plans.map((plan) => (
          <Card key={plan.id} className={plan.popular ? "border-primary shadow-lg" : ""}>
            <CardHeader>
              <div className="flex items-center justify-between">
                <CardTitle>{plan.name}</CardTitle>
                {plan.popular && (
                  <span className="rounded-full bg-primary px-3 py-1 text-xs font-medium text-primary-foreground">
                    Popular
                  </span>
                )}
              </div>
              <CardDescription>{plan.description}</CardDescription>
              <div className="mt-2">
                <span className="text-4xl font-bold">{plan.price}</span>
                {plan.price !== "$0" && <span className="text-muted-foreground">/month</span>}
              </div>
            </CardHeader>
            <CardContent>
              <ul className="space-y-2">
                {plan.features.map((feature, i) => (
                  <li key={i} className="flex items-center gap-2">
                    <Check className="h-4 w-4 text-green-500" />
                    <span className="text-sm">{feature}</span>
                  </li>
                ))}
              </ul>
              <Button
                className="mt-6 w-full"
                variant={plan.popular ? "default" : "outline"}
                onClick={() => handleCheckout(plan.stripePlan)}
                disabled={loading !== null || isCurrentPlan(plan.id)}
              >
                {loading === plan.stripePlan ? (
                  <>
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    Processing...
                  </>
                ) : isCurrentPlan(plan.id) ? (
                  "Current Plan"
                ) : plan.price === "$0" ? (
                  "Free"
                ) : (
                  <>
                    <CreditCard className="mr-2 h-4 w-4" />
                    Get Started
                  </>
                )}
              </Button>
            </CardContent>
          </Card>
        ))}
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Current Subscription</CardTitle>
          <CardDescription>Your current plan and usage</CardDescription>
        </CardHeader>
        <CardContent>
          {subLoading ? (
            <div className="flex items-center gap-2 text-muted-foreground">
              <Loader2 className="h-4 w-4 animate-spin" />
              Loading subscription...
            </div>
          ) : (
            <div className="space-y-4">
              <div className="flex items-center justify-between">
                <div>
                  <p className="font-medium">{tierLabels[currentTier] || currentTier} Plan</p>
                  <p className="text-sm text-muted-foreground">
                    Status: {subscription?.status || "active"}
                  </p>
                </div>
                {currentTier === "selfHosted" || currentTier === "starter" ? (
                  <Button variant="outline" size="sm" onClick={() => handleCheckout("pro")}>
                    Upgrade to Pro
                  </Button>
                ) : null}
              </div>
              {usage && (
                <div className="grid gap-4 md:grid-cols-2">
                  <div className="rounded-lg border p-4">
                    <p className="text-sm text-muted-foreground">Memories</p>
                    <p className="text-lg font-semibold">
                      {usage.memory_count?.toLocaleString() ?? 0}
                    </p>
                  </div>
                  <div className="rounded-lg border p-4">
                    <p className="text-sm text-muted-foreground">Searches</p>
                    <p className="text-lg font-semibold">
                      {usage.search_count?.toLocaleString() ?? 0}
                    </p>
                  </div>
                </div>
              )}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
