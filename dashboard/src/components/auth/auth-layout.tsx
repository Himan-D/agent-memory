import Link from "next/link";
import { Brain, Network, Zap } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { HystersisLogo } from "@/components/auth/hystersis-logo";

const LANDING_URL = process.env.NEXT_PUBLIC_LANDING_URL || "https://hystersis.com";

const features = [
  { icon: Brain, label: "ProMem extraction with 97%+ accuracy" },
  { icon: Network, label: "Graph memory with spreading activation" },
  { icon: Zap, label: "85% token compression for agent context" },
];

interface AuthLayoutProps {
  children: React.ReactNode;
}

export function AuthLayout({ children }: AuthLayoutProps) {
  return (
    <div className="min-h-screen lg:grid lg:grid-cols-2">
      {/* Brand panel */}
      <div className="relative hidden flex-col justify-between overflow-hidden bg-gradient-to-br from-indigo-600 via-violet-600 to-purple-700 p-10 text-white lg:flex">
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_20%_20%,rgba(255,255,255,0.15),transparent_40%),radial-gradient(circle_at_80%_80%,rgba(255,255,255,0.1),transparent_35%)]" />
        <div className="relative z-10">
          <Link href={LANDING_URL} className="inline-block transition-opacity hover:opacity-90">
            <HystersisLogo
              showWordmark
              wordmarkClassName="text-2xl text-white"
              iconClassName="h-8 w-8"
            />
          </Link>
          <Badge variant="secondary" className="mt-4 bg-white/15 text-white hover:bg-white/20">
            Agent Memory Dashboard
          </Badge>
        </div>

        <div className="relative z-10 space-y-8">
          <div className="space-y-3">
            <h1 className="text-3xl font-semibold leading-tight tracking-tight">
              Memory that agents
              <br />
              actually remember.
            </h1>
            <p className="max-w-md text-base text-white/80">
              Manage memories, entities, skills, and compression settings from one
              place — built for production AI agents.
            </p>
          </div>
          <ul className="space-y-3">
            {features.map(({ icon: Icon, label }) => (
              <li key={label} className="flex items-center gap-3 text-sm text-white/90">
                <span className="flex h-8 w-8 items-center justify-center rounded-lg bg-white/15">
                  <Icon className="h-4 w-4" />
                </span>
                {label}
              </li>
            ))}
          </ul>
        </div>

        <p className="relative z-10 text-sm text-white/60">
          © {new Date().getFullYear()} Hystersis
        </p>
      </div>

      {/* Form panel */}
      <div className="flex min-h-screen flex-col items-center justify-center bg-gradient-to-b from-background via-background to-muted/30 p-6 sm:p-10">
        <div className="w-full max-w-md">{children}</div>
      </div>
    </div>
  );
}
