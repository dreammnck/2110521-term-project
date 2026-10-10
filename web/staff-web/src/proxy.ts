import { NextRequest, NextResponse } from "next/server";

// Route guard: pages require the staff session cookie. The gateway still enforces roles.
export function proxy(req: NextRequest) {
  const signedIn = req.cookies.has("condo_staff_token");
  const onLogin = req.nextUrl.pathname === "/login";
  if (!signedIn && !onLogin) return NextResponse.redirect(new URL("/login", req.url));
  if (signedIn && onLogin) return NextResponse.redirect(new URL("/", req.url));
  return NextResponse.next();
}

export const config = { matcher: ["/((?!api|_next|favicon.ico).*)"] };
