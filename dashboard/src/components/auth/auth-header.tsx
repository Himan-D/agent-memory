import { Sparkles } from "lucide-react";

// AUTH_PAGE: Reusable logo header for all authentication pages
// Ensures consistent branding across signin, signup, error pages
export function AuthHeader() {
  return (
    <div className="flex items-center justify-center gap-3 mb-8">
      <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-primary">
        <Sparkles className="h-6 w-6 text-primary-foreground" />
      </div>
      <span className="text-2xl font-bold">Hystersis</span>
    </div>
  );
}
