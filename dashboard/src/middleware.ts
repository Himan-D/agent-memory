import { NextResponse } from "next/server";
import { auth } from "@/lib/auth";

export default auth((req) => {
  const { pathname } = req.nextUrl;
  
  // Allow auth pages without redirect
  if (pathname.startsWith("/auth/")) {
    return NextResponse.next();
  }
  
  // Allow API proxy endpoint for frontend-backend communication
  if (pathname.startsWith("/api/proxy")) {
    return NextResponse.next();
  }
  
  // Redirect unauthenticated users to sign in
  if (!req.auth) {
    const url = new URL("/auth/signin", req.url);
    return NextResponse.redirect(url);
  }
  
  return NextResponse.next();
});

export const config = {
  matcher: [
    "/((?!api/auth|api/proxy|_next/static|_next/image|favicon.ico).*)",
  ],
};