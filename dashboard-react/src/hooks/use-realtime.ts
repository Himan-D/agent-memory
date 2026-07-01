"use client";

import { useState, useEffect, useCallback, useRef } from "react";

interface RealtimeEvent {
  type: string;
  payload: Record<string, unknown>;
  timestamp: string;
}

interface RealtimeOptions {
  url: string;
  reconnectInterval?: number;
  maxReconnectAttempts?: number;
  onMessage?: (event: RealtimeEvent) => void;
  onOpen?: () => void;
  onClose?: () => void;
  onError?: (error: Event) => void;
}

interface RealtimeState {
  connected: boolean;
  lastEvent: RealtimeEvent | null;
  error: Error | null;
  messageCount: number;
}

export function useRealtime(options: RealtimeOptions) {
  const {
    url,
    reconnectInterval = 3000,
    maxReconnectAttempts = 10,
    onMessage,
    onOpen,
    onClose,
    onError,
  } = options;

  const [state, setState] = useState<RealtimeState>({
    connected: false,
    lastEvent: null,
    error: null,
    messageCount: 0,
  });

  const wsRef = useRef<WebSocket | null>(null);
  const reconnectAttemptsRef = useRef(0);
  const reconnectTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const eventHandlersRef = useRef<Record<string, Array<(payload: Record<string, unknown>) => void>>>({});

  const connect = useCallback(() => {
    if (typeof window === "undefined") return;

    try {
      const ws = new WebSocket(url);

      ws.onopen = () => {
        reconnectAttemptsRef.current = 0;
        setState((prev) => ({ ...prev, connected: true, error: null }));
        onOpen?.();
      };

      ws.onmessage = (event) => {
        try {
          const data: RealtimeEvent = JSON.parse(event.data);
          setState((prev) => ({
            ...prev,
            lastEvent: data,
            messageCount: prev.messageCount + 1,
          }));

          // Call global handler
          onMessage?.(data);

          // Call type-specific handlers
          const handlers = eventHandlersRef.current[data.type];
          if (handlers) {
            handlers.forEach((handler) => handler(data.payload));
          }
        } catch (err) {
          console.error("Failed to parse realtime message:", err);
        }
      };

      ws.onclose = () => {
        setState((prev) => ({ ...prev, connected: false }));
        onClose?.();

        // Attempt reconnect
        if (reconnectAttemptsRef.current < maxReconnectAttempts) {
          reconnectAttemptsRef.current += 1;
          reconnectTimerRef.current = setTimeout(() => {
            connect();
          }, reconnectInterval * reconnectAttemptsRef.current);
        }
      };

      ws.onerror = (error) => {
        setState((prev) => ({ ...prev, error: new Error("WebSocket error") }));
        onError?.(error as unknown as Event);
      };

      wsRef.current = ws;
    } catch (error) {
      setState((prev) => ({
        ...prev,
        error: error instanceof Error ? error : new Error("Failed to connect"),
      }));
    }
  }, [url, reconnectInterval, maxReconnectAttempts, onMessage, onOpen, onClose, onError]);

  const disconnect = useCallback(() => {
    if (reconnectTimerRef.current) {
      clearTimeout(reconnectTimerRef.current);
    }
    wsRef.current?.close();
    wsRef.current = null;
    reconnectAttemptsRef.current = 0;
    setState((prev) => ({ ...prev, connected: false }));
  }, []);

  const send = useCallback(
    (data: Record<string, unknown>) => {
      if (wsRef.current?.readyState === WebSocket.OPEN) {
        wsRef.current.send(JSON.stringify(data));
      }
    },
    []
  );

  const subscribe = useCallback(
    (eventType: string, handler: (payload: Record<string, unknown>) => void) => {
      if (!eventHandlersRef.current[eventType]) {
        eventHandlersRef.current[eventType] = [];
      }
      eventHandlersRef.current[eventType].push(handler);

      return () => {
        eventHandlersRef.current[eventType] = eventHandlersRef.current[eventType].filter(
          (h) => h !== handler
        );
      };
    },
    []
  );

  useEffect(() => {
    connect();
    return () => {
      disconnect();
    };
  }, [connect, disconnect]);

  return {
    ...state,
    connect,
    disconnect,
    send,
    subscribe,
  };
}

// Simple event emitter for client-side real-time updates
export function createEventBus() {
  const handlers = new Map<string, Set<(...args: any[]) => void>>();

  return {
    on: (event: string, handler: (...args: any[]) => void) => {
      if (!handlers.has(event)) {
        handlers.set(event, new Set());
      }
      handlers.get(event)!.add(handler);
      return () => handlers.get(event)!.delete(handler);
    },
    off: (event: string, handler: (...args: any[]) => void) => {
      handlers.get(event)?.delete(handler);
    },
    emit: (event: string, ...args: any[]) => {
      handlers.get(event)?.forEach((handler) => handler(...args));
    },
    clear: () => {
      handlers.clear();
    },
  };
}