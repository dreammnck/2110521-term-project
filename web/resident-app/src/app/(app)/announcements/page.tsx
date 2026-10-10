"use client";

import Link from "next/link";
import { useApi } from "@condo/shared/hooks";
import { Async, Badge, Card } from "@condo/shared/ui";
import { dateOnly, type Announcement } from "@condo/shared";

export default function AnnouncementsPage() {
  const { data, error } = useApi<Announcement[]>("announcements", { refreshInterval: 60000 });
  return (
    <div>
      <h1 className="mb-4 text-xl font-semibold">Announcements</h1>
      <Async data={data} error={error} empty="No announcements right now.">
        {(items) => (
          <ul className="space-y-3">
            {items.map((a) => (
              <li key={a.id}>
                <Link href={`/announcements/${a.id}`}>
                  <Card className="transition hover:border-[var(--brand)]">
                    <div className="flex items-start justify-between gap-2">
                      <h2 className={a.read ? "font-medium" : "font-semibold"}>{a.title}</h2>
                      {!a.read && <Badge tone="blue">New</Badge>}
                    </div>
                    <p className="mt-1 line-clamp-2 text-sm text-slate-600">{a.body}</p>
                    <p className="mt-2 text-xs text-slate-500">
                      {a.category} · {dateOnly(a.publishAt)}
                    </p>
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
