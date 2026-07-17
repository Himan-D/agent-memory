"use client";

/**
 * Realtime hook — prefers Server-Sent Events on GET /events.
 * Kept as a thin adapter so older call sites that imported useRealtime
 * continue to work alongside useSSE / RealtimeProvider.
 */
export { useSSE as useRealtime } from "./use-sse";
export type { SSEEvent as RealtimeEvent } from "./use-sse";
