import NextAuth from "next-auth";
import Credentials from "next-auth/providers/credentials";
import { setApiKey, userApiKeysApi } from "./api";

export const { handlers, signIn, signOut, auth } = NextAuth({
  providers: [
    Credentials({
      name: "Demo",
      credentials: {
        email: { label: "Email", type: "email", placeholder: "demo@example.com" },
        password: { label: "Password", type: "password" },
      },
      async authorize(credentials) {
        if (
          credentials?.email === "demo@hystersis.ai" &&
          credentials?.password === "demo123"
        ) {
          return {
            id: "1",
            name: "Demo User",
            email: "demo@hystersis.ai",
          };
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
      }
      return token;
    },
    async session({ session, token }) {
      if (session.user) {
        session.user.id = token.id as string;
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
      if (user?.email === "demo@hystersis.ai") {
        try {
          const result = await userApiKeysApi.create({
            label: "Demo User Key",
            scope: "write",
          });
          if (result.key) {
            setApiKey(result.key);
          }
        } catch (e) {
          console.log("API key creation skipped:", e);
        }
      }
    },
  },
});