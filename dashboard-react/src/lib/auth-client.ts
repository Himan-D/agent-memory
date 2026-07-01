import { createAuthClient } from "better-auth/react";
import { dashClient } from "@better-auth/infra/client";
import {
  credentialsClient,
  defaultCredentialsSchema,
} from "better-auth-credentials-plugin/client";
import { clearSessionToken } from "./api";

// We define a generic user for the client since we don't have the backend auth.ts
interface AuthUser {
  id: string;
  name: string;
  email: string;
  role: string;
  avatar_url?: string;
  token?: string;
  createdAt: Date;
  updatedAt: Date;
  emailVerified: boolean;
  image?: string | null;
}

export const authClient = createAuthClient({
  baseURL: import.meta.env.VITE_BETTER_AUTH_URL ?? "https://app.hystersis.com",
  plugins: [
    // @ts-expect-error dashClient types might mismatch in Vite env without backend auth type
    dashClient(),
    credentialsClient<
      AuthUser,
      "/sign-in/credentials",
      typeof defaultCredentialsSchema
    >(),
  ],
});

export async function signOutAndClear() {
  clearSessionToken();
  await authClient.signOut();
  window.location.href = "/auth/signin";
}
