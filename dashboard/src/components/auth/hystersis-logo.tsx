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
      <div className="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl bg-gradient-to-br from-indigo-500 to-violet-600 p-2 shadow-lg shadow-indigo-500/20">
        <svg
          viewBox="0 0 128 128"
          className={cn("h-7 w-7 text-white", iconClassName)}
          fill="none"
          xmlns="http://www.w3.org/2000/svg"
          aria-hidden
        >
          <circle cx="64" cy="64" r="12" fill="currentColor" />
          <circle cx="32" cy="40" r="7" fill="currentColor" opacity="0.9" />
          <circle cx="96" cy="40" r="7" fill="currentColor" opacity="0.9" />
          <circle cx="32" cy="88" r="7" fill="currentColor" opacity="0.9" />
          <circle cx="96" cy="88" r="7" fill="currentColor" opacity="0.9" />
          <circle cx="64" cy="24" r="5" fill="currentColor" opacity="0.7" />
          <circle cx="64" cy="104" r="5" fill="currentColor" opacity="0.7" />
          <line x1="64" y1="52" x2="32" y2="40" stroke="currentColor" strokeWidth="2.5" opacity="0.5" />
          <line x1="64" y1="52" x2="96" y2="40" stroke="currentColor" strokeWidth="2.5" opacity="0.5" />
          <line x1="64" y1="52" x2="32" y2="88" stroke="currentColor" strokeWidth="2.5" opacity="0.5" />
          <line x1="64" y1="52" x2="96" y2="88" stroke="currentColor" strokeWidth="2.5" opacity="0.5" />
          <line x1="64" y1="52" x2="64" y2="24" stroke="currentColor" strokeWidth="2" opacity="0.35" />
          <line x1="64" y1="76" x2="64" y2="104" stroke="currentColor" strokeWidth="2" opacity="0.35" />
        </svg>
      </div>
      {showWordmark && (
        <span className={cn("font-semibold tracking-tight", wordmarkClassName)}>
          Hystersis
        </span>
      )}
    </div>
  );
}
