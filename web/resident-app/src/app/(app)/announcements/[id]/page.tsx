"use client";

import { use, useEffect } from "react";
import { mutate } from "swr";
import { post, useApi } from "@condo/shared/hooks";
import { Card, ErrorNote, Loading } from "@condo/shared/ui";
import { dateTime, type Announcement } from "@condo/shared";

export default function AnnouncementDetail({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params);
  const { data: a, error } = useApi<Announcement>(`announcements/${id}`, { revalidateOnFocus: false });

  // Opening an announcement marks it as read.
  const shouldMark = a && !a.read;
  useEffect(() => {
    if (!shouldMark) return;
    post(`announcements/${id}/read`)
      .then(() => mutate((k) => typeof k === "string" && k.startsWith("announcements")))
      .catch(() => undefined);
  }, [shouldMark, id]);

  if (error) return <ErrorNote error={error} />;
  if (!a) return <Loading />;
  return (
    <div className="space-y-4">
      <Card className="space-y-2">
        <p className="text-xs text-slate-500">
          {a.category} · {dateTime(a.publishAt)}
        </p>
        <h1 className="text-xl font-semibold">{a.title}</h1>
        <p className="whitespace-pre-line text-slate-700">{a.body}</p>
      </Card>
    </div>
  );
}
