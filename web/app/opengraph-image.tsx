import { readFileSync } from "fs";
import { join } from "path";

import { ImageResponse } from "next/og";

export const alt = "ports — see every port, stop it by name";
export const size = { width: 1200, height: 630 };
export const contentType = "image/png";

const fontDir = join(
  process.cwd(),
  "node_modules/geist/dist/fonts/geist-sans",
);
const geistRegular = readFileSync(join(fontDir, "Geist-Regular.ttf"));
const geistSemiBold = readFileSync(join(fontDir, "Geist-SemiBold.ttf"));

const LOGO = `data:image/svg+xml,${encodeURIComponent(
  '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100"><mask id="m"><rect width="100" height="100" fill="white"/><line x1="12" y1="88" x2="88" y2="12" stroke="black" stroke-width="14" stroke-linecap="square"/></mask><circle cx="50" cy="50" r="35" stroke="#fafafa" stroke-width="14" fill="none" mask="url(#m)"/><circle cx="38" cy="38" r="6" fill="#fafafa"/><circle cx="62" cy="62" r="6" fill="#fafafa"/></svg>',
)}`;

export default function OpengraphImage() {
  return new ImageResponse(
    (
      <div
        style={{
          display: "flex",
          flexDirection: "column",
          width: "100%",
          height: "100%",
          background: "#000000",
          color: "#fafafa",
          padding: "80px",
          justifyContent: "space-between",
          fontFamily: "Geist",
        }}
      >
        <div style={{ display: "flex", alignItems: "center", gap: "16px" }}>
          {/* eslint-disable-next-line @next/next/no-img-element */}
          <img src={LOGO} width={46} height={46} alt="" />
          <div style={{ fontSize: 30, fontWeight: 600, letterSpacing: "-0.02em" }}>
            ports
          </div>
        </div>
        <div style={{ display: "flex", flexDirection: "column" }}>
          <div
            style={{
              fontSize: 78,
              fontWeight: 600,
              letterSpacing: "-0.04em",
              lineHeight: 1.04,
            }}
          >
            See every port.
          </div>
          <div
            style={{
              fontSize: 78,
              fontWeight: 600,
              letterSpacing: "-0.04em",
              lineHeight: 1.04,
              color: "#a1a1a1",
            }}
          >
            Stop it by name.
          </div>
        </div>
        <div
          style={{
            display: "flex",
            justifyContent: "space-between",
            alignItems: "center",
            fontSize: 26,
            color: "#a1a1a1",
          }}
        >
          <div style={{ display: "flex" }}>Zero config. One static binary.</div>
          <div style={{ display: "flex", color: "#4ade80" }}>ports.tools</div>
        </div>
      </div>
    ),
    {
      ...size,
      fonts: [
        { name: "Geist", data: geistRegular, weight: 400, style: "normal" },
        { name: "Geist", data: geistSemiBold, weight: 600, style: "normal" },
      ],
    },
  );
}
