import { createAuthClient } from "better-auth/react";
import { dashClient } from "@better-auth/infra/client";
import {
  credentialsClient,
  defaultCredentialsSchema,
} from "better-auth-credentials-plugin/client";
import type { auth } from "./auth";
import { clearSessionToken } from "./api";

export const authClient = createAuthClient({
  baseURL: process.env.NEXT_PUBLIC_BETTER_AUTH_URL || process.env.NEXT_PUBLIC_APP_URL || "http://localhost:3000",
  plugins: [
    dashClient(),
    credentialsClient<
      typeof auth.$Infer.Session.user,
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
