"use client";

import {
  createContext,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";
import { useSSE, type SSEEvent } from "@/hooks/use-sse";
import { useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

interface RealtimeContextValue {
  connected: boolean;
  error: string | null;
  lastEvent: SSEEvent | null;
  eventCount: number;
  recentEvents: SSEEvent[];
}

const RealtimeContext = createContext<RealtimeContextValue>({
  connected: false,
  error: null,
  lastEvent: null,
  eventCount: 0,
  recentEvents: [],
});

const MAX_RECENT = 40;

export function RealtimeProvider({ children }: { children: ReactNode }) {
  const { connected, error, lastEvent, eventCount, subscribe } = useSSE(true);
  const [recentEvents, setRecentEvents] = useState<SSEEvent[]>([]);
  const queryClient = useQueryClient();

  useEffect(() => {
    return subscribe("*", (payload) => {
      const type = String(payload.type || "message");
      if (type === "ping") return;

      const evt: SSEEvent = {
        type,
        data: payload,
        timestamp: String(payload.timestamp || new Date().toISOString()),
      };
      setRecentEvents((prev) => [evt, ...prev].slice(0, MAX_RECENT));

      // Invalidate relevant caches for live UI
      if (type.startsWith("memory.")) {
        queryClient.invalidateQueries({ queryKey: ["recent-memories"] });
        queryClient.invalidateQueries({ queryKey: ["analytics"] });
        queryClient.invalidateQueries({ queryKey: ["memories"] });
      }
      if (type.startsWith("notification.")) {
        queryClient.invalidateQueries({ queryKey: ["notifications"] });
      }
      if (type.startsWith("webhook.")) {
        queryClient.invalidateQueries({ queryKey: ["webhooks"] });
        queryClient.invalidateQueries({ queryKey: ["webhook-deliveries"] });
        queryClient.invalidateQueries({ queryKey: ["webhook-dead-letter"] });
      }
      if (type.startsWith("agent.") || type.startsWith("session.")) {
        queryClient.invalidateQueries({ queryKey: ["agents-count"] });
        queryClient.invalidateQueries({ queryKey: ["agents"] });
        queryClient.invalidateQueries({ queryKey: ["sessions"] });
      }
      if (type === "alert.triggered") {
        queryClient.invalidateQueries({ queryKey: ["alerts"] });
        toast.warning("Alert triggered", {
          description: String(
            (payload as { message?: string }).message || type,
          ),
        });
      }
    });
  }, [subscribe, queryClient]);

  const value = useMemo(
    () => ({
      connected,
      error,
      lastEvent,
      eventCount,
      recentEvents,
    }),
    [connected, error, lastEvent, eventCount, recentEvents],
  );

  return (
    <RealtimeContext.Provider value={value}>{children}</RealtimeContext.Provider>
  );
}

export function useRealtimeContext() {
  return useContext(RealtimeContext);
}
