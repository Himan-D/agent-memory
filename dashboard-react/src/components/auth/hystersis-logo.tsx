import { cn } from "@/lib/utils";

interface HystersisLogoProps {
  className?: string;
  iconClassName?: string;
  showWordmark?: boolean;
  wordmarkClassName?: string;
}

export function HystersisLogo({
  className,
  iconClassName = "h-9 w-9",
  showWordmark = true,
  wordmarkClassName,
}: HystersisLogoProps) {
  return (
    <div className={cn("flex items-center gap-3", className)}>
      <div className="relative flex h-11 w-11 shrink-0 items-center justify-center rounded-xl border border-zinc-200/80 bg-white p-2 shadow-[0_12px_32px_rgba(15,23,42,0.12)] dark:border-white/10 dark:bg-white/95">
        <span className="absolute inset-0 rounded-xl bg-[radial-gradient(circle_at_35%_25%,rgba(99,102,241,0.22),transparent_48%)]" />
        <img
          src="/logo.svg"
          alt=""
          width={28}
          height={28}
          className={cn("relative h-7 w-7", iconClassName)}
        />
      </div>
      {showWordmark && (
        <span className={cn("font-semibold tracking-tight", wordmarkClassName)}>
          Hystersis
        </span>
      )}
    </div>
  );
}
