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
    <Card className="overflow-hidden rounded-2xl border border-zinc-200/80 bg-white/92 shadow-[0_24px_80px_rgba(15,23,42,0.12)] ring-1 ring-white/70 backdrop-blur">
      <CardHeader className="space-y-5 border-b border-zinc-100 px-7 pb-5 pt-7">
        {mode && <AuthModeSwitch mode={mode} />}
        <div className="space-y-2">
          <CardTitle className="text-2xl font-semibold tracking-tight text-zinc-950">
            {title}
          </CardTitle>
          <CardDescription className="text-sm leading-6 text-zinc-600">{description}</CardDescription>
        </div>
      </CardHeader>
      <CardContent className="space-y-5 px-7 py-6">{children}</CardContent>
      {footer && (
        <CardFooter className="justify-center border-t border-zinc-100 bg-zinc-50/80 px-7 py-4 text-sm text-zinc-600">
          {footer}
        </CardFooter>
      )}
    </Card>
  );
}
