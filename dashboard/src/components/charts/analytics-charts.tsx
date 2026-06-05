"use client";

import dynamic from "next/dynamic";
import { Skeleton } from "@/components/ui/skeleton";

const LazyAnalyticsCharts = dynamic(
  () => import("./analytics-charts-inner"),
  {
    ssr: false,
    loading: () => (
      <div className="space-y-4">
        <Skeleton className="h-[300px]" />
        <Skeleton className="h-[300px]" />
      </div>
    ),
  }
);

export { LazyAnalyticsCharts };
