"use client";

import { createContext, useCallback, useContext, useRef, useState, type ReactNode } from "react";

type Tone = "success" | "error";
const ToastCtx = createContext<(msg: string, tone?: Tone) => void>(() => undefined);
export const useToast = () => useContext(ToastCtx);

/** Haptic tick on devices that support it. */
export const haptic = (ms = 12) => {
  try {
    navigator.vibrate?.(ms);
  } catch {
    /* unsupported */
  }
};

export function ToastProvider({ children }: { children: ReactNode }) {
  const [toast, setToast] = useState<{ msg: string; tone: Tone; id: number } | null>(null);
  const timer = useRef<ReturnType<typeof setTimeout>>(undefined);

  const show = useCallback((msg: string, tone: Tone = "success") => {
    clearTimeout(timer.current);
    setToast({ msg, tone, id: Date.now() });
    if (tone === "success") haptic();
    timer.current = setTimeout(() => setToast(null), 3000);
  }, []);

  return (
    <ToastCtx.Provider value={show}>
      {children}
      <div aria-live="polite" className="pointer-events-none fixed inset-x-0 top-[calc(4rem+env(safe-area-inset-top))] z-50 flex justify-center px-4">
        {toast && (
          <div
            key={toast.id}
            className={`toast-in rounded-full px-4 py-2 text-sm font-medium text-white shadow-lg ${toast.tone === "success" ? "bg-slate-900" : "bg-red-600"}`}
          >
            {toast.msg}
          </div>
        )}
      </div>
    </ToastCtx.Provider>
  );
}
