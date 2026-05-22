"use client";

import { useState, useCallback } from "react";

interface AuditEntry {
  action: string;
  resource: string;
  resourceId: string;
  details?: Record<string, unknown>;
  timestamp: string;
  userId?: string;
}

const STORAGE_KEY = "hystersis_audit_log";
const MAX_ENTRIES = 1000;

export function useAuditLogger() {
  const [logs, setLogs] = useState<AuditEntry[]>(() => {
    if (typeof window === "undefined") return [];
    try {
      const stored = localStorage.getItem(STORAGE_KEY);
      return stored ? JSON.parse(stored) : [];
    } catch {
      return [];
    }
  });

  const addLog = useCallback((entry: Omit<AuditEntry, "timestamp">) => {
    const newEntry: AuditEntry = {
      ...entry,
      timestamp: new Date().toISOString(),
    };

    setLogs((prev) => {
      const updated = [newEntry, ...prev].slice(0, MAX_ENTRIES);
      try {
        localStorage.setItem(STORAGE_KEY, JSON.stringify(updated));
      } catch {
        // Storage full or unavailable
      }
      return updated;
    });

    return newEntry;
  }, []);

  const logCreate = useCallback(
    (resource: string, resourceId: string, details?: Record<string, unknown>) => {
      return addLog({ action: "CREATE", resource, resourceId, details });
    },
    [addLog]
  );

  const logUpdate = useCallback(
    (resource: string, resourceId: string, details?: Record<string, unknown>) => {
      return addLog({ action: "UPDATE", resource, resourceId, details });
    },
    [addLog]
  );

  const logDelete = useCallback(
    (resource: string, resourceId: string, details?: Record<string, unknown>) => {
      return addLog({ action: "DELETE", resource, resourceId, details });
    },
    [addLog]
  );

  const logAction = useCallback(
    (action: string, resource: string, resourceId: string, details?: Record<string, unknown>) => {
      return addLog({ action, resource, resourceId, details });
    },
    [addLog]
  );

  const clearLogs = useCallback(() => {
    setLogs([]);
    try {
      localStorage.removeItem(STORAGE_KEY);
    } catch {
      // Ignore
    }
  }, []);

  const exportLogs = useCallback((): string => {
    return JSON.stringify(logs, null, 2);
  }, [logs]);

  const getLogsByResource = useCallback(
    (resource: string): AuditEntry[] => {
      return logs.filter((log) => log.resource === resource);
    },
    [logs]
  );

  const getRecentLogs = useCallback(
    (count: number = 50): AuditEntry[] => {
      return logs.slice(0, count);
    },
    [logs]
  );

  return {
    logs,
    addLog,
    logCreate,
    logUpdate,
    logDelete,
    logAction,
    clearLogs,
    exportLogs,
    getLogsByResource,
    getRecentLogs,
  };
}