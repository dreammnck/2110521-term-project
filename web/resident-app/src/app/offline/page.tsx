export default function Offline() {
  return (
    <main className="grid min-h-dvh place-items-center px-6 text-center">
      <div>
        <p className="text-5xl">📡</p>
        <h1 className="mt-3 text-xl font-semibold">You&apos;re offline</h1>
        <p className="mt-1 text-sm text-slate-500">Check your connection and try again.</p>
        <a href="/" className="mt-5 inline-block rounded-lg bg-[var(--brand)] px-5 py-2.5 text-sm font-medium text-white">
          Retry
        </a>
      </div>
    </main>
  );
}
