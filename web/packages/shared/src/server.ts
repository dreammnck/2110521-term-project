import { NextRequest, NextResponse } from "next/server";
import type { Role, User } from "./types";

const gateway = () => process.env.GATEWAY_URL ?? "http://localhost:8080";

/**
 * Server-side auth + proxy for a Next.js app. The JWT lives in an httpOnly cookie
 * and is attached to gateway calls here, so it is never exposed to browser JS.
 */
export function createHandlers(opts: { cookieName: string; role: Role }) {
  const { cookieName, role } = opts;

  async function login(req: NextRequest) {
    const body = await req.json().catch(() => null);
    if (!body?.email || !body?.password) {
      return NextResponse.json({ error: "Email and password are required" }, { status: 400 });
    }
    let res: Response;
    try {
      res = await fetch(`${gateway()}/api/v1/auth/login`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email: body.email, password: body.password }),
        cache: "no-store",
      });
    } catch {
      return NextResponse.json({ error: "Service unavailable, please try again" }, { status: 503 });
    }
    const data = (await res.json().catch(() => ({}))) as { token?: string; user?: User; error?: string };
    if (!res.ok || !data.token || !data.user) {
      return NextResponse.json({ error: data.error ?? "Login failed" }, { status: res.status === 200 ? 502 : res.status });
    }
    if (data.user.role !== role) {
      const need = role === "staff" ? "juristic staff" : "residents";
      return NextResponse.json({ error: `This app is for ${need} only` }, { status: 403 });
    }
    const out = NextResponse.json({ user: data.user });
    out.cookies.set(cookieName, data.token, {
      httpOnly: true,
      sameSite: "lax",
      path: "/",
      maxAge: 12 * 60 * 60,
      secure: process.env.COOKIE_SECURE === "true",
    });
    return out;
  }

  async function logout() {
    const out = NextResponse.json({ ok: true });
    out.cookies.delete(cookieName);
    return out;
  }

  async function proxy(req: NextRequest, ctx: { params: Promise<{ path: string[] }> }) {
    const token = req.cookies.get(cookieName)?.value;
    if (!token) return NextResponse.json({ error: "Not signed in" }, { status: 401 });
    const { path } = await ctx.params;
    const url = `${gateway()}/api/v1/${path.map(encodeURIComponent).join("/")}${req.nextUrl.search}`;

    const headers = new Headers({ Authorization: `Bearer ${token}` });
    const ct = req.headers.get("content-type");
    if (ct) headers.set("content-type", ct);
    const hasBody = req.method !== "GET" && req.method !== "HEAD";

    let res: Response;
    try {
      res = await fetch(url, {
        method: req.method,
        headers,
        body: hasBody ? req.body : undefined,
        // @ts-expect-error duplex is required by undici when streaming a request body
        duplex: "half",
        cache: "no-store",
      });
    } catch {
      return NextResponse.json({ error: "Service unavailable" }, { status: 503 });
    }
    const out = new Headers({ "Cache-Control": "no-store" });
    const rct = res.headers.get("content-type");
    if (rct) out.set("content-type", rct);
    return new Response(res.status === 204 ? null : res.body, { status: res.status, headers: out });
  }

  return { login, logout, proxy, cookieName };
}
