import { betterAuth } from "better-auth";
import { dash } from "@better-auth/infra";
import { nextCookies } from "better-auth/next-js";
import { credentials } from "better-auth-credentials-plugin";

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "https://api.hystersis.com";
const APP_BASE_URL = process.env.BETTER_AUTH_URL || "https://app.hystersis.com";
const SECRET = process.env.BETTER_AUTH_SECRET || "fallback_secret_for_build_only_replace_in_prod_123";

export const auth = betterAuth({
  database: undefined,
  secret: SECRET,
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
      role: {
        type: "string",
        returned: true,
        required: false,
      },
      avatar_url: {
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
          image: data.user?.image || data.user?.avatar_url,
          avatar_url: data.user?.avatar_url || data.user?.image,
          token: data.token,
          role: data.user?.role || "user",
        };
      },
    }),
  ],
});

export type Session = typeof auth.$Infer.Session;
