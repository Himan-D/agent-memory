"use client";

import {
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  AreaChart,
  Area,
  BarChart,
  Bar,
} from "recharts";

interface AnalyticsChartsInnerProps {
  memoryGrowthData: { date: string; count: number }[];
  skillData: { domain: string; count: number }[];
}

export default function AnalyticsChartsInner({
  memoryGrowthData,
  skillData,
}: AnalyticsChartsInnerProps) {
  return (
    <div className="grid gap-4 lg:grid-cols-2">
      <div className="h-[300px]">
        {memoryGrowthData.length > 0 ? (
          <ResponsiveContainer width="100%" height="100%">
            <AreaChart data={memoryGrowthData}>
              <defs>
                <linearGradient id="memoryGradient" x1="0" y1="0" x2="0" y2="1">
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
                fill="url(#memoryGradient)"
              />
            </AreaChart>
          </ResponsiveContainer>
        ) : (
          <div className="flex items-center justify-center text-muted-foreground h-full">
            <p>No memory data yet</p>
          </div>
        )}
      </div>

      <div className="h-[300px]">
        {skillData.length > 0 ? (
          <ResponsiveContainer width="100%" height="100%">
            <BarChart data={skillData} layout="vertical">
              <CartesianGrid strokeDasharray="3 3" className="stroke-border" />
              <XAxis type="number" className="text-xs" />
              <YAxis dataKey="domain" type="category" className="text-xs" width={100} />
              <Tooltip
                contentStyle={{
                  backgroundColor: "hsl(var(--card))",
                  border: "1px solid hsl(var(--border))",
                  borderRadius: "8px",
                }}
              />
              <Bar dataKey="count" fill="hsl(var(--primary))" radius={[0, 4, 4, 0]} />
            </BarChart>
          </ResponsiveContainer>
        ) : (
          <div className="flex items-center justify-center text-muted-foreground h-full">
            <p>No skill data yet</p>
          </div>
        )}
      </div>
    </div>
  );
}
