"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Check, CreditCard, Loader2 } from "lucide-react";
import { useSession } from "next-auth/react";

const plans = [
  {
    id: "starter",
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

export default function BillingPage() {
  const { data: session } = useSession();
  const [loading, setLoading] = useState<string | null>(null);

  const handleCheckout = async (planId: string) => {
    setLoading(planId);
    try {
      const res = await fetch(`/api/proxy?endpoint=/stripe/checkout`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ plan: planId }),
      });
      const data = await res.json();
      if (data.url) {
        window.location.href = data.url;
      }
    } catch (error) {
      console.error("Checkout error:", error);
    } finally {
      setLoading(null);
    }
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
                onClick={() => handleCheckout(plan.id)}
                disabled={loading !== null}
              >
                {loading === plan.id ? (
                  <>
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    Processing...
                  </>
                ) : plan.price === "$0" ? (
                  "Current Plan"
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
          <div className="space-y-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="font-medium">Free Tier</p>
                <p className="text-sm text-muted-foreground">Basic features included</p>
              </div>
              <Button variant="outline" size="sm">
                Upgrade
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
