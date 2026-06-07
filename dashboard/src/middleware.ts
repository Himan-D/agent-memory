import { NextRequest, NextResponse } from "next/server";
import { getSessionCookie } from "better-auth/cookies";

export function middleware(request: NextRequest) {
  const { pathname } = request.nextUrl;
  const sessionCookie = getSessionCookie(request);
  const isAuthenticated = Boolean(sessionCookie);

  // DEMO_PAGE: Allow public access to /demo (compression playground without auth)
  if (pathname.startsWith("/demo")) {
    return NextResponse.next();
  }

  // AUTH_PAGES: Redirect signed-in users away from auth pages
  if (pathname.startsWith("/auth/")) {
    if (isAuthenticated && pathname !== "/auth/error") {
      return NextResponse.redirect(new URL("/", request.url));
    }
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
  if (!isAuthenticated) {
    return NextResponse.redirect(new URL("/auth/signin", request.url));
  }

  return NextResponse.next();
}

export const config = {
  matcher: [
    "/((?!api/auth|api/proxy|_next/static|_next/image|favicon.ico|icon|apple-icon|logo.svg|sitemap.xml|robots.txt).*)",
  ],
};
