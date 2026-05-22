"use client";

import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { memoriesApi } from "@/lib/api";
import { toast } from "sonner";
import { formatDateTime, truncate } from "@/lib/utils";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
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
import { Skeleton } from "@/components/ui/skeleton";
import { Eye, Edit, Trash2, MoreHorizontal, Copy, Link } from "lucide-react";
import { MemoryForm } from "./memory-form";
import { useConfirmation } from "@/hooks";

interface MemoryTableProps {
  memories: Array<{
    id: string;
    content: string;
    type: string;
    category?: string;
    tags?: string[];
    importance?: string;
    created_at: string;
    updated_at: string;
    metadata?: Record<string, unknown>;
  }>;
  onDelete?: (id: string) => void;
  onView?: (memory: any) => void;
  onEdit?: (memory: any) => void;
  selectedIds?: Set<string>;
  onSelect?: (id: string, checked: boolean) => void;
  loading?: boolean;
}

function getTypeColors(type: string) {
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
}

function getImportanceBadge(importance?: string) {
  switch (importance) {
    case "critical":
      return "bg-red-500/10 text-red-600 border-red-500/20";
    case "high":
      return "bg-orange-500/10 text-orange-600 border-orange-500/20";
    case "medium":
      return "bg-yellow-500/10 text-yellow-600 border-yellow-500/20";
    case "low":
      return "bg-green-500/10 text-green-600 border-green-500/20";
    default:
      return "bg-gray-500/10 text-gray-600 border-gray-500/20";
  }
}

