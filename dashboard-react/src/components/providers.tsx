"use client";

import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useState, useEffect, Component, type ReactNode } from "react";
import { Toaster } from "@/components/ui/sonner";
import { NotificationProvider } from "@/contexts/notification-context";
import { AuthProvider } from "@/contexts/auth-context";
import * as amplitude from "@amplitude/analytics-browser";

const AMPLITUDE_API_KEY = import.meta.env.VITE_AMPLITUDE_API_KEY || "";

/**
 * Lightweight error boundary specifically for provider-level errors.
 * If NotificationProvider (or similar) throws during hydration, this
 * catches it so the rest of the tree (and CSS) still renders correctly.
 */
class ProviderErrorBoundary extends Component<{ children: ReactNode }, { hasError: boolean }> {
  constructor(props: { children: ReactNode }) {
    super(props);
    this.state = { hasError: false };
  }

  static getDerivedStateFromError(): { hasError: boolean } {
    return { hasError: true };
  }

  componentDidCatch(error: Error) {
    console.error("[ProviderErrorBoundary] Caught during render:", error);
  }

  render() {
    if (this.state.hasError) {
      // Render children without the crashed provider so the page
      // still shows with CSS intact rather than a blank/unstyled page.
      return this.props.children;
    }
    return this.props.children;
  }
}

export function Providers({ children }: { children: React.ReactNode }) {
  const [queryClient] = useState(() => new QueryClient({
    defaultOptions: {
      queries: {
        staleTime: 30 * 1000,
        gcTime: 5 * 60 * 1000,
        retry: 1,
        refetchOnWindowFocus: false,
      },
      mutations: {
        retry: 0,
      },
    },
  }));

  useEffect(() => {
    if (AMPLITUDE_API_KEY) {
      amplitude.init(AMPLITUDE_API_KEY, {
        defaultTracking: {
          sessions: true,
          pageViews: true,
          formInteractions: true,
          fileDownloads: true,
        },
      });
    }
  }, []);

  return (
    <QueryClientProvider client={queryClient}>
      <ProviderErrorBoundary>
        <AuthProvider>
          <NotificationProvider>
            {children}
            <Toaster />
          </NotificationProvider>
        </AuthProvider>
      </ProviderErrorBoundary>
    </QueryClientProvider>
  );
}