import { auth } from "@/auth";
import { NextResponse } from "next/server";

const PUBLIC_PATHS = ["/login", "/denied", "/api/auth"];

export default auth((req) => {
  const { pathname } = req.nextUrl;
  const isPublic = PUBLIC_PATHS.some((p) => pathname.startsWith(p));
  const isAuthed = !!req.auth;

  if (!isAuthed && !isPublic) {
    return NextResponse.redirect(new URL("/login", req.url));
  }

  if (isAuthed && pathname === "/login") {
    return NextResponse.redirect(new URL("/watchlist", req.url));
  }

  return NextResponse.next();
});

export const config = {
  matcher: ["/((?!_next/static|_next/image|favicon.ico|.*\\.svg$).*)"],
};
