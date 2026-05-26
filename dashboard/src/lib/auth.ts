import NextAuth, { DefaultSession } from "next-auth";
import Credentials from "next-auth/providers/credentials";
import { setSessionToken, clearSessionToken } from "./api";

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "https://api.hystersis.ai";

declare module "next-auth" {
  interface Session {
    user: {
      id?: string;
      name: string;
      email: string;
      token?: string;
    } & DefaultSession["user"];
  }
}

// JWT token extension — token/id/role fields are set in authorize() callback
// and accessed in the jwt/session callbacks below.
// Using inline type assertions instead of module augmentation to avoid
// build issues with next-auth v5 beta module resolution.

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

        try {
          const response = await fetch(`${API_BASE}/auth/login`, {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ email, password }),
          });

          if (response.ok) {
            const data = await response.json();
            if (data.success && data.token) {
              // Store token in localStorage for API calls
              setSessionToken(data.token);
              
              return {
                id: data.user?.id || email,
                name: data.user?.name || email.split("@")[0],
                email: email,
                token: data.token, // Store backend session token
              };
            }
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
        token.token = (user as { token?: string }).token;
      }
      return token;
    },
    async session({ session, token }) {
      if (session.user) {
        session.user.id = token.id as string;
        session.user.email = token.email as string;
        session.user.token = token.token as string | undefined;
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
    async signOut() {
      clearSessionToken();
    },
  },
});