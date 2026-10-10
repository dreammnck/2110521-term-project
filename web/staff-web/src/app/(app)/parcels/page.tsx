"use client";

import { useRef, useState } from "react";
import { mutate } from "swr";
import { api, useApi, useRooms } from "@condo/shared/hooks";
import { Async, Button, Card, ErrorNote, Field, Input, Notice, PageHeader, Select, StatusBadge, Tabs } from "@condo/shared/ui";
import { dateTime, type Parcel } from "@condo/shared";

function RegisterForm({ onDone }: { onDone: () => void }) {
  const { rooms, buildings } = useRooms();
  const [building, setBuilding] = useState("");
  const [error, setError] = useState<unknown>();
  const [ok, setOk] = useState("");
  const [busy, setBusy] = useState(false);
  const formRef = useRef<HTMLFormElement>(null);
  const roomOptions = (rooms ?? []).filter((r) => !building || r.buildingId === building);

  return (
    <Card className="mb-6">
      <h2 className="mb-3 font-semibold">Register incoming parcel</h2>
      <form
        ref={formRef}
        className="grid gap-3 sm:grid-cols-2"
        onSubmit={async (e) => {
          e.preventDefault();
          setBusy(true);
          setError(undefined);
          setOk("");
          try {
            const fd = new FormData(e.currentTarget);
            if (!(fd.get("photo") as File)?.size) fd.delete("photo");
            await api("parcels", { method: "POST", body: fd });
            formRef.current?.reset();
            setOk("Parcel registered. The resident has been notified.");
            onDone();
          } catch (err) {
            setError(err);
          } finally {
            setBusy(false);
          }
        }}
      >
        <Field label="Building">
          <Select value={building} onChange={(e) => setBuilding(e.target.value)}>
            <option value="">All buildings</option>
            {buildings.map((b) => (
              <option key={b.id} value={b.id}>
                {b.name}
              </option>
            ))}
          </Select>
        </Field>
        <Field label="Room (recipient)">
          <Select name="roomId" required defaultValue="">
            <option value="" disabled>
              Select a room…
            </option>
            {roomOptions.map((r) => (
              <option key={r.id} value={r.id}>
                {r.number} · {r.building}
              </option>
            ))}
          </Select>
        </Field>
        <Field label="Carrier">
          <Input name="carrier" placeholder="Kerry, Flash, Thailand Post…" />
        </Field>
        <Field label="Tracking number">
          <Input name="trackingNo" />
        </Field>
        <Field label="Note">
          <Input name="note" placeholder="e.g. large box" />
        </Field>
        <Field label="Photo (optional)" hint="JPG, PNG or WebP, up to 5 MB">
          <Input name="photo" type="file" accept="image/jpeg,image/png,image/webp" />
        </Field>
        <div className="space-y-3 sm:col-span-2">
          <ErrorNote error={error} />
          {ok && <Notice>{ok}</Notice>}
          <Button type="submit" disabled={busy}>
            {busy ? "Saving…" : "Register parcel"}
          </Button>
        </div>
      </form>
    </Card>
  );
}

export default function ParcelsPage() {
  const [tab, setTab] = useState<"WAITING" | "PICKED_UP">("WAITING");
  const key = `parcels?status=${tab}`;
  const { data, error } = useApi<Parcel[]>(key, { refreshInterval: 20000 });
  const { label } = useRooms();

  return (
    <>
      <PageHeader title="Parcels" subtitle="Register arrivals and track pickups" />
      <RegisterForm onDone={() => mutate((k) => typeof k === "string" && k.startsWith("parcels"))} />
      <Tabs
        value={tab}
        onChange={setTab}
        options={[
          { value: "WAITING", label: "Awaiting pickup" },
          { value: "PICKED_UP", label: "Picked up" },
        ]}
      />
      <Async data={data} error={error} empty={tab === "WAITING" ? "No parcels are waiting for pickup." : "No parcels have been picked up yet."}>
        {(items) => (
          <ul className="grid gap-3 sm:grid-cols-2">
            {items.map((p) => (
              <li key={p.id}>
                <Card className="flex gap-3">
                  {p.hasPhoto ? (
                    // eslint-disable-next-line @next/next/no-img-element
                    <img src={`/api/gw/parcels/${p.id}/photo`} alt="Parcel" className="size-20 shrink-0 rounded-lg object-cover" />
                  ) : (
                    <div className="grid size-20 shrink-0 place-items-center rounded-lg bg-slate-100 text-2xl">📦</div>
                  )}
                  <div className="min-w-0 flex-1">
                    <div className="flex items-start justify-between gap-2">
                      <p className="font-semibold">{label(p.roomId)}</p>
                      <StatusBadge status={p.status} />
                    </div>
                    <p className="truncate text-sm text-slate-600">
                      {p.carrier || "Parcel"}
                      {p.trackingNo && ` · ${p.trackingNo}`}
                    </p>
                    {p.note && <p className="truncate text-sm text-slate-500">{p.note}</p>}
                    <p className="mt-1 text-xs text-slate-500">
                      Received {dateTime(p.receivedAt)}
                      {p.pickedUpAt && ` · picked up ${dateTime(p.pickedUpAt)}`}
                    </p>
                  </div>
                </Card>
              </li>
            ))}
          </ul>
        )}
      </Async>
    </>
  );
}
