import { lazy, Suspense } from "react";
import { Skeleton } from "@/components/ui/skeleton";

const LazyAnalyticsChartsInner = lazy(() => import("./analytics-charts-inner"));

export function LazyAnalyticsCharts({ memoryGrowthData, skillData }: { memoryGrowthData: any[]; skillData: any[] }) {
  return (
    <Suspense
      fallback={
        <div className="space-y-4">
          <Skeleton className="h-[300px]" />
          <Skeleton className="h-[300px]" />
        </div>
      }
    >
      <LazyAnalyticsChartsInner memoryGrowthData={memoryGrowthData} skillData={skillData} />
    </Suspense>
  );
}
