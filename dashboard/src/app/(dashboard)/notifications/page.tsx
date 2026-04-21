"use client";

import { useState } from "react";
import { useNotifications } from "@/contexts/notification-context";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { toast } from "sonner";
import { Check, Archive, Trash2, Bell, CheckCheck, ArchiveIcon } from "lucide-react";
import { cn } from "@/lib/utils";

export default function NotificationsPage() {
  const {
    notifications,
    summary,
    isLoading,
    fetchNotifications,
    markAsRead,
    markAllAsRead,
    archive,
    archiveAll,
    deleteNotification,
  } = useNotifications();

  const [filter, setFilter] = useState<"all" | "unread" | "read" | "archived">("all");

  const filteredNotifications = notifications.filter((n) => {
    if (filter === "unread") return n.status === "unread";
    if (filter === "read") return n.status === "read";
    if (filter === "archived") return n.status === "archived";
    return true;
  });

  const getTypeStyles = (type: string) => {
    switch (type) {
      case "success":
        return "bg-green-500/10 text-green-500 border-green-500/20";
      case "warning":
        return "bg-yellow-500/10 text-yellow-500 border-yellow-500/20";
      case "error":
        return "bg-red-500/10 text-red-500 border-red-500/20";
      default:
        return "bg-blue-500/10 text-blue-500 border-blue-500/20";
    }
  };

  const getStatusBadge = (status: string) => {
    switch (status) {
      case "unread":
        return <Badge variant="default">Unread</Badge>;
      case "read":
        return <Badge variant="secondary">Read</Badge>;
      case "archived":
        return <Badge variant="outline">Archived</Badge>;
      default:
        return null;
    }
  };

  const handleMarkAsRead = async (id: string) => {
    await markAsRead(id);
    toast.success("Notification marked as read");
  };

  const handleArchive = async (id: string) => {
    await archive(id);
    toast.success("Notification archived");
  };

  const handleDelete = async (id: string) => {
    await deleteNotification(id);
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Notifications</h1>
          <p className="text-muted-foreground">Manage your notifications and preferences</p>
        </div>
      </div>

      <div className="grid gap-4 md:grid-cols-4">
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">Total</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{summary?.total ?? 0}</div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">Unread</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-blue-500">{summary?.unread ?? 0}</div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">Read</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-green-500">{summary?.read ?? 0}</div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">Archived</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-muted-foreground">{summary?.archived ?? 0}</div>
          </CardContent>
        </Card>
      </div>

      <div className="flex items-center gap-2">
        <Button variant="outline" onClick={markAllAsRead} disabled={!summary?.unread}>
          <CheckCheck className="mr-2 h-4 w-4" />
          Mark All Read
        </Button>
        <Button variant="outline" onClick={archiveAll}>
          <ArchiveIcon className="mr-2 h-4 w-4" />
          Archive All
        </Button>
      </div>

      <Tabs value={filter} onValueChange={(v) => setFilter(v as typeof filter)}>
        <TabsList>
          <TabsTrigger value="all">All</TabsTrigger>
          <TabsTrigger value="unread">Unread</TabsTrigger>
          <TabsTrigger value="read">Read</TabsTrigger>
          <TabsTrigger value="archived">Archived</TabsTrigger>
        </TabsList>
        <TabsContent value={filter} className="mt-4">
          <Card>
            <CardContent className="p-0">
              {filteredNotifications.length === 0 ? (
                <div className="flex flex-col items-center justify-center py-12 text-muted-foreground">
                  <Bell className="h-12 w-12 mb-4" />
                  <p>No notifications</p>
                </div>
              ) : (
                <div className="divide-y">
                  {filteredNotifications.map((notification) => (
                    <div
                      key={notification.id}
                      className={cn(
                        "flex items-start gap-4 p-4 hover:bg-muted/50 transition-colors",
                        notification.status === "unread" && "bg-blue-500/5"
                      )}
                    >
                      <div className={cn("mt-1 rounded-full p-2 border", getTypeStyles(notification.type))}>
                        <Bell className="h-4 w-4" />
                      </div>
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2">
                          <p className="font-medium truncate">{notification.title}</p>
                          {getStatusBadge(notification.status)}
                        </div>
                        <p className="text-sm text-muted-foreground mt-1">
                          {notification.message}
                        </p>
                        <div className="flex items-center gap-4 mt-2">
                          <p className="text-xs text-muted-foreground">
                            {new Date(notification.created_at).toLocaleString()}
                          </p>
                          {notification.link && (
                            <a
                              href={notification.link}
                              className="text-xs text-blue-500 hover:underline"
                            >
                              View details
                            </a>
                          )}
                        </div>
                      </div>
                      <div className="flex items-center gap-2">
                        {notification.status === "unread" && (
                          <Button
                            variant="ghost"
                            size="icon"
                            onClick={() => handleMarkAsRead(notification.id)}
                          >
                            <Check className="h-4 w-4" />
                          </Button>
                        )}
                        <Button
                          variant="ghost"
                          size="icon"
                          onClick={() => handleArchive(notification.id)}
                        >
                          <Archive className="h-4 w-4" />
                        </Button>
                        <Button
                          variant="ghost"
                          size="icon"
                          onClick={() => handleDelete(notification.id)}
                        >
                          <Trash2 className="h-4 w-4" />
                        </Button>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  );
}