export const baht = (n: number) =>
  new Intl.NumberFormat("th-TH", { style: "currency", currency: "THB" }).format(n);

export const dateTime = (iso?: string) =>
  iso ? new Intl.DateTimeFormat("en-GB", { dateStyle: "medium", timeStyle: "short" }).format(new Date(iso)) : "";

export const dateOnly = (iso?: string) =>
  iso ? new Intl.DateTimeFormat("en-GB", { dateStyle: "medium" }).format(new Date(iso)) : "";

/** "2026-10" -> "Oct 2026" */
export const cycleLabel = (cycle: string) => {
  const [y, m] = cycle.split("-").map(Number);
  if (!y || !m) return cycle;
  return new Intl.DateTimeFormat("en-GB", { month: "short", year: "numeric" }).format(new Date(y, m - 1, 1));
};

export const currentCycle = () => new Date().toISOString().slice(0, 7);

export const isoDatePlusDays = (days: number) => {
  const d = new Date();
  d.setDate(d.getDate() + days);
  return d.toISOString().slice(0, 10);
};

export const describeTarget = (a: { targetType: string; targetFloor?: number }, buildingName?: string) => {
  switch (a.targetType) {
    case "BUILDING":
      return buildingName ?? "Specific building";
    case "FLOOR":
      return `${buildingName ?? "Building"} · floor ${a.targetFloor}`;
    default:
      return "All residents";
  }
};
