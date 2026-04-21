"use client";

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { alertsApi, type AlertRule, type Alert } from "@/lib/api";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { Textarea } from "@/components/ui/textarea";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Switch } from "@/components/ui/switch";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  AlertTriangle,
  Bell,
  BellOff,
  CheckCircle,
  Eye,
  MoreHorizontal,
  Plus,
  RefreshCw,
  Trash2,
  XCircle,
} from "lucide-react";
import { toast } from "sonner";

const severityColors = {
  info: "bg-blue-100 text-blue-800",
  warning: "bg-yellow-100 text-yellow-800",
  critical: "bg-red-100 text-red-800",
};

const typeLabels = {
  retention: "Low Retention Rate",
  usage: "Declining Usage",
  negative_feedback: "Negative Feedback",
  storage: "Storage Warning",
  api_quota: "API Quota",
  agent_offline: "Agent Offline",
};

const operatorLabels = {
  lt: "Less than",
  gt: "Greater than",
  eq: "Equals",
};

export default function AlertsPage() {
  const [isCreateOpen, setIsCreateOpen] = useState(false);
  const [editingRule, setEditingRule] = useState<AlertRule | null>(null);
  const queryClient = useQueryClient();

  const { data: rulesData, isLoading, refetch } = useQuery({
    queryKey: ["alert-rules"],
    queryFn: () => alertsApi.listRules(),
  });

  const { data: alertsData, refetch: refetchAlerts } = useQuery({
    queryKey: ["active-alerts"],
    queryFn: () => alertsApi.listActive(),
  });

  const { data: statsData } = useQuery({
    queryKey: ["alert-stats"],
    queryFn: () => alertsApi.getStats(),
  });

  const [newRule, setNewRule] = useState({
    name: "",
    description: "",
    type: "retention" as AlertRule["type"],
    severity: "warning" as AlertRule["severity"],
    condition: "",
    threshold: 30,
    operator: "lt" as AlertRule["operator"],
    notify_email: false,
    notify_webhook: false,
    notify_in_app: true,
  });

  const createRuleMutation = useMutation({
    mutationFn: (data: Partial<AlertRule>) => alertsApi.createRule(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["alert-rules"] });
      setIsCreateOpen(false);
      setNewRule({
        name: "",
        description: "",
        type: "retention",
        severity: "warning",
        condition: "",
        threshold: 30,
        operator: "lt",
        notify_email: false,
        notify_webhook: false,
        notify_in_app: true,
      });
      toast.success("Alert rule created");
    },
    onError: () => {
      toast.error("Failed to create alert rule");
    },
  });

  const updateRuleMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<AlertRule> }) =>
      alertsApi.updateRule(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["alert-rules"] });
      setEditingRule(null);
      toast.success("Alert rule updated");
    },
    onError: () => {
      toast.error("Failed to update alert rule");
    },
  });

  const deleteRuleMutation = useMutation({
    mutationFn: (id: string) => alertsApi.deleteRule(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["alert-rules"] });
      toast.success("Alert rule deleted");
    },
    onError: () => {
      toast.error("Failed to delete alert rule");
    },
  });

  const enableRuleMutation = useMutation({
    mutationFn: ({ id, enabled }: { id: string; enabled: boolean }) =>
      alertsApi.enableRule(id, enabled),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["alert-rules"] });
      toast.success("Rule updated");
    },
    onError: () => {
      toast.error("Failed to update rule");
    },
  });

  const resolveAlertMutation = useMutation({
    mutationFn: (id: string) => alertsApi.resolveAlert(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["active-alerts"] });
      refetchAlerts();
      toast.success("Alert resolved");
    },
    onError: () => {
      toast.error("Failed to resolve alert");
    },
  });

  const dismissAlertMutation = useMutation({
    mutationFn: (id: string) => alertsApi.dismissAlert(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["active-alerts"] });
      refetchAlerts();
      toast.success("Alert dismissed");
    },
    onError: () => {
      toast.error("Failed to dismiss alert");
    },
  });

  const rules = rulesData?.rules || [];
  const activeAlerts = alertsData?.alerts || [];
  const stats = statsData || {};

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Alert Rules</h1>
          <p className="text-muted-foreground">Configure analytics alerts and notifications</p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" onClick={() => refetch()}>
            <RefreshCw className="mr-2 h-4 w-4" />
            Refresh
          </Button>
          <Dialog open={isCreateOpen} onOpenChange={setIsCreateOpen}>
            <DialogTrigger asChild>
              <Button>
                <Plus className="mr-2 h-4 w-4" />
                Create Rule
              </Button>
            </DialogTrigger>
            <DialogContent className="max-w-lg">
              <DialogHeader>
                <DialogTitle>Create Alert Rule</DialogTitle>
                <DialogDescription>
                  Set up automatic alerts based on analytics metrics
                </DialogDescription>
              </DialogHeader>
              <div className="space-y-4 py-4">
                <div className="space-y-2">
                  <Label htmlFor="name">Rule Name</Label>
                  <Input
                    id="name"
                    placeholder="Low Retention Alert"
                    value={newRule.name}
                    onChange={(e) => setNewRule({ ...newRule, name: e.target.value })}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="description">Description</Label>
                  <Input
                    id="description"
                    placeholder="Alert when user retention drops below threshold"
                    value={newRule.description}
                    onChange={(e) => setNewRule({ ...newRule, description: e.target.value })}
                  />
                </div>
                <div className="grid grid-cols-2 gap-4">
                  <div className="space-y-2">
                    <Label>Alert Type</Label>
                    <Select
                      value={newRule.type}
                      onValueChange={(v) => setNewRule({ ...newRule, type: v as AlertRule["type"] })}
                    >
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="retention">Retention Rate</SelectItem>
                        <SelectItem value="usage">Usage / DAU</SelectItem>
                        <SelectItem value="negative_feedback">Negative Feedback</SelectItem>
                        <SelectItem value="storage">Storage</SelectItem>
                        <SelectItem value="api_quota">API Quota</SelectItem>
                        <SelectItem value="agent_offline">Agent Offline</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                  <div className="space-y-2">
                    <Label>Severity</Label>
                    <Select
                      value={newRule.severity}
                      onValueChange={(v) => setNewRule({ ...newRule, severity: v as AlertRule["severity"] })}
                    >
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="info">Info</SelectItem>
                        <SelectItem value="warning">Warning</SelectItem>
                        <SelectItem value="critical">Critical</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                </div>
                <div className="grid grid-cols-2 gap-4">
                  <div className="space-y-2">
                    <Label>Condition</Label>
                    <Select
                      value={newRule.operator}
                      onValueChange={(v) => setNewRule({ ...newRule, operator: v as AlertRule["operator"] })}
                    >
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="lt">Less than</SelectItem>
                        <SelectItem value="gt">Greater than</SelectItem>
                        <SelectItem value="eq">Equals</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="threshold">Threshold</Label>
                    <Input
                      id="threshold"
                      type="number"
                      value={newRule.threshold}
                      onChange={(e) => setNewRule({ ...newRule, threshold: parseFloat(e.target.value) })}
                    />
                  </div>
                </div>
                <div className="space-y-3">
                  <Label>Notification Channels</Label>
                  <div className="flex items-center gap-2">
                    <Switch
                      checked={newRule.notify_in_app}
                      onCheckedChange={(v) => setNewRule({ ...newRule, notify_in_app: v })}
                    />
                    <Label>In-app notifications</Label>
                  </div>
                  <div className="flex items-center gap-2">
                    <Switch
                      checked={newRule.notify_email}
                      onCheckedChange={(v) => setNewRule({ ...newRule, notify_email: v })}
                    />
                    <Label>Email notifications</Label>
                  </div>
                  <div className="flex items-center gap-2">
                    <Switch
                      checked={newRule.notify_webhook}
                      onCheckedChange={(v) => setNewRule({ ...newRule, notify_webhook: v })}
                    />
                    <Label>Webhook notifications</Label>
                  </div>
                </div>
                <Button
                  className="w-full"
                  onClick={() => createRuleMutation.mutate(newRule)}
                  disabled={!newRule.name || createRuleMutation.isPending}
                >
                  Create Rule
                </Button>
              </div>
            </DialogContent>
          </Dialog>
        </div>
      </div>

      <div className="grid gap-4 md:grid-cols-4">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Total Rules</CardTitle>
            <Bell className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{stats.total_rules || 0}</div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Enabled</CardTitle>
            <CheckCircle className="h-4 w-4 text-green-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{stats.enabled_rules || 0}</div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Active Alerts</CardTitle>
            <AlertTriangle className="h-4 w-4 text-red-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{stats.active_alerts || 0}</div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Disabled</CardTitle>
            <BellOff className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{(stats.total_rules || 0) - (stats.enabled_rules || 0)}</div>
          </CardContent>
        </Card>
      </div>

      {activeAlerts.length > 0 && (
        <Card className="border-red-200 bg-red-50">
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-red-800">
              <AlertTriangle className="h-5 w-5" />
              Active Alerts ({activeAlerts.length})
            </CardTitle>
            <CardDescription className="text-red-600">
              These conditions have been triggered and require attention
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Alert</TableHead>
                  <TableHead>Severity</TableHead>
                  <TableHead>Value</TableHead>
                  <TableHead>Threshold</TableHead>
                  <TableHead>Triggered</TableHead>
                  <TableHead className="text-right">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {activeAlerts.map((alert) => (
                  <TableRow key={alert.id}>
                    <TableCell className="font-medium">
                      <div className="flex items-center gap-2">
                        <AlertTriangle className={`h-4 w-4 ${alert.severity === "critical" ? "text-red-500" : "text-yellow-500"}`} />
                        {alert.rule_name}
                      </div>
                    </TableCell>
                    <TableCell>
                      <Badge className={severityColors[alert.severity as keyof typeof severityColors]}>
                        {alert.severity}
                      </Badge>
                    </TableCell>
                    <TableCell className="font-mono">{alert.value.toFixed(1)}</TableCell>
                    <TableCell className="font-mono text-muted-foreground">{alert.threshold}</TableCell>
                    <TableCell className="text-muted-foreground">
                      {new Date(alert.triggered_at).toLocaleString()}
                    </TableCell>
                    <TableCell className="text-right">
                      <div className="flex justify-end gap-2">
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => resolveAlertMutation.mutate(alert.id)}
                          disabled={resolveAlertMutation.isPending}
                        >
                          <CheckCircle className="mr-1 h-4 w-4" />
                          Resolve
                        </Button>
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => dismissAlertMutation.mutate(alert.id)}
                          disabled={dismissAlertMutation.isPending}
                        >
                          <XCircle className="mr-1 h-4 w-4" />
                          Dismiss
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      )}

      <Card>
        <CardHeader>
          <CardTitle>Alert Rules</CardTitle>
          <CardDescription>Configure when and how you receive alerts</CardDescription>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="flex items-center justify-center py-8">
              <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
            </div>
          ) : rules.length === 0 ? (
            <div className="text-center py-8 text-muted-foreground">
              <Bell className="mx-auto h-12 w-12 mb-4 opacity-50" />
              <p>No alert rules configured</p>
              <p className="text-sm">Create your first rule to start monitoring</p>
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Name</TableHead>
                  <TableHead>Type</TableHead>
                  <TableHead>Condition</TableHead>
                  <TableHead>Severity</TableHead>
                  <TableHead>Notifications</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead className="text-right">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {rules.map((rule) => (
                  <TableRow key={rule.id}>
                    <TableCell className="font-medium">
                      <div>
                        {rule.name}
                        {rule.description && (
                          <p className="text-xs text-muted-foreground">{rule.description}</p>
                        )}
                      </div>
                    </TableCell>
                    <TableCell>
                      <Badge variant="outline">{typeLabels[rule.type]}</Badge>
                    </TableCell>
                    <TableCell className="font-mono text-sm">
                      {rule.condition} {operatorLabels[rule.operator]} {rule.threshold}
                    </TableCell>
                    <TableCell>
                      <Badge className={severityColors[rule.severity]}>{rule.severity}</Badge>
                    </TableCell>
                    <TableCell>
                      <div className="flex gap-1">
                        {rule.notify_in_app && <Badge variant="secondary">In-app</Badge>}
                        {rule.notify_email && <Badge variant="secondary">Email</Badge>}
                        {rule.notify_webhook && <Badge variant="secondary">Webhook</Badge>}
                      </div>
                    </TableCell>
                    <TableCell>
                      <Switch
                        checked={rule.enabled}
                        onCheckedChange={(enabled) =>
                          enableRuleMutation.mutate({ id: rule.id, enabled })
                        }
                      />
                    </TableCell>
                    <TableCell className="text-right">
                      <Dialog>
                        <DropdownMenu>
                          <DropdownMenuTrigger asChild>
                            <Button variant="ghost" size="sm">
                              <MoreHorizontal className="h-4 w-4" />
                            </Button>
                          </DropdownMenuTrigger>
                          <DropdownMenuContent align="end">
                            <DialogTrigger asChild>
                              <DropdownMenuItem onClick={() => setEditingRule(rule)}>
                                <Eye className="mr-2 h-4 w-4" />
                                View / Edit
                              </DropdownMenuItem>
                            </DialogTrigger>
                            <DropdownMenuItem
                              className="text-destructive"
                              onClick={() => {
                                if (confirm(`Delete rule "${rule.name}"?`)) {
                                  deleteRuleMutation.mutate(rule.id);
                                }
                              }}
                            >
                              <Trash2 className="mr-2 h-4 w-4" />
                              Delete
                            </DropdownMenuItem>
                          </DropdownMenuContent>
                        </DropdownMenu>
                        <DialogContent>
                          <DialogHeader>
                            <DialogTitle>Edit Alert Rule</DialogTitle>
                            <DialogDescription>{rule.description}</DialogDescription>
                          </DialogHeader>
                          {editingRule && (
                            <div className="space-y-4 py-4">
                              <div className="grid grid-cols-2 gap-4">
                                <div className="space-y-2">
                                  <Label>Type</Label>
                                  <Badge variant="outline">{typeLabels[editingRule.type]}</Badge>
                                </div>
                                <div className="space-y-2">
                                  <Label>Severity</Label>
                                  <Select
                                    value={editingRule.severity}
                                    onValueChange={(v) =>
                                      setEditingRule({ ...editingRule, severity: v as typeof editingRule.severity })
                                    }
                                  >
                                    <SelectTrigger>
                                      <SelectValue />
                                    </SelectTrigger>
                                    <SelectContent>
                                      <SelectItem value="info">Info</SelectItem>
                                      <SelectItem value="warning">Warning</SelectItem>
                                      <SelectItem value="critical">Critical</SelectItem>
                                    </SelectContent>
                                  </Select>
                                </div>
                              </div>
                              <div className="grid grid-cols-2 gap-4">
                                <div className="space-y-2">
                                  <Label>Operator</Label>
                                  <Select
                                    value={editingRule.operator}
                                    onValueChange={(v) =>
                                      setEditingRule({ ...editingRule, operator: v as typeof editingRule.operator })
                                    }
                                  >
                                    <SelectTrigger>
                                      <SelectValue />
                                    </SelectTrigger>
                                    <SelectContent>
                                      <SelectItem value="lt">Less than</SelectItem>
                                      <SelectItem value="gt">Greater than</SelectItem>
                                      <SelectItem value="eq">Equals</SelectItem>
                                    </SelectContent>
                                  </Select>
                                </div>
                                <div className="space-y-2">
                                  <Label>Threshold</Label>
                                  <Input
                                    type="number"
                                    value={editingRule.threshold}
                                    onChange={(e) =>
                                      setEditingRule({ ...editingRule, threshold: parseFloat(e.target.value) })
                                    }
                                  />
                                </div>
                              </div>
                              <div className="space-y-3">
                                <Label>Notifications</Label>
                                <div className="flex items-center gap-2">
                                  <Switch
                                    checked={editingRule.notify_in_app}
                                    onCheckedChange={(v) =>
                                      setEditingRule({ ...editingRule, notify_in_app: v })
                                    }
                                  />
                                  <Label>In-app</Label>
                                </div>
                                <div className="flex items-center gap-2">
                                  <Switch
                                    checked={editingRule.notify_email}
                                    onCheckedChange={(v) =>
                                      setEditingRule({ ...editingRule, notify_email: v })
                                    }
                                  />
                                  <Label>Email</Label>
                                </div>
                                <div className="flex items-center gap-2">
                                  <Switch
                                    checked={editingRule.notify_webhook}
                                    onCheckedChange={(v) =>
                                      setEditingRule({ ...editingRule, notify_webhook: v })
                                    }
                                  />
                                  <Label>Webhook</Label>
                                </div>
                              </div>
                              <Button
                                className="w-full"
                                onClick={() =>
                                  updateRuleMutation.mutate({
                                    id: editingRule.id,
                                    data: {
                                      severity: editingRule.severity,
                                      threshold: editingRule.threshold,
                                      operator: editingRule.operator,
                                      notify_email: editingRule.notify_email,
                                      notify_webhook: editingRule.notify_webhook,
                                      notify_in_app: editingRule.notify_in_app,
                                    },
                                  })
                                }
                                disabled={updateRuleMutation.isPending}
                              >
                                Save Changes
                              </Button>
                            </div>
                          )}
                        </DialogContent>
                      </Dialog>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>
    </div>
  );
}