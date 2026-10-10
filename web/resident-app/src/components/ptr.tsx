"use client";

import { useEffect, useRef, useState, type ReactNode } from "react";
import { useSWRConfig } from "swr";

const THRESHOLD = 64;

/** Native-style pull-to-refresh: revalidates every SWR query. */
export function PullToRefresh({ children }: { children: ReactNode }) {
  const { mutate } = useSWRConfig();
  const [pull, setPull] = useState(0);
  const [busy, setBusy] = useState(false);
  const startY = useRef<number | null>(null);
  const pullRef = useRef(0);

  useEffect(() => {
    const reset = () => {
      startY.current = null;
      pullRef.current = 0;
    };
    const onStart = (e: TouchEvent) => {
      if (window.scrollY <= 0 && !document.body.style.overflow) startY.current = e.touches[0].clientY;
    };
    const onMove = (e: TouchEvent) => {
      if (startY.current === null) return;
      const d = e.touches[0].clientY - startY.current;
      if (d > 0 && window.scrollY <= 0) {
        pullRef.current = Math.min(d * 0.5, 90);
        setPull(pullRef.current);
      } else {
        reset();
        setPull(0);
      }
    };
    const onEnd = async () => {
      const v = pullRef.current;
      reset();
      if (v >= THRESHOLD) {
        setBusy(true);
        setPull(48);
        await mutate(() => true);
        setBusy(false);
      }
      setPull(0);
    };
    window.addEventListener("touchstart", onStart, { passive: true });
    window.addEventListener("touchmove", onMove, { passive: true });
    window.addEventListener("touchend", onEnd);
    window.addEventListener("touchcancel", onEnd);
    return () => {
      window.removeEventListener("touchstart", onStart);
      window.removeEventListener("touchmove", onMove);
      window.removeEventListener("touchend", onEnd);
      window.removeEventListener("touchcancel", onEnd);
    };
  }, [mutate]);

  const dragging = pull > 0 && !busy;
  return (
    <>
      <div
        aria-hidden
        className="pointer-events-none fixed inset-x-0 z-20 flex justify-center"
        style={{
          top: "calc(env(safe-area-inset-top) + 3.5rem)",
          transform: `translateY(${pull - 36}px)`,
          opacity: Math.min(pull / THRESHOLD, 1),
          transition: dragging ? "none" : "transform .2s, opacity .2s",
        }}
      >
        <div className="grid size-9 place-items-center rounded-full bg-white shadow-md">
          <svg viewBox="0 0 24 24" className={`size-5 text-[var(--brand)] ${busy ? "animate-spin" : ""}`} style={busy ? undefined : { transform: `rotate(${pull * 4}deg)` }} fill="none" stroke="currentColor" strokeWidth={2.2} strokeLinecap="round">
            <path d="M21 12a9 9 0 1 1-3-6.7" />
            <path d="M21 3v6h-6" />
          </svg>
        </div>
      </div>
      <div style={pull > 0 ? { transform: `translateY(${pull * 0.4}px)`, transition: dragging ? "none" : "transform .2s" } : undefined}>{children}</div>
    </>
  );
}
