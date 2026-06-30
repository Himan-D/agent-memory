"use client";

import { useState } from "react";
import { Label } from "@/components/ui/label";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { Eye, EyeOff, type LucideIcon } from "lucide-react";

interface AuthFieldProps extends React.ComponentProps<typeof Input> {
  label: string;
  icon?: LucideIcon;
}

export function AuthField({ label, icon: Icon, className, id, type, ...props }: AuthFieldProps) {
  const fieldId = id || props.name;
  const isPassword = type === "password";
  const [showPassword, setShowPassword] = useState(false);

  return (
    <div className="space-y-2">
      <Label htmlFor={fieldId} className="text-sm font-medium text-zinc-800">
        {label}
      </Label>
      <div className="relative">
        {Icon && (
          <Icon className="pointer-events-none absolute left-3.5 top-1/2 h-4 w-4 -translate-y-1/2 text-zinc-400" />
        )}
        <Input
          id={fieldId}
          type={isPassword ? (showPassword ? "text" : "password") : type}
          className={cn(
            "h-12 rounded-xl border-zinc-200 bg-zinc-50/70 px-3.5 text-zinc-950 shadow-inner shadow-zinc-950/[0.02] placeholder:text-zinc-400 focus-visible:border-zinc-900 focus-visible:bg-white focus-visible:ring-zinc-900/10",
            Icon && "pl-10",
            isPassword && "pr-12",
            className
          )}
          {...props}
        />
        {isPassword && (
          <Button
            type="button"
            variant="ghost"
            size="icon"
            aria-label={showPassword ? "Hide password" : "Show password"}
            aria-pressed={showPassword}
            onClick={() => setShowPassword((prev) => !prev)}
            className="absolute right-1 top-1/2 h-9 w-9 -translate-y-1/2 text-zinc-400 hover:text-zinc-700"
          >
            {showPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
          </Button>
        )}
      </div>
    </div>
  );
}
