import { useEffect } from "react";
import { useLocation } from "react-router-dom";
import { authClient } from "@/lib/auth-client";
import { setSessionToken, clearSessionToken } from "@/lib/api";

export function AuthSessionSync() {
  const location = useLocation();
  const pathname = location.pathname;
  const isPublicPage = pathname?.startsWith("/auth") || pathname?.startsWith("/demo");

  // On public pages there's no session to sync — skip the hook entirely
  // to avoid errors that could break hydration.
  const { data: session } = authClient.useSession();

  useEffect(() => {
    if (isPublicPage) return;
    // @ts-expect-error token exists in our custom schema but better-auth client inference might omit it
    const token = session?.user?.token;
    if (token) {
      setSessionToken(token);
    }
  }, [(session?.user as any)?.token, isPublicPage]);

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

