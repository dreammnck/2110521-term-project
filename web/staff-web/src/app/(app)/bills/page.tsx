"use client";

import { useMemo, useState } from "react";
import { mutate } from "swr";
import { post, useApi, useRooms } from "@condo/shared/hooks";
import { Async, Button, Card, ErrorNote, Field, Input, Notice, PageHeader, Select, StatusBadge, Tabs } from "@condo/shared/ui";
import { baht, currentCycle, cycleLabel, dateOnly, isoDatePlusDays, type Bill, type BillInput } from "@condo/shared";

function IssueSingle({ onDone }: { onDone: (msg: string) => void }) {
  const { rooms } = useRooms();
  const [error, setError] = useState<unknown>();
  const [busy, setBusy] = useState(false);
  return (
    <form
      className="grid gap-3 sm:grid-cols-2"
      onSubmit={async (e) => {
        e.preventDefault();
        const fd = new FormData(e.currentTarget);
        setBusy(true);
        setError(undefined);
        try {
          const body: BillInput = {
            roomId: String(fd.get("roomId")),
            cycle: String(fd.get("cycle")),
            waterUnits: Number(fd.get("waterUnits") || 0),
            electricUnits: Number(fd.get("electricUnits") || 0),
            dueDate: String(fd.get("dueDate")),
          };
          const bill = await post<Bill>("utilities/bills", body);
          onDone(`Bill issued: ${baht(bill.total)} for ${cycleLabel(bill.cycle)}.`);
        } catch (err) {
          setError(err);
        } finally {
          setBusy(false);
        }
      }}
    >
      <Field label="Room">
        <Select name="roomId" required defaultValue="">
          <option value="" disabled>
            Select a room…
          </option>
          {(rooms ?? []).map((r) => (
            <option key={r.id} value={r.id}>
              {r.number} · {r.building}
            </option>
          ))}
        </Select>
      </Field>
      <Field label="Billing cycle">
        <Input name="cycle" type="month" required defaultValue={currentCycle()} />
      </Field>
      <Field label="Water (units)">
        <Input name="waterUnits" type="number" min="0" step="0.01" defaultValue="0" />
      </Field>
      <Field label="Electricity (kWh)">
        <Input name="electricUnits" type="number" min="0" step="0.01" defaultValue="0" />
      </Field>
      <Field label="Due date">
        <Input name="dueDate" type="date" required defaultValue={isoDatePlusDays(14)} />
      </Field>
      <div className="space-y-3 sm:col-span-2">
        <ErrorNote error={error} />
        <Button type="submit" disabled={busy}>
          {busy ? "Issuing…" : "Issue bill"}
        </Button>
      </div>
    </form>
  );
}

