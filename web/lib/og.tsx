import { readFileSync } from "fs";
import { join } from "path";

import { ImageResponse } from "next/og";

/**
 * Shared OG image factory. Every card uses the site's neutral palette —
 * ink on white, hairline borders — in Geist, read from the geist
 * package's TTFs since Satori needs real font files.
 */

type OgFont = {
  name: string;
  data: Buffer;
  weight: 400 | 600;
  style: "normal";
};

let fonts: OgFont[] = [];

function ogFonts(): OgFont[] {
  if (!fonts.length) {
    const fontDir = join(
      process.cwd(),
      "node_modules/geist/dist/fonts/geist-sans",
    );
    fonts = [
      {
        name: "Geist",
        data: readFileSync(join(fontDir, "Geist-Regular.ttf")),
        weight: 400,
        style: "normal",
      },
      {
        name: "Geist",
        data: readFileSync(join(fontDir, "Geist-SemiBold.ttf")),
        weight: 600,
        style: "normal",
      },
    ];
  }
  return fonts;
}

export const ogSize = { width: 1200, height: 630 };
export const ogContentType = "image/png";

export function ogImageAlt(parts: string[]): string {
  return parts.join(" ");
}

// The mark (components/site/mark.tsx): a grid of ports, one become a name.
const LOGO = `data:image/svg+xml,${encodeURIComponent(
  '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 13 13" fill="#0a0a0a"><rect width="3" height="3" rx="0.8"/><rect x="5" width="3" height="3" rx="0.8"/><rect x="10" width="3" height="3" rx="0.8"/><rect y="5" width="3" height="3" rx="0.8"/><rect x="5" y="5" width="8" height="3" rx="1.5" fill="#0070f3"/><rect y="10" width="3" height="3" rx="0.8"/><rect x="5" y="10" width="3" height="3" rx="0.8"/><rect x="10" y="10" width="3" height="3" rx="0.8"/></svg>',
)}`;

export function makeOgImage({
  lines,
  footnote,
  label = "ports",
}: {
  /** Headline. The last line renders muted, the rest in foreground. */
  lines: string[];
  footnote: string;
  label?: string;
}) {
  return new ImageResponse(
    (
      <div
        style={{
          display: "flex",
          flexDirection: "column",
          width: "100%",
          height: "100%",
          background: "#ffffff",
          color: "#0a0a0a",
          padding: "72px 80px",
          justifyContent: "space-between",
          fontFamily: "Geist",
          position: "relative",
        }}
      >
        {/* static radar sweep, decorative */}
        <div
          style={{
            display: "flex",
            position: "absolute",
            right: "-140px",
            top: "-160px",
            width: "640px",
            height: "640px",
            alignItems: "center",
            justifyContent: "center",
            border: "1px solid #e5e5e5",
            borderRadius: "9999px",
          }}
        >
          <div
            style={{
              display: "flex",
              width: "440px",
              height: "440px",
              border: "1px solid #e5e5e5",
              borderRadius: "9999px",
              alignItems: "center",
              justifyContent: "center",
            }}
          >
            <div
              style={{
                display: "flex",
                width: "240px",
                height: "240px",
                border: "1px solid #e5e5e5",
                borderRadius: "9999px",
              }}
            />
          </div>
        </div>
        <div
          style={{
            display: "flex",
            width: "10px",
            height: "10px",
            background: "#0a0a0a",
            borderRadius: "9999px",
            position: "absolute",
            right: "300px",
            top: "210px",
          }}
        />

        <div style={{ display: "flex", alignItems: "center", gap: "16px" }}>
          {/* satori cards must use plain img — next/image does not run here */}
          {/* eslint-disable-next-line @next/next/no-img-element */}
          <img src={LOGO} width={39} height={39} alt="" />
          <div
            style={{
              fontSize: 30,
              fontWeight: 600,
              letterSpacing: "-0.02em",
            }}
          >
            {label}
          </div>
        </div>

        <div style={{ display: "flex", flexDirection: "column" }}>
          {lines.map((line, i) => (
            <div
              key={line}
              style={{
                fontSize: 78,
                fontWeight: 600,
                letterSpacing: "-0.04em",
                lineHeight: 1.04,
                color: i === lines.length - 1 ? "#737373" : "#0a0a0a",
              }}
            >
              {line}
            </div>
          ))}
        </div>

        <div
          style={{
            display: "flex",
            justifyContent: "space-between",
            alignItems: "center",
            fontSize: 26,
            color: "#737373",
          }}
        >
          <div style={{ display: "flex" }}>{footnote}</div>
          <div style={{ display: "flex", color: "#0a0a0a" }}>ports.tools</div>
        </div>
      </div>
    ),
    {
      ...ogSize,
      fonts: ogFonts(),
    },
  );
}