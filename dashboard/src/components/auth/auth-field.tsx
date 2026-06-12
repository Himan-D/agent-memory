import { Label } from "@/components/ui/label";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";
import type { LucideIcon } from "lucide-react";

interface AuthFieldProps extends React.ComponentProps<typeof Input> {
  label: string;
  icon?: LucideIcon;
}

export function AuthField({ label, icon: Icon, className, id, ...props }: AuthFieldProps) {
  const fieldId = id || props.name;

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
          className={cn(
            "h-12 rounded-xl border-zinc-200 bg-zinc-50/70 px-3.5 text-zinc-950 shadow-inner shadow-zinc-950/[0.02] placeholder:text-zinc-400 focus-visible:border-zinc-900 focus-visible:bg-white focus-visible:ring-zinc-900/10",
            Icon && "pl-10",
            className
          )}
          {...props}
        />
      </div>
    </div>
  );
}
