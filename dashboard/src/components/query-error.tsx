"use client";

import { AlertTriangle, RefreshCw } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

interface QueryErrorProps {
  title?: string;
  message?: string;
  onRetry?: () => void;
  isRetrying?: boolean;
}

/** Inline error state with optional retry for failed data fetches. */
export function QueryError({
  title = "Failed to load",
  message = "Something went wrong while loading this section.",
  onRetry,
  isRetrying,
}: QueryErrorProps) {
  return (
    <Card className="border-destructive/40">
      <CardHeader className="pb-2">
        <CardTitle className="flex items-center gap-2 text-base text-destructive">
          <AlertTriangle className="h-4 w-4" />
          {title}
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        <p className="text-sm text-muted-foreground">{message}</p>
        {onRetry && (
          <Button
            variant="outline"
            size="sm"
            onClick={onRetry}
            disabled={isRetrying}
          >
            <RefreshCw
              className={`mr-2 h-4 w-4 ${isRetrying ? "animate-spin" : ""}`}
            />
            {isRetrying ? "Retrying…" : "Retry"}
          </Button>
        )}
      </CardContent>
    </Card>
  );
}
