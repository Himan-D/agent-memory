"use client";

import { Suspense, useState } from "react";
import { signIn } from "next-auth/react";
import { useRouter, useSearchParams } from "next/navigation";
import { Sparkles, Loader2, CheckCircle } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { AuthHeader } from "@/components/auth/auth-header";
import { trackEvent, initAmplitude } from "@/lib/amplitude";

function SignInForm() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const registered = searchParams.get("registered") === "true";
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState("");

  async function handleSubmit(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setIsLoading(true);
    setError("");

    const formData = new FormData(e.currentTarget);
    const email = formData.get("email") as string;
    const password = formData.get("password") as string;

    trackEvent("sign_in_attempt", { email });

    const result = await signIn("credentials", {
      email,
      password,
      redirect: false,
    });

    if (result?.error) {
      trackEvent("sign_in_error", { email, error: result.error });
      setError("Invalid email or password");
      setIsLoading(false);
    } else {
      trackEvent("sign_in_success", { email });
      router.push("/");
      router.refresh();
    }
  }

  return (
    <Card className="shadow-2xl">
      <CardHeader className="space-y-4">
        <CardTitle className="text-3xl font-bold tracking-tight">Sign in</CardTitle>
        <CardDescription className="text-base">
          Welcome back! Enter your email below to sign in to your account
        </CardDescription>
      </CardHeader>
      <CardContent className="pt-6">
        {registered && (
          <div className="rounded-md bg-green-500/15 p-4 text-sm text-green-600 mb-6 flex items-center">
            <CheckCircle className="h-4 w-4 mr-2" />
            Account created successfully! Please sign in.
          </div>
        )}
        <form onSubmit={handleSubmit} className="space-y-6">
          <div className="space-y-3">
            <Label htmlFor="email" className="text-base font-medium">Email</Label>
            <Input
              id="email"
              name="email"
              type="email"
              placeholder="demo@hystersis.ai"
              required
              defaultValue="demo@hystersis.ai"
              className="h-12 text-lg"
            />
          </div>
          <div className="space-y-3">
            <Label htmlFor="password" className="text-base font-medium">Password</Label>
            <Input
              id="password"
              name="password"
              type="password"
              required
              defaultValue="demo123"
              className="h-12 text-lg"
            />
          </div>
          {error && (
            <div className="rounded-md bg-destructive/15 p-4 text-sm text-destructive">
              {error}
            </div>
          )}
          <Button type="submit" className="w-full h-12 text-lg" disabled={isLoading}>
            {isLoading && <Loader2 className="mr-2 h-5 w-5 animate-spin" />}
            {isLoading ? "Signing in..." : "Sign in"}
          </Button>
        </form>
        <div className="mt-6 text-center text-sm">
          Don&apos;t have an account?{" "}
          <Button
            variant="link"
            className="p-0 h-auto font-semibold"
            onClick={() => router.push("/auth/signup")}
          >
            Sign up
          </Button>
        </div>
        <div className="mt-8 p-4 bg-muted rounded-lg text-center">
          <p className="text-sm font-medium mb-1">Demo Credentials</p>
          <code className="text-sm bg-background px-3 py-2 rounded">
            demo@hystersis.ai / demo123
          </code>
        </div>
      </CardContent>
    </Card>
  );
}

export default function SignInPage() {
  return (
    <div className="min-h-screen flex flex-col items-center justify-center bg-gradient-to-br from-background via-background to-muted/20 p-4">
      <div className="w-full max-w-md">
        <AuthHeader />
        <Suspense fallback={<div className="text-center py-8">Loading...</div>}>
          <SignInForm />
        </Suspense>
      </div>
    </div>
  );
}