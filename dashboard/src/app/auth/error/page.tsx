"use client";

import { useSearchParams } from "next/navigation";
import { Suspense } from "react";
import Link from "next/link";
import { Button } from "@/components/ui/button";

function ErrorContent() {
  const searchParams = useSearchParams();
  const error = searchParams.get("error");

  const errorMessages: Record<string, string> = {
    CredentialsSignin: "Invalid email or password",
    default: "An error occurred",
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-background via-background to-muted/20 p-4">
      <div className="w-full max-w-md text-center space-y-6">
        <h1 className="text-4xl font-bold">Authentication Error</h1>
        <p className="text-muted-foreground">
          {error ? errorMessages[error] || errorMessages.default : errorMessages.default}
        </p>
        <Link href="/auth/signin">
          <Button>Try Again</Button>
        </Link>
      </div>
    </div>
  );
}

export default function ErrorPage() {
  return (
    <Suspense fallback={<div>Loading...</div>}>
      <ErrorContent />
    </Suspense>
  );
}