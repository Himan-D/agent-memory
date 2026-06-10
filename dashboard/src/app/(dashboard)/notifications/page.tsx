"use client";

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useNotifications } from "@/contexts/notification-context";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { toast } from "sonner";
import {
  Check,
  Archive,
  Trash2,
  Bell,
  CheckCheck,
  ArchiveIcon,
  Info,
  CheckCircle2,
  AlertTriangle,
  XCircle,
  Settings,
  Search,
} from "lucide-react";
import { cn } from "@/lib/utils";
import { notificationsApi } from "@/lib/api";

type TypeFilter = "all" | "info" | "success" | "warning" | "error";

const typeFilterOptions: { value: TypeFilter; label: string; className: string }[] = [
  { value: "all", label: "All", className: "" },
  { value: "info", label: "Info", className: "border-blue-500/30 text-blue-600" },
  { value: "success", label: "Success", className: "border-green-500/30 text-green-600" },
  { value: "warning", label: "Warning", className: "border-yellow-500/30 text-yellow-700" },
  { value: "error", label: "Error", className: "border-red-500/30 text-red-600" },
];

function TypeIcon({ type }: { type: string }) {
  switch (type) {
    case "success":
      return <CheckCircle2 className="h-4 w-4" />;
    case "warning":
      return <AlertTriangle className="h-4 w-4" />;
    case "error":
      return <XCircle className="h-4 w-4" />;
    default:
      return <Info className="h-4 w-4" />;
  }
}

const typeIconStyles: Record<string, string> = {
  success: "bg-green-500/10 text-green-500 border-green-500/20",
  warning: "bg-yellow-500/10 text-yellow-500 border-yellow-500/20",
  error: "bg-red-500/10 text-red-500 border-red-500/20",
  info: "bg-blue-500/10 text-blue-500 border-blue-500/20",
};

function getTypeStyles(type: string): string {
  return typeIconStyles[type] || typeIconStyles.info;
}

function getStatusBadge(status: string) {
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
}

