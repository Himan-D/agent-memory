"use client";

import Link from "next/link";
import { cn } from "@/lib/utils";

type AuthMode = "signin" | "signup";

interface AuthModeSwitchProps {
  mode: AuthMode;
}

const modes: { id: AuthMode; label: string; href: string }[] = [
  { id: "signin", label: "Sign in", href: "/auth/signin" },
  { id: "signup", label: "Sign up", href: "/auth/signup" },
];

export function AuthModeSwitch({ mode }: AuthModeSwitchProps) {
  return (
    <div className="inline-flex w-full rounded-md border bg-muted/60 p-1">
      {modes.map((item) => (
        <Link
          key={item.id}
          href={item.href}
          className={cn(
            "flex-1 rounded px-3 py-2 text-center text-sm font-medium transition-all",
            mode === item.id
              ? "bg-background text-foreground shadow-sm ring-1 ring-border/60"
              : "text-muted-foreground hover:text-foreground"
          )}
        >
          {item.label}
        </Link>
      ))}
    </div>
  );
}
