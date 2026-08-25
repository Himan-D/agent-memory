"use client";

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Card, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { Skeleton } from "@/components/ui/skeleton";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { FilterComponent } from "@/components/ui/filter-component";
import {
  Webhook,
  Plus,
  Trash2,
  Play,
  AlertCircle,
  RefreshCw,
  Edit,
  Copy,
  History,
  RotateCcw,
  CheckCircle2,
  XCircle,
  Clock,
} from "lucide-react";
import { toast } from "sonner";
import { cn } from "@/lib/utils";
import {
  webhooksApi,
  type Webhook as WebhookType,
  type WebhookDelivery,
  type WebhookDeadLetterEntry,
} from "@/lib/api";

const availableEvents = [
  "memory.created",
  "memory.updated",
  "memory.deleted",
  "memory.archived",
  "entity.created",
  "entity.updated",
  "entity.deleted",
  "session.created",
  "session.ended",
  "skill.executed",
  "alert.triggered",
  "search.performed",
  "agent.connected",
  "agent.disconnected",
];

const availableFields = ["id", "content", "user_id", "agent_id", "session_id", "metadata", "created_at", "updated_at"];

function getEventBadgeClass(event: string): string {
  if (event.startsWith("memory.")) return "bg-blue-500/10 text-blue-600 border-blue-500/20";
  if (event.startsWith("entity.")) return "bg-green-500/10 text-green-600 border-green-500/20";
  if (event.startsWith("session.")) return "bg-purple-500/10 text-purple-600 border-purple-500/20";
  if (event.startsWith("agent.")) return "bg-orange-500/10 text-orange-600 border-orange-500/20";
  if (event.startsWith("chain.")) return "bg-yellow-500/10 text-yellow-700 border-yellow-500/20";
  if (event.startsWith("skill.")) return "bg-pink-500/10 text-pink-600 border-pink-500/20";
  return "";
}

function formatRelativeTime(dateStr: string): string {
  const date = new Date(dateStr);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffSecs = Math.floor(diffMs / 1000);
  const diffMins = Math.floor(diffSecs / 60);
  const diffHours = Math.floor(diffMins / 60);
  const diffDays = Math.floor(diffHours / 24);

  if (diffSecs < 60) return "just now";
  if (diffMins < 60) return `${diffMins} minute${diffMins !== 1 ? "s" : ""} ago`;
  if (diffHours < 24) return `${diffHours} hour${diffHours !== 1 ? "s" : ""} ago`;
  if (diffDays < 30) return `${diffDays} day${diffDays !== 1 ? "s" : ""} ago`;
  return date.toLocaleDateString();
}

function getHealthDot(webhook: WebhookType): { color: string; label: string } {
  if (!webhook.active) return { color: "bg-red-500", label: "Inactive" };
  const lastAt = webhook.last_delivery_at || webhook.last_triggered;
  if (!lastAt) return { color: "bg-yellow-500", label: "No deliveries yet" };
  const lastTriggered = new Date(lastAt);
  const hourAgo = new Date(Date.now() - 60 * 60 * 1000);
  const failures = webhook.failure_count ?? 0;
  const successes = webhook.success_count ?? 0;
  if (failures > 0 && successes === 0) return { color: "bg-red-500", label: "All deliveries failing" };
  if (failures > successes) return { color: "bg-yellow-500", label: "Elevated failure rate" };
  if (lastTriggered > hourAgo) return { color: "bg-green-500", label: "Healthy" };
  return { color: "bg-yellow-500", label: "Not triggered recently" };
}

