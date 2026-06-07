"use client";

import { useSearchParams } from "next/navigation";
import { Suspense, useEffect } from "react";
import Link from "next/link";
import { AlertTriangle, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { AuthLayout } from "@/components/auth/auth-layout";
import { AuthCard } from "@/components/auth/auth-card";
import { trackPageView } from "@/lib/amplitude";

function ErrorContent() {
  const searchParams = useSearchParams();
  const error = searchParams.get("error");

  const errorMessages: Record<string, string> = {
    CredentialsSignin: "Invalid email or password",
    default: "An error occurred during authentication",
  };

  const message = error ? errorMessages[error] || errorMessages.default : errorMessages.default;

  useEffect(() => {
    trackPageView("auth_error");
  }, []);

  return (
    <AuthLayout>
      <AuthCard title="Authentication error" description="We couldn't complete your sign in.">
        <div className="flex flex-col items-center gap-4 rounded-lg border border-destructive/30 bg-destructive/5 px-6 py-8 text-center">
          <div className="flex h-12 w-12 items-center justify-center rounded-full bg-destructive/10">
            <AlertTriangle className="h-6 w-6 text-destructive" />
          </div>
          <p className="text-sm text-muted-foreground">{message}</p>
        </div>
        <Button className="w-full" size="lg" render={<Link href="/auth/signin" />}>
          Back to sign in
        </Button>
      </AuthCard>
    </AuthLayout>
  );
}

export default function ErrorPage() {
  return (
    <Suspense
      fallback={
        <div className="flex min-h-screen items-center justify-center text-muted-foreground">
          <Loader2 className="mr-2 h-5 w-5 animate-spin" />
          Loading...
        </div>
      }
    >
      <ErrorContent />
    </Suspense>
  );
}
