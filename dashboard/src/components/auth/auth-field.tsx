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
      <Label htmlFor={fieldId} className="text-sm font-medium">
        {label}
      </Label>
      <div className="relative">
        {Icon && (
          <Icon className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
        )}
        <Input
          id={fieldId}
          className={cn("h-11", Icon && "pl-10", className)}
          {...props}
        />
      </div>
    </div>
  );
}
