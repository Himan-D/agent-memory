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
    <div className="inline-flex w-full rounded-xl border border-zinc-200 bg-zinc-100/80 p-1">
      {modes.map((item) => (
        <Link
          key={item.id}
          href={item.href}
          className={cn(
            "flex-1 rounded-lg px-3 py-2 text-center text-sm font-medium transition-all",
            mode === item.id
              ? "bg-white text-zinc-950 shadow-sm ring-1 ring-zinc-200/80"
              : "text-zinc-500 hover:text-zinc-900"
          )}
        >
          {item.label}
        </Link>
      ))}
    </div>
  );
}
