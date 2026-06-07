"use client";

import { useSearchParams } from "next/navigation";
import { Suspense, useEffect } from "react";
import Link from "next/link";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { AuthLayout } from "@/components/auth/auth-layout";
import { trackPageView } from "@/lib/amplitude";

function ErrorContent() {
  const searchParams = useSearchParams();
  const error = searchParams.get("error");

  const errorMessages: Record<string, string> = {
    CredentialsSignin: "Invalid email or password",
    default: "An error occurred",
  };

  useEffect(() => {
    trackPageView("auth_error");
  }, []);

  return (
    <AuthLayout>
        <Card className="shadow-2xl">
          <CardHeader className="space-y-4 text-center">
            <CardTitle className="text-3xl font-bold tracking-tight">Authentication Error</CardTitle>
            <CardDescription className="text-base">
              {error ? errorMessages[error] || errorMessages.default : errorMessages.default}
            </CardDescription>
          </CardHeader>
          <CardContent className="pt-6">
            <div className="rounded-lg bg-muted p-4 mb-6">
              <p className="text-sm text-center">
                {error ? errorMessages[error] || errorMessages.default : errorMessages.default}
              </p>
            </div>
            <Link href="/auth/signin" className="w-full">
              <Button className="w-full h-12 text-lg">Try Again</Button>
            </Link>
          </CardContent>
        </Card>
    </AuthLayout>
  );
}

export default function ErrorPage() {
  return (
    <Suspense fallback={<div>Loading...</div>}>
      <ErrorContent />
    </Suspense>
  );
}