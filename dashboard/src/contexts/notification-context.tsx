"use client";

import React, { createContext, useContext, useState, useCallback, useEffect } from "react";
import { notificationsApi, type Notification, type NotificationSummary } from "@/lib/api";
import { toast } from "sonner";

interface NotificationContextType {
  notifications: Notification[];
  unreadCount: number;
  summary: NotificationSummary | null;
  isLoading: boolean;
  fetchNotifications: () => Promise<void>;
  fetchSummary: () => Promise<void>;
  markAsRead: (id: string) => Promise<void>;
  markAllAsRead: () => Promise<void>;
  archive: (id: string) => Promise<void>;
  archiveAll: () => Promise<void>;
  deleteNotification: (id: string) => Promise<void>;
  createNotification: (data: { user_id: string; type: Notification["type"]; title: string; message: string; channel?: Notification["channel"]; data?: Record<string, unknown>; link?: string }) => Promise<void>;
}

const NotificationContext = createContext<NotificationContextType | undefined>(undefined);

export function NotificationProvider({ children }: { children: React.ReactNode }) {
  const [notifications, setNotifications] = useState<Notification[]>([]);
  const [summary, setSummary] = useState<NotificationSummary | null>(null);
  const [isLoading, setIsLoading] = useState(false);

  const fetchNotifications = useCallback(async () => {
    setIsLoading(true);
    try {
      const response = await notificationsApi.list({ limit: 20 });
      setNotifications(response.notifications);
    } catch (error) {
      console.error("Failed to fetch notifications:", error);
    } finally {
      setIsLoading(false);
    }
  }, []);

  const fetchSummary = useCallback(async () => {
    try {
      const sum = await notificationsApi.summary();
      setSummary(sum);
    } catch (error) {
      console.error("Failed to fetch notification summary:", error);
    }
  }, []);

  const markAsRead = useCallback(async (id: string) => {
    try {
      await notificationsApi.markRead(id);
      setNotifications(prev =>
        prev.map(n => n.id === id ? { ...n, status: "read" as const } : n)
      );
      fetchSummary();
    } catch (error) {
      toast.error("Failed to mark notification as read");
    }
  }, [fetchSummary, toast]);

  const markAllAsRead = useCallback(async () => {
    try {
      await notificationsApi.markAllRead();
      setNotifications(prev =>
        prev.map(n => ({ ...n, status: "read" as const }))
      );
      fetchSummary();
      toast.success("All notifications marked as read");
    } catch (error) {
      toast.error("Failed to mark all as read");
    }
  }, [fetchSummary, toast]);

  const archive = useCallback(async (id: string) => {
    try {
      await notificationsApi.archive(id);
      setNotifications(prev => prev.filter(n => n.id !== id));
      fetchSummary();
    } catch (error) {
      toast.error("Failed to archive notification");
    }
  }, [fetchSummary, toast]);

  const archiveAll = useCallback(async () => {
    try {
      await notificationsApi.archiveAll();
      setNotifications([]);
      fetchSummary();
      toast.success("All notifications archived");
    } catch (error) {
      toast.error("Failed to archive all notifications");
    }
  }, [fetchSummary, toast]);

  const deleteNotification = useCallback(async (id: string) => {
    try {
      await notificationsApi.delete(id);
      setNotifications(prev => prev.filter(n => n.id !== id));
      fetchSummary();
      toast.success("Notification deleted");
    } catch (error) {
      toast.error("Failed to delete notification");
    }
  }, [fetchSummary, toast]);

  const createNotification = useCallback(async (data: { user_id: string; type: Notification["type"]; title: string; message: string; channel?: Notification["channel"]; data?: Record<string, unknown>; link?: string }) => {
    try {
      await notificationsApi.create(data);
      fetchNotifications();
      fetchSummary();
    } catch (error) {
      toast.error("Failed to create notification");
    }
  }, [fetchNotifications, fetchSummary, toast]);

  useEffect(() => {
    fetchNotifications();
    fetchSummary();
  }, [fetchNotifications, fetchSummary]);

  const unreadCount = summary?.unread ?? 0;

  return (
    <NotificationContext.Provider
      value={{
        notifications,
        unreadCount,
        summary,
        isLoading,
        fetchNotifications,
        fetchSummary,
        markAsRead,
        markAllAsRead,
        archive,
        archiveAll,
        deleteNotification,
        createNotification,
      }}
    >
      {children}
    </NotificationContext.Provider>
  );
}

export function useNotifications() {
  const context = useContext(NotificationContext);
  if (context === undefined) {
    throw new Error("useNotifications must be used within a NotificationProvider");
  }
  return context;
}