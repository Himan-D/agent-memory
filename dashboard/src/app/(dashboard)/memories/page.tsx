"use client";

import { useState, useEffect } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { memoriesApi, type Memory } from "@/lib/api";
import { formatDateTime, truncate } from "@/lib/utils";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Textarea } from "@/components/ui/textarea";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Calendar } from "@/components/ui/calendar";
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
import { cn } from "@/lib/utils";
import { FilterComponent } from "@/components/ui/filter-component";
import { Database, Trash2, Edit, Eye, RefreshCw, MoreHorizontal, Plus, Download } from "lucide-react";
import { toast } from "sonner";
import { MemoryTable } from "@/components/memories/memory-table";
import { BulkOperations } from "@/components/bulk-operations";
import { useAuditLogger } from "@/hooks/use-audit-logger";
import { useSelection } from "@/hooks/use-selection";
import { useConfirmation } from "@/hooks/use-confirmation";

export default function MemoriesPage() {
  const [searchQuery, setSearchQuery] = useState("");
  const [typeFilter, setTypeFilter] = useState<string>("all");
  const [dateFrom, setDateFrom] = useState<Date | null>(null);
  const [dateTo, setDateTo] = useState<Date | null>(null);
  const [isCreateOpen, setIsCreateOpen] = useState(false);
  const [isViewOpen, setIsViewOpen] = useState(false);
  const [isEditOpen, setIsEditOpen] = useState(false);
  const [selectedMemory, setSelectedMemory] = useState<Memory | null>(null);
  const [newMemory, setNewMemory] = useState({
    content: "",
    type: "user" as Memory["type"],
    category: "",
    tags: [] as string[],
  });
  const [editMemory, setEditMemory] = useState({
    content: "",
    type: "user" as Memory["type"],
    category: "",
    tags: [] as string[],
  });
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);

  const queryClient = useQueryClient();
  const { logCreate, logUpdate, logDelete } = useAuditLogger();
  const { selectedIds, toggle, selectAll, isSelected, isAllSelected } = useSelection<string>({ multiple: true, maxSelections: 100 });
  const { confirm } = useConfirmation();

  const { data: memoriesData, isLoading, refetch } = useQuery({
    queryKey: ["memories", page, pageSize],
    queryFn: () => memoriesApi.list({ limit: pageSize, offset: (page - 1) * pageSize }),
  });

  const createMutation = useMutation({
    mutationFn: async (data: Partial<Memory>) => {
      return memoriesApi.create(data);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["memories"] });
      setIsCreateOpen(false);
      setNewMemory({ content: "", type: "user", category: "", tags: [] });
      logCreate("memory", "new", { type: newMemory.type });
      toast.success("Memory created successfully");
    },
    onError: (err) => {
      toast.error(`Failed to create memory: ${err}`);
    },
  });

  const updateMutation = useMutation({
    mutationFn: async ({ id, data }: { id: string; data: Partial<Memory> }) => {
      return memoriesApi.update(id, data);
    },
    onSuccess: (_: unknown, variables: { id: string; data: Partial<Memory> }) => {
      queryClient.invalidateQueries({ queryKey: ["memories"] });
      setIsEditOpen(false);
      setSelectedMemory(null);
      logUpdate("memory", variables.id, { updatedFields: Object.keys(variables.data) });
      toast.success("Memory updated successfully");
    },
    onError: (err) => {
      toast.error(`Failed to update memory: ${err}`);
    },
  });

  const deleteMutation = useMutation({
    mutationFn: async (id: string) => {
      return memoriesApi.delete(id);
    },
    onSuccess: (_: unknown, variables: string) => {
      queryClient.invalidateQueries({ queryKey: ["memories"] });
      logDelete("memory", variables, { reason: "manual_delete" });
      toast.success("Memory deleted");
    },
    onError: (err) => {
      toast.error(`Failed to delete memory: ${err}`);
    },
  });

  const bulkDeleteMutation = useMutation({
    mutationFn: async (ids: string[]) => {
      return Promise.all(ids.map(id => memoriesApi.delete(id)));
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["memories"] });
      logDelete("memory", "bulk", { count: selectedIds.size, ids: Array.from(selectedIds) });
      toast.success(`Deleted ${selectedIds.size} memories`);
    },
    onError: (err) => {
      toast.error(`Failed to delete memories: ${err}`);
    },
  });

  const filteredMemories = memoriesData?.memories?.filter((memory) => {
    const matchesSearch = searchQuery === "" ||
      memory.content.toLowerCase().includes(searchQuery.toLowerCase()) ||
      memory.type.toLowerCase().includes(searchQuery.toLowerCase()) ||
      memory.tags?.some((tag) => tag.toLowerCase().includes(searchQuery.toLowerCase()));

    const matchesType = typeFilter === "all" || memory.type === typeFilter;

    const memoryDate = new Date(memory.created_at);
    const matchesFrom = !dateFrom || memoryDate >= dateFrom;
    const matchesTo = !dateTo || memoryDate <= dateTo;

    return matchesSearch && matchesType && matchesFrom && matchesTo;
  }) || [];

  // Apply pagination
  const paginatedMemories = filteredMemories.slice((page - 1) * pageSize, page * pageSize);
  const totalPages = Math.ceil(filteredMemories.length / pageSize);

  const handleBulkDelete = (ids: string[]) => {
    confirm({
      title: "Bulk Delete",
      description: `Are you sure you want to delete ${ids.length} memories? This action cannot be undone.`,
      confirmText: "Delete",
      confirmVariant: "destructive",
    }, () => bulkDeleteMutation.mutate(ids));
  };

  const handleExport = () => {
    const csvContent = [
      ["ID", "Content", "Type", "Category", "Tags", "Created At"].join(","),
      ...filteredMemories.map(memory => [
        memory.id,
        `"${memory.content.replace(/"/g, '""')}"`,
        memory.type,
        memory.category || "",
        memory.tags?.join(" ") || "",
        memory.created_at
      ].join(","))
    ].join("\n");

    const blob = new Blob([csvContent], { type: "text/csv;charset=utf-8;" });
    const link = document.createElement("a");
    const url = URL.createObjectURL(blob);
    link.setAttribute("href", url);
    link.setAttribute("download", `memories_${new Date().toISOString().split('T')[0]}.csv`);
    link.style.visibility = "hidden";
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
  };

  const handleDelete = (id: string) => {
    confirm({
      title: "Delete Memory",
      description: "Are you sure you want to delete this memory? This action cannot be undone.",
      confirmText: "Delete",
      confirmVariant: "destructive",
    }, () => deleteMutation.mutate(id));
  };

  const clearFilters = () => {
    setSearchQuery("");
    setTypeFilter("all");
    setDateFrom(null);
    setDateTo(null);
  };
 
  const prevPage = () => {
    if (page > 1) setPage(page - 1);
  };

  const nextPage = () => {
    if (page < totalPages) setPage(page + 1);
  };

  const hasActiveFilters = searchQuery !== "" || typeFilter !== "all" || dateFrom !== null || dateTo !== null;

  const getTypeColors = (type: string) => {
    switch (type) {
      case "conversation":
        return "bg-blue-500/10 text-blue-600 border-blue-500/20";
      case "session":
        return "bg-purple-500/10 text-purple-600 border-purple-500/20";
      case "user":
        return "bg-green-500/10 text-green-600 border-green-500/20";
      case "org":
        return "bg-orange-500/10 text-orange-600 border-orange-500/20";
      default:
        return "bg-gray-500/10 text-gray-600 border-gray-500/20";
    }
  };

  const handleView = (memory: Memory) => {
    setSelectedMemory(memory);
    setIsViewOpen(true);
  };

  const handleEdit = (memory: Memory) => {
    setSelectedMemory(memory);
    setEditMemory({
      content: memory.content,
      type: memory.type,
      category: memory.category || "",
      tags: memory.tags || [],
    });
    setIsEditOpen(true);
  };

  const handleUpdate = () => {
    if (!selectedMemory) return;
    updateMutation.mutate({
      id: selectedMemory.id,
      data: {
        content: editMemory.content,
        type: editMemory.type,
        category: editMemory.category,
        tags: editMemory.tags,
      },
    });
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">Memories</h1>
          <p className="text-muted-foreground">
            {memoriesData?.count ? `${memoriesData.count} memories stored` : "Manage your AI agent memories"}
          </p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" size="icon" onClick={() => refetch()}>
            <RefreshCw className="h-4 w-4" />
          </Button>
          <Dialog open={isCreateOpen} onOpenChange={setIsCreateOpen}>
            <DialogTrigger asChild>
              <Button>
                <Plus className="mr-2 h-4 w-4" />
                Create Memory
              </Button>
            </DialogTrigger>
            <DialogContent className="sm:max-w-[600px]">
              <DialogHeader>
                <DialogTitle>Create New Memory</DialogTitle>
                <DialogDescription>
                  Add a new memory to your agent&apos;s knowledge base
                </DialogDescription>
              </DialogHeader>
              <div className="grid gap-4 py-4">
                <div className="grid gap-2">
                  <Label htmlFor="content">Content</Label>
                  <Textarea
                    id="content"
                    placeholder="Enter memory content..."
                    value={newMemory.content}
                    onChange={(e) =>
                      setNewMemory({ ...newMemory, content: e.target.value })
                    }
                    className="min-h-[100px]"
                  />
                </div>
                <div className="grid grid-cols-2 gap-4">
                  <div className="grid gap-2">
                    <Label htmlFor="type">Type</Label>
                    <Select
                      value={newMemory.type}
                      onValueChange={(value) =>
                        setNewMemory({ ...newMemory, type: value as Memory["type"] })
                      }
                    >
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="conversation">Conversation</SelectItem>
                        <SelectItem value="session">Session</SelectItem>
                        <SelectItem value="user">User</SelectItem>
                        <SelectItem value="org">Organization</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                  <div className="grid gap-2">
                    <Label htmlFor="category">Category</Label>
                    <Input
                      id="category"
                      placeholder="e.g., preferences"
                      value={newMemory.category}
                      onChange={(e) =>
                        setNewMemory({ ...newMemory, category: e.target.value })
                      }
                    />
                  </div>
                </div>
                <div className="grid gap-2">
                  <Label htmlFor="tags">Tags (comma-separated)</Label>
                  <Input
                    id="tags"
                    placeholder="e.g., important, follow-up"
                    value={newMemory.tags.join(", ")}
                    onChange={(e) =>
                      setNewMemory({ ...newMemory, tags: e.target.value.split(",").map(t => t.trim()).filter(Boolean) })
                    }
                  />
                </div>
              </div>
              <DialogFooter>
                <Button
                  variant="outline"
                  onClick={() => setIsCreateOpen(false)}
                >
                  Cancel
                </Button>
                <Button
                  onClick={() => createMutation.mutate(newMemory)}
                  disabled={!newMemory.content.trim() || createMutation.isPending}
                >
                  {createMutation.isPending ? "Creating..." : "Create Memory"}
                </Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>
        </div>
      </div>

      {/* Bulk Operations */}
      {selectedIds.size > 0 && (
        <div className="flex gap-2">
          <BulkOperations
            resource="memory"
            selectedIds={Array.from(selectedIds)}
            apiDelete={memoriesApi.delete}
            apiBatchDelete={async (ids) => { await bulkDeleteMutation.mutateAsync(ids); }}
            onBulkExport={handleExport}
            onBulkDelete={handleBulkDelete}
          />
        </div>
      )}

      <FilterComponent
        searchValue={searchQuery}
        onSearchChange={setSearchQuery}
        searchPlaceholder="Search memories..."
        typeValue={typeFilter}
        onTypeChange={setTypeFilter}
        typeOptions={[
          { label: "All Types", value: "all" },
          { label: "Conversation", value: "conversation" },
          { label: "Session", value: "session" },
          { label: "User", value: "user" },
          { label: "Organization", value: "org" },
        ]}
        dateFrom={dateFrom}
        onDateFromChange={setDateFrom}
        dateTo={dateTo}
        onDateToChange={setDateTo}
        onClear={clearFilters}
      />

      <Card>
        <CardContent className="p-0">
          <MemoryTable
            memories={paginatedMemories}
            onDelete={handleDelete}
            onView={setIsViewOpen}
            onEdit={setIsEditOpen}
            selectedIds={selectedIds}
            onSelect={(id, checked) => toggle(id)}
            loading={isLoading}
          />
        </CardContent>
      </Card>

      {/* Pagination */}
      {totalPages > 1 && (
        <div className="flex items-center justify-between">
          <div className="text-sm text-muted-foreground">
            Showing {paginatedMemories.length} of {filteredMemories.length} memories
          </div>
          <div className="flex gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={() => prevPage()}
              disabled={page === 1}
            >
              Previous
            </Button>
            <Button
              variant="outline"
              size="sm"
              onClick={() => nextPage()}
              disabled={page === totalPages}
            >
              Next
            </Button>
          </div>
        </div>
      )}

      {/* View Memory Dialog */}
      <Dialog open={isViewOpen} onOpenChange={setIsViewOpen}>
        <DialogContent className="sm:max-w-[600px]">
          <DialogHeader>
            <DialogTitle>Memory Details</DialogTitle>
          </DialogHeader>
          {selectedMemory && (
            <div className="space-y-4">
              <div>
                <Label className="text-muted-foreground">Content</Label>
                <p className="mt-1 whitespace-pre-wrap">{selectedMemory.content}</p>
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <Label className="text-muted-foreground">Type</Label>
                  <p className="mt-1 capitalize">{selectedMemory.type}</p>
                </div>
                <div>
                  <Label className="text-muted-foreground">Category</Label>
                  <p className="mt-1">{selectedMemory.category || "N/A"}</p>
                </div>
              </div>
              {selectedMemory.tags && selectedMemory.tags.length > 0 && (
                <div>
                  <Label className="text-muted-foreground">Tags</Label>
                  <div className="mt-1 flex flex-wrap gap-1">
                    {selectedMemory.tags.map((tag) => (
                      <Badge key={tag} variant="outline">
                        {tag}
                      </Badge>
                    ))}
                  </div>
                </div>
              )}
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <Label className="text-muted-foreground">Created</Label>
                  <p className="mt-1">{formatDateTime(selectedMemory.created_at)}</p>
                </div>
                <div>
                  <Label className="text-muted-foreground">Updated</Label>
                  <p className="mt-1">{formatDateTime(selectedMemory.updated_at)}</p>
                </div>
              </div>
            </div>
          )}
        </DialogContent>
      </Dialog>

      {/* Edit Memory Dialog */}
      <Dialog open={isEditOpen} onOpenChange={(open) => {
        setIsEditOpen(open);
        if (!open) setSelectedMemory(null);
      }}>
        <DialogContent className="sm:max-w-[600px]">
          <DialogHeader>
            <DialogTitle>Edit Memory</DialogTitle>
            <DialogDescription>Update memory content and metadata</DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 py-4">
            <div className="grid gap-2">
              <Label htmlFor="edit-content">Content</Label>
              <Textarea
                id="edit-content"
                value={editMemory.content}
                onChange={(e) => setEditMemory({ ...editMemory, content: e.target.value })}
                className="min-h-[100px]"
              />
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div className="grid gap-2">
                <Label htmlFor="edit-type">Type</Label>
                <Select
                  value={editMemory.type}
                  onValueChange={(value) => setEditMemory({ ...editMemory, type: value as Memory["type"] })}
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="conversation">Conversation</SelectItem>
                    <SelectItem value="session">Session</SelectItem>
                    <SelectItem value="user">User</SelectItem>
                    <SelectItem value="org">Organization</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div className="grid gap-2">
                <Label htmlFor="edit-category">Category</Label>
                <Input
                  id="edit-category"
                  value={editMemory.category}
                  onChange={(e) => setEditMemory({ ...editMemory, category: e.target.value })}
                />
              </div>
            </div>
            <div className="grid gap-2">
              <Label htmlFor="edit-tags">Tags (comma-separated)</Label>
              <Input
                id="edit-tags"
                value={editMemory.tags.join(", ")}
                onChange={(e) => setEditMemory({ ...editMemory, tags: e.target.value.split(",").map(t => t.trim()).filter(Boolean) })}
              />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setIsEditOpen(false)}>Cancel</Button>
            <Button onClick={handleUpdate} disabled={updateMutation.isPending}>
              {updateMutation.isPending ? "Saving..." : "Save Changes"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}