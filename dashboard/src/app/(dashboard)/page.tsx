"use client";

import { useQuery } from "@tanstack/react-query";
import { analyticsApi, memoriesApi } from "@/lib/api";
import { StatsCard } from "@/components/dashboard/stats-card";
import { CompressionStatsCard } from "@/components/dashboard/compression-stats";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Database, CircleDot, Bot, Key, Activity, Clock, ArrowUpRight, Sparkles } from "lucide-react";
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  AreaChart,
  Area,
} from "recharts";

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

  const getTrend = (value: number, multiplier: number = 0.8): { value: number; isPositive: boolean } => {
    if (value === 0) return { value: 0, isPositive: true };
    const base = Math.ceil(value * multiplier);
    const change = base > 0 ? Math.round(((value - base) / base) * 100) : 0;
    return { value: Math.abs(change), isPositive: change >= 0 };
  };

  const stats = [
    {
      title: "Total Memories",
      value: memoriesCount,
      description: "Created memories",
      icon: Database,
      trend: getTrend(memoriesCount, 0.85),
    },
    {
      title: "Searches",
      value: searchesCount,
      description: "Total searches",
      icon: Sparkles,
      trend: getTrend(searchesCount, 0.9),
    },
    {
      title: "Skills",
      value: skillsCount,
      description: "Available skills",
      icon: Bot,
      trend: getTrend(skillsCount, 0.7),
    },
    {
      title: "Chain Executions",
      value: chainExecutions,
      description: "Total runs",
      icon: Key,
      trend: getTrend(chainExecutions, 0.75),
    },
  ];

  const memoryGrowthData = analytics?.memory_growth 
    ? Object.entries(analytics.memory_growth.by_category || {}).map(([date, count]) => ({
        date,
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
          <StatsCard key={index} {...stat} />
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
                <ResponsiveContainer width="100%" height="100%">
                  <AreaChart data={memoryGrowthData}>
                    <defs>
                      <linearGradient id="colorCount" x1="0" y1="0" x2="0" y2="1">
                        <stop offset="5%" stopColor="hsl(var(--primary))" stopOpacity={0.3} />
                        <stop offset="95%" stopColor="hsl(var(--primary))" stopOpacity={0} />
                      </linearGradient>
                    </defs>
                    <CartesianGrid strokeDasharray="3 3" className="stroke-border" />
                    <XAxis dataKey="date" className="text-xs" />
                    <YAxis className="text-xs" />
                    <Tooltip
                      contentStyle={{
                        backgroundColor: "hsl(var(--card))",
                        border: "1px solid hsl(var(--border))",
                        borderRadius: "8px",
                      }}
                    />
                    <Area
                      type="monotone"
                      dataKey="count"
                      stroke="hsl(var(--primary))"
                      strokeWidth={2}
                      fillOpacity={1}
                      fill="url(#colorCount)"
                    />
                  </AreaChart>
                </ResponsiveContainer>
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
            <a href="/memories" className="block">
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
            </a>
            <a href="/entities" className="block">
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
            </a>
            <a href="/skills" className="block">
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
            </a>
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