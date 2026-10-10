"use client";

import Link from "next/link";
import { Suspense } from "react";
import { useSearchParams } from "next/navigation";
import { useApi } from "@condo/shared/hooks";
import { Button, Card, ErrorNote, Loading } from "@condo/shared/ui";
import { baht, type Bill } from "@condo/shared";

// The gateway redirects here after checkout. The webhook may land slightly after
// the redirect, so poll the bill until its status settles.
function Result() {
  const params = useSearchParams();
  const billId = params.get("billId");
  const gatewayStatus = params.get("status");
  const { data: bill, error } = useApi<Bill>(billId ? `utilities/bills/${billId}` : null, {
    refreshInterval: (b) => (b?.status === "PENDING" ? 2000 : 0),
  });

  if (!billId) return <ErrorNote error="Missing bill reference." />;
  if (error) return <ErrorNote error={error} />;
  if (!bill) return <Loading />;

  if (bill.status === "PAID") {
    return (
      <Card className="space-y-3 text-center">
        <p className="text-4xl">✅</p>
        <h1 className="text-xl font-semibold">Payment successful</h1>
        <p className="text-slate-600">We received {baht(bill.total)}. Thank you!</p>
        <Link href={`/bills/${bill.id}/receipt`}>
          <Button className="w-full">View receipt</Button>
        </Link>
      </Card>
    );
  }
  if (bill.status === "PENDING" && gatewayStatus !== "FAILED") {
    return (
      <Card className="space-y-2 text-center">
        <p className="text-4xl">⏳</p>
        <h1 className="text-xl font-semibold">Confirming your payment…</h1>
        <p className="text-sm text-slate-500">This usually takes a few seconds.</p>
      </Card>
    );
  }
  return (
    <Card className="space-y-3 text-center">
      <p className="text-4xl">⚠️</p>
      <h1 className="text-xl font-semibold">Payment not completed</h1>
      <p className="text-slate-600">No money was taken. You can try again.</p>
      <Link href={`/bills/${bill.id}`}>
        <Button className="w-full">Back to bill</Button>
      </Link>
    </Card>
  );
}

export default function ReturnPage() {
  return (
    <Suspense fallback={<Loading />}>
      <Result />
    </Suspense>
  );
}
