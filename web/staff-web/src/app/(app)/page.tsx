"use client";

import Link from "next/link";
import { useApi, useRooms } from "@condo/shared/hooks";
import { Async, Card, PageHeader, StatusBadge } from "@condo/shared/ui";
import { baht, cycleLabel, dateOnly, dateTime, type Announcement, type Bill, type Parcel } from "@condo/shared";

function Stat({ label, value, href, tone }: { label: string; value: number | string; href: string; tone?: string }) {
  return (
    <Link href={href} className="block">
      <Card className="transition hover:border-[var(--brand)]">
        <p className="text-sm text-slate-500">{label}</p>
        <p className={`mt-1 text-3xl font-semibold ${tone ?? ""}`}>{value}</p>
      </Card>
    </Link>
  );
}

export default function Dashboard() {
  const parcels = useApi<Parcel[]>("parcels?status=WAITING", { refreshInterval: 30000 });
  const bills = useApi<Bill[]>("utilities/bills", { refreshInterval: 30000 });
  const anns = useApi<Announcement[]>("announcements");
  const { label } = useRooms();

  const openBills = bills.data?.filter((b) => b.status !== "PAID");
  const overdue = bills.data?.filter((b) => b.status === "OVERDUE").length;
  const active = anns.data?.filter((a) => a.status === "PUBLISHED").length;

  return (
    <>
      <PageHeader title="Dashboard" subtitle="Items that need follow-up" />
      <div className="mb-6 grid grid-cols-2 gap-3 md:grid-cols-4">
        <Stat label="Parcels awaiting pickup" value={parcels.data?.length ?? "–"} href="/parcels" tone="text-amber-600" />
        <Stat label="Unsettled bills" value={openBills?.length ?? "–"} href="/bills" />
        <Stat label="Overdue bills" value={overdue ?? "–"} href="/bills" tone={overdue ? "text-red-600" : ""} />
        <Stat label="Active announcements" value={active ?? "–"} href="/announcements" />
      </div>

      <div className="grid gap-6 lg:grid-cols-2">
        <section>
          <h2 className="mb-2 font-semibold">Parcels waiting longest</h2>
          <Async data={parcels.data && [...parcels.data].reverse().slice(0, 6)} error={parcels.error} empty="No parcels are waiting 🎉">
            {(items) => (
              <ul className="space-y-2">
                {items.map((p) => (
                  <li key={p.id}>
                    <Card className="flex items-center justify-between gap-3 !p-3">
                      <div>
                        <p className="font-medium">{label(p.roomId)}</p>
                        <p className="text-xs text-slate-500">
                          {p.carrier || "Parcel"} · received {dateTime(p.receivedAt)}
                        </p>
                      </div>
                      <StatusBadge status={p.status} />
                    </Card>
                  </li>
                ))}
              </ul>
            )}
          </Async>
        </section>

        <section>
          <h2 className="mb-2 font-semibold">Bills needing attention</h2>
          <Async
            data={openBills && [...openBills].sort((a, b) => a.dueDate.localeCompare(b.dueDate)).slice(0, 6)}
            error={bills.error}
            empty="All bills are settled"
          >
            {(items) => (
              <ul className="space-y-2">
                {items.map((b) => (
                  <li key={b.id}>
                    <Card className="flex items-center justify-between gap-3 !p-3">
                      <div>
                        <p className="font-medium">
                          {label(b.roomId)} · {cycleLabel(b.cycle)}
                        </p>
                        <p className="text-xs text-slate-500">
                          {baht(b.total)} · due {dateOnly(b.dueDate)}
                        </p>
                      </div>
                      <StatusBadge status={b.status} />
                    </Card>
                  </li>
                ))}
              </ul>
            )}
          </Async>
        </section>
      </div>
    </>
  );
}
