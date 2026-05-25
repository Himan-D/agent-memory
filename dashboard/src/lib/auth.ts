import NextAuth from "next-auth";
import Credentials from "next-auth/providers/credentials";
import { setApiKey, userApiKeysApi } from "./api";

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

export const { handlers, signIn, signOut, auth } = NextAuth({
  providers: [
    Credentials({
      name: "credentials",
      credentials: {
        email: { label: "Email", type: "email", placeholder: "demo@example.com" },
        password: { label: "Password", type: "password" },
      },
      async authorize(credentials) {
        if (!credentials?.email || !credentials?.password) {
          return null;
        }

        const email = credentials.email as string;
        const password = credentials.password as string;

        if (email === "demo@hystersis.ai" && password === "demo123") {
          return {
            id: "demo-1",
            name: "Demo User",
            email: "demo@hystersis.ai",
          };
        }

        try {
          const response = await fetch(`${API_BASE}/auth/login`, {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ email, password }),
          });

          if (response.ok) {
            const user = await response.json();
            return {
              id: user.id || email,
              name: user.name || email.split("@")[0],
              email: email,
            };
          }
        } catch (e) {
          console.log("Backend auth check failed:", e);
        }

        return null;
      },
    }),
  ],
  pages: {
    signIn: "/auth/signin",
    error: "/auth/error",
  },
  callbacks: {
    async jwt({ token, user }) {
      if (user) {
        token.id = user.id;
        token.email = user.email;
      }
      return token;
    },
    async session({ session, token }) {
      if (session.user) {
        session.user.id = token.id as string;
        session.user.email = token.email as string;
      }
      return session;
    },
  },
  session: {
    strategy: "jwt",
  },
  secret: process.env.NEXTAUTH_SECRET,
  trustHost: true,
  events: {
    async signIn({ user }) {
      try {
        const result = await userApiKeysApi.create({
          label: `${user.email} API Key`,
          scope: "write",
        });
        if (result.key) {
          setApiKey(result.key);
        }
      } catch (e) {
        console.log("API key creation skipped:", e);
      }
    },
  },
});