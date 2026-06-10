"use client";

import { useQuery } from "@tanstack/react-query";
import { analyticsApi, memoriesApi, agentsApi } from "@/lib/api";
import { registerWebComponents } from "@/components/ui/web-component";
import { CompressionStatsCard } from "@/components/dashboard/compression-stats";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import Link from "next/link";
import {
  Database,
  CircleDot,
  Bot,
  Activity,
  Clock,
  ArrowUpRight,
  Sparkles,
  Webhook,
  AlertTriangle,
  Workflow,
  BarChart3,
  Search,
} from "lucide-react";
import { LazyMemoryChart } from "@/components/charts/memory-chart";

registerWebComponents();

const ICON_SVG = {
  database:
    '<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><ellipse cx="12" cy="5" rx="9" ry="3"/><path d="M3 5V19A9 3 0 0 0 21 19V5"/><path d="M3 12A9 3 0 0 0 21 12"/></svg>',
  sparkles:
    '<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m12 3-1.912 5.813a2 2 0 0 1-1.275 1.275L3 12l5.813 1.912a2 2 0 0 1 1.275 1.275L12 21l1.912-5.813a2 2 0 0 1 1.275-1.275L21 12l-5.813-1.912a2 2 0 0 1-1.275-1.275L12 3Z"/></svg>',
  bot: '<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 8V4H8"/><rect width="16" height="12" x="4" y="8" rx="2"/><path d="M2 14h2"/><path d="M20 14h2"/><path d="M15 13v2"/><path d="M9 13v2"/></svg>',
  search:
    '<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/></svg>',
};

interface QuickAction {
  href: string;
  icon: React.ComponentType<{ className?: string }>;
  label: string;
  description: string;
}

const quickActions: QuickAction[] = [
  { href: "/memories", icon: Database, label: "Memories", description: "Search and manage memories" },
  { href: "/entities", icon: CircleDot, label: "Knowledge Graph", description: "Visualize entity relationships" },
  { href: "/skills", icon: Sparkles, label: "Skills", description: "Manage agent capabilities" },
  { href: "/webhooks", icon: Webhook, label: "Webhooks", description: "Manage event integrations" },
  { href: "/alerts", icon: AlertTriangle, label: "Alerts", description: "Configure monitoring rules" },
  { href: "/analytics", icon: BarChart3, label: "Analytics", description: "View detailed analytics" },
];

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

  const { data: agents } = useQuery({
    queryKey: ["agents-count"],
    queryFn: () => agentsApi.list(),
  });

  const memoriesCount = analytics?.memory_growth?.total_created || 0;
  const searchesCount = analytics?.search_analytics?.total_searches || 0;
  const agentCount = Array.isArray(agents) ? agents.length : agents?.agents?.length || 0;
  const skillsCount = analytics?.skill_metrics?.total_skills || 0;

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
      iconSvg: ICON_SVG.search,
    },
    {
      title: "Agents",
      value: String(agentCount),
      description: "Connected agents",
      iconSvg: ICON_SVG.bot,
    },
    {
      title: "Skills",
      value: String(skillsCount),
      description: "Available skills",
      iconSvg: ICON_SVG.sparkles,
    },
  ];

  const memoryGrowthData = analytics?.memory_growth?.daily_trend?.length
    ? analytics.memory_growth.daily_trend.map((point: { date: string; count: number }) => ({
        date: point.date,
        count: point.count,
      }))
    : [];

  const agentActivity = analytics?.agent_activity || [];

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
      </div>

      <div className="grid gap-4 md:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle className="text-lg font-semibold">Quick Actions</CardTitle>
          </CardHeader>
          <CardContent className="grid gap-2 sm:grid-cols-2">
            {quickActions.map((action) => (
              <Link key={action.href} href={action.href} className="block">
                <div className="flex items-center gap-3 p-3 rounded-lg border hover:bg-accent transition-colors cursor-pointer">
                  <div className="rounded-lg bg-primary/10 p-2">
                    <action.icon className="h-4 w-4 text-primary" />
                  </div>
                  <div className="flex-1 min-w-0">
                    <p className="font-medium text-sm">{action.label}</p>
                    <p className="text-xs text-muted-foreground truncate">{action.description}</p>
                  </div>
                  <ArrowUpRight className="h-3.5 w-3.5 text-muted-foreground shrink-0" />
                </div>
              </Link>
            ))}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-lg font-semibold">
              <Bot className="h-5 w-5" />
              Agent Activity
            </CardTitle>
          </CardHeader>
          <CardContent>
            {agentActivity.length > 0 ? (
              <div className="space-y-3">
                {agentActivity.slice(0, 5).map((agent: { agent_id: string; agent_name: string; session_count: number; memory_count: number; last_active: string | null }) => (
                  <div
                    key={agent.agent_id}
                    className="flex items-center justify-between rounded-lg border p-3"
                  >
                    <div className="flex items-center gap-3">
                      <div className="rounded-full bg-green-500/10 p-2">
                        <Bot className="h-4 w-4 text-green-600" />
                      </div>
                      <div>
                        <p className="font-medium text-sm">{agent.agent_name || agent.agent_id}</p>
                        <p className="text-xs text-muted-foreground">
                          {agent.session_count} sessions &middot; {agent.memory_count} memories
                        </p>
                      </div>
                    </div>
                    {agent.last_active && (
                      <span className="text-xs text-muted-foreground">
                        {new Date(agent.last_active).toLocaleDateString()}
                      </span>
                    )}
                  </div>
                ))}
              </div>
            ) : (
              <div className="flex flex-col items-center justify-center py-8 text-muted-foreground">
                <Bot className="h-12 w-12 mb-4 opacity-50" />
                <p>No agent activity yet</p>
                <p className="text-sm">Connect agents to see their activity</p>
              </div>
            )}
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
              {recentMemories.memories.map((memory: { id: string; content: string; type: string; tags?: string[]; created_at: string }) => (
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
