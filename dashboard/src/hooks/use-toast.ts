"use client";

import { useState, useCallback } from "react";
import { toast } from "sonner";

interface ToastOptions {
  duration?: number;
  position?: "top-left" | "top-right" | "bottom-left" | "bottom-right";
}

export function useToast() {
  const show = useCallback((message: string, type: "success" | "error" | "info" | "warning", options?: ToastOptions) => {
    const { duration = 4000 } = options || {};
    
    switch (type) {
      case "success":
        toast.success(message, { duration });
        break;
      case "error":
        toast.error(message, { duration });
        break;
      case "info":
        toast.info(message, { duration });
        break;
      case "warning":
        toast.warning(message, { duration });
        break;
    }
  }, []);

  const success = useCallback((message: string, options?: ToastOptions) => {
    show(message, "success", options);
  }, [show]);

  const error = useCallback((message: string, options?: ToastOptions) => {
    show(message, "error", options);
  }, [show]);

  const info = useCallback((message: string, options?: ToastOptions) => {
    show(message, "info", options);
  }, [show]);

  const warning = useCallback((message: string, options?: ToastOptions) => {
    show(message, "warning", options);
  }, [show]);

  return { success, error, info, warning };
}