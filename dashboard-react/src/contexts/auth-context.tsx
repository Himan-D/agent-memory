import { createContext, useContext } from "react";
import { authClient, signOutAndClear } from "@/lib/auth-client";
import type { ReactNode } from "react";

export interface User {
  id: string;
  name: string;
  email: string;
  avatar_url?: string;
  role: "admin" | "member" | "viewer";
}

interface AuthContextType {
  user: User | null;
  isLoading: boolean;
  logout: () => void;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export function AuthProvider({ children }: { children: ReactNode }) {
  const { data: session, isPending: isLoading } = authClient.useSession();
  const sessionUser = session?.user as any;

  const user = sessionUser ? {
    id: sessionUser.id,
    name: sessionUser.name,
    email: sessionUser.email,
    avatar_url: sessionUser.avatar_url || undefined,
    role: (sessionUser.role || "member") as "admin" | "member" | "viewer",
  } : null;

  const logout = async () => {
    await signOutAndClear();
  };

  return (
    <AuthContext.Provider value={{ user, isLoading, logout }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error("useAuth must be used within an AuthProvider");
  }
  return context;
}