function IssueBatch({ onDone }: { onDone: (msg: string) => void }) {
  const { rooms } = useRooms();
  const [cycle, setCycle] = useState(currentCycle());
  const [due, setDue] = useState(isoDatePlusDays(14));
  const [units, setUnits] = useState<Record<string, { w: string; e: string }>>({});
  const [error, setError] = useState<unknown>();
  const [busy, setBusy] = useState(false);
  const filled = useMemo(
    () => Object.entries(units).filter(([, v]) => v.w !== "" || v.e !== ""),
    [units],
  );
  const set = (id: string, k: "w" | "e", v: string) => setUnits((u) => ({ ...u, [id]: { w: u[id]?.w ?? "", e: u[id]?.e ?? "", [k]: v } }));

  return (
    <form
      className="space-y-3"
      onSubmit={async (e) => {
        e.preventDefault();
        setBusy(true);
        setError(undefined);
        try {
          const items: BillInput[] = filled.map(([roomId, v]) => ({
            roomId,
            cycle,
            waterUnits: Number(v.w || 0),
            electricUnits: Number(v.e || 0),
            dueDate: due,
          }));
          const res = await post<{ created: Bill[]; skippedRoomIds: string[] }>("utilities/bills", { items });
          setUnits({});
          onDone(`Issued ${res.created.length} bill(s)${res.skippedRoomIds.length ? `, skipped ${res.skippedRoomIds.length} that already existed` : ""}.`);
        } catch (err) {
          setError(err);
        } finally {
          setBusy(false);
        }
      }}
    >
      <div className="grid gap-3 sm:grid-cols-2">
        <Field label="Billing cycle">
          <Input type="month" required value={cycle} onChange={(e) => setCycle(e.target.value)} />
        </Field>
        <Field label="Due date">
          <Input type="date" required value={due} onChange={(e) => setDue(e.target.value)} />
        </Field>
      </div>
      <p className="text-sm text-slate-500">Enter meter usage for each room. Rooms left blank are skipped.</p>
      <div className="max-h-80 overflow-auto rounded-lg border border-slate-200">
        <table className="w-full text-sm">
          <thead className="sticky top-0 bg-slate-50 text-left text-slate-600">
            <tr>
              <th className="px-3 py-2">Room</th>
              <th className="px-3 py-2">Water (units)</th>
              <th className="px-3 py-2">Electricity (kWh)</th>
            </tr>
          </thead>
          <tbody>
            {(rooms ?? []).map((r) => (
              <tr key={r.id} className="border-t border-slate-100">
                <td className="px-3 py-1.5 whitespace-nowrap">
                  {r.number} · {r.building}
                </td>
                <td className="px-3 py-1.5">
                  <Input type="number" min="0" step="0.01" aria-label={`Water ${r.number}`} value={units[r.id]?.w ?? ""} onChange={(e) => set(r.id, "w", e.target.value)} />
                </td>
                <td className="px-3 py-1.5">
                  <Input type="number" min="0" step="0.01" aria-label={`Electricity ${r.number}`} value={units[r.id]?.e ?? ""} onChange={(e) => set(r.id, "e", e.target.value)} />
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <ErrorNote error={error} />
      <Button type="submit" disabled={busy || filled.length === 0}>
        {busy ? "Issuing…" : `Issue ${filled.length} bill(s)`}
      </Button>
    </form>
  );
}

export default function BillsPage() {
  const [mode, setMode] = useState<"single" | "batch">("single");
  const [status, setStatus] = useState("");
  const [cycle, setCycle] = useState("");
  const [notice, setNotice] = useState("");
  const qs = new URLSearchParams();
  if (status) qs.set("status", status);
  if (cycle) qs.set("cycle", cycle);
  const { data, error } = useApi<Bill[]>(`utilities/bills${qs.size ? `?${qs}` : ""}`, { refreshInterval: 20000 });
  const { label } = useRooms();

  const done = (msg: string) => {
    setNotice(msg);
    mutate((k) => typeof k === "string" && k.startsWith("utilities/bills"));
  };

  return (
    <>
      <PageHeader title="Utility bills" subtitle="Issue water and electricity bills and track payments" />
      <Card className="mb-6">
        <h2 className="mb-3 font-semibold">Issue bills</h2>
        <Tabs
          value={mode}
          onChange={setMode}
          options={[
            { value: "single", label: "Single room" },
            { value: "batch", label: "Whole cycle (batch)" },
          ]}
        />
        {mode === "single" ? <IssueSingle onDone={done} /> : <IssueBatch onDone={done} />}
        {notice && (
          <div className="mt-3">
            <Notice>{notice}</Notice>
          </div>
        )}
      </Card>

      <div className="mb-3 flex flex-wrap items-end gap-3">
        <h2 className="mr-auto font-semibold">All bills</h2>
        <Field label="Status">
          <Select value={status} onChange={(e) => setStatus(e.target.value)}>
            <option value="">All</option>
            {["UNPAID", "PENDING", "OVERDUE", "PAID"].map((s) => (
              <option key={s}>{s}</option>
            ))}
          </Select>
        </Field>
        <Field label="Cycle">
          <Input type="month" value={cycle} onChange={(e) => setCycle(e.target.value)} />
        </Field>
      </div>
      <Async data={data} error={error} empty="No bills match these filters.">
        {(items) => (
          <div className="overflow-x-auto rounded-xl border border-slate-200 bg-white">
            <table className="w-full text-sm">
              <thead className="bg-slate-50 text-left text-slate-600">
                <tr>
                  <th className="px-3 py-2">Room</th>
                  <th className="px-3 py-2">Cycle</th>
                  <th className="px-3 py-2 text-right">Water</th>
                  <th className="px-3 py-2 text-right">Electricity</th>
                  <th className="px-3 py-2 text-right">Total</th>
                  <th className="px-3 py-2">Due</th>
                  <th className="px-3 py-2">Status</th>
                </tr>
              </thead>
              <tbody>
                {items.map((b) => (
                  <tr key={b.id} className="border-t border-slate-100">
                    <td className="px-3 py-2 whitespace-nowrap">{label(b.roomId)}</td>
                    <td className="px-3 py-2">{cycleLabel(b.cycle)}</td>
                    <td className="px-3 py-2 text-right">{baht(b.waterAmount)}</td>
                    <td className="px-3 py-2 text-right">{baht(b.electricAmount)}</td>
                    <td className="px-3 py-2 text-right font-medium">{baht(b.total)}</td>
                    <td className="px-3 py-2 whitespace-nowrap">{dateOnly(b.dueDate)}</td>
                    <td className="px-3 py-2">
                      <StatusBadge status={b.status} />
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </Async>
    </>
  );
}
