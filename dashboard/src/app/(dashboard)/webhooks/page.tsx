"use client";

import { useState, useEffect, useCallback } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { FilterComponent } from "@/components/ui/filter-component";
import { Webhook, Plus, Trash2, Play, ExternalLink, AlertCircle, RefreshCw } from "lucide-react";
import { toast } from "sonner";
import { webhooksApi, type Webhook as WebhookType } from "@/lib/api";

const availableEvents = [
  "memory.created",
  "memory.updated",
  "memory.deleted",
  "entity.created",
  "entity.updated",
  "agent.created",
  "agent.deleted",
  "chain.executed",
  "skill.executed",
  "session.created",
];

export default function WebhooksPage() {
  const [searchQuery, setSearchQuery] = useState("");
  const [statusFilter, setStatusFilter] = useState("all");
  const [dateFrom, setDateFrom] = useState<Date | null>(null);
  const [dateTo, setDateTo] = useState<Date | null>(null);
  const [webhooks, setWebhooks] = useState<WebhookType[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [isCreateOpen, setIsCreateOpen] = useState(false);
  const [newWebhook, setNewWebhook] = useState({ url: "", events: [] as string[] });
  const [isCreating, setIsCreating] = useState(false);
  const [deletingId, setDeletingId] = useState<string | null>(null);
  const [testingId, setTestingId] = useState<string | null>(null);

  const fetchWebhooks = useCallback(async () => {
    try {
      setIsLoading(true);
      const response = await webhooksApi.list();
      setWebhooks(response.webhooks || []);
    } catch (error) {
      console.error("Failed to fetch webhooks:", error);
      toast.error("Failed to load webhooks");
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchWebhooks();
  }, [fetchWebhooks]);

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
      (webhook.events || []).some(e => e.toLowerCase().includes(searchQuery.toLowerCase()));

    const matchesStatus = statusFilter === "all" || 
      (statusFilter === "active" && webhook.active) ||
      (statusFilter === "inactive" && !webhook.active);

    const webhookDate = new Date(webhook.created_at || Date.now());
    const matchesFrom = !dateFrom || webhookDate >= dateFrom;
    const matchesTo = !dateTo || webhookDate <= dateTo;

    return matchesSearch && matchesStatus && matchesFrom && matchesTo;
  });

  const handleCreate = async () => {
    if (!newWebhook.url.trim()) {
      toast.error("Webhook URL is required");
      return;
    }

    if (newWebhook.events.length === 0) {
      toast.error("Select at least one event");
      return;
    }

    try {
      setIsCreating(true);
      const created = await webhooksApi.create({
        url: newWebhook.url,
        events: newWebhook.events,
        active: true,
      });
      setWebhooks(prev => [...prev, created]);
      setIsCreateOpen(false);
      setNewWebhook({ url: "", events: [] });
      toast.success("Webhook created successfully");
    } catch (error) {
      console.error("Failed to create webhook:", error);
      toast.error("Failed to create webhook");
    } finally {
      setIsCreating(false);
    }
  };

  const handleDelete = async (id: string) => {
    if (!confirm("Are you sure you want to delete this webhook?")) return;

    try {
      setDeletingId(id);
      await webhooksApi.delete(id);
      setWebhooks(prev => prev.filter(w => w.id !== id));
      toast.success("Webhook deleted");
    } catch (error) {
      console.error("Failed to delete webhook:", error);
      toast.error("Failed to delete webhook");
    } finally {
      setDeletingId(null);
    }
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
    } catch (error) {
      console.error("Failed to test webhook:", error);
      toast.error("Failed to test webhook");
    } finally {
      setTestingId(null);
    }
  };

  const toggleEvent = (event: string) => {
    setNewWebhook(prev => ({
      ...prev,
      events: prev.events.includes(event)
        ? prev.events.filter(e => e !== event)
        : [...prev.events, event],
    }));
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">Webhooks</h1>
          <p className="text-muted-foreground">Configure real-time event notifications</p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" size="icon" onClick={fetchWebhooks}>
            <RefreshCw className="h-4 w-4" />
          </Button>
          <Dialog open={isCreateOpen} onOpenChange={setIsCreateOpen}>
            <DialogTrigger asChild>
              <Button>
                <Plus className="mr-2 h-4 w-4" />
                Create Webhook
              </Button>
            </DialogTrigger>
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
                          id={event} 
                          checked={newWebhook.events.includes(event)}
                          onCheckedChange={() => toggleEvent(event)}
                        />
                        <Label htmlFor={event} className="text-sm font-normal">{event}</Label>
                      </div>
                    ))}
                  </div>
                </div>
              </div>
              <DialogFooter>
                <Button variant="outline" onClick={() => setIsCreateOpen(false)}>Cancel</Button>
                <Button onClick={handleCreate} disabled={isCreating}>
                  {isCreating ? "Creating..." : "Create Webhook"}
                </Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>
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
            <p className="text-sm">Each request includes a signature header for verification. See documentation for details.</p>
          </div>
        </div>
      </div>

      {isLoading ? (
        <div className="space-y-4">
          {[1, 2, 3].map(i => (
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
          {filteredWebhooks.map((webhook) => (
            <Card key={webhook.id}>
              <CardContent className="p-4">
                <div className="flex items-start justify-between">
                  <div className="flex items-start gap-4">
                    <div className="rounded-lg bg-muted p-2">
                      <Webhook className="h-5 w-5 text-muted-foreground" />
                    </div>
                    <div className="space-y-2">
                      <div className="flex items-center gap-2">
                        <p className="font-mono text-sm">{webhook.id}</p>
                        <Badge variant={webhook.active ? "default" : "secondary"}>
                          {webhook.active ? "Active" : "Inactive"}
                        </Badge>
                      </div>
                      <p className="text-sm">{webhook.url}</p>
                      <div className="flex flex-wrap gap-1">
                        {(webhook.events || []).map((event) => (
                          <Badge key={event} variant="outline" className="text-xs">
                            {event}
                          </Badge>
                        ))}
                      </div>
                      {webhook.last_triggered && (
                        <p className="text-xs text-muted-foreground">
                          Last triggered: {webhook.last_triggered}
                        </p>
                      )}
                    </div>
                  </div>
                  <div className="flex gap-2">
                    <Button 
                      variant="outline" 
                      size="sm"
                      onClick={() => handleTest(webhook.id)}
                      disabled={testingId === webhook.id}
                    >
                      <Play className="mr-1 h-3 w-3" />
                      {testingId === webhook.id ? "Testing..." : "Test"}
                    </Button>
                    <Button variant="ghost" size="sm" onClick={() => handleDelete(webhook.id)} disabled={deletingId === webhook.id}>
                      <Trash2 className="h-4 w-4 text-destructive" />
                    </Button>
                  </div>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}
    </div>
  );
}