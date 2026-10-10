"use client";

import { useMemo } from "react";
import useSWR, { type SWRConfiguration } from "swr";
import type { Room } from "./types";

export class ApiError extends Error {
  constructor(message: string, public status: number) {
    super(message);
  }
}

/** Calls the gateway through the app's same-origin proxy (/api/gw). */
export async function api<T = unknown>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers);
  if (init.body && !(init.body instanceof FormData) && !headers.has("content-type")) {
    headers.set("content-type", "application/json");
  }
  const res = await fetch(`/api/gw/${path.replace(/^\//, "")}`, { ...init, headers });
  if (res.status === 401 && typeof window !== "undefined" && !window.location.pathname.startsWith("/login")) {
    // Clear the (expired) cookie first, otherwise the route guard would bounce us back here.
    await fetch("/api/logout", { method: "POST" }).catch(() => undefined);
    window.location.href = "/login";
  }
  if (!res.ok) {
    const data = await res.json().catch(() => ({}));
    throw new ApiError(data.error ?? res.statusText, res.status);
  }
  if (res.status === 204) return undefined as T;
  return res.json() as Promise<T>;
}

export const post = <T = unknown>(path: string, body?: unknown) =>
  api<T>(path, { method: "POST", body: body === undefined ? undefined : JSON.stringify(body) });

export const patch = <T = unknown>(path: string, body: unknown) =>
  api<T>(path, { method: "PATCH", body: JSON.stringify(body) });

export function useApi<T>(path: string | null, config?: SWRConfiguration<T, ApiError>) {
  return useSWR<T, ApiError>(path, (p: string) => api<T>(p), { revalidateOnFocus: true, ...config });
}

export async function logout(): Promise<void> {
  await fetch("/api/logout", { method: "POST" });
  window.location.href = "/login";
}

/** Staff-only: rooms plus lookup helpers (the gateway restricts GET /rooms to staff). */
export function useRooms() {
  const { data: rooms, error } = useApi<Room[]>("rooms", { revalidateOnFocus: false });
  const byId = useMemo(() => new Map((rooms ?? []).map((r) => [r.id, r])), [rooms]);
  const label = (id: string) => {
    const r = byId.get(id);
    return r ? `${r.number} · ${r.building}` : id.slice(0, 8);
  };
  const buildings = useMemo(() => {
    const m = new Map<string, string>();
    (rooms ?? []).forEach((r) => m.set(r.buildingId, r.building));
    return [...m].map(([id, name]) => ({ id, name }));
  }, [rooms]);
  return { rooms, error, byId, label, buildings };
}
