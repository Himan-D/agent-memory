import { useSearchParams, Link } from "react-router-dom";
import { Suspense, useEffect } from "react";
import { AlertTriangle, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { AuthLayout } from "@/components/auth/auth-layout";
import { AuthCard } from "@/components/auth/auth-card";
import { trackPageView } from "@/lib/amplitude";

function ErrorContent() {
  const [searchParams] = useSearchParams();
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
        <Alert variant="destructive">
          <AlertTriangle />
          <AlertTitle>Sign in failed</AlertTitle>
          <AlertDescription>{message}</AlertDescription>
        </Alert>
        <Button className="w-full" size="lg" render={<Link to="/auth/signin" />}>
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
