import Link from "next/link";
import { ArrowUpRight, Database, KeyRound, Network, ShieldCheck } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { HystersisLogo } from "@/components/auth/hystersis-logo";

const LANDING_URL = process.env.NEXT_PUBLIC_LANDING_URL || "https://hystersis.com";

const features = [
  { icon: Database, label: "Persistent memory, entities, and sessions" },
  { icon: Network, label: "Graph search with source-aware retrieval" },
  { icon: KeyRound, label: "API keys, webhooks, and team access" },
];

interface AuthLayoutProps {
  children: React.ReactNode;
}

export function AuthLayout({ children }: AuthLayoutProps) {
  return (
    <div className="min-h-screen bg-background lg:grid lg:grid-cols-[minmax(0,0.92fr)_minmax(420px,0.74fr)]">
      <div className="relative hidden min-h-screen flex-col justify-between overflow-hidden border-r bg-zinc-950 p-10 text-white lg:flex">
        <div className="absolute inset-0 opacity-30 [background-image:linear-gradient(rgba(255,255,255,.08)_1px,transparent_1px),linear-gradient(90deg,rgba(255,255,255,.08)_1px,transparent_1px)] [background-size:48px_48px]" />
        <div className="absolute left-0 top-24 h-px w-full bg-gradient-to-r from-transparent via-white/30 to-transparent" />
        <div className="relative z-10">
          <Link href={LANDING_URL} className="inline-block transition-opacity hover:opacity-90">
            <HystersisLogo
              showWordmark
              wordmarkClassName="text-2xl text-white"
              iconClassName="h-8 w-8"
            />
          </Link>
          <div className="mt-6 flex items-center gap-3">
            <Badge variant="secondary" className="bg-white/10 text-white hover:bg-white/15">
              Production dashboard
            </Badge>
            <Link
              href={LANDING_URL}
              className="inline-flex items-center gap-1 text-sm text-white/70 transition-colors hover:text-white"
            >
              Back to site
              <ArrowUpRight className="h-3.5 w-3.5" />
            </Link>
          </div>
        </div>

        <div className="relative z-10 max-w-xl space-y-8">
          <div className="space-y-4">
            <p className="text-sm font-medium uppercase tracking-[0.18em] text-white/50">
              Hystersis control plane
            </p>
            <h1 className="text-4xl font-semibold leading-tight">
              Sign in to operate agent memory in production.
            </h1>
            <p className="max-w-lg text-base leading-7 text-white/68">
              Review memory writes, tune compression, manage API access, and monitor the
              retrieval layer your agents depend on.
            </p>
          </div>
          <ul className="grid gap-3">
            {features.map(({ icon: Icon, label }) => (
              <li key={label} className="flex items-center gap-3 rounded-lg border border-white/10 bg-white/[0.04] p-3 text-sm text-white/82">
                <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-md bg-white/10">
                  <Icon className="h-4 w-4" />
                </span>
                {label}
              </li>
            ))}
          </ul>
        </div>

        <div className="relative z-10 flex items-center gap-2 text-sm text-white/60">
          <ShieldCheck className="h-4 w-4" />
          Secure access for Hystersis workspaces
        </div>
      </div>

      <div className="flex min-h-screen flex-col items-center justify-center bg-background p-6 sm:p-10">
        <div className="mb-8 flex justify-center lg:hidden">
          <Link href={LANDING_URL} className="transition-opacity hover:opacity-90">
            <HystersisLogo wordmarkClassName="text-xl" />
          </Link>
        </div>
        <div className="w-full max-w-[440px]">{children}</div>
      </div>
    </div>
  );
}
