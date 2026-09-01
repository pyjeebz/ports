import type { Metadata } from "next";
import { GeistSans } from "geist/font/sans";
import { GeistMono } from "geist/font/mono";
import { RootProvider } from "fumadocs-ui/provider/next";
import { SITE } from "@/lib/site";
import "./globals.css";

// Drawn from SITE so the layout, OG images and manifest can't drift.
const title = `ports — ${SITE.tagline}`;
const description = SITE.description;

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
    <html
      lang="en"
      className={`${GeistSans.variable} ${GeistMono.variable}`}
      suppressHydrationWarning
    >
      <body className="antialiased">
        {/* The site is a single dark palette; keep next-themes from
            flipping it. Fumadocs tokens are overridden to neutral in CSS. */}
        <RootProvider theme={{ defaultTheme: "dark", enableSystem: false }}>
          {children}
        </RootProvider>
      </body>
    </html>
  );
}
