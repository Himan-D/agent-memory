import { Suspense } from "react";
import { Outlet, Navigate, useLocation } from "react-router-dom";
import { useAuth } from "@/contexts/auth-context";
import { Loader2 } from "lucide-react";
import { Sidebar } from "@/components/dashboard/sidebar";
import { Header } from "@/components/dashboard/header";
import { Toaster } from "@/components/ui/sonner";
import { Skeleton } from "@/components/ui/skeleton";
import { ErrorBoundary } from "@/components/error-boundary";

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

export function DashboardLayout() {
  const { user, isLoading } = useAuth();
  const location = useLocation();

  if (isLoading) {
    return (
      <div className="flex min-h-screen flex-col items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    );
  }

  if (!user) {
    return <Navigate to={`/auth/signin?callbackUrl=${encodeURIComponent(location.pathname)}`} replace />;
  }

  return (
    <div className="min-h-screen bg-background">
      <Sidebar />
      <div className="pl-64 transition-all duration-300">
        <Header />
        <main className="p-6">
          <ErrorBoundary>
            <Suspense fallback={<PageLoader />}>
              <Outlet />
            </Suspense>
          </ErrorBoundary>
        </main>
      </div>
      <Toaster />
    </div>
  );
}
