"use client";

import dynamic from "next/dynamic";
import { Skeleton } from "@/components/ui/skeleton";
import { Clock } from "lucide-react";

const LazyMemoryChart = dynamic(
  () => import("./memory-chart-inner"),
  {
    ssr: false,
    loading: () => (
      <div className="h-[250px] flex items-center justify-center">
        <Clock className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    ),
  }
);

export { LazyMemoryChart };
