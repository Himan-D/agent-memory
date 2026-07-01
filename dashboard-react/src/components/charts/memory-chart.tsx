import { lazy, Suspense } from "react";
import { Clock } from "lucide-react";

const LazyMemoryChartInner = lazy(() => import("./memory-chart-inner"));

export function LazyMemoryChart({ data }: { data: any[] }) {
  return (
    <Suspense
      fallback={
        <div className="h-[250px] flex items-center justify-center">
          <Clock className="h-8 w-8 animate-spin text-muted-foreground" />
        </div>
      }
    >
      <LazyMemoryChartInner data={data} />
    </Suspense>
  );
}
