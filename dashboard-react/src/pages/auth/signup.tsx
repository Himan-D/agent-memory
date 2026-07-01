import { useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { Loader2, User, Mail, Lock, AlertCircle } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { AuthLayout } from "@/components/auth/auth-layout";
import { AuthCard } from "@/components/auth/auth-card";
import { AuthField } from "@/components/auth/auth-field";
import { trackSignUpAttempt, trackSignUpSuccess, trackSignUpError } from "@/lib/amplitude";

export default function SignUpPage() {
  const navigate = useNavigate();
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
      setError("Both password fields must match.");
      setIsLoading(false);
      return;
    }

    if (password.length < 6) {
      setError("Use at least 6 characters for your password.");
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
        navigate("/auth/signin?registered=true");
      } else {
        trackSignUpError(email, data.error || "Registration failed");
        setError(data.error || "We could not create the account. Check the details and try again.");
      }
    } catch {
      trackSignUpError(email, "Network error");
      setError("The dashboard could not reach the API. Please try again in a moment.");
    }

    setIsLoading(false);
  }

  return (
    <AuthLayout>
      <AuthCard
        mode="signup"
        title="Create a Hystersis account"
        description="Set up access to the dashboard for memory search, compression controls, API keys, and agent observability."
        footer={
          <>
            Already have an account?{" "}
            <Link to="/auth/signin" className="font-medium text-primary hover:underline">
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
            <Alert variant="destructive">
              <AlertCircle />
              <AlertDescription>{error}</AlertDescription>
            </Alert>
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
