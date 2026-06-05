import { NextResponse } from "next/server";
import { auth } from "@/lib/auth";

export default auth((req) => {
  const { pathname } = req.nextUrl;

  // DEMO_PAGE: Allow public access to /demo (compression playground without auth)
  if (pathname.startsWith("/demo")) {
    return NextResponse.next();
  }

  // AUTH_PAGES: Allow auth pages without redirect
  if (pathname.startsWith("/auth/")) {
    return NextResponse.next();
  }

  // API_PROXY: Allow API proxy for frontend-backend communication
  if (pathname.startsWith("/api/proxy")) {
    return NextResponse.next();
  }

  // STATIC: Allow static files
  if (pathname.startsWith("/_next")) {
    return NextResponse.next();
  }

  // SEO: Allow sitemap and robots.txt without auth
  if (pathname === "/sitemap.xml" || pathname === "/robots.txt") {
    return NextResponse.next();
  }

  // AUTH_REQUIRED: Redirect unauthenticated users to sign in for all other routes
  if (!req.auth) {
    const url = new URL("/auth/signin", req.url);
    return NextResponse.redirect(url);
  }

  return NextResponse.next();
});

export const config = {
  matcher: [
    // Match all routes except static files and API routes
    "/((?!api/auth|api/proxy|_next/static|_next/image|favicon.ico|sitemap.xml|robots.txt).*)",
  ],
};