"use client";

import { useState, useCallback } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Loader2, Trash2, Upload, FileSpreadsheet, Download } from "lucide-react";
import { useAuditLogger } from "@/hooks/use-audit-logger";

interface BulkOperationsProps {
  resource: string;
  selectedIds: string[];
  onBulkDelete?: (ids: string[]) => void;
  onBulkExport?: () => void;
  onBulkUpdate?: (ids: string[], data: Record<string, unknown>) => void;
  apiDelete?: (id: string) => Promise<void>;
  apiBatchDelete?: (ids: string[]) => Promise<void>;
}

export function BulkOperations({
  resource,
  selectedIds,
  onBulkDelete,
  onBulkExport,
  onBulkUpdate,
  apiDelete,
}: BulkOperationsProps) {
  const [isDeleteOpen, setIsDeleteOpen] = useState(false);
  const [isExportOpen, setIsExportOpen] = useState(false);
  const [isUpdateOpen, setIsUpdateOpen] = useState(false);
  const [updateData, setUpdateData] = useState("");
  const [isProcessing, setIsProcessing] = useState(false);
  const [progress, setProgress] = useState(0);

  const { logDelete, logAction } = useAuditLogger();

  const handleBulkDelete = useCallback(async () => {
    if (selectedIds.length === 0) return;
    setIsProcessing(true);
    setProgress(0);

    try {
      if (apiDelete) {
        // Sequential deletion with progress
        for (let i = 0; i < selectedIds.length; i++) {
          await apiDelete(selectedIds[i]);
          setProgress(Math.round(((i + 1) / selectedIds.length) * 100));
        }
      } else if (onBulkDelete) {
        onBulkDelete(selectedIds);
        setProgress(100);
      }

      // Audit log
      logDelete(resource, `bulk-${selectedIds.length}`, {
        count: selectedIds.length,
        ids: selectedIds,
      });

      toast.success(`Deleted ${selectedIds.length} ${resource} successfully`);
      setIsDeleteOpen(false);
    } catch (err: any) {
      toast.error(`Failed to delete: ${err.message}`);
    } finally {
      setIsProcessing(false);
      setProgress(0);
    }
  }, [selectedIds, apiDelete, onBulkDelete, resource, logDelete]);

  const handleExport = useCallback(() => {
    if (onBulkExport) {
      onBulkExport();
    }
    logAction("EXPORT", resource, `export-${Date.now()}`, {
      count: selectedIds.length,
      type: "csv",
    });
    toast.success("Export started");
    setIsExportOpen(false);
  }, [selectedIds, onBulkExport, resource, logAction]);

  const handleBulkUpdate = useCallback(async () => {
    if (selectedIds.length === 0 || !updateData.trim()) return;
    setIsProcessing(true);

    try {
      let parsedData: Record<string, unknown>;
      try {
        parsedData = JSON.parse(updateData);
      } catch {
        toast.error("Invalid JSON data");
        setIsProcessing(false);
        return;
      }

      if (onBulkUpdate) {
        onBulkUpdate(selectedIds, parsedData);
      }

      logAction("BULK_UPDATE", resource, `bulk-update-${Date.now()}`, {
        count: selectedIds.length,
        fields: Object.keys(parsedData),
      });

      toast.success(`Updated ${selectedIds.length} ${resource} successfully`);
      setIsUpdateOpen(false);
      setUpdateData("");
    } catch (err: any) {
      toast.error(`Failed to update: ${err.message}`);
    } finally {
      setIsProcessing(false);
    }
  }, [selectedIds, updateData, onBulkUpdate, resource, logAction]);

  if (selectedIds.length === 0) {
    return null;
  }

  return (
    <div className="flex items-center gap-2 p-2 bg-muted/50 rounded-lg border">
      <span className="text-sm text-muted-foreground">
        {selectedIds.length} selected
      </span>
      <Button
        variant="destructive"
        size="sm"
        onClick={() => setIsDeleteOpen(true)}
        disabled={isProcessing}
      >
        <Trash2 className="mr-2 h-4 w-4" />
        Delete Selected
      </Button>
      <Button
        variant="outline"
        size="sm"
        onClick={handleExport}
        disabled={isProcessing}
      >
        <Download className="mr-2 h-4 w-4" />
        Export
      </Button>
      <Dialog open={isDeleteOpen} onOpenChange={setIsDeleteOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Confirm Bulk Delete</DialogTitle>
            <DialogDescription>
              Are you sure you want to delete {selectedIds.length} {resource}?
              This action cannot be undone.
            </DialogDescription>
          </DialogHeader>
          {isProcessing && (
            <div className="w-full bg-secondary rounded-full h-2 mt-4">
              <div
                className="bg-primary h-2 rounded-full transition-all duration-300"
                style={{ width: `${progress}%` }}
              />
              <p className="text-sm text-muted-foreground mt-2 text-center">
                {progress}% complete
              </p>
            </div>
          )}
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setIsDeleteOpen(false)}
              disabled={isProcessing}
            >
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={handleBulkDelete}
              disabled={isProcessing}
            >
              {isProcessing ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Deleting...
                </>
              ) : (
                `Delete ${selectedIds.length} Items`
              )}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}