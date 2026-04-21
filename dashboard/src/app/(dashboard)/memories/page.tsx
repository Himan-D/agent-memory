"use client";

import { useState } from "react";
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
import { Database, Trash2, Edit, Eye, RefreshCw, MoreHorizontal, Plus } from "lucide-react";
import { toast } from "sonner";

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

  const queryClient = useQueryClient();

  const { data: memoriesData, isLoading, refetch } = useQuery({
    queryKey: ["memories"],
    queryFn: () => memoriesApi.list({ limit: 200 }),
  });

  const createMutation = useMutation({
    mutationFn: async (data: Partial<Memory>) => {
      return memoriesApi.create(data);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["memories"] });
      setIsCreateOpen(false);
      setNewMemory({ content: "", type: "user", category: "", tags: [] });
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
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["memories"] });
      setIsEditOpen(false);
      setSelectedMemory(null);
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
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["memories"] });
      toast.success("Memory deleted");
    },
    onError: (err) => {
      toast.error(`Failed to delete memory: ${err}`);
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
  });

  const clearFilters = () => {
    setSearchQuery("");
    setTypeFilter("all");
    setDateFrom(null);
    setDateTo(null);
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
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-[50%]">Content</TableHead>
                <TableHead>Type</TableHead>
                <TableHead>Tags</TableHead>
                <TableHead>Created</TableHead>
                <TableHead className="w-[50px]"></TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {isLoading ? (
                Array.from({ length: 5 }).map((_, i) => (
                  <TableRow key={i}>
                    <TableCell><Skeleton className="h-4 w-full" /></TableCell>
                    <TableCell><Skeleton className="h-4 w-16" /></TableCell>
                    <TableCell><Skeleton className="h-4 w-24" /></TableCell>
                    <TableCell><Skeleton className="h-4 w-20" /></TableCell>
                    <TableCell><Skeleton className="h-8 w-8" /></TableCell>
                  </TableRow>
                ))
              ) : filteredMemories?.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={5} className="text-center py-8">
                    <Database className="mx-auto h-12 w-12 text-muted-foreground/50" />
                    <p className="mt-2 font-medium">No memories found</p>
                    <p className="text-sm text-muted-foreground">
                      {searchQuery || hasActiveFilters ? "Try different filters" : "Create your first memory to get started"}
                    </p>
                  </TableCell>
                </TableRow>
              ) : (
                filteredMemories?.map((memory) => (
                  <TableRow key={memory.id}>
                    <TableCell>
                      <div className="space-y-1">
                        <p className="font-medium line-clamp-2">
                          {truncate(memory.content, 100)}
                        </p>
                        {memory.tags && memory.tags.length > 0 && (
                          <div className="flex gap-1 flex-wrap">
                            {memory.tags.slice(0, 3).map((tag) => (
                              <Badge key={tag} variant="outline" className="text-xs">
                                {tag}
                              </Badge>
                            ))}
                          </div>
                        )}
                      </div>
                    </TableCell>
                    <TableCell>
                      <Badge
                        variant="outline"
                        className={getTypeColors(memory.type)}
                      >
                        {memory.type}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-sm text-muted-foreground">
                      {memory.tags?.length || 0}
                    </TableCell>
                    <TableCell className="text-sm text-muted-foreground">
                      {formatDateTime(memory.created_at)}
                    </TableCell>
                    <TableCell>
                      <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                          <Button variant="ghost" size="icon">
                            <MoreHorizontal className="h-4 w-4" />
                          </Button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end">
                          <DropdownMenuItem onClick={() => handleView(memory)}>
                            <Eye className="mr-2 h-4 w-4" />
                            View
                          </DropdownMenuItem>
                          <DropdownMenuItem onClick={() => handleEdit(memory)}>
                            <Edit className="mr-2 h-4 w-4" />
                            Edit
                          </DropdownMenuItem>
                          <DropdownMenuSeparator />
                          <DropdownMenuItem
                            className="text-destructive"
                            onClick={() => deleteMutation.mutate(memory.id)}
                          >
                            <Trash2 className="mr-2 h-4 w-4" />
                            Delete
                          </DropdownMenuItem>
                        </DropdownMenuContent>
                      </DropdownMenu>
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      <div className="flex items-center justify-between text-sm text-muted-foreground">
        <span>Showing {filteredMemories?.length || 0} of {memoriesData?.memories?.length || 0} memories</span>
        {hasActiveFilters && (
          <Button variant="link" onClick={clearFilters}>Clear filters</Button>
        )}
      </div>

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