export default function NotificationsPage() {
  const queryClient = useQueryClient();
  const {
    notifications,
    summary,
    markAsRead,
    markAllAsRead,
    archive,
    archiveAll,
    deleteNotification,
  } = useNotifications();

  const [filter, setFilter] = useState<"all" | "unread" | "read" | "archived">("all");
  const [typeFilter, setTypeFilter] = useState<TypeFilter>("all");
  const [searchQuery, setSearchQuery] = useState("");
  const [isPrefsOpen, setIsPrefsOpen] = useState(false);

  const { data: prefsData, isLoading: isLoadingPrefs } = useQuery({
    queryKey: ["notification-preferences"],
    queryFn: () => notificationsApi.getPreferences(),
    enabled: isPrefsOpen,
  });

  const [localPrefs, setLocalPrefs] = useState({
    in_app_enabled: true,
    email_enabled: false,
    webhook_enabled: false,
    email_address: "",
    webhook_url: "",
  });

  const updatePrefsMutation = useMutation({
    mutationFn: (data: typeof localPrefs) => notificationsApi.updatePreferences(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["notification-preferences"] });
      setIsPrefsOpen(false);
      toast.success("Preferences saved");
    },
    onError: () => toast.error("Failed to save preferences"),
  });

  const handleOpenPrefs = () => {
    if (prefsData) {
      setLocalPrefs({
        in_app_enabled: prefsData.in_app_enabled,
        email_enabled: prefsData.email_enabled,
        webhook_enabled: prefsData.webhook_enabled,
        email_address: prefsData.email_address ?? "",
        webhook_url: prefsData.webhook_url ?? "",
      });
    }
    setIsPrefsOpen(true);
  };

  const filteredNotifications = notifications.filter((n) => {
    const matchesStatus =
      filter === "all" ||
      (filter === "unread" && n.status === "unread") ||
      (filter === "read" && n.status === "read") ||
      (filter === "archived" && n.status === "archived");

    const matchesType = typeFilter === "all" || n.type === typeFilter;

    const matchesSearch =
      searchQuery === "" ||
      n.title.toLowerCase().includes(searchQuery.toLowerCase()) ||
      n.message.toLowerCase().includes(searchQuery.toLowerCase());

    return matchesStatus && matchesType && matchesSearch;
  });

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
    toast.success("Notification deleted");
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Notifications</h1>
          <p className="text-muted-foreground">Manage your notifications and preferences</p>
        </div>
        <Button variant="outline" onClick={handleOpenPrefs}>
          <Settings className="mr-2 h-4 w-4" />
          Preferences
        </Button>
      </div>

      <div className="grid gap-4 md:grid-cols-4">
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">Total</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold tabular-nums">{summary?.total ?? 0}</div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">Unread</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-blue-500 tabular-nums">
              {summary?.unread ?? 0}
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">Read</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-green-500 tabular-nums">
              {summary?.read ?? 0}
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">Archived</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-muted-foreground tabular-nums">
              {summary?.archived ?? 0}
            </div>
          </CardContent>
        </Card>
      </div>

      <div className="flex flex-wrap items-center gap-2">
        <Button variant="outline" onClick={markAllAsRead} disabled={!summary?.unread}>
          <CheckCheck className="mr-2 h-4 w-4" />
          Mark All Read
        </Button>
        <Button variant="outline" onClick={archiveAll}>
          <ArchiveIcon className="mr-2 h-4 w-4" />
          Archive All
        </Button>
      </div>

      {/* Type filter chips */}
      <div className="flex flex-wrap gap-2">
        {typeFilterOptions.map((opt) => (
          <button
            key={opt.value}
            onClick={() => setTypeFilter(opt.value)}
            className={cn(
              "inline-flex items-center gap-1.5 rounded-full border px-3 py-1 text-sm transition-colors",
              typeFilter === opt.value
                ? "bg-foreground text-background border-foreground"
                : cn("bg-background hover:bg-muted/60", opt.className || "text-muted-foreground"),
            )}
          >
            {opt.value !== "all" && (
              <span
                className={cn(
                  "h-2 w-2 rounded-full",
                  opt.value === "info" && "bg-blue-500",
                  opt.value === "success" && "bg-green-500",
                  opt.value === "warning" && "bg-yellow-500",
                  opt.value === "error" && "bg-red-500",
                )}
              />
            )}
            {opt.label}
          </button>
        ))}
      </div>

      {/* Search */}
      <div className="relative">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
        <Input
          placeholder="Search notifications..."
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          className="pl-9"
        />
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
                  {(searchQuery || typeFilter !== "all") && (
                    <Button
                      variant="ghost"
                      className="mt-2"
                      onClick={() => {
                        setSearchQuery("");
                        setTypeFilter("all");
                      }}
                    >
                      Clear filters
                    </Button>
                  )}
                </div>
              ) : (
                <div className="divide-y">
                  {filteredNotifications.map((notification) => (
                    <div
                      key={notification.id}
                      className={cn(
                        "flex items-start gap-4 p-4 hover:bg-muted/50 transition-colors",
                        notification.status === "unread" && "bg-blue-500/5",
                      )}
                    >
                      <div
                        className={cn(
                          "mt-1 rounded-full p-2 border",
                          getTypeStyles(notification.type),
                        )}
                      >
                        <TypeIcon type={notification.type} />
                      </div>
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2 flex-wrap">
                          <p className="font-medium truncate">{notification.title}</p>
                          {getStatusBadge(notification.status)}
                        </div>
                        <p className="text-sm text-muted-foreground mt-1">{notification.message}</p>
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
                            aria-label="Mark as read"
                            onClick={() => handleMarkAsRead(notification.id)}
                          >
                            <Check className="h-4 w-4" />
                          </Button>
                        )}
                        <Button
                          variant="ghost"
                          size="icon"
                          aria-label="Archive notification"
                          onClick={() => handleArchive(notification.id)}
                        >
                          <Archive className="h-4 w-4" />
                        </Button>
                        <Button
                          variant="ghost"
                          size="icon"
                          aria-label="Delete notification"
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

      {/* Preferences Dialog */}
      <Dialog open={isPrefsOpen} onOpenChange={setIsPrefsOpen}>
        <DialogContent className="sm:max-w-[420px]">
          <DialogHeader>
            <DialogTitle>Notification Preferences</DialogTitle>
            <DialogDescription>
              Choose which channels you want to receive notifications on.
            </DialogDescription>
          </DialogHeader>
          {isLoadingPrefs ? (
            <div className="py-8 text-center text-muted-foreground text-sm">Loading…</div>
          ) : (
            <div className="grid gap-4 py-4">
              <div className="flex items-center justify-between">
                <div>
                  <Label htmlFor="pref-in-app" className="font-medium">
                    In-app notifications
                  </Label>
                  <p className="text-xs text-muted-foreground">Show in the notification panel</p>
                </div>
                <Switch
                  id="pref-in-app"
                  checked={localPrefs.in_app_enabled}
                  onCheckedChange={(v) => setLocalPrefs({ ...localPrefs, in_app_enabled: v })}
                />
              </div>
              <div className="flex items-center justify-between">
                <div>
                  <Label htmlFor="pref-email" className="font-medium">
                    Email notifications
                  </Label>
                  <p className="text-xs text-muted-foreground">Send to your account email</p>
                </div>
                <Switch
                  id="pref-email"
                  checked={localPrefs.email_enabled}
                  onCheckedChange={(v) => setLocalPrefs({ ...localPrefs, email_enabled: v })}
                />
              </div>
              <div className="flex items-center justify-between">
                <div>
                  <Label htmlFor="pref-webhook" className="font-medium">
                    Webhook notifications
                  </Label>
                  <p className="text-xs text-muted-foreground">POST to your configured webhook</p>
                </div>
                <Switch
                  id="pref-webhook"
                  checked={localPrefs.webhook_enabled}
                  onCheckedChange={(v) => setLocalPrefs({ ...localPrefs, webhook_enabled: v })}
                />
              </div>
              {localPrefs.email_enabled && (
                <div className="grid gap-2">
                  <Label htmlFor="pref-email-address">Email address</Label>
                  <Input
                    id="pref-email-address"
                    type="email"
                    placeholder="you@example.com"
                    value={localPrefs.email_address}
                    onChange={(e) => setLocalPrefs({ ...localPrefs, email_address: e.target.value })}
                  />
                </div>
              )}
              {localPrefs.webhook_enabled && (
                <div className="grid gap-2">
                  <Label htmlFor="pref-webhook-url">Webhook URL</Label>
                  <Input
                    id="pref-webhook-url"
                    type="url"
                    placeholder="https://example.com/webhook"
                    value={localPrefs.webhook_url}
                    onChange={(e) => setLocalPrefs({ ...localPrefs, webhook_url: e.target.value })}
                  />
                </div>
              )}
            </div>
          )}
          <DialogFooter>
            <Button variant="outline" onClick={() => setIsPrefsOpen(false)}>
              Cancel
            </Button>
            <Button
              onClick={() => updatePrefsMutation.mutate(localPrefs)}
              disabled={updatePrefsMutation.isPending}
            >
              {updatePrefsMutation.isPending ? "Saving..." : "Save Preferences"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
