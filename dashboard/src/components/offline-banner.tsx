"use client";

import { useEffect, useState } from "react";
import { WifiOff } from "lucide-react";
import { cn } from "@/lib/utils";

export function OfflineBanner() {
  const [offline, setOffline] = useState(false);

  useEffect(() => {
    if (typeof window === "undefined") return;
    const update = () => setOffline(!navigator.onLine);
    update();
    window.addEventListener("online", update);
    window.addEventListener("offline", update);
    return () => {
      window.removeEventListener("online", update);
      window.removeEventListener("offline", update);
    };
  }, []);

  if (!offline) return null;

  return (
    <div
      role="status"
      aria-live="polite"
      className={cn(
        "sticky top-16 z-40 flex items-center justify-center gap-2 border-b",
        "border-yellow-500/40 bg-yellow-500/15 px-4 py-2 text-sm text-yellow-900 dark:text-yellow-100",
      )}
    >
      <WifiOff className="h-4 w-4 shrink-0" />
      <span>
        You are offline. Changes may fail until your connection is restored.
      </span>
    </div>
  );
}
