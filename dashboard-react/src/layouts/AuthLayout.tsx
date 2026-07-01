import { Outlet, Navigate, useLocation } from "react-router-dom";
import { useAuth } from "@/contexts/auth-context";
import { Loader2 } from "lucide-react";

export function AuthLayout() {
  const { user, isLoading } = useAuth();
  const location = useLocation();

  if (isLoading) {
    return (
      <div className="flex min-h-screen flex-col items-center justify-center">
        <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
      </div>
    );
  }

  if (user && location.pathname !== "/auth/error") {
    return <Navigate to="/" replace />;
  }

  return <Outlet />;
}
