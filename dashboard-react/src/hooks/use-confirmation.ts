"use client";

import { useState, useCallback } from "react";

export interface ConfirmationOptions {
  title: string;
  description?: string;
  confirmText?: string;
  cancelText?: string;
  confirmVariant?: "default" | "destructive";
}

export function useConfirmation() {
  const [open, setOpen] = useState(false);
  const [options, setOptions] = useState<ConfirmationOptions>({
    title: "",
    description: "",
    confirmText: "Confirm",
    cancelText: "Cancel",
    confirmVariant: "default",
  });
  const [onConfirm, setOnConfirm] = useState<(() => void) | null>(null);

  const confirm = useCallback((opts: ConfirmationOptions, callback: () => void) => {
    setOptions(opts);
    setOnConfirm(() => callback);
    setOpen(true);
  }, []);

  const handleConfirm = useCallback(() => {
    onConfirm?.();
    setOpen(false);
    setOnConfirm(null);
  }, [onConfirm]);

  const cancel = useCallback(() => {
    setOpen(false);
    setOnConfirm(null);
  }, []);

  const isOpen = open;

  return {
    confirm,
    handleConfirm,
    cancel,
    isOpen,
    options,
  };
}