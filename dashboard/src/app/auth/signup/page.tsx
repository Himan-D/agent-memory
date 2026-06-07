"use client";

import { useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { Loader2, User, Mail, Lock, AlertCircle } from "lucide-react";
import { Button } from "@/components/ui/button";
import { AuthLayout } from "@/components/auth/auth-layout";
import { AuthCard } from "@/components/auth/auth-card";
import { AuthField } from "@/components/auth/auth-field";
import { trackSignUpAttempt, trackSignUpSuccess, trackSignUpError } from "@/lib/amplitude";

export default function SignUpPage() {
  const router = useRouter();
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState("");
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");

  async function handleSubmit(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setIsLoading(true);
    setError("");

    if (password !== confirmPassword) {
      setError("Passwords do not match");
      setIsLoading(false);
      return;
    }

    if (password.length < 6) {
      setError("Password must be at least 6 characters");
      setIsLoading(false);
      return;
    }

    trackSignUpAttempt(email);

    try {
      const response = await fetch("/api/proxy?endpoint=/auth/register", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email, password, name }),
      });

      const data = await response.json();

      if (data.success) {
        trackSignUpSuccess(email);
        router.push("/auth/signin?registered=true");
      } else {
        trackSignUpError(email, data.error || "Registration failed");
        setError(data.error || "Failed to create account");
      }
    } catch {
      trackSignUpError(email, "Network error");
      setError("Network error. Please try again.");
    }

    setIsLoading(false);
  }

  return (
    <AuthLayout>
      <AuthCard
        mode="signup"
        title="Create your account"
        description="Get started with the Hystersis agent memory dashboard."
        footer={
          <>
            Already have an account?{" "}
            <Link href="/auth/signin" className="font-medium text-primary hover:underline">
              Sign in
            </Link>
          </>
        }
      >
        <form onSubmit={handleSubmit} className="space-y-4">
          <AuthField
            label="Full name"
            name="name"
            type="text"
            icon={User}
            placeholder="Jane Doe"
            required
            autoComplete="name"
            value={name}
            onChange={(e) => setName(e.target.value)}
          />
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
            placeholder="At least 6 characters"
            required
            autoComplete="new-password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
          />
          <AuthField
            label="Confirm password"
            name="confirmPassword"
            type="password"
            icon={Lock}
            placeholder="Repeat your password"
            required
            autoComplete="new-password"
            value={confirmPassword}
            onChange={(e) => setConfirmPassword(e.target.value)}
          />

          {error && (
            <div className="flex items-start gap-3 rounded-lg border border-destructive/30 bg-destructive/10 px-4 py-3 text-sm text-destructive">
              <AlertCircle className="mt-0.5 h-4 w-4 shrink-0" />
              <p>{error}</p>
            </div>
          )}

          <Button type="submit" className="w-full" size="lg" disabled={isLoading}>
            {isLoading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
            {isLoading ? "Creating account..." : "Create account"}
          </Button>
        </form>
      </AuthCard>
    </AuthLayout>
  );
}
