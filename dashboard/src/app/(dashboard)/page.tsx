"use client";

import { useQuery } from "@tanstack/react-query";
import { analyticsApi, memoriesApi } from "@/lib/api";
import { registerWebComponents } from "@/components/ui/web-component";
import { CompressionStatsCard } from "@/components/dashboard/compression-stats";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Database, CircleDot, Bot, Key, Activity, Clock, ArrowUpRight, Sparkles } from "lucide-react";
import { LazyMemoryChart } from "@/components/charts/memory-chart";
import Link from "next/link";

registerWebComponents();

const ICON_SVG = {
  database: '<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><ellipse cx="12" cy="5" rx="9" ry="3"/><path d="M3 5V19A9 3 0 0 0 21 19V5"/><path d="M3 12A9 3 0 0 0 21 12"/></svg>',
  sparkles: '<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m12 3-1.912 5.813a2 2 0 0 1-1.275 1.275L3 12l5.813 1.912a2 2 0 0 1 1.275 1.275L12 21l1.912-5.813a2 2 0 0 1 1.275-1.275L21 12l-5.813-1.912a2 2 0 0 1-1.275-1.275L12 3Z"/></svg>',
  bot: '<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 8V4H8"/><rect width="16" height="12" x="4" y="8" rx="2"/><path d="M2 14h2"/><path d="M20 14h2"/><path d="M15 13v2"/><path d="M9 13v2"/></svg>',
  key: '<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="7.5" cy="15.5" r="5.5"/><path d="m21 2-9.6 9.6"/><path d="m15.5 7.5 3 3L22 7l-3-3"/></svg>',
};

