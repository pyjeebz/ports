import type { Metadata } from "next";
import { GeistSans } from "geist/font/sans";
import { GeistMono } from "geist/font/mono";
import "./globals.css";

const title = "ports — see every port, stop it by name";
const description =
  "A zero-config CLI that shows every listening port with its framework, project, and uptime — and lets you stop things by name.";

export const metadata: Metadata = {
  metadataBase: new URL("https://ports.tools"),
  title,
  description,
  openGraph: {
    title,
    description,
    url: "https://ports.tools",
    siteName: "ports",
    type: "website",
  },
  twitter: { card: "summary_large_image", title, description },
};

export default function RootLayout({
  children,
}: Readonly<{ children: React.ReactNode }>) {
  return (
    <html lang="en" className={`${GeistSans.variable} ${GeistMono.variable}`}>
      <body className="antialiased">{children}</body>
    </html>
  );
}
