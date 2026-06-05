"use client";

import { useState, useCallback } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { memoriesApi } from "@/lib/api";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Label } from "@/components/ui/label";
import { Loader2 } from "lucide-react";

interface MemoryFormProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  mode: "create" | "edit";
  initialData?: {
    id?: string;
    content: string;
    type: string;
    category?: string;
    tags?: string[];
  };
  onSuccess?: () => void;
}

export function MemoryForm({ open, onOpenChange, mode, initialData, onSuccess }: MemoryFormProps) {
  const [formData, setFormData] = useState({
    content: initialData?.content || "",
    type: initialData?.type || "user",
    category: initialData?.category || "",
    tags: initialData?.tags?.join(", ") || "",
    importance: initialData?.tags ? "medium" : "medium",
  });
  const [isSubmitting, setIsSubmitting] = useState(false);

  const queryClient = useQueryClient();

  const createMutation = useMutation({
    mutationFn: (data: Record<string, unknown>) => memoriesApi.create(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["memories"] });
      toast.success("Memory created successfully");
      setIsSubmitting(false);
      onOpenChange(false);
      onSuccess?.();
    },
    onError: (err: Error) => {
      toast.error(`Failed to create memory: ${err.message}`);
      setIsSubmitting(false);
    },
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: Record<string, unknown> }) =>
      memoriesApi.update(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["memories"] });
      toast.success("Memory updated successfully");
      setIsSubmitting(false);
      onOpenChange(false);
      onSuccess?.();
    },
    onError: (err: Error) => {
      toast.error(`Failed to update memory: ${err.message}`);
      setIsSubmitting(false);
    },
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    setIsSubmitting(true);

    const data = {
      content: formData.content,
      type: formData.type,
      category: formData.category || undefined,
      tags: formData.tags ? formData.tags.split(",").map((t) => t.trim()).filter(Boolean) : undefined,
    };

    if (mode === "create") {
      createMutation.mutate(data);
    } else if (initialData?.id) {
      updateMutation.mutate({ id: initialData.id, data });
    }
  };

  const isLoading = createMutation.isPending || updateMutation.isPending;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[600px]">
        <DialogHeader>
          <DialogTitle>{mode === "create" ? "Create New Memory" : "Edit Memory"}</DialogTitle>
          <DialogDescription>
            {mode === "create"
              ? "Add a new memory to your agent's knowledge base"
              : "Update existing memory content and metadata"}
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <Label htmlFor="content">Content *</Label>
            <Textarea
              id="content"
              placeholder="Enter memory content..."
              value={formData.content}
              onChange={(e) => setFormData((prev) => ({ ...prev, content: e.target.value }))}
              className="min-h-[100px] mt-1.5"
              required
              maxLength={50000}
            />
            <p className="text-xs text-muted-foreground mt-1">
              {formData.content.length}/50,000 characters
            </p>
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <Label htmlFor="type">Type *</Label>
              <Select
                value={formData.type}
                onValueChange={(value) => setFormData((prev) => ({ ...prev, type: value || "user" }))}
              >
                <SelectTrigger className="mt-1.5">
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
            <div>
              <Label htmlFor="category">Category</Label>
              <Input
                id="category"
                placeholder="e.g., preferences"
                value={formData.category}
                onChange={(e) => setFormData((prev) => ({ ...prev, category: e.target.value }))}
                className="mt-1.5"
                maxLength={100}
              />
            </div>
          </div>
          <div>
            <Label htmlFor="tags">Tags (comma-separated)</Label>
            <Input
              id="tags"
              placeholder="e.g., important, follow-up, project-alpha"
              value={formData.tags}
              onChange={(e) => setFormData((prev) => ({ ...prev, tags: e.target.value }))}
              className="mt-1.5"
            />
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)} disabled={isSubmitting}>
              Cancel
            </Button>
            <Button type="submit" disabled={isSubmitting || !formData.content.trim()}>
              {isSubmitting && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
              {mode === "create" ? "Create Memory" : "Save Changes"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}