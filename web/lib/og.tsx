import { readFileSync } from "fs";
import { join } from "path";

import { ImageResponse } from "next/og";

/**
 * Shared OG image factory. Every card uses the paper palette from
 * globals.css so the preview matches the site. (Satori needs real font
 * files; the hand-drawn Google fonts ship as woff2 via next/font, so the
 * cards keep Geist while the site draws in Kalam.)
 *
 * Satori has no <mask>, so the logo's slash cut is faked by overpainting
 * the ring with the card background instead of masking it away.
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

const LOGO = `data:image/svg+xml,${encodeURIComponent(
  '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100"><circle cx="50" cy="50" r="35" stroke="#2c2c2a" stroke-width="14" fill="none"/><line x1="12" y1="88" x2="88" y2="12" stroke="#f4f1ea" stroke-width="18" stroke-linecap="square"/><circle cx="38" cy="38" r="6" fill="#2c2c2a"/><circle cx="62" cy="62" r="6" fill="#2c2c2a"/></svg>',
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
          background: "#f4f1ea",
          color: "#2c2c2a",
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
            border: "1px solid #c9c4b6",
            borderRadius: "9999px",
          }}
        >
          <div
            style={{
              display: "flex",
              width: "440px",
              height: "440px",
              border: "1px solid #c9c4b6",
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
                border: "1px solid #c9c4b6",
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
            background: "#3d63dd",
            borderRadius: "9999px",
            position: "absolute",
            right: "300px",
            top: "210px",
          }}
        />

        <div style={{ display: "flex", alignItems: "center", gap: "16px" }}>
          {/* satori cards must use plain img — next/image does not run here */}
          {/* eslint-disable-next-line @next/next/no-img-element */}
          <img src={LOGO} width={46} height={46} alt="" />
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
                color: i === lines.length - 1 ? "#595955" : "#2c2c2a",
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
            color: "#595955",
          }}
        >
          <div style={{ display: "flex" }}>{footnote}</div>
          <div style={{ display: "flex", color: "#3d63dd" }}>ports.tools</div>
        </div>
      </div>
    ),
    {
      ...ogSize,
      fonts: ogFonts(),
    },
  );
}