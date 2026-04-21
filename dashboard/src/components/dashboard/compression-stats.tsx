"use client";

import { useEffect, useState } from "react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { api, CompressionStats } from "@/lib/api";

export function CompressionStatsCard() {
  const [stats, setStats] = useState<CompressionStats | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    async function fetchStats() {
      try {
        const data = await api.compression.getStats();
        setStats(data);
      } catch (error) {
        console.error("Failed to fetch compression stats:", error);
      } finally {
        setLoading(false);
      }
    }

    fetchStats();
    const interval = setInterval(fetchStats, 30000);
    return () => clearInterval(interval);
  }, []);

  if (loading && !stats) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>Compression Engine</CardTitle>
          <CardDescription>Loading...</CardDescription>
        </CardHeader>
      </Card>
    );
  }

  const accuracyPercent = stats ? Math.round(stats.accuracy_retention * 100) : 0;
  const reductionPercent = stats ? Math.round(stats.token_reduction * 100) : 0;

  return (
    <Card>
      <CardHeader>
        <CardTitle>Compression Engine</CardTitle>
        <CardDescription>Proprietary Memory Compression</CardDescription>
      </CardHeader>
      <CardContent>
        <div className="grid gap-4">
          <div className="flex items-center justify-between">
            <span className="text-sm font-medium">Accuracy Retention</span>
            <span className="text-sm text-muted-foreground">{accuracyPercent}%</span>
          </div>
          <div className="h-2 w-full rounded-full bg-muted overflow-hidden">
            <div
              className="h-full bg-green-500 transition-all duration-300"
              style={{ width: `${accuracyPercent}%` }}
            />
          </div>

          <div className="flex items-center justify-between">
            <span className="text-sm font-medium">Token Reduction</span>
            <span className="text-sm text-muted-foreground">{reductionPercent}%</span>
          </div>
          <div className="h-2 w-full rounded-full bg-muted overflow-hidden">
            <div
              className="h-full bg-blue-500 transition-all duration-300"
              style={{ width: `${reductionPercent}%` }}
            />
          </div>

          <div className="pt-2 border-t">
            <div className="grid grid-cols-2 gap-4 text-center">
              <div>
                <div className="text-2xl font-bold">{stats?.extractions_performed || 0}</div>
                <div className="text-xs text-muted-foreground">Extractions</div>
              </div>
              <div>
                <div className="text-2xl font-bold">{stats?.spreading_activations || 0}</div>
                <div className="text-xs text-muted-foreground">Spreading</div>
              </div>
            </div>
          </div>

          <div className="pt-2 border-t">
            <div className="grid grid-cols-2 gap-4 text-center">
              <div>
                <div className="text-lg font-semibold">
                  {stats?.total_tokens_saved 
                    ? stats.total_tokens_saved >= 1000000 
                      ? `${(stats.total_tokens_saved / 1000000).toFixed(1)}M`
                      : stats.total_tokens_saved >= 1000 
                        ? `${(stats.total_tokens_saved / 1000).toFixed(0)}K`
                        : stats.total_tokens_saved
                    : 0}
                </div>
                <div className="text-xs text-muted-foreground">Tokens Saved</div>
              </div>
              <div>
                <div className="text-lg font-semibold">{stats?.p95_latency_ms || 0}ms</div>
                <div className="text-xs text-muted-foreground">P95 Latency</div>
              </div>
            </div>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}