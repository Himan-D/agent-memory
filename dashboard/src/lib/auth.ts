import { betterAuth } from "better-auth";
import { dash } from "@better-auth/infra";
import { nextCookies } from "better-auth/next-js";
import { credentials } from "better-auth-credentials-plugin";

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "https://api.hystersis.com";
const APP_BASE_URL = process.env.BETTER_AUTH_URL || "https://app.hystersis.com";

export const auth = betterAuth({
  database: undefined,
  // Use fallback during build time to prevent BetterAuthError when secret is missing.
  // In production runtime, BETTER_AUTH_SECRET must be set via environment variables.
  secret: process.env.BETTER_AUTH_SECRET || "build-time-fallback-secret-for-opennext",
  baseURL: APP_BASE_URL,
  trustedOrigins: [
    APP_BASE_URL,
    "https://app.hystersis.com",
    "http://localhost:3000",
  ].filter(Boolean) as string[],
  emailAndPassword: {
    enabled: false,
  },
  session: {
    cookieCache: {
      enabled: true,
      strategy: "jwt",
    },
  },
  user: {
    additionalFields: {
      token: {
        type: "string",
        returned: true,
        required: false,
      },
    },
  },
  plugins: [
    dash({
      apiKey: process.env.BETTER_AUTH_API_KEY,
    }),
    nextCookies(),
    credentials({
      autoSignUp: true,
      linkAccountIfExisting: true,
      providerId: "hystersis-backend",
      async callback(_ctx, parsed) {
        const { email, password } = parsed;

        const response = await fetch(`${API_BASE}/auth/login`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ email, password }),
        });

        if (!response.ok) {
          return null;
        }

        const data = await response.json();
        if (!data.success || !data.token) {
          return null;
        }

        return {
          email,
          name: data.user?.name || email.split("@")[0],
          token: data.token,
        };
      },
    }),
  ],
});

export type Session = typeof auth.$Infer.Session;
