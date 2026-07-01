
import { useQuery } from "@tanstack/react-query";
import { analyticsApi } from "@/lib/api";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { LazyAnalyticsCharts } from "@/components/charts/analytics-charts";
import { Database, Search, Bot, Zap, TrendingUp } from "lucide-react";

export default function AnalyticsPage() {
  const { data: analytics, isLoading } = useQuery({
    queryKey: ["analytics"],
    queryFn: () => analyticsApi.dashboard(),
    refetchInterval: 60000,
  });

  const memoryGrowthData = analytics?.memory_growth?.by_category 
    ? Object.entries(analytics.memory_growth.by_category).map(([date, count]) => ({
        date,
        count,
      }))
    : [];

  const skillData = analytics?.skill_metrics?.skills_by_domain
    ? Object.entries(analytics.skill_metrics.skills_by_domain).map(([domain, count]) => ({
        domain,
        count,
      }))
    : [];

  const topQueries = analytics?.search_analytics?.top_queries || [];

  if (isLoading) {
    return (
      <div className="space-y-6">
        <div className="space-y-2">
          <Skeleton className="h-9 w-48" />
          <Skeleton className="h-5 w-64" />
        </div>
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
          {Array.from({ length: 4 }).map((_, i) => (
            <Skeleton key={i} className="h-32" />
          ))}
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold">Analytics</h1>
        <p className="text-muted-foreground">Monitor your memory infrastructure</p>
      </div>

      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Total Memories</CardTitle>
            <Database className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {analytics?.memory_growth?.total_created || 0}
            </div>
            <p className="text-xs text-muted-foreground">
              {analytics?.memory_growth?.total_deleted || 0} deleted, {analytics?.memory_growth?.total_archived || 0} archived
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Total Searches</CardTitle>
            <Search className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {analytics?.search_analytics?.total_searches || 0}
            </div>
            <p className="text-xs text-muted-foreground">
              {analytics?.search_analytics?.avg_results_per_query?.toFixed(1) || 0} avg results
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Skills</CardTitle>
            <Bot className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {analytics?.skill_metrics?.total_skills || 0}
            </div>
            <p className="text-xs text-muted-foreground">
              {analytics?.skill_metrics?.active_skills || 0} active
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Chain Executions</CardTitle>
            <Zap className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {analytics?.skill_metrics?.chain_usage?.total_executions || 0}
            </div>
            <p className="text-xs text-muted-foreground">
              {(analytics?.skill_metrics?.chain_usage?.success_rate || 0).toFixed(1)}% success rate
            </p>
          </CardContent>
        </Card>
      </div>

      <LazyAnalyticsCharts memoryGrowthData={memoryGrowthData} skillData={skillData} />

      <Card>
        <CardHeader>
          <CardTitle className="text-lg font-semibold">Top Search Queries</CardTitle>
        </CardHeader>
        <CardContent>
          {topQueries.length > 0 ? (
            <div className="space-y-3">
              {topQueries.map((item, i) => {
                const queryText = typeof item === 'string' ? item : item.query;
                const queryCount = typeof item === 'string' ? null : item.count;
                return (
                <div key={queryText || i} className="flex items-center justify-between">
                  <div className="flex items-center gap-3">
                    <span className="text-sm font-mono text-muted-foreground">#{i + 1}</span>
                    <span>{queryText}</span>
                  </div>
                  {queryCount != null ? (
                    <span className="text-sm tabular-nums text-muted-foreground">{queryCount}</span>
                  ) : (
                    <TrendingUp className="h-4 w-4 text-muted-foreground" />
                  )}
                </div>
                );
              })}
            </div>
          ) : (
            <div className="flex items-center justify-center py-8 text-muted-foreground">
              <Search className="mr-2 h-5 w-5 opacity-50" />
              <span>No search queries yet</span>
            </div>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-lg font-semibold">Retention Metrics</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid gap-4 md:grid-cols-4">
            <div className="text-center">
              <div className="text-2xl font-bold">{analytics?.retention?.active_users || 0}</div>
              <p className="text-xs text-muted-foreground">Active Users</p>
            </div>
            <div className="text-center">
              <div className="text-2xl font-bold">{analytics?.retention?.returning_users || 0}</div>
              <p className="text-xs text-muted-foreground">Returning Users</p>
            </div>
            <div className="text-center">
              <div className="text-2xl font-bold">{(analytics?.retention?.retention_rate || 0).toFixed(1)}%</div>
              <p className="text-xs text-muted-foreground">Retention Rate</p>
            </div>
            <div className="text-center">
              <div className="text-2xl font-bold">{analytics?.retention?.avg_memories_per_user || 0}</div>
              <p className="text-xs text-muted-foreground">Avg Memories/User</p>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}