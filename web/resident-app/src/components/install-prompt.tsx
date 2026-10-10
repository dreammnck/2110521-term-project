"use client";

import { useEffect, useState } from "react";
import { Icon } from "./icons";

interface InstallEvent extends Event {
  prompt: () => Promise<void>;
}

const KEY = "condo-install-dismissed";

/** "Add to home screen" banner (Android install prompt, iOS instructions). */
export function InstallPrompt() {
  const [evt, setEvt] = useState<InstallEvent | null>(null);
  const [ios, setIos] = useState(false);

  useEffect(() => {
    const standalone = window.matchMedia("(display-mode: standalone)").matches || (navigator as Navigator & { standalone?: boolean }).standalone;
    let dismissed = false;
    try {
      dismissed = localStorage.getItem(KEY) === "1";
    } catch {
      /* storage unavailable */
    }
    if (standalone || dismissed) return;
    if (/iphone|ipad|ipod/i.test(navigator.userAgent)) setIos(true);
    const onPrompt = (e: Event) => {
      e.preventDefault();
      setEvt(e as InstallEvent);
    };
    window.addEventListener("beforeinstallprompt", onPrompt);
    return () => window.removeEventListener("beforeinstallprompt", onPrompt);
  }, []);

  if (!evt && !ios) return null;
  const dismiss = () => {
    try {
      localStorage.setItem(KEY, "1");
    } catch {
      /* ignore */
    }
    setEvt(null);
    setIos(false);
  };

  return (
    <div className="flex items-center gap-3 rounded-2xl bg-slate-900 p-3 text-white">
      <Icon name="download" className="size-6 shrink-0" />
      <p className="flex-1 text-sm">
        {evt ? "Install Condo Resident for quick access." : "Install: tap Share, then “Add to Home Screen”."}
      </p>
      {evt && (
        <button onClick={() => evt.prompt().then(dismiss)} className="rounded-full bg-white px-3 py-1.5 text-sm font-semibold text-slate-900 active:scale-95">
          Install
        </button>
      )}
      <button onClick={dismiss} aria-label="Dismiss" className="p-1 text-white/70">
        <Icon name="close" className="size-5" />
      </button>
    </div>
  );
}
