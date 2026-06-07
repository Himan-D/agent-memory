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
    <Card className="border-0 bg-card/80 shadow-xl shadow-black/5 ring-1 ring-border/60 backdrop-blur-sm dark:shadow-black/20">
      <CardHeader className="space-y-4 pb-2">
        {mode && <AuthModeSwitch mode={mode} />}
        <div className="space-y-1.5">
          <CardTitle className="text-2xl font-semibold tracking-tight">
            {title}
          </CardTitle>
          <CardDescription className="text-base">{description}</CardDescription>
        </div>
      </CardHeader>
      <CardContent className="space-y-5">{children}</CardContent>
      {footer && (
        <CardFooter className="justify-center border-t text-sm text-muted-foreground">
          {footer}
        </CardFooter>
      )}
    </Card>
  );
}
