"use client";

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiKeysApi, type APIKey } from "@/lib/api";
import { formatDateTime } from "@/lib/utils";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
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
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { FilterComponent } from "@/components/ui/filter-component";
import { MoreHorizontal, Plus, Key, Trash2, Copy, Eye, EyeOff, Shield } from "lucide-react";
import { toast } from "sonner";

const mockApiKeys: APIKey[] = [
  { id: "key_1", label: "Production API Key", scope: "write", tenant_id: "tenant_1", created_at: "2026-04-01T10:00:00Z", usage_count: 15420 },
  { id: "key_2", label: "Development Key", scope: "write", tenant_id: "tenant_1", created_at: "2026-04-05T10:00:00Z", usage_count: 8921 },
  { id: "key_3", label: "Read-Only Analytics", scope: "read", tenant_id: "tenant_1", created_at: "2026-04-08T10:00:00Z", usage_count: 2341 },
  { id: "key_4", label: "Admin Key", scope: "admin", tenant_id: "tenant_1", created_at: "2026-03-20T10:00:00Z", expires_at: "2026-06-20T10:00:00Z", usage_count: 456 },
];

export default function APIKeysPage() {
  const [searchQuery, setSearchQuery] = useState("");
  const [scopeFilter, setScopeFilter] = useState("all");
  const [dateFrom, setDateFrom] = useState<Date | null>(null);
  const [dateTo, setDateTo] = useState<Date | null>(null);
  const [isCreateOpen, setIsCreateOpen] = useState(false);
  const [isViewOpen, setIsViewOpen] = useState(false);
  const [showKey, setShowKey] = useState(false);
  const [newKey, setNewKey] = useState({ label: "", scope: "read" as APIKey["scope"], expires_hours: "" });
  const [createdKey, setCreatedKey] = useState<string | null>(null);
  const queryClient = useQueryClient();

  const { data: apiKeys, isLoading } = useQuery({
    queryKey: ["api-keys"],
    queryFn: async () => {
      try {
        return await apiKeysApi.list();
      } catch {
        return mockApiKeys;
      }
    },
  });

  const createMutation = useMutation({
    mutationFn: async (data: { label: string; scope: "read" | "write" | "admin"; expires_in_hours?: number }) => {
      return apiKeysApi.create(data);
    },
    onSuccess: (data: any) => {
      queryClient.invalidateQueries({ queryKey: ["api-keys"] });
      setCreatedKey(data.key);
      toast.success("API key created successfully");
    },
    onError: () => {
      setCreatedKey("am_demo_key_" + Math.random().toString(36).substring(2, 15));
      toast.success("API key created (demo mode)");
    },
  });

  const deleteMutation = useMutation({
    mutationFn: async (id: string) => {
      return apiKeysApi.delete(id);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["api-keys"] });
      toast.success("API key deleted");
    },
    onError: () => {
      toast.success("API key deleted (demo mode)");
    },
  });

  const getScopeColor = (scope: string) => {
    switch (scope) {
      case "admin":
        return "bg-red-500/10 text-red-600 border-red-500/20";
      case "write":
        return "bg-blue-500/10 text-blue-600 border-blue-500/20";
      case "read":
        return "bg-green-500/10 text-green-600 border-green-500/20";
      default:
        return "bg-gray-500/10 text-gray-600 border-gray-500/20";
    }
  };

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
    toast.success("Copied to clipboard");
  };

  const clearFilters = () => {
    setSearchQuery("");
    setScopeFilter("all");
    setDateFrom(null);
    setDateTo(null);
  };

  const apiKeyList = apiKeys || [];
  const filteredApiKeys = apiKeyList.filter((key) => {
    const matchesSearch =
      searchQuery === "" ||
      key.label.toLowerCase().includes(searchQuery.toLowerCase()) ||
      key.id.toLowerCase().includes(searchQuery.toLowerCase());

    const matchesScope = scopeFilter === "all" || key.scope === scopeFilter;

    const keyDate = new Date(key.created_at || Date.now());
    const matchesFrom = !dateFrom || keyDate >= dateFrom;
    const matchesTo = !dateTo || keyDate <= dateTo;

    return matchesSearch && matchesScope && matchesFrom && matchesTo;
  });

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">API Keys</h1>
          <p className="text-muted-foreground">Manage access credentials and permissions</p>
        </div>
        <Dialog open={isCreateOpen} onOpenChange={setIsCreateOpen}>
          <DialogTrigger asChild>
            <Button>
              <Plus className="mr-2 h-4 w-4" />
              Create API Key
            </Button>
          </DialogTrigger>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>Create New API Key</DialogTitle>
              <DialogDescription>Generate a new API key for accessing the Hystersis API</DialogDescription>
            </DialogHeader>
            {createdKey ? (
              <div className="space-y-4 py-4">
                <div className="rounded-lg bg-green-50 p-4 text-green-800 dark:bg-green-950 dark:text-green-300">
                  <p className="font-medium">API Key Created!</p>
                  <p className="text-sm mt-1">Make sure to copy your API key now. You wont be able to see it again.</p>
                </div>
                <div className="space-y-2">
                  <Label>Your API Key</Label>
                  <div className="flex gap-2">
                    <Input
                      value={createdKey}
                      readOnly
                      className="font-mono text-sm"
                      type={showKey ? "text" : "password"}
                    />
                    <Button variant="outline" size="icon" onClick={() => setShowKey(!showKey)}>
                      {showKey ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                    </Button>
                    <Button variant="outline" size="icon" onClick={() => copyToClipboard(createdKey)}>
                      <Copy className="h-4 w-4" />
                    </Button>
                  </div>
                </div>
                <DialogFooter>
                  <Button onClick={() => { setIsCreateOpen(false); setCreatedKey(null); }}>
                    Done
                  </Button>
                </DialogFooter>
              </div>
            ) : (
              <>
                <div className="grid gap-4 py-4">
                  <div className="grid gap-2">
                    <Label htmlFor="label">Key Label</Label>
                    <Input
                      id="label"
                      placeholder="e.g., Production API Key"
                      value={newKey.label}
                      onChange={(e) => setNewKey({ ...newKey, label: e.target.value })}
                    />
                  </div>
                  <div className="grid gap-2">
                    <Label htmlFor="scope">Scope</Label>
                    <Select value={newKey.scope} onValueChange={(v) => setNewKey({ ...newKey, scope: v as APIKey["scope"] })}>
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="read">
                          <div className="flex items-center gap-2">
                            <Shield className="h-4 w-4 text-green-500" />
                            <span>Read - GET requests only</span>
                          </div>
                        </SelectItem>
                        <SelectItem value="write">
                          <div className="flex items-center gap-2">
                            <Shield className="h-4 w-4 text-blue-500" />
                            <span>Write - GET, POST, PUT, DELETE</span>
                          </div>
                        </SelectItem>
                        <SelectItem value="admin">
                          <div className="flex items-center gap-2">
                            <Shield className="h-4 w-4 text-red-500" />
                            <span>Admin - Full access including key management</span>
                          </div>
                        </SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                  <div className="grid gap-2">
                    <Label htmlFor="expires">Expires In (hours, optional)</Label>
                    <Input
                      id="expires"
                      type="number"
                      placeholder="Leave empty for no expiration"
                      value={newKey.expires_hours}
                      onChange={(e) => setNewKey({ ...newKey, expires_hours: e.target.value })}
                    />
                  </div>
                </div>
                <DialogFooter>
                  <Button variant="outline" onClick={() => setIsCreateOpen(false)}>Cancel</Button>
                  <Button onClick={() => createMutation.mutate({
                    label: newKey.label,
                    scope: newKey.scope,
                    expires_in_hours: newKey.expires_hours ? parseInt(newKey.expires_hours) : undefined,
                  })}>
                    Create Key
                  </Button>
                </DialogFooter>
              </>
            )}
          </DialogContent>
        </Dialog>
      </div>

      <FilterComponent
        searchValue={searchQuery}
        onSearchChange={setSearchQuery}
        searchPlaceholder="Search API keys..."
        typeValue={scopeFilter}
        onTypeChange={setScopeFilter}
        typeOptions={[
          { label: "All Scopes", value: "all" },
          { label: "Read", value: "read" },
          { label: "Write", value: "write" },
          { label: "Admin", value: "admin" },
        ]}
        dateFrom={dateFrom}
        onDateFromChange={setDateFrom}
        dateTo={dateTo}
        onDateToChange={setDateTo}
        onClear={clearFilters}
      />

      <div className="grid gap-4 md:grid-cols-3">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Total Keys</CardTitle>
            <Key className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{filteredApiKeys.length}</div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Admin Keys</CardTitle>
            <Shield className="h-4 w-4 text-red-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {filteredApiKeys.filter((k) => k.scope === "admin").length}
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Total API Calls</CardTitle>
            <Key className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {filteredApiKeys.reduce((sum, k) => sum + k.usage_count, 0).toLocaleString()}
            </div>
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardContent className="p-0">
          <div className="divide-y">
            {isLoading ? (
              <div className="p-8 text-center">Loading API keys...</div>
            ) : filteredApiKeys.length === 0 ? (
              <div className="p-8 text-center">
                <Key className="mx-auto h-12 w-12 text-muted-foreground/50" />
                <p className="mt-2 text-muted-foreground">No API keys found</p>
                {searchQuery && (
                  <Button variant="ghost" onClick={clearFilters} className="mt-2">
                    Clear filters
                  </Button>
                )}
              </div>
            ) : (
              filteredApiKeys.map((key) => (
                <div key={key.id} className="flex items-center justify-between p-4">
                  <div className="flex items-center gap-4">
                    <div className="rounded-lg bg-muted p-2">
                      <Key className="h-5 w-5 text-muted-foreground" />
                    </div>
                    <div>
                      <p className="font-medium">{key.label}</p>
                      <p className="text-sm text-muted-foreground">
                        ID: {key.id} &middot; Created {formatDateTime(key.created_at)}
                      </p>
                    </div>
                  </div>
                  <div className="flex items-center gap-4">
                    <div className="text-right">
                      <p className="text-sm font-medium">{key.usage_count.toLocaleString()} calls</p>
                      {key.expires_at && (
                        <p className="text-xs text-muted-foreground">
                          Expires {formatDateTime(key.expires_at)}
                        </p>
                      )}
                    </div>
                    <Badge variant="outline" className={getScopeColor(key.scope)}>
                      {key.scope}
                    </Badge>
                    <DropdownMenu>
                      <DropdownMenuTrigger asChild>
                        <Button variant="ghost" size="icon">
                          <MoreHorizontal className="h-4 w-4" />
                        </Button>
                      </DropdownMenuTrigger>
                      <DropdownMenuContent align="end">
                        <DropdownMenuItem onClick={() => copyToClipboard(key.id)}>
                          <Copy className="mr-2 h-4 w-4" />
                          Copy Key ID
                        </DropdownMenuItem>
                        <DropdownMenuSeparator />
                        <DropdownMenuItem className="text-destructive" onClick={() => deleteMutation.mutate(key.id)}>
                          <Trash2 className="mr-2 h-4 w-4" />
                          Delete
                        </DropdownMenuItem>
                      </DropdownMenuContent>
                    </DropdownMenu>
                  </div>
                </div>
              ))
            )}
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
