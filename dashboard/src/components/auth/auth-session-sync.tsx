"use client";

import { useEffect } from "react";
import { authClient } from "@/lib/auth-client";
import { setSessionToken, clearSessionToken } from "@/lib/api";

export function AuthSessionSync() {
  const { data: session } = authClient.useSession();

  useEffect(() => {
    const token = session?.user?.token;
    if (token) {
      setSessionToken(token);
    }
  }, [session?.user?.token]);

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
