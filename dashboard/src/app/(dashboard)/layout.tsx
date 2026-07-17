"use client";

import { Suspense } from "react";
import { Sidebar } from "@/components/dashboard/sidebar";
import { Header } from "@/components/dashboard/header";
import { PageBreadcrumb } from "@/components/dashboard/page-breadcrumb";
import { Toaster } from "@/components/ui/sonner";
import { Skeleton } from "@/components/ui/skeleton";
import { ErrorBoundary } from "@/components/error-boundary";
import { OfflineBanner } from "@/components/offline-banner";
import { RealtimeProvider } from "@/contexts/realtime-context";

const PageLoader = () => (
  <div className="space-y-6">
    <div className="space-y-2">
      <Skeleton className="h-9 w-48" />
      <Skeleton className="h-5 w-64" />
    </div>
    <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
      {Array.from({ length: 4 }).map((_, i) => (
        <Skeleton key={i} className="h-32" />
      ))}
    </div>
    <div className="grid gap-4 md:grid-cols-2">
      <Skeleton className="h-[300px]" />
      <Skeleton className="h-[300px]" />
    </div>
  </div>
);

export default function DashboardLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <RealtimeProvider>
      <div className="min-h-screen bg-background">
        <Sidebar />
        <div className="pl-64 transition-all duration-300">
          <Header />
          <OfflineBanner />
          <main className="p-6">
            <PageBreadcrumb />
            <ErrorBoundary title="Page failed to render">
              <Suspense fallback={<PageLoader />}>{children}</Suspense>
            </ErrorBoundary>
          </main>
        </div>
        <Toaster />
      </div>
    </RealtimeProvider>
  );
}
