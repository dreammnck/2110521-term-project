"use client";

import { useRouter } from "next/navigation";
import { mutate } from "swr";
import { patch, post, useApi } from "@condo/shared/hooks";
import { Async, Button, Card } from "@condo/shared/ui";
import { dateTime, type Notification } from "@condo/shared";

const hrefFor = (n: Notification) => {
  switch (n.refType) {
    case "parcel":
      return "/parcels";
    case "announcement":
      return `/announcements/${n.refId}`;
    case "bill":
      return `/bills/${n.refId}`;
  }
  return "/";
};

export default function NotificationsPage() {
  const router = useRouter();
  const { data, error } = useApi<Notification[]>("notifications", { refreshInterval: 15000 });
  const unread = data?.filter((n) => !n.readAt).length ?? 0;

  return (
    <div>
      <div className="mb-4 flex items-center justify-between">
        <h1 className="text-xl font-semibold">Notifications</h1>
        {unread > 0 && (
          <Button
            variant="ghost"
            onClick={async () => {
              await post("notifications/read-all");
              mutate("notifications");
            }}
          >
            Mark all read
          </Button>
        )}
      </div>
      <Async data={data} error={error} empty="You're all caught up.">
        {(items) => (
          <ul className="space-y-2">
            {items.map((n) => (
              <li key={n.id}>
                <button
                  className="w-full text-left"
                  onClick={async () => {
                    if (!n.readAt) await patch(`notifications/${n.id}/read`, {}).catch(() => undefined);
                    mutate("notifications");
                    router.push(hrefFor(n));
                  }}
                >
                  <Card className={`!p-3 ${n.readAt ? "" : "border-[var(--brand)]/40 bg-rose-50/40"}`}>
                    <div className="flex items-start justify-between gap-2">
                      <p className={n.readAt ? "font-medium" : "font-semibold"}>{n.title}</p>
                      {!n.readAt && <span className="mt-1.5 size-2 shrink-0 rounded-full bg-[var(--brand)]" aria-label="Unread" />}
                    </div>
                    <p className="text-sm text-slate-600">{n.body}</p>
                    <p className="mt-1 text-xs text-slate-400">{dateTime(n.createdAt)}</p>
                  </Card>
                </button>
              </li>
            ))}
          </ul>
        )}
      </Async>
    </div>
  );
}
