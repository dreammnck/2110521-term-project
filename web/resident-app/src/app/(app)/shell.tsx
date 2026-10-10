"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { logout, useApi } from "@condo/shared/hooks";
import type { Notification } from "@condo/shared";
import { Icon, type IconName } from "@/components/icons";
import { PullToRefresh } from "@/components/ptr";
import { ToastProvider } from "@/components/toast";

const tabs: { href: string; label: string; icon: IconName }[] = [
  { href: "/", label: "Home", icon: "home" },
  { href: "/parcels", label: "Parcels", icon: "box" },
  { href: "/bills", label: "Bills", icon: "bill" },
  { href: "/announcements", label: "News", icon: "news" },
  { href: "/notifications", label: "Alerts", icon: "bell" },
];

const titles: [RegExp, string][] = [
  [/^\/bills\/return/, "Payment"],
  [/\/receipt$/, "Receipt"],
  [/^\/bills\/[^/]+$/, "Bill"],
  [/^\/announcements\/[^/]+$/, "Announcement"],
];

export function useUnreadCount() {
  const { data } = useApi<Notification[]>("notifications", { refreshInterval: 15000 });
  return data?.filter((n) => !n.readAt).length ?? 0;
}

export function Shell({ children }: { children: React.ReactNode }) {
  const path = usePathname();
  const router = useRouter();
  const unread = useUnreadCount();

  // Detail screens behave like pushed native screens: back button, no tab bar.
  const nested = path.split("/").filter(Boolean).length > 1;
  const title = titles.find(([re]) => re.test(path))?.[1] ?? "";
  const goBack = () => {
    if (window.history.length > 1) router.back();
    else router.push(path.replace(/\/[^/]+$/, "") || "/");
  };

  return (
    <ToastProvider>
      <div className="relative mx-auto min-h-dvh max-w-md bg-slate-50 md:shadow-2xl">
        <header className="sticky top-0 z-30 border-b border-slate-200 bg-white/95 pt-[env(safe-area-inset-top)] backdrop-blur">
          <div className="flex h-14 items-center justify-between px-2">
            {nested ? (
              <>
                <button onClick={goBack} aria-label="Back" className="flex size-11 items-center justify-center rounded-full text-slate-700 active:bg-slate-100">
                  <Icon name="back" />
                </button>
                <h1 className="text-base font-semibold">{title}</h1>
                <div className="size-11" />
              </>
            ) : (
              <>
                <div className="flex items-center gap-2 pl-2">
                  <div className="grid size-8 place-items-center rounded-lg bg-[var(--brand)] font-bold text-white">C</div>
                  <span className="font-semibold">Condo</span>
                </div>
                <div className="flex items-center">
                  <Link href="/notifications" aria-label={`Notifications, ${unread} unread`} className="relative flex size-11 items-center justify-center rounded-full text-slate-700 active:bg-slate-100">
                    <Icon name="bell" />
                    {unread > 0 && <span className="absolute right-2 top-2 size-2.5 rounded-full bg-red-600 ring-2 ring-white" />}
                  </Link>
                  <button onClick={logout} aria-label="Sign out" className="flex size-11 items-center justify-center rounded-full text-slate-700 active:bg-slate-100">
                    <Icon name="logout" className="size-5" />
                  </button>
                </div>
              </>
            )}
          </div>
        </header>

        <PullToRefresh>
          <main key={path} className={`page-enter px-4 pt-5 ${nested ? "pb-32" : "pb-[calc(6rem+env(safe-area-inset-bottom))]"}`}>
            {children}
          </main>
        </PullToRefresh>

        {!nested && (
          <nav className="no-print fixed bottom-0 left-1/2 z-30 w-full max-w-md -translate-x-1/2 border-t border-slate-200 bg-white/95 pb-[env(safe-area-inset-bottom)] backdrop-blur">
            <ul className="grid grid-cols-5">
              {tabs.map((t) => {
                const active = t.href === "/" ? path === "/" : path.startsWith(t.href);
                return (
                  <li key={t.href}>
                    <Link
                      href={t.href}
                      aria-current={active ? "page" : undefined}
                      className={`relative flex h-16 flex-col items-center justify-center gap-0.5 text-[11px] font-medium ${active ? "text-[var(--brand)]" : "text-slate-500"}`}
                    >
                      {active && <span className="absolute inset-x-5 top-0 h-0.5 rounded-b bg-[var(--brand)]" />}
                      <span className="relative">
                        <Icon name={t.icon} className="size-6" />
                        {t.href === "/notifications" && unread > 0 && (
                          <span className="absolute -right-2 -top-1 grid min-w-4 place-items-center rounded-full bg-red-600 px-1 text-[10px] font-semibold leading-4 text-white">
                            {unread > 9 ? "9+" : unread}
                          </span>
                        )}
                      </span>
                      {t.label}
                    </Link>
                  </li>
                );
              })}
            </ul>
          </nav>
        )}
      </div>
    </ToastProvider>
  );
}
