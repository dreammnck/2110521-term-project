"use client";

import Link from "next/link";
import { use, useState } from "react";
import { post, useApi } from "@condo/shared/hooks";
import { Button, Card, ErrorNote, Loading, StatusBadge } from "@condo/shared/ui";
import { baht, cycleLabel, dateOnly, type Bill } from "@condo/shared";
import { Sheet } from "@/components/sheet";
import { haptic } from "@/components/toast";

export default function BillDetail({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params);
  const { data: bill, error } = useApi<Bill>(`utilities/bills/${id}`);
  const [busy, setBusy] = useState(false);
  const [payError, setPayError] = useState<unknown>();
  const [open, setOpen] = useState(false);

  if (error) return <ErrorNote error={error} />;
  if (!bill) return <Loading />;

  const pay = async () => {
    setBusy(true);
    setPayError(undefined);
    try {
      haptic();
      const res = await post<{ checkoutUrl: string }>("payments", { billId: bill.id });
      window.location.href = res.checkoutUrl; // hand off to the payment gateway
    } catch (err) {
      setPayError(err);
      setBusy(false);
    }
  };

  return (
    <div className="space-y-4">
      <div className="flex items-start justify-between">
        <div>
          <h1 className="text-xl font-semibold">{cycleLabel(bill.cycle)}</h1>
          <p className="text-sm text-slate-500">Due {dateOnly(bill.dueDate)}</p>
        </div>
        <StatusBadge status={bill.status} />
      </div>

      <Card>
        <dl className="space-y-2 text-sm">
          <div className="flex justify-between">
            <dt className="text-slate-600">Water · {bill.waterUnits} units</dt>
            <dd>{baht(bill.waterAmount)}</dd>
          </div>
          <div className="flex justify-between">
            <dt className="text-slate-600">Electricity · {bill.electricUnits} kWh</dt>
            <dd>{baht(bill.electricAmount)}</dd>
          </div>
          <div className="flex justify-between border-t border-slate-200 pt-2 text-base font-semibold">
            <dt>Total</dt>
            <dd>{baht(bill.total)}</dd>
          </div>
        </dl>
      </Card>

      {bill.status === "PENDING" && (
        <p className="rounded-lg bg-amber-50 px-3 py-2 text-sm text-amber-800">
          A payment is in progress. If you did not finish it, you can start again below.
        </p>
      )}

      {/* Sticky action bar, like a native checkout screen */}
      <div className="fixed bottom-0 left-1/2 z-30 w-full max-w-md -translate-x-1/2 border-t border-slate-200 bg-white/95 px-4 pb-[calc(0.75rem+env(safe-area-inset-bottom))] pt-3 backdrop-blur">
        {bill.status === "PAID" ? (
          <Link href={`/bills/${bill.id}/receipt`} className="block">
            <Button className="w-full !py-3 !text-base">View receipt</Button>
          </Link>
        ) : (
          <Button onClick={() => setOpen(true)} className="w-full !py-3 !text-base">
            Pay {baht(bill.total)}
          </Button>
        )}
      </div>

      <Sheet open={open} onClose={() => !busy && setOpen(false)} title={`Pay ${baht(bill.total)}`}>
        <p className="mb-4 text-sm text-slate-600">
          You will be taken to our payment partner to complete the payment for {cycleLabel(bill.cycle)}. Return here afterwards to see your receipt.
        </p>
        <div className="space-y-3">
          <ErrorNote error={payError} />
          <Button onClick={pay} disabled={busy} className="w-full !py-3 !text-base">
            {busy ? "Redirecting…" : "Continue to payment"}
          </Button>
          <Button variant="secondary" onClick={() => setOpen(false)} disabled={busy} className="w-full !py-3 !text-base">
            Cancel
          </Button>
        </div>
      </Sheet>
    </div>
  );
}
