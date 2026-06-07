import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { AuthModeSwitch } from "@/components/auth/auth-mode-switch";

type AuthMode = "signin" | "signup";

interface AuthCardProps {
  mode?: AuthMode;
  title: string;
  description: string;
  children: React.ReactNode;
  footer?: React.ReactNode;
}

export function AuthCard({
  mode,
  title,
  description,
  children,
  footer,
}: AuthCardProps) {
  return (
    <Card className="overflow-hidden border bg-card shadow-sm ring-1 ring-border/70">
      <CardHeader className="space-y-5 pb-3">
        {mode && <AuthModeSwitch mode={mode} />}
        <div className="space-y-2">
          <CardTitle className="text-2xl font-semibold">
            {title}
          </CardTitle>
          <CardDescription className="text-sm leading-6">{description}</CardDescription>
        </div>
      </CardHeader>
      <CardContent className="space-y-5 pt-2">{children}</CardContent>
      {footer && (
        <CardFooter className="justify-center border-t bg-muted/30 px-6 py-4 text-sm text-muted-foreground">
          {footer}
        </CardFooter>
      )}
    </Card>
  );
}
