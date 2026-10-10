"use client";

import Link from "next/link";
import { useState } from "react";
import { useApi } from "@condo/shared/hooks";
import { Async, Card, StatusBadge, Tabs } from "@condo/shared/ui";
import { baht, cycleLabel, dateOnly, type Bill } from "@condo/shared";

export default function BillsPage() {
  const [tab, setTab] = useState<"open" | "history">("open");
  const { data, error } = useApi<Bill[]>("utilities/bills/me", { refreshInterval: 30000 });
  const items = data?.filter((b) => (tab === "open" ? b.status !== "PAID" : b.status === "PAID"));

  return (
    <div>
      <h1 className="mb-4 text-xl font-semibold">Utility bills</h1>
      <Tabs
        value={tab}
        onChange={setTab}
        options={[
          { value: "open", label: "To pay" },
          { value: "history", label: "History" },
        ]}
      />
      <Async data={items} error={error} empty={tab === "open" ? "You have no bills to pay. 🎉" : "No payment history yet."}>
        {(list) => (
          <ul className="space-y-3">
            {list.map((b) => (
              <li key={b.id}>
                <Link href={`/bills/${b.id}`}>
                  <Card className="flex items-center justify-between gap-3 transition hover:border-[var(--brand)]">
                    <div>
                      <p className="font-semibold">{cycleLabel(b.cycle)}</p>
                      <p className="text-xs text-slate-500">Due {dateOnly(b.dueDate)}</p>
                    </div>
                    <div className="text-right">
                      <p className="font-semibold">{baht(b.total)}</p>
                      <StatusBadge status={b.status} />
                    </div>
                  </Card>
                </Link>
              </li>
            ))}
          </ul>
        )}
      </Async>
    </div>
  );
}
