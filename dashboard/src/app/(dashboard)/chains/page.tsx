"use client";

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { chainsApi, type Chain } from "@/lib/api";
import { formatDateTime } from "@/lib/utils";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
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
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { FilterComponent } from "@/components/ui/filter-component";
import { MoreHorizontal, Plus, Zap, Play, Trash2, Edit, RefreshCw, Workflow, Clock } from "lucide-react";
import { toast } from "sonner";

export default function ChainsPage() {
  const [searchQuery, setSearchQuery] = useState("");
  const [dateFrom, setDateFrom] = useState<Date | null>(null);
  const [dateTo, setDateTo] = useState<Date | null>(null);
  const [isCreateOpen, setIsCreateOpen] = useState(false);
  const [isExecuteOpen, setIsExecuteOpen] = useState(false);
  const [isEditOpen, setIsEditOpen] = useState(false);
  const [selectedChain, setSelectedChain] = useState<Chain | null>(null);
  const [newChain, setNewChain] = useState({
    name: "",
    trigger: "",
  });
  const [editChain, setEditChain] = useState({
    name: "",
    trigger: "",
  });
  const [executeContext, setExecuteContext] = useState("");
  const [executeResult, setExecuteResult] = useState<string | null>(null);

  const queryClient = useQueryClient();

  const { data: chainsData, isLoading, refetch } = useQuery({
    queryKey: ["chains"],
    queryFn: () => chainsApi.list(),
  });

  const createMutation = useMutation({
    mutationFn: (data: { name: string; trigger: string }) => chainsApi.create(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["chains"] });
      setIsCreateOpen(false);
      setNewChain({ name: "", trigger: "" });
      toast.success("Chain created successfully");
    },
    onError: () => {
      toast.error("Failed to create chain");
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => chainsApi.delete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["chains"] });
      toast.success("Chain deleted");
    },
    onError: () => {
      toast.error("Failed to delete chain");
    },
  });

  const executeMutation = useMutation({
    mutationFn: ({ id, context }: { id: string; context?: Record<string, unknown> }) => 
      chainsApi.execute(id, context),
    onSuccess: (result) => {
      setExecuteResult(JSON.stringify(result.result, null, 2));
      toast.success("Chain executed successfully");
    },
    onError: () => {
      toast.error("Failed to execute chain");
    },
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: { name?: string; trigger?: string } }) =>
      chainsApi.update(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["chains"] });
      setIsEditOpen(false);
      setSelectedChain(null);
      toast.success("Chain updated successfully");
    },
    onError: () => {
      toast.error("Failed to update chain");
    },
  });

  const handleCreate = () => {
    if (!newChain.name.trim()) {
      toast.error("Chain name is required");
      return;
    }
    if (!newChain.trigger.trim()) {
      toast.error("Trigger is required");
      return;
    }
    createMutation.mutate({ name: newChain.name, trigger: newChain.trigger });
  };

  const handleExecute = () => {
    if (!selectedChain) return;
    let context: Record<string, unknown> | undefined;
    if (executeContext.trim()) {
      try {
        context = JSON.parse(executeContext);
      } catch {
        toast.error("Invalid JSON context");
        return;
      }
    }
    executeMutation.mutate({ id: selectedChain.id, context });
  };

  const handleDelete = (id: string) => {
    if (!confirm("Are you sure you want to delete this chain?")) return;
    deleteMutation.mutate(id);
  };

  const openExecuteDialog = (chain: Chain) => {
    setSelectedChain(chain);
    setExecuteContext("");
    setExecuteResult(null);
    setIsExecuteOpen(true);
  };

  const openEditDialog = (chain: Chain) => {
    setSelectedChain(chain);
    setEditChain({ name: chain.name, trigger: chain.trigger || "" });
    setIsEditOpen(true);
  };

  const handleEdit = () => {
    if (!selectedChain || !editChain.name.trim()) {
      toast.error("Chain name is required");
      return;
    }
    updateMutation.mutate({
      id: selectedChain.id,
      data: { name: editChain.name, trigger: editChain.trigger },
    });
  };

  const chains = chainsData?.chains || [];

  const clearFilters = () => {
    setSearchQuery("");
    setDateFrom(null);
    setDateTo(null);
  };

  const filteredChains = chains.filter((chain) => {
    const matchesSearch =
      searchQuery === "" ||
      chain.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      (chain.trigger?.toLowerCase().includes(searchQuery.toLowerCase()) ?? false);

    const chainDate = new Date(chain.created_at || Date.now());
    const matchesFrom = !dateFrom || chainDate >= dateFrom;
    const matchesTo = !dateTo || chainDate <= dateTo;

    return matchesSearch && matchesFrom && matchesTo;
  });

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">Chains</h1>
          <p className="text-muted-foreground">Create and manage skill chains for complex workflows</p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" size="icon" onClick={() => refetch()}>
            <RefreshCw className="h-4 w-4" />
          </Button>
          <Dialog open={isCreateOpen} onOpenChange={setIsCreateOpen}>
            <DialogTrigger asChild>
              <Button>
                <Plus className="mr-2 h-4 w-4" />
                Create Chain
              </Button>
            </DialogTrigger>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>Create New Chain</DialogTitle>
                <DialogDescription>Create a workflow chain of skills</DialogDescription>
              </DialogHeader>
              <div className="grid gap-4 py-4">
                <div className="grid gap-2">
                  <Label htmlFor="name">Chain Name</Label>
                  <Input
                    id="name"
                    placeholder="e.g., Code Review Pipeline"
                    value={newChain.name}
                    onChange={(e) => setNewChain({ ...newChain, name: e.target.value })}
                  />
                </div>
                <div className="grid gap-2">
                  <Label htmlFor="trigger">Trigger</Label>
                  <Input
                    id="trigger"
                    placeholder="e.g., code_review_requested"
                    value={newChain.trigger}
                    onChange={(e) => setNewChain({ ...newChain, trigger: e.target.value })}
                  />
                </div>
              </div>
              <DialogFooter>
                <Button variant="outline" onClick={() => setIsCreateOpen(false)}>Cancel</Button>
                <Button onClick={handleCreate} disabled={createMutation.isPending}>
                  {createMutation.isPending ? "Creating..." : "Create Chain"}
                </Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>
        </div>
      </div>

      <FilterComponent
        searchValue={searchQuery}
        onSearchChange={setSearchQuery}
        searchPlaceholder="Search chains..."
        typeValue="all"
        onTypeChange={() => {}}
        typeOptions={[]}
        dateFrom={dateFrom}
        onDateFromChange={setDateFrom}
        dateTo={dateTo}
        onDateToChange={setDateTo}
        onClear={clearFilters}
      />

      {isLoading ? (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {[1, 2, 3].map(i => (
            <Card key={i}>
              <CardHeader>
                <Skeleton className="h-5 w-32" />
              </CardHeader>
              <CardContent>
                <Skeleton className="h-4 w-full mb-2" />
                <Skeleton className="h-8 w-full" />
              </CardContent>
            </Card>
          ))}
        </div>
      ) : filteredChains.length === 0 ? (
        <Card>
          <CardContent className="flex flex-col items-center justify-center py-12">
            <Zap className="h-12 w-12 text-muted-foreground mb-4" />
            <p className="text-muted-foreground">No chains found</p>
            {searchQuery && (
              <Button variant="ghost" onClick={clearFilters} className="mt-2">
                Clear filters
              </Button>
            )}
          </CardContent>
        </Card>
      ) : (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {filteredChains.map((chain) => (
            <Card key={chain.id} className="card-hover">
              <CardHeader className="space-y-0 pb-2">
                <div className="flex items-start justify-between">
                  <div className="flex items-center gap-3">
                    <div className="rounded-lg bg-primary/10 p-2">
                      <Workflow className="h-5 w-5 text-primary" />
                    </div>
                    <div>
                      <CardTitle className="text-lg">{chain.name}</CardTitle>
                      <p className="text-sm text-muted-foreground">Trigger: {chain.trigger}</p>
                    </div>
                  </div>
                  <DropdownMenu>
                    <DropdownMenuTrigger asChild>
                      <Button variant="ghost" size="icon">
                        <MoreHorizontal className="h-4 w-4" />
                      </Button>
                    </DropdownMenuTrigger>
                    <DropdownMenuContent align="end">
                      <DropdownMenuItem onClick={() => openExecuteDialog(chain)}>
                        <Play className="mr-2 h-4 w-4" />
                        Execute
                      </DropdownMenuItem>
                      <DropdownMenuSeparator />
                      <DropdownMenuItem 
                        className="text-destructive"
                        onClick={() => handleDelete(chain.id)}
                      >
                        <Trash2 className="mr-2 h-4 w-4" />
                        Delete
                      </DropdownMenuItem>
                    </DropdownMenuContent>
                  </DropdownMenu>
                </div>
              </CardHeader>
              <CardContent>
                <div className="space-y-3">
                  <div className="flex items-center gap-2 text-sm text-muted-foreground">
                    <Clock className="h-4 w-4" />
                    <span>{chain.steps?.length || 0} steps</span>
                  </div>
                  <div className="flex items-center gap-2">
                    <Badge variant={chain.confidence >= 0.8 ? "default" : "secondary"}>
                      {Math.round(chain.confidence * 100)}% confidence
                    </Badge>
                  </div>
                  <div className="flex gap-2">
                    <Button 
                      variant="outline" 
                      size="sm" 
                      className="flex-1"
                      onClick={() => openExecuteDialog(chain)}
                    >
                      <Play className="mr-1 h-3 w-3" />
                      Execute
                    </Button>
                    <Button variant="outline" size="sm" onClick={() => openEditDialog(chain)}>
                      <Edit className="h-4 w-4" />
                    </Button>
                  </div>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      {/* Execute Chain Dialog */}
      <Dialog open={isExecuteOpen} onOpenChange={(open) => {
        setIsExecuteOpen(open);
        if (!open) {
          setSelectedChain(null);
          setExecuteResult(null);
        }
      }}>
        <DialogContent className="sm:max-w-[600px]">
          <DialogHeader>
            <DialogTitle>Execute Chain</DialogTitle>
            <DialogDescription>
              Run "{selectedChain?.name}" with optional context
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-4">
            <div className="grid gap-2">
              <Label htmlFor="context">Context (JSON)</Label>
              <textarea
                id="context"
                className="flex min-h-[100px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2"
                placeholder='{"key": "value"}'
                value={executeContext}
                onChange={(e) => setExecuteContext(e.target.value)}
              />
              <p className="text-xs text-muted-foreground">Optional context data to pass to the chain</p>
            </div>
            {executeResult && (
              <div className="grid gap-2">
                <Label>Result</Label>
                <pre className="rounded-md bg-muted p-4 text-sm overflow-auto max-h-48">
                  {executeResult}
                </pre>
              </div>
            )}
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setIsExecuteOpen(false)}>Close</Button>
            <Button onClick={handleExecute} disabled={executeMutation.isPending}>
              {executeMutation.isPending ? "Executing..." : "Execute Chain"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={isEditOpen} onOpenChange={(open) => {
        setIsEditOpen(open);
        if (!open) {
          setSelectedChain(null);
        }
      }}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Edit Chain</DialogTitle>
            <DialogDescription>
              Update chain configuration
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-4">
            <div className="grid gap-2">
              <Label htmlFor="edit-name">Chain Name</Label>
              <Input
                id="edit-name"
                value={editChain.name}
                onChange={(e) => setEditChain({ ...editChain, name: e.target.value })}
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="edit-trigger">Trigger</Label>
              <Input
                id="edit-trigger"
                value={editChain.trigger}
                onChange={(e) => setEditChain({ ...editChain, trigger: e.target.value })}
              />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setIsEditOpen(false)}>Cancel</Button>
            <Button onClick={handleEdit} disabled={updateMutation.isPending}>
              {updateMutation.isPending ? "Saving..." : "Save Changes"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}