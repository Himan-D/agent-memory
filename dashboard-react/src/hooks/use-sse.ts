"use client";

import { useState, useEffect, useRef, useCallback } from "react";

interface SSEEvent {
  type: string;
  data: Record<string, unknown>;
  timestamp: string;
}

interface SSEState {
  connected: boolean;
  lastEvent: SSEEvent | null;
  error: string | null;
}

export function useSSE(enabled = true) {
  const [state, setState] = useState<SSEState>({
    connected: false,
    lastEvent: null,
    error: null,
  });

  const sourceRef = useRef<EventSource | null>(null);
  const handlersRef = useRef<Map<string, Set<(data: Record<string, unknown>) => void>>>(new Map());
  const reconnectTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const attemptsRef = useRef(0);

  useEffect(() => {
    if (!enabled || typeof window === "undefined") return;

    function connect() {
      const apiUrl = import.meta.env.VITE_API_URL || "http://127.0.0.1:8080";
      const token = typeof window !== "undefined" ? localStorage.getItem("hystersis_session_token") : null;
      const url = token ? `${apiUrl}/events?token=${encodeURIComponent(token)}` : `${apiUrl}/events`;

      const source = new EventSource(url);
      sourceRef.current = source;

      source.onopen = () => {
        attemptsRef.current = 0;
        setState((prev) => ({ ...prev, connected: true, error: null }));
      };

      const handleEvent = (event: MessageEvent, explicitType?: string) => {
        try {
          const data = JSON.parse(event.data) as Record<string, unknown>;
          const parsed: SSEEvent = {
            type: explicitType || String(data.type || "message"),
            data,
            timestamp: String(data.timestamp || new Date().toISOString()),
          };
          setState((prev) => ({ ...prev, lastEvent: parsed }));

          const typeHandlers = handlersRef.current.get(parsed.type);
          if (typeHandlers) {
            typeHandlers.forEach((handler) => handler(parsed.data));
          }

          const allHandlers = handlersRef.current.get("*");
          if (allHandlers) {
            allHandlers.forEach((handler) => handler({ type: parsed.type, ...parsed.data }));
          }
        } catch {
          // ignore unparseable messages
        }
      };

      source.onmessage = (event) => handleEvent(event);
      [
        "notification.created",
        "notification.updated",
        "notification.deleted",
        "notification.bulk_updated",
      ].forEach((eventType) => {
        source.addEventListener(eventType, (event) => handleEvent(event as MessageEvent, eventType));
      });

      source.onerror = () => {
        source.close();
        setState((prev) => ({ ...prev, connected: false, error: "Connection lost" }));

        if (attemptsRef.current < 10) {
          attemptsRef.current += 1;
          const delay = Math.min(1000 * Math.pow(2, attemptsRef.current), 30000);
          reconnectTimerRef.current = setTimeout(connect, delay);
        }
      };
    }

    connect();

    return () => {
      if (reconnectTimerRef.current) clearTimeout(reconnectTimerRef.current);
      sourceRef.current?.close();
      sourceRef.current = null;
    };
  }, [enabled]);

  const subscribe = useCallback((eventType: string, handler: (data: Record<string, unknown>) => void) => {
    if (!handlersRef.current.has(eventType)) {
      handlersRef.current.set(eventType, new Set());
    }
    handlersRef.current.get(eventType)!.add(handler);

    return () => {
      handlersRef.current.get(eventType)?.delete(handler);
    };
  }, []);

  return { ...state, subscribe };
}
