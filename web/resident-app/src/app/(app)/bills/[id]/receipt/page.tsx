"use client";

import { use } from "react";
import { useApi } from "@condo/shared/hooks";
import { Button, Card, ErrorNote, Loading } from "@condo/shared/ui";
import { baht, cycleLabel, dateTime, type Receipt } from "@condo/shared";

export default function ReceiptPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params);
  const { data: r, error } = useApi<Receipt>(`payments/${id}/receipt`);
  if (error) return <ErrorNote error={error} />;
  if (!r) return <Loading />;

  return (
    <div className="space-y-4">
      <Card className="space-y-4">
        <div className="text-center">
          <p className="text-3xl">✅</p>
          <h1 className="text-lg font-semibold">Payment receipt</h1>
          <p className="text-sm text-slate-500">{r.receiptNo}</p>
        </div>
        <dl className="space-y-2 text-sm">
          <Row k="Billing cycle" v={cycleLabel(r.bill.cycle)} />
          <Row k="Water" v={baht(r.bill.waterAmount)} />
          <Row k="Electricity" v={baht(r.bill.electricAmount)} />
          <Row k="Paid at" v={dateTime(r.paidAt)} />
          <Row k="Payment reference" v={r.paymentReference} mono />
          <div className="flex justify-between border-t border-slate-200 pt-2 text-base font-semibold">
            <dt>Amount paid</dt>
            <dd>{baht(r.amount)}</dd>
          </div>
        </dl>
      </Card>
      <Button variant="secondary" className="no-print w-full" onClick={() => window.print()}>
        Print / save as PDF
      </Button>
    </div>
  );
}

function Row({ k, v, mono }: { k: string; v: string; mono?: boolean }) {
  return (
    <div className="flex justify-between gap-4">
      <dt className="text-slate-600">{k}</dt>
      <dd className={`text-right ${mono ? "break-all font-mono text-xs" : ""}`}>{v}</dd>
    </div>
  );
}
