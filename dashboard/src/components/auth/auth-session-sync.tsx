"use client";

import { useEffect } from "react";
import { usePathname } from "next/navigation";
import { authClient } from "@/lib/auth-client";
import { setSessionToken, clearSessionToken } from "@/lib/api";

export function AuthSessionSync() {
  const pathname = usePathname();
  const isPublicPage = pathname?.startsWith("/auth") || pathname?.startsWith("/demo");

  // On public pages there's no session to sync — skip the hook entirely
  // to avoid errors that could break hydration.
  const { data: session } = authClient.useSession();

  useEffect(() => {
    if (isPublicPage) return;
    const token = session?.user?.token;
    if (token) {
      setSessionToken(token);
    }
  }, [session?.user?.token, isPublicPage]);

  useEffect(() => {
    const handleStorage = (event: StorageEvent) => {
      if (event.key === "better-auth.session" && !event.newValue) {
        clearSessionToken();
      }
    };

    window.addEventListener("storage", handleStorage);
    return () => window.removeEventListener("storage", handleStorage);
  }, []);

  return null;
}

