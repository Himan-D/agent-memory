"use client";

import { Suspense, useState } from "react";
import { signIn } from "next-auth/react";
import { useRouter, useSearchParams } from "next/navigation";
import { Loader2, CheckCircle } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { AuthLayout } from "@/components/auth/auth-layout";
import { DemoCredentials } from "@/components/auth/demo-credentials";
import { trackSignInAttempt, trackSignInSuccess, trackSignInError } from "@/lib/amplitude";

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

    trackSignInAttempt(email);

    const result = await signIn("credentials", {
      email,
      password,
      redirect: false,
    });

    if (result?.error) {
      trackSignInError(email, result.error);
      setError("Invalid email or password");
      setIsLoading(false);
    } else {
      trackSignInSuccess(email);
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
              placeholder="you@example.com"
              required
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
        <DemoCredentials />
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
      </CardContent>
    </Card>
  );
}

export default function SignInPage() {
  return (
    <AuthLayout>
      <Suspense fallback={<div className="text-center py-8">Loading...</div>}>
        <SignInForm />
      </Suspense>
    </AuthLayout>
  );
}