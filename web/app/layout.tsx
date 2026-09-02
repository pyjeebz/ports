import type { Metadata } from "next";
import { Architects_Daughter, JetBrains_Mono, Kalam } from "next/font/google";
import { RootProvider } from "fumadocs-ui/provider/next";
import { SITE } from "@/lib/site";
import "./globals.css";

// The Napkin trio: Architects Daughter (display — the Sharpie), Kalam
// (body — the ballpoint), JetBrains Mono (code, data, tags — the rigid
// technical contrast).
const architectsDaughter = Architects_Daughter({
  weight: "400",
  subsets: ["latin"],
  variable: "--font-architects",
  display: "swap",
});
const kalam = Kalam({
  weight: ["300", "400", "700"],
  subsets: ["latin"],
  variable: "--font-kalam",
  display: "swap",
});
const jetbrainsMono = JetBrains_Mono({
  subsets: ["latin"],
  variable: "--font-jetbrains",
  display: "swap",
});

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
      className={`${architectsDaughter.variable} ${kalam.variable} ${jetbrainsMono.variable}`}
      suppressHydrationWarning
    >
      <body className="paper-texture antialiased">
        {/* The site is a single paper palette; keep next-themes from
            flipping it. Fumadocs tokens are retinted in CSS. */}
        <RootProvider theme={{ defaultTheme: "light", enableSystem: false }}>
          {children}
        </RootProvider>
      </body>
    </html>
  );
}