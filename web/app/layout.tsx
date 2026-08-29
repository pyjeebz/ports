import type { Metadata } from "next";
import { GeistSans } from "geist/font/sans";
import { GeistMono } from "geist/font/mono";
import { RootProvider } from "fumadocs-ui/provider/next";
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
    <html
      lang="en"
      className={`${GeistSans.variable} ${GeistMono.variable}`}
      suppressHydrationWarning
    >
      <body className="antialiased">
        {/* Static CRT texture: scanlines + vignette. The only animated
            pixels on the site are the radar sweep and the flap board. */}
        <div className="crt" aria-hidden="true" />
        {/* The site is a single dark palette; keep next-themes from
            flipping it. Fumadocs tokens are overridden to phosphor in CSS. */}
        <RootProvider theme={{ defaultTheme: "dark", enableSystem: false }}>
          {children}
        </RootProvider>
      </body>
    </html>
  );
}
