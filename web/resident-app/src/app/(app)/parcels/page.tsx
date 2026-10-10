"use client";

import { useState } from "react";
import { mutate } from "swr";
import { post, useApi } from "@condo/shared/hooks";
import { Async, Button, Card, ErrorNote, StatusBadge } from "@condo/shared/ui";
import { dateTime, type Parcel } from "@condo/shared";
import { Sheet } from "@/components/sheet";
import { useToast } from "@/components/toast";

function ParcelCard({ p }: { p: Parcel }) {
  const [open, setOpen] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<unknown>();
  const toast = useToast();

  const confirm = async () => {
    setBusy(true);
    setError(undefined);
    try {
      await post(`parcels/${p.id}/pickup`);
      await mutate("parcels/me");
      setOpen(false);
      toast("Pickup confirmed");
    } catch (err) {
      setError(err);
      setBusy(false);
    }
  };

  return (
    <Card className="space-y-3">
      <div className="flex gap-3">
        {p.hasPhoto ? (
          // eslint-disable-next-line @next/next/no-img-element
          <img src={`/api/gw/parcels/${p.id}/photo`} alt="Parcel" className="size-20 shrink-0 rounded-lg object-cover" />
        ) : (
          <div className="grid size-20 shrink-0 place-items-center rounded-lg bg-slate-100 text-2xl">📦</div>
        )}
        <div className="min-w-0 flex-1">
          <div className="flex items-start justify-between gap-2">
            <p className="font-semibold">{p.carrier || "Parcel"}</p>
            <StatusBadge status={p.status} />
          </div>
          {p.trackingNo && <p className="truncate text-sm text-slate-600">{p.trackingNo}</p>}
          {p.note && <p className="truncate text-sm text-slate-500">{p.note}</p>}
          <p className="mt-1 text-xs text-slate-500">
            Arrived {dateTime(p.receivedAt)}
            {p.pickedUpAt && ` · collected ${dateTime(p.pickedUpAt)}`}
          </p>
        </div>
      </div>
      {p.status === "WAITING" && (
        <>
          <Button onClick={() => setOpen(true)} className="w-full !py-3 !text-base">
            Confirm pickup
          </Button>
          <Sheet open={open} onClose={() => !busy && setOpen(false)} title="Confirm pickup">
            <p className="mb-4 text-sm text-slate-600">
              Only confirm once you are holding the parcel{p.carrier ? ` from ${p.carrier}` : ""}. The juristic office will be notified.
            </p>
            <div className="space-y-3">
              <ErrorNote error={error} />
              <Button onClick={confirm} disabled={busy} className="w-full !py-3 !text-base">
                {busy ? "Confirming…" : "Yes, I have it"}
              </Button>
              <Button variant="secondary" onClick={() => setOpen(false)} disabled={busy} className="w-full !py-3 !text-base">
                Not yet
              </Button>
            </div>
          </Sheet>
        </>
      )}
    </Card>
  );
}

export default function ParcelsPage() {
  const { data, error } = useApi<Parcel[]>("parcels/me", { refreshInterval: 20000 });
  const waiting = data?.filter((p) => p.status === "WAITING");
  const done = data?.filter((p) => p.status !== "WAITING");

  return (
    <div className="space-y-6">
      <h1 className="text-xl font-semibold">My parcels</h1>
      <section className="space-y-3">
        <h2 className="text-sm font-medium text-slate-500">Waiting for you</h2>
        <Async data={waiting} error={error} empty="No parcels are waiting for you.">
          {(items) => items.map((p) => <ParcelCard key={p.id} p={p} />)}
        </Async>
      </section>
      {done && done.length > 0 && (
        <section className="space-y-3">
          <h2 className="text-sm font-medium text-slate-500">Collected</h2>
          {done.map((p) => (
            <ParcelCard key={p.id} p={p} />
          ))}
        </section>
      )}
    </div>
  );
}