export function MemoryTable({
  memories,
  onDelete,
  onView,
  onEdit,
  selectedIds,
  onSelect,
  loading,
}: MemoryTableProps) {
  const [isViewOpen, setIsViewOpen] = useState(false);
  const [isEditOpen, setIsEditOpen] = useState(false);
  const [selectedMemory, setSelectedMemory] = useState<any>(null);
  const queryClient = useQueryClient();

  const { confirm, open, title, description, handleConfirm, cancel } = useConfirmation();

  const deleteMutation = useMutation({
    mutationFn: (id: string) => memoriesApi.delete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["memories"] });
      toast.success("Memory deleted successfully");
    },
    onError: (err: Error) => {
      toast.error(`Failed to delete memory: ${err.message}`);
    },
  });

  const handleDelete = (memory: any) => {
    confirm({
      title: "Delete Memory",
      description: `Are you sure you want to delete this memory? This action cannot be undone.`,
      onConfirm: () => {
        deleteMutation.mutate(memory.id);
        onDelete?.(memory.id);
      },
    });
  };

  const handleView = (memory: any) => {
    setSelectedMemory(memory);
    setIsViewOpen(true);
    onView?.(memory);
  };

  const handleEdit = (memory: any) => {
    setSelectedMemory(memory);
    setIsEditOpen(true);
    onEdit?.(memory);
  };

  const handleCopyContent = (content: string) => {
    if (typeof navigator !== "undefined" && navigator.clipboard) {
      navigator.clipboard.writeText(content).then(() => {
        toast.success("Content copied to clipboard");
      }).catch(() => {
        toast.error("Failed to copy content");
      });
    }
  };

  const handleCopyLink = (id: string) => {
    const url = `${window.location.origin}/memories#${id}`;
    if (typeof navigator !== "undefined" && navigator.clipboard) {
      navigator.clipboard.writeText(url).then(() => {
        toast.success("Link copied to clipboard");
      });
    }
  };

  const toggleSelect = (id: string, checked: boolean) => {
    onSelect?.(id, checked);
  };

  if (loading) {
    return (
      <div className="space-y-4">
        {Array.from({ length: 5 }).map((_, i) => (
          <div key={i} className="flex items-center space-x-4 p-4">
            <Skeleton className="h-6 w-6" />
            <Skeleton className="h-4 flex-1" />
            <Skeleton className="h-4 w-16" />
            <Skeleton className="h-4 w-24" />
            <Skeleton className="h-8 w-8" />
          </div>
        ))}
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <Table>
        <TableHeader>
          <TableRow>
            {selectedIds !== undefined && (
              <TableHead className="w-[40px]">
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => {
                    const allChecked = memories.every((m) => selectedIds.has(m.id));
                    memories.forEach((m) => onSelect?.(m.id, !allChecked));
                  }}
                >
                  {memories.every((m) => selectedIds.has(m.id)) ? "✓" : "☐"}
                </Button>
              </TableHead>
            )}
            <TableHead className="min-w-[300px]">Content</TableHead>
            <TableHead>Type</TableHead>
            <TableHead>Importance</TableHead>
            <TableHead>Tags</TableHead>
            <TableHead>Created</TableHead>
            <TableHead className="w-[50px]"></TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {memories.length === 0 ? (
            <TableRow>
              <TableCell colSpan={6} className="text-center py-12">
                <div className="flex flex-col items-center gap-2 text-muted-foreground">
                  <p className="text-lg font-medium">No memories found</p>
                  <p className="text-sm">Create your first memory to get started</p>
                </div>
              </TableCell>
            </TableRow>
          ) : (
            memories.map((memory) => (
              <TableRow
                key={memory.id}
                className={selectedIds?.has(memory.id) ? "bg-accent" : undefined}
              >
                {selectedIds !== undefined && (
                  <TableCell>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => toggleSelect(memory.id, !selectedIds.has(memory.id))}
                    >
                      {selectedIds.has(memory.id) ? "✓" : "☐"}
                    </Button>
                  </TableCell>
                )}
                <TableCell>
                  <div className="space-y-1">
                    <p className="font-medium line-clamp-2 text-sm">
                      {truncate(memory.content, 100)}
                    </p>
                    {memory.category && (
                      <Badge variant="secondary" className="text-xs">
                        {memory.category}
                      </Badge>
                    )}
                  </div>
                </TableCell>
                <TableCell>
                  <Badge variant="outline" className={getTypeColors(memory.type)}>
                    {memory.type}
                  </Badge>
                </TableCell>
                <TableCell>
                  {memory.importance && (
                    <Badge variant="outline" className={getImportanceBadge(memory.importance)}>
                      {memory.importance}
                    </Badge>
                  )}
                </TableCell>
                <TableCell className="text-sm text-muted-foreground">
                  {(memory.tags || []).slice(0, 3).map((tag: string) => (
                    <Badge key={tag} variant="outline" className="text-xs mr-1">
                      {tag}
                    </Badge>
                  ))}
                  {(memory.tags || []).length > 3 && (
                    <span className="text-xs text-muted-foreground">
                      +{(memory.tags || []).length - 3} more
                    </span>
                  )}
                </TableCell>
                <TableCell className="text-sm text-muted-foreground whitespace-nowrap">
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
                        View Details
                      </DropdownMenuItem>
                      <DropdownMenuItem onClick={() => handleEdit(memory)}>
                        <Edit className="mr-2 h-4 w-4" />
                        Edit
                      </DropdownMenuItem>
                      <DropdownMenuItem onClick={() => handleCopyContent(memory.content)}>
                        <Copy className="mr-2 h-4 w-4" />
                        Copy Content
                      </DropdownMenuItem>
                      <DropdownMenuItem onClick={() => handleCopyLink(memory.id)}>
                        <Link className="mr-2 h-4 w-4" />
                        Copy Link
                      </DropdownMenuItem>
                      <DropdownMenuSeparator />
                      <DropdownMenuItem
                        className="text-destructive"
                        onClick={() => handleDelete(memory)}
                        disabled={deleteMutation.isPending}
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

      {/* View Dialog */}
      <MemoryForm
        open={isViewOpen}
        onOpenChange={setIsViewOpen}
        mode="edit"
        initialData={selectedMemory}
        onSuccess={() => setSelectedMemory(null)}
      />

      {/* Edit Dialog */}
      <MemoryForm
        open={isEditOpen}
        onOpenChange={setIsEditOpen}
        mode="edit"
        initialData={selectedMemory}
        onSuccess={() => setSelectedMemory(null)}
      />
    </div>
  );
}