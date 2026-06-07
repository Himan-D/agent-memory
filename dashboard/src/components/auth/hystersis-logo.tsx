import Image from "next/image";
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
        <Image
          src="/logo.svg"
          alt=""
          width={28}
          height={28}
          className={cn("h-7 w-7", iconClassName)}
          priority
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