export default function DashboardPage() {
  const { data: analytics, isLoading: analyticsLoading } = useQuery({
    queryKey: ["analytics"],
    queryFn: () => analyticsApi.dashboard(),
    refetchInterval: 30000,
  });

  const { data: recentMemories } = useQuery({
    queryKey: ["recent-memories"],
    queryFn: () => memoriesApi.list({ limit: 5 }),
  });

  const memoriesCount = analytics?.memory_growth?.total_created || 0;
  const searchesCount = analytics?.search_analytics?.total_searches || 0;
  const skillsCount = analytics?.skill_metrics?.total_skills || 0;
  const chainExecutions = analytics?.skill_metrics?.chain_usage?.total_executions || 0;

  const stats = [
    {
      title: "Total Memories",
      value: String(memoriesCount),
      description: "Created memories",
      iconSvg: ICON_SVG.database,
    },
    {
      title: "Searches",
      value: String(searchesCount),
      description: "Total searches",
      iconSvg: ICON_SVG.sparkles,
    },
    {
      title: "Skills",
      value: String(skillsCount),
      description: "Available skills",
      iconSvg: ICON_SVG.bot,
    },
    {
      title: "Chain Executions",
      value: String(chainExecutions),
      description: "Total runs",
      iconSvg: ICON_SVG.key,
    },
  ];

  // by_category is category→count, not a time series — render as categorical bar chart
  const memoryGrowthData = analytics?.memory_growth
    ? Object.entries(analytics.memory_growth.by_category || {}).map(([category, count]) => ({
        date: category,
        count: count as number,
      }))
    : [];

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold">Dashboard</h1>
        <p className="text-muted-foreground">Monitor your memory infrastructure</p>
      </div>

      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        {stats.map((stat, index) => (
          <hyst-stats-card
            key={index}
            title={stat.title}
            value={stat.value}
            description={stat.description}
            icon-svg={stat.iconSvg}
          />
        ))}
      </div>

      <div className="grid gap-4 md:grid-cols-2">
        <CompressionStatsCard />

        <Card>
          <CardHeader>
            <CardTitle className="text-lg font-semibold">Memory Growth</CardTitle>
          </CardHeader>
          <CardContent>
            {analyticsLoading ? (
              <div className="h-[250px] flex items-center justify-center">
                <Clock className="h-8 w-8 animate-spin text-muted-foreground" />
              </div>
            ) : memoryGrowthData.length > 0 ? (
              <div className="h-[250px]">
                <LazyMemoryChart data={memoryGrowthData} />
              </div>
            ) : (
              <div className="h-[250px] flex flex-col items-center justify-center text-muted-foreground">
                <Database className="h-12 w-12 mb-4 opacity-50" />
                <p>No memory data yet</p>
                <p className="text-sm">Create memories to see growth trends</p>
              </div>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="text-lg font-semibold">Quick Actions</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            <Link href="/memories" className="block">
              <div className="flex items-center gap-3 p-3 rounded-lg border hover:bg-accent transition-colors cursor-pointer">
                <div className="rounded-lg bg-primary/10 p-2">
                  <Database className="h-4 w-4 text-primary" />
                </div>
                <div className="flex-1">
                  <p className="font-medium">View Memories</p>
                  <p className="text-sm text-muted-foreground">Search and manage memories</p>
                </div>
                <ArrowUpRight className="h-4 w-4 text-muted-foreground" />
              </div>
            </Link>
            <Link href="/entities" className="block">
              <div className="flex items-center gap-3 p-3 rounded-lg border hover:bg-accent transition-colors cursor-pointer">
                <div className="rounded-lg bg-primary/10 p-2">
                  <CircleDot className="h-4 w-4 text-primary" />
                </div>
                <div className="flex-1">
                  <p className="font-medium">Knowledge Graph</p>
                  <p className="text-sm text-muted-foreground">Visualize entity relationships</p>
                </div>
                <ArrowUpRight className="h-4 w-4 text-muted-foreground" />
              </div>
            </Link>
            <Link href="/skills" className="block">
              <div className="flex items-center gap-3 p-3 rounded-lg border hover:bg-accent transition-colors cursor-pointer">
                <div className="rounded-lg bg-primary/10 p-2">
                  <Bot className="h-4 w-4 text-primary" />
                </div>
                <div className="flex-1">
                  <p className="font-medium">Skills</p>
                  <p className="text-sm text-muted-foreground">Manage agent capabilities</p>
                </div>
                <ArrowUpRight className="h-4 w-4 text-muted-foreground" />
              </div>
            </Link>
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-lg font-semibold">
            <Activity className="h-5 w-5" />
            Recent Memories
          </CardTitle>
        </CardHeader>
        <CardContent>
          {recentMemories?.memories && recentMemories.memories.length > 0 ? (
            <div className="space-y-3">
              {recentMemories.memories.map((memory) => (
                <div
                  key={memory.id}
                  className="flex items-center justify-between rounded-lg border p-3 transition-colors hover:bg-accent"
                >
                  <div className="flex items-center gap-3 flex-1 min-w-0">
                    <div className="rounded-full bg-primary/10 p-2 shrink-0">
                      <Clock className="h-4 w-4 text-primary" />
                    </div>
                    <div className="min-w-0 flex-1">
                      <p className="font-medium truncate">{memory.content}</p>
                      <div className="flex items-center gap-2 mt-1">
                        <Badge variant="outline" className="text-xs">
                          {memory.type}
                        </Badge>
                        {memory.tags?.slice(0, 2).map((tag) => (
                          <Badge key={tag} variant="secondary" className="text-xs">
                            {tag}
                          </Badge>
                        ))}
                      </div>
                    </div>
                  </div>
                  <span className="text-xs text-muted-foreground shrink-0 ml-4">
                    {new Date(memory.created_at).toLocaleDateString()}
                  </span>
                </div>
              ))}
            </div>
          ) : (
            <div className="flex flex-col items-center justify-center py-8 text-muted-foreground">
              <Database className="h-12 w-12 mb-4 opacity-50" />
              <p>No memories yet</p>
              <p className="text-sm">Start adding memories to see them here</p>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
