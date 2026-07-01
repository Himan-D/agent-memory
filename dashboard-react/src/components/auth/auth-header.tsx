import { Link } from "react-router-dom";
import { HystersisLogo } from "@/components/auth/hystersis-logo";

const LANDING_URL = process.env.NEXT_PUBLIC_LANDING_URL || "https://hystersis.com";

/** @deprecated Use AuthLayout brand panel or HystersisLogo directly */
export function AuthHeader() {
  return (
    <div className="mb-8 flex flex-col items-center gap-2">
      <Link to={LANDING_URL} className="transition-opacity hover:opacity-90">
        <HystersisLogo wordmarkClassName="text-2xl" />
      </Link>
      <p className="text-sm text-muted-foreground">Dashboard</p>
    </div>
  );
}
