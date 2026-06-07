"use client";

import { Sparkles } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";

const DEMO_EMAIL = "demo@hystersis.ai";
const DEMO_PASSWORD = "demo123";

interface DemoCredentialsProps {
  onFill?: (email: string, password: string) => void;
}

export function DemoCredentials({ onFill }: DemoCredentialsProps) {
  function handleFill() {
    onFill?.(DEMO_EMAIL, DEMO_PASSWORD);
  }

  return (
    <div className="rounded-lg border border-dashed border-border/80 bg-muted/40 p-4">
      <div className="mb-3 flex items-center justify-between gap-2">
        <div className="flex items-center gap-2">
          <Sparkles className="h-4 w-4 text-primary" />
          <span className="text-sm font-medium">Try the demo</span>
        </div>
        <Badge variant="secondary">No signup needed</Badge>
      </div>
      <div className="mb-3 space-y-1 rounded-md bg-background/80 px-3 py-2 font-mono text-xs text-muted-foreground">
        <p>
          <span className="text-foreground/70">Email</span> {DEMO_EMAIL}
        </p>
        <p>
          <span className="text-foreground/70">Password</span> {DEMO_PASSWORD}
        </p>
      </div>
      <Button type="button" variant="outline" size="sm" className="w-full" onClick={handleFill}>
        Fill demo credentials
      </Button>
    </div>
  );
}
