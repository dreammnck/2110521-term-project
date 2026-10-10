import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "Condo Staff",
  description: "Juristic staff portal: parcels, utility bills and announcements",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  );
}