export default function WebhooksPage() {
  const queryClient = useQueryClient();
  const [searchQuery, setSearchQuery] = useState("");
  const [statusFilter, setStatusFilter] = useState("all");
  const [dateFrom, setDateFrom] = useState<Date | null>(null);
  const [dateTo, setDateTo] = useState<Date | null>(null);
  const [isCreateOpen, setIsCreateOpen] = useState(false);
  const [newWebhook, setNewWebhook] = useState({ url: "", events: [] as string[], fields: [] as string[] });
  const [editingWebhook, setEditingWebhook] = useState<WebhookType | null>(null);
  const [deliveriesWebhookId, setDeliveriesWebhookId] = useState<string | null>(null);
  const [testingId, setTestingId] = useState<string | null>(null);
  const [deletingId, setDeletingId] = useState<string | null>(null);
  const [activeTab, setActiveTab] = useState("webhooks");

  const { data: webhooksData, isLoading, refetch } = useQuery({
    queryKey: ["webhooks"],
    queryFn: () => webhooksApi.list(),
  });

  const { data: deliveriesData, isLoading: isLoadingDeliveries } = useQuery({
    queryKey: ["webhook-deliveries", deliveriesWebhookId],
    queryFn: () => webhooksApi.getDeliveries(deliveriesWebhookId!),
    enabled: !!deliveriesWebhookId,
  });

  const { data: deadLetterData, isLoading: isLoadingDeadLetter } = useQuery({
    queryKey: ["webhook-dead-letter"],
    queryFn: () => webhooksApi.getDeadLetter(),
    enabled: activeTab === "dead-letter",
  });

  const createMutation = useMutation({
    mutationFn: (data: Partial<WebhookType>) => webhooksApi.create(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["webhooks"] });
      setIsCreateOpen(false);
      setNewWebhook({ url: "", events: [], fields: [] });
      toast.success("Webhook created successfully");
    },
    onError: () => toast.error("Failed to create webhook"),
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<WebhookType> }) =>
      webhooksApi.update(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["webhooks"] });
      setEditingWebhook(null);
      toast.success("Webhook updated successfully");
    },
    onError: () => toast.error("Failed to update webhook"),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => webhooksApi.delete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["webhooks"] });
      setDeletingId(null);
      toast.success("Webhook deleted");
    },
    onError: () => {
      setDeletingId(null);
      toast.error("Failed to delete webhook");
    },
  });

  const retryDeadLetterMutation = useMutation({
    mutationFn: ({ webhookId, event }: { webhookId: string; event: string }) =>
      webhooksApi.retryDeadLetter(webhookId, event),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["webhook-dead-letter"] });
      toast.success("Retry queued");
    },
    onError: () => toast.error("Failed to queue retry"),
  });

  const webhooks = webhooksData?.webhooks || [];

  const clearFilters = () => {
    setSearchQuery("");
    setStatusFilter("all");
    setDateFrom(null);
    setDateTo(null);
  };

  const filteredWebhooks = webhooks.filter((webhook) => {
    const matchesSearch =
      searchQuery === "" ||
      webhook.url.toLowerCase().includes(searchQuery.toLowerCase()) ||
      webhook.id.toLowerCase().includes(searchQuery.toLowerCase()) ||
      (webhook.events || []).some((e) => e.toLowerCase().includes(searchQuery.toLowerCase()));

    const matchesStatus =
      statusFilter === "all" ||
      (statusFilter === "active" && webhook.active) ||
      (statusFilter === "inactive" && !webhook.active);

    const webhookDate = new Date(webhook.created_at || Date.now());
    const matchesFrom = !dateFrom || webhookDate >= dateFrom;
    const matchesTo = !dateTo || webhookDate <= dateTo;

    return matchesSearch && matchesStatus && matchesFrom && matchesTo;
  });

  const handleCreate = () => {
    if (!newWebhook.url.trim()) {
      toast.error("Webhook URL is required");
      return;
    }
    if (newWebhook.events.length === 0) {
      toast.error("Select at least one event");
      return;
    }
    createMutation.mutate({ url: newWebhook.url, events: newWebhook.events, fields: newWebhook.fields, active: true });
  };

  const handleUpdate = () => {
    if (!editingWebhook) return;
    if (!editingWebhook.url.trim()) {
      toast.error("Webhook URL is required");
      return;
    }
    updateMutation.mutate({
      id: editingWebhook.id,
      data: {
        url: editingWebhook.url,
        events: editingWebhook.events,
        fields: editingWebhook.fields || [],
        active: editingWebhook.active,
      },
    });
  };

  const handleDelete = (id: string) => {
    if (!confirm("Are you sure you want to delete this webhook?")) return;
    setDeletingId(id);
    deleteMutation.mutate(id);
  };

  const handleTest = async (id: string) => {
    try {
      setTestingId(id);
      const result = await webhooksApi.test(id);
      if (result.success) {
        toast.success(result.message || "Test webhook sent successfully");
      } else {
        toast.error(result.message || "Test webhook failed");
      }
    } catch (error: any) {
      toast.error(error?.message || "Failed to test webhook");
    } finally {
      setTestingId(null);
    }
  };

  const handleCopySecret = (secret: string) => {
    navigator.clipboard.writeText(secret).then(
      () => toast.success("Secret copied to clipboard"),
      () => toast.error("Failed to copy secret"),
    );
  };

  const toggleNewEvent = (event: string) => {
    setNewWebhook((prev) => ({
      ...prev,
      events: prev.events.includes(event)
        ? prev.events.filter((e) => e !== event)
        : [...prev.events, event],
    }));
  };

  const toggleNewField = (field: string) => {
    setNewWebhook((prev) => ({
      ...prev,
      fields: prev.fields.includes(field)
        ? prev.fields.filter((f) => f !== field)
        : [...prev.fields, field],
    }));
  };

  const toggleEditEvent = (event: string) => {
    if (!editingWebhook) return;
    setEditingWebhook({
      ...editingWebhook,
      events: editingWebhook.events.includes(event)
        ? editingWebhook.events.filter((e) => e !== event)
        : [...editingWebhook.events, event],
    });
  };

  const toggleEditField = (field: string) => {
    if (!editingWebhook) return;
    const fields = editingWebhook.fields || [];
    setEditingWebhook({
      ...editingWebhook,
      fields: fields.includes(field)
        ? fields.filter((f) => f !== field)
        : [...fields, field],
    });
  };

  const deliveries: WebhookDelivery[] = deliveriesData?.deliveries || [];
  const deadLetterEntries: WebhookDeadLetterEntry[] = deadLetterData?.entries || [];
  const deliveriesWebhook = webhooks.find((w) => w.id === deliveriesWebhookId);

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">Webhooks</h1>
          <p className="text-muted-foreground">Configure real-time event notifications</p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" size="icon" onClick={() => refetch()} aria-label="Refresh data">
            <RefreshCw className="h-4 w-4" />
          </Button>
          <Button onClick={() => setIsCreateOpen(true)}>
            <Plus className="mr-2 h-4 w-4" />
            Create Webhook
          </Button>
        </div>
      </div>

      <FilterComponent
        searchValue={searchQuery}
        onSearchChange={setSearchQuery}
        searchPlaceholder="Search webhooks..."
        typeValue={statusFilter}
        onTypeChange={setStatusFilter}
        typeOptions={[
          { label: "All Status", value: "all" },
          { label: "Active", value: "active" },
          { label: "Inactive", value: "inactive" },
        ]}
        dateFrom={dateFrom}
        onDateFromChange={setDateFrom}
        dateTo={dateTo}
        onDateToChange={setDateTo}
        onClear={clearFilters}
      />

      <div className="rounded-lg border bg-yellow-50 p-4 text-yellow-800 dark:bg-yellow-950 dark:text-yellow-300">
        <div className="flex items-start gap-3">
          <AlertCircle className="mt-0.5 h-5 w-5 flex-shrink-0" />
          <div>
            <p className="font-medium">Webhook payloads are signed</p>
            <p className="text-sm">
              Each request includes a signature header for verification. See documentation for details.
            </p>
          </div>
        </div>
      </div>

      <Tabs value={activeTab} onValueChange={setActiveTab}>
        <TabsList>
          <TabsTrigger value="webhooks">
            Webhooks ({filteredWebhooks.length})
          </TabsTrigger>
          <TabsTrigger value="dead-letter">Dead Letter Queue</TabsTrigger>
        </TabsList>

        <TabsContent value="webhooks" className="mt-4">
          {isLoading ? (
            <div className="space-y-4">
              {[1, 2, 3].map((i) => (
                <Card key={i}>
                  <CardContent className="p-4">
                    <Skeleton className="h-20 w-full" />
                  </CardContent>
                </Card>
              ))}
            </div>
          ) : filteredWebhooks.length === 0 ? (
            <Card>
              <CardContent className="flex flex-col items-center justify-center py-12">
                <Webhook className="h-12 w-12 text-muted-foreground mb-4" />
                <p className="text-muted-foreground">No webhooks found</p>
                {searchQuery && (
                  <Button variant="ghost" onClick={clearFilters} className="mt-2">
                    Clear filters
                  </Button>
                )}
              </CardContent>
            </Card>
          ) : (
            <div className="space-y-4">
              {filteredWebhooks.map((webhook) => {
                const health = getHealthDot(webhook);
                return (
                  <Card key={webhook.id}>
                    <CardContent className="p-4">
                      <div className="flex items-start justify-between gap-4">
                        <div className="flex items-start gap-4 min-w-0">
                          <div className="relative mt-1 rounded-lg bg-muted p-2 flex-shrink-0">
                            <Webhook className="h-5 w-5 text-muted-foreground" />
                            <span
                              className={cn(
                                "absolute -top-1 -right-1 h-3 w-3 rounded-full border-2 border-background",
                                health.color,
                              )}
                              title={health.label}
                            />
                          </div>
                          <div className="space-y-2 min-w-0">
                            <div className="flex items-center gap-2 flex-wrap">
                              <p className="font-mono text-sm">{webhook.id}</p>
                              <Badge variant={webhook.active ? "default" : "secondary"}>
                                {webhook.active ? "Active" : "Inactive"}
                              </Badge>
                            </div>
                            <p className="text-sm break-all">{webhook.url}</p>
                            <div className="flex flex-wrap gap-1">
                              {(webhook.events || []).map((event) => (
                                <Badge
                                  key={event}
                                  variant="outline"
                                  className={cn("text-xs", getEventBadgeClass(event))}
                                >
                                  {event}
                                </Badge>
                              ))}
                            </div>
                            {(webhook.last_delivery_at || webhook.last_triggered) && (
                              <p className="text-xs text-muted-foreground">
                                Last triggered:{" "}
                                {formatRelativeTime(
                                  webhook.last_delivery_at || webhook.last_triggered || "",
                                )}
                              </p>
                            )}
                          </div>
                        </div>
                        <div className="flex items-center gap-1 flex-shrink-0 flex-wrap justify-end">
                          {webhook.secret && (
                            <Button
                              variant="ghost"
                              size="sm"
                              onClick={() => handleCopySecret(webhook.secret!)}
                              title="Copy webhook secret"
                            >
                              <Copy className="h-3 w-3" />
                            </Button>
                          )}
                          <Button
                            variant="outline"
                            size="sm"
                            onClick={() => setDeliveriesWebhookId(webhook.id)}
                          >
                            <History className="mr-1 h-3 w-3" />
                            Deliveries
                          </Button>
                          <Button
                            variant="outline"
                            size="sm"
                            onClick={() => handleTest(webhook.id)}
                            disabled={testingId === webhook.id}
                          >
                            <Play className="mr-1 h-3 w-3" />
                            {testingId === webhook.id ? "Testing..." : "Test"}
                          </Button>
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => setEditingWebhook({ ...webhook })}
                          >
                            <Edit className="h-4 w-4" />
                          </Button>
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => handleDelete(webhook.id)}
                            disabled={deletingId === webhook.id}
                          >
                            <Trash2 className="h-4 w-4 text-destructive" />
                          </Button>
                        </div>
                      </div>
                    </CardContent>
                  </Card>
                );
              })}
            </div>
          )}
        </TabsContent>

        <TabsContent value="dead-letter" className="mt-4">
          {isLoadingDeadLetter ? (
            <div className="space-y-4">
              {[1, 2, 3].map((i) => (
                <Card key={i}>
                  <CardContent className="p-4">
                    <Skeleton className="h-16 w-full" />
                  </CardContent>
                </Card>
              ))}
            </div>
          ) : deadLetterEntries.length === 0 ? (
            <Card>
              <CardContent className="flex flex-col items-center justify-center py-12">
                <CheckCircle2 className="h-12 w-12 text-green-500 mb-4" />
                <p className="text-muted-foreground">No failed deliveries</p>
              </CardContent>
            </Card>
          ) : (
            <div className="space-y-3">
              {deadLetterEntries.map((entry) => (
                <Card key={entry.id}>
                  <CardContent className="p-4">
                    <div className="flex items-start justify-between gap-4">
                      <div className="space-y-1 min-w-0">
                        <div className="flex items-center gap-2 flex-wrap">
                          <Badge
                            variant="outline"
                            className={cn("text-xs", getEventBadgeClass(entry.event))}
                          >
                            {entry.event}
                          </Badge>
                          <span className="text-xs text-muted-foreground font-mono truncate">
                            {entry.webhook_id}
                          </span>
                        </div>
                        {entry.error && (
                          <p className="text-sm text-destructive truncate">{entry.error}</p>
                        )}
                        <p className="text-xs text-muted-foreground">
                          {entry.attempts} attempt{entry.attempts !== 1 ? "s" : ""} ·{" "}
                          {formatRelativeTime(entry.created_at)}
                        </p>
                      </div>
                      <Button
                        variant="outline"
                        size="sm"
                        className="flex-shrink-0"
                        onClick={() =>
                          retryDeadLetterMutation.mutate({
                            webhookId: entry.webhook_id,
                            event: entry.event,
                          })
                        }
                        disabled={retryDeadLetterMutation.isPending}
                      >
                        <RotateCcw className="mr-1 h-3 w-3" />
                        Retry
                      </Button>
                    </div>
                  </CardContent>
                </Card>
              ))}
            </div>
          )}
        </TabsContent>
      </Tabs>

      {/* Create Webhook Dialog */}
      <Dialog open={isCreateOpen} onOpenChange={setIsCreateOpen}>
        <DialogContent className="sm:max-w-[500px]">
          <DialogHeader>
            <DialogTitle>Create New Webhook</DialogTitle>
            <DialogDescription>Set up a webhook to receive event notifications</DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 py-4">
            <div className="grid gap-2">
              <Label htmlFor="url">Endpoint URL</Label>
              <Input
                id="url"
                placeholder="https://api.example.com/webhook"
                value={newWebhook.url}
                onChange={(e) => setNewWebhook({ ...newWebhook, url: e.target.value })}
              />
            </div>
            <div className="grid gap-2">
              <Label>Events</Label>
              <div className="grid grid-cols-2 gap-2">
                {availableEvents.map((event) => (
                  <div key={event} className="flex items-center gap-2">
                    <Switch
                      id={`new-${event}`}
                      checked={newWebhook.events.includes(event)}
                      onCheckedChange={() => toggleNewEvent(event)}
                    />
                    <Label htmlFor={`new-${event}`} className="text-sm font-normal">
                      {event}
                    </Label>
                  </div>
                ))}
              </div>
            </div>
            <div className="grid gap-2">
              <Label>Payload fields</Label>
              <div className="grid grid-cols-2 gap-2">
                {availableFields.map((field) => (
                  <div key={field} className="flex items-center gap-2">
                    <Switch
                      id={`new-field-${field}`}
                      checked={newWebhook.fields.includes(field)}
                      onCheckedChange={() => toggleNewField(field)}
                    />
                    <Label htmlFor={`new-field-${field}`} className="text-sm font-normal">
                      {field}
                    </Label>
                  </div>
                ))}
              </div>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setIsCreateOpen(false)}>
              Cancel
            </Button>
            <Button onClick={handleCreate} disabled={createMutation.isPending}>
              {createMutation.isPending ? "Creating..." : "Create Webhook"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Edit Webhook Dialog */}
      <Dialog open={!!editingWebhook} onOpenChange={() => setEditingWebhook(null)}>
        <DialogContent className="sm:max-w-[500px]">
          <DialogHeader>
            <DialogTitle>Edit Webhook</DialogTitle>
            <DialogDescription>Update webhook configuration</DialogDescription>
          </DialogHeader>
          {editingWebhook && (
            <div className="grid gap-4 py-4">
              <div className="grid gap-2">
                <Label htmlFor="edit-url">Endpoint URL</Label>
                <Input
                  id="edit-url"
                  value={editingWebhook.url}
                  onChange={(e) =>
                    setEditingWebhook({ ...editingWebhook, url: e.target.value })
                  }
                />
              </div>
              <div className="flex items-center gap-3">
                <Switch
                  id="edit-active"
                  checked={editingWebhook.active}
                  onCheckedChange={(checked) =>
                    setEditingWebhook({ ...editingWebhook, active: checked })
                  }
                />
                <Label htmlFor="edit-active">Active</Label>
              </div>
              <div className="grid gap-2">
                <Label>Events</Label>
                <div className="grid grid-cols-2 gap-2">
                  {availableEvents.map((event) => (
                    <div key={event} className="flex items-center gap-2">
                      <Switch
                        id={`edit-${event}`}
                        checked={editingWebhook.events.includes(event)}
                        onCheckedChange={() => toggleEditEvent(event)}
                      />
                      <Label htmlFor={`edit-${event}`} className="text-sm font-normal">
                        {event}
                      </Label>
                    </div>
                  ))}
                </div>
              </div>
              <div className="grid gap-2">
                <Label>Payload fields</Label>
                <div className="grid grid-cols-2 gap-2">
                  {availableFields.map((field) => (
                    <div key={field} className="flex items-center gap-2">
                      <Switch
                        id={`edit-field-${field}`}
                        checked={(editingWebhook.fields || []).includes(field)}
                        onCheckedChange={() => toggleEditField(field)}
                      />
                      <Label htmlFor={`edit-field-${field}`} className="text-sm font-normal">
                        {field}
                      </Label>
                    </div>
                  ))}
                </div>
              </div>
            </div>
          )}
          <DialogFooter>
            <Button variant="outline" onClick={() => setEditingWebhook(null)}>
              Cancel
            </Button>
            <Button onClick={handleUpdate} disabled={updateMutation.isPending}>
              {updateMutation.isPending ? "Saving..." : "Save Changes"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delivery History Dialog */}
      <Dialog open={!!deliveriesWebhookId} onOpenChange={() => setDeliveriesWebhookId(null)}>
        <DialogContent className="sm:max-w-[600px]">
          <DialogHeader>
            <DialogTitle>Delivery History</DialogTitle>
            <DialogDescription>
              Recent deliveries for{" "}
              <span className="font-mono text-xs">
                {deliveriesWebhook?.url || deliveriesWebhookId}
              </span>
            </DialogDescription>
          </DialogHeader>
          <div className="py-2 max-h-[420px] overflow-y-auto space-y-2">
            {isLoadingDeliveries ? (
              <div className="space-y-3">
                {[1, 2, 3].map((i) => (
                  <Skeleton key={i} className="h-14 w-full" />
                ))}
              </div>
            ) : deliveries.length === 0 ? (
              <div className="flex flex-col items-center justify-center py-10 text-muted-foreground">
                <History className="h-10 w-10 mb-3" />
                <p>No deliveries yet</p>
              </div>
            ) : (
              deliveries.map((delivery) => (
                <div
                  key={delivery.id}
                  className="flex items-center gap-3 p-3 rounded-lg border"
                >
                  {delivery.status === "success" ? (
                    <CheckCircle2 className="h-4 w-4 text-green-500 flex-shrink-0" />
                  ) : delivery.status === "failed" ? (
                    <XCircle className="h-4 w-4 text-red-500 flex-shrink-0" />
                  ) : (
                    <Clock className="h-4 w-4 text-yellow-500 flex-shrink-0" />
                  )}
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2 flex-wrap">
                      <Badge
                        variant="outline"
                        className={cn("text-xs", getEventBadgeClass(delivery.event))}
                      >
                        {delivery.event}
                      </Badge>
                      {delivery.response_code !== undefined && (
                        <span className="text-xs text-muted-foreground">
                          HTTP {delivery.response_code}
                        </span>
                      )}
                    </div>
                    {delivery.error && (
                      <p className="text-xs text-destructive mt-1 truncate">{delivery.error}</p>
                    )}
                  </div>
                  <div className="text-right text-xs text-muted-foreground flex-shrink-0">
                    <p>{formatRelativeTime(delivery.created_at)}</p>
                    {delivery.duration_ms !== undefined && (
                      <p className="tabular-nums">{delivery.duration_ms}ms</p>
                    )}
                  </div>
                </div>
              ))
            )}
          </div>
        </DialogContent>
      </Dialog>
    </div>
  );
}
