import Link from "next/link";
import { ArrowUpRight, Database, KeyRound, Network, ShieldCheck, Sparkles } from "lucide-react";
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
    <div className="min-h-screen bg-[#f7f8fb] text-zinc-950 lg:grid lg:grid-cols-[minmax(0,0.98fr)_minmax(430px,0.72fr)]">
      <div className="relative hidden min-h-screen flex-col justify-between overflow-hidden border-r border-white/10 bg-[#080b12] p-10 text-white lg:flex">
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_18%_18%,rgba(20,184,166,0.18),transparent_30%),radial-gradient(circle_at_75%_12%,rgba(99,102,241,0.22),transparent_26%),linear-gradient(135deg,rgba(255,255,255,0.08),transparent_32%)]" />
        <div className="absolute inset-0 opacity-[0.18] [background-image:linear-gradient(rgba(255,255,255,.14)_1px,transparent_1px),linear-gradient(90deg,rgba(255,255,255,.14)_1px,transparent_1px)] [background-size:56px_56px]" />
        <div className="absolute -left-20 top-1/3 h-72 w-72 rounded-full border border-cyan-300/20" />
        <div className="absolute bottom-12 right-10 h-56 w-56 rounded-full border border-indigo-300/20" />
        <div className="absolute left-0 top-28 h-px w-full bg-gradient-to-r from-transparent via-cyan-200/35 to-transparent" />
        <div className="relative z-10">
          <Link href={LANDING_URL} className="inline-block transition-opacity hover:opacity-90">
            <HystersisLogo
              showWordmark
              wordmarkClassName="text-2xl text-white"
              iconClassName="h-8 w-8"
            />
          </Link>
          <div className="mt-6 flex items-center gap-3">
            <Badge variant="secondary" className="border border-white/10 bg-white/10 text-white shadow-none hover:bg-white/15">
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
            <p className="inline-flex items-center gap-2 text-sm font-medium uppercase tracking-[0.18em] text-cyan-100/70">
              <Sparkles className="h-4 w-4 text-cyan-200" />
              Hystersis control plane
            </p>
            <h1 className="max-w-[12ch] text-5xl font-semibold leading-[0.96] tracking-tight">
              Sign in to operate agent memory in production.
            </h1>
            <p className="max-w-lg text-base leading-7 text-white/68">
              Review memory writes, tune compression, manage API access, and monitor the
              retrieval layer your agents depend on.
            </p>
          </div>
          <ul className="grid gap-3">
            {features.map(({ icon: Icon, label }) => (
              <li key={label} className="flex items-center gap-3 rounded-lg border border-white/10 bg-white/[0.055] p-3 text-sm text-white/82 shadow-[inset_0_1px_0_rgba(255,255,255,0.08)] backdrop-blur">
                <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-md border border-white/10 bg-white/10 text-cyan-100">
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

      <div className="relative flex min-h-screen flex-col items-center justify-center overflow-hidden bg-[linear-gradient(180deg,#ffffff_0%,#f4f6fb_100%)] p-6 sm:p-10">
        <div className="absolute inset-x-0 top-0 h-40 bg-[radial-gradient(circle_at_50%_0%,rgba(99,102,241,0.12),transparent_52%)]" />
        <div className="absolute bottom-0 left-1/2 h-48 w-[34rem] -translate-x-1/2 rounded-full bg-cyan-100/35 blur-3xl" />
        <div className="mb-8 flex justify-center lg:hidden">
          <Link href={LANDING_URL} className="transition-opacity hover:opacity-90">
            <HystersisLogo wordmarkClassName="text-xl" />
          </Link>
        </div>
        <div className="relative z-10 w-full max-w-[456px]">{children}</div>
      </div>
    </div>
  );
}
