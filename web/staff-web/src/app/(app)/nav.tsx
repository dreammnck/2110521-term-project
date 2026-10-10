"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { logout } from "@condo/shared/hooks";
import { Button } from "@condo/shared/ui";

const links = [
  { href: "/", label: "Dashboard" },
  { href: "/parcels", label: "Parcels" },
  { href: "/bills", label: "Utility bills" },
  { href: "/announcements", label: "Announcements" },
];

export function Nav() {
  const path = usePathname();
  return (
    <aside className="border-b border-slate-200 bg-white md:min-h-dvh md:w-56 md:border-b-0 md:border-r">
      <div className="flex items-center justify-between px-4 py-3 md:block md:py-5">
        <div className="flex items-center gap-2">
          <div className="grid size-8 place-items-center rounded-lg bg-[var(--brand)] font-bold text-white">C</div>
          <span className="font-semibold">Condo Staff</span>
        </div>
        <Button variant="ghost" className="md:mt-6 md:w-full md:justify-start" onClick={logout}>
          Sign out
        </Button>
      </div>
      <nav className="flex gap-1 overflow-x-auto px-2 pb-2 md:flex-col md:px-3">
        {links.map((l) => {
          const active = l.href === "/" ? path === "/" : path.startsWith(l.href);
          return (
            <Link
              key={l.href}
              href={l.href}
              aria-current={active ? "page" : undefined}
              className={`whitespace-nowrap rounded-lg px-3 py-2 text-sm font-medium ${
                active ? "bg-[var(--brand)] text-white" : "text-slate-700 hover:bg-slate-100"
              }`}
            >
              {l.label}
            </Link>
          );
        })}
      </nav>
    </aside>
  );
}
