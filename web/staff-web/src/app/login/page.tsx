"use client";

import { LoginForm } from "@condo/shared/ui";

export default function LoginPage() {
  return (
    <LoginForm
      title="Condo Staff"
      subtitle="Juristic staff portal"
      hint="Demo: staff@condo.local / password123"
      onLogin={async (email, password) => {
        const res = await fetch("/api/login", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ email, password }),
        });
        const data = await res.json().catch(() => ({}));
        if (!res.ok) throw new Error(data.error ?? "Login failed");
        window.location.href = "/";
      }}
    />
  );
}
