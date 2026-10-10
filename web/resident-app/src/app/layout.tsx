import type { Metadata, Viewport } from "next";
import { RegisterSW } from "@/components/register-sw";
import "./globals.css";

export const metadata: Metadata = {
  title: "Condo Resident",
  description: "Check parcels, pay utility bills and read announcements",
  applicationName: "Condo Resident",
  appleWebApp: { capable: true, title: "Condo", statusBarStyle: "default" },
  icons: { icon: "/icons/64", apple: "/icons/180" },
  formatDetection: { telephone: false },
};

export const viewport: Viewport = {
  width: "device-width",
  initialScale: 1,
  viewportFit: "cover",
  themeColor: "#9b1c31",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body>
        {children}
        <RegisterSW />
      </body>
    </html>
  );
}
