"use client";

import { Suspense, useState } from "react";
import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { authClient } from "@/lib/auth-client";
import { Loader2, Mail, Lock, CheckCircle, AlertCircle } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { AuthLayout } from "@/components/auth/auth-layout";
import { AuthCard } from "@/components/auth/auth-card";
import { AuthField } from "@/components/auth/auth-field";
import { trackSignInAttempt, trackSignInSuccess, trackSignInError } from "@/lib/amplitude";

function SignInForm() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const registered = searchParams.get("registered") === "true";
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");

  async function handleSubmit(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setIsLoading(true);
    setError("");

    trackSignInAttempt(email);

    const { error: signInError } = await authClient.signIn.credentials({
      email,
      password,
    });

    if (signInError) {
      trackSignInError(email, signInError.message || "sign_in_failed");
      setError("The email or password is incorrect. Check your credentials and try again.");
      setIsLoading(false);
    } else {
      trackSignInSuccess(email);
      router.push("/");
      router.refresh();
    }
  }

  return (
    <AuthCard
      mode="signin"
      title="Sign in to Hystersis"
      description="Use your workspace credentials to access memory operations, API keys, analytics, and agent configuration."
      footer={
        <>
          Don&apos;t have an account?{" "}
          <Link href="/auth/signup" className="font-medium text-primary hover:underline">
            Create one
          </Link>
        </>
      }
    >
      {registered && (
        <Alert variant="success">
          <CheckCircle />
          <AlertDescription>
            Your account is ready. Sign in to open the dashboard.
          </AlertDescription>
        </Alert>
      )}

      <form onSubmit={handleSubmit} className="space-y-4">
        <AuthField
          label="Email"
          name="email"
          type="email"
          icon={Mail}
          placeholder="you@company.com"
          required
          autoComplete="email"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
        />
        <AuthField
          label="Password"
          name="password"
          type="password"
          icon={Lock}
          placeholder="Enter your password"
          required
          autoComplete="current-password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
        />

        {error && (
          <Alert variant="destructive">
            <AlertCircle />
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        )}

        <Button type="submit" className="w-full" size="lg" disabled={isLoading}>
          {isLoading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
          {isLoading ? "Signing in..." : "Sign in"}
        </Button>
      </form>
    </AuthCard>
  );
}

export default function SignInPage() {
  return (
    <AuthLayout>
      <Suspense
        fallback={
          <div className="flex items-center justify-center py-16 text-muted-foreground">
            <Loader2 className="mr-2 h-5 w-5 animate-spin" />
            Loading...
          </div>
        }
      >
        <SignInForm />
      </Suspense>
    </AuthLayout>
  );
}
