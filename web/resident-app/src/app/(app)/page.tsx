"use client";

import Link from "next/link";
import { useApi } from "@condo/shared/hooks";
import { Async, Badge, Card } from "@condo/shared/ui";
import { InstallPrompt } from "@/components/install-prompt";
import { baht, dateOnly, type Announcement, type Bill, type Parcel, type User } from "@condo/shared";

export default function Home() {
  const me = useApi<User>("users/me", { revalidateOnFocus: false });
  const parcels = useApi<Parcel[]>("parcels/me", { refreshInterval: 20000 });
  const bills = useApi<Bill[]>("utilities/bills/me", { refreshInterval: 30000 });
  const anns = useApi<Announcement[]>("announcements", { refreshInterval: 60000 });

  const waiting = parcels.data?.filter((p) => p.status === "WAITING").length;
  const open = bills.data?.filter((b) => b.status !== "PAID");
  const owed = open?.reduce((sum, b) => sum + b.total, 0);
  const overdue = bills.data?.some((b) => b.status === "OVERDUE");

  return (
    <div className="space-y-5">
      <InstallPrompt />
      <div>
        <h1 className="text-xl font-semibold">Hello{me.data ? `, ${me.data.name}` : ""}</h1>
        {me.data?.roomNumber && (
          <p className="text-sm text-slate-500">
            Room {me.data.roomNumber} · {me.data.building}
          </p>
        )}
      </div>

      <div className="grid grid-cols-2 gap-3">
        <Link href="/parcels">
          <Card className="h-full">
            <p className="text-sm text-slate-500">Parcels to collect</p>
            <p className={`mt-1 text-3xl font-semibold ${waiting ? "text-amber-600" : ""}`}>{waiting ?? "–"}</p>
          </Card>
        </Link>
        <Link href="/bills">
          <Card className="h-full">
            <p className="text-sm text-slate-500">Amount due</p>
            <p className={`mt-1 text-2xl font-semibold ${overdue ? "text-red-600" : ""}`}>{owed === undefined ? "–" : baht(owed)}</p>
            {overdue && <p className="text-xs text-red-600">Includes overdue bills</p>}
          </Card>
        </Link>
      </div>

      <section>
        <div className="mb-2 flex items-center justify-between">
          <h2 className="font-semibold">Latest announcements</h2>
          <Link href="/announcements" className="text-sm text-[var(--brand)]">
            See all
          </Link>
        </div>
        <Async data={anns.data?.slice(0, 3)} error={anns.error} empty="No announcements right now.">
          {(items) => (
            <ul className="space-y-2">
              {items.map((a) => (
                <li key={a.id}>
                  <Link href={`/announcements/${a.id}`}>
                    <Card className="!p-3">
                      <div className="flex items-center justify-between gap-2">
                        <p className="font-medium">{a.title}</p>
                        {!a.read && <Badge tone="blue">New</Badge>}
                      </div>
                      <p className="text-xs text-slate-500">{dateOnly(a.publishAt)}</p>
                    </Card>
                  </Link>
                </li>
              ))}
            </ul>
          )}
        </Async>
      </section>
    </div>
  );
}
