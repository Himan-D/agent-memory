"use client";

import { useEffect, useState } from "react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { api, TierPolicy } from "@/lib/api";

const POLICIES = [
  { value: TierPolicy.AGGRESSIVE, label: "Aggressive", description: "1-day hot storage", retention: "1 day" },
  { value: TierPolicy.BALANCED, label: "Balanced", description: "7-day hot storage", retention: "7 days" },
  { value: TierPolicy.CONSERVATIVE, label: "Conservative", description: "30-day hot storage", retention: "30 days" },
];

export function TierPolicySelector() {
  const [policy, setPolicy] = useState<string>(TierPolicy.BALANCED);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    async function fetchPolicy() {
      try {
        const data = await api.compression.getTierPolicy();
        setPolicy(data.policy);
      } catch (error) {
        console.error("Failed to fetch tier policy:", error);
      } finally {
        setLoading(false);
      }
    }

    fetchPolicy();
  }, []);

  const handlePolicyChange = async (newPolicy: string) => {
    setSaving(true);
    try {
      await api.compression.setTierPolicy(newPolicy);
      setPolicy(newPolicy);
    } catch (error) {
      console.error("Failed to set tier policy:", error);
    } finally {
      setSaving(false);
    }
  };

  if (loading) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>Memory Tier Policy</CardTitle>
          <CardDescription>Loading...</CardDescription>
        </CardHeader>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>Memory Tier Policy</CardTitle>
        <CardDescription>Configure hot storage retention</CardDescription>
      </CardHeader>
      <CardContent>
        <div className="space-y-3">
          {POLICIES.map((p) => (
            <div
              key={p.value}
              className={`flex items-center justify-between p-3 rounded-lg border cursor-pointer transition-colors ${
                policy === p.value ? "border-green-500 bg-green-500/10" : "hover:bg-muted"
              }`}
              onClick={() => !saving && handlePolicyChange(p.value)}
            >
              <div>
                <div className="font-medium">{p.label}</div>
                <div className="text-sm text-muted-foreground">{p.description}</div>
              </div>
              <div className="text-right">
                <div className="text-sm font-medium">{p.retention}</div>
              </div>
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  );
}