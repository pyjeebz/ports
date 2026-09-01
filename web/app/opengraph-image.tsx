import { makeOgImage, ogContentType, ogImageAlt, ogSize } from "@/lib/og";

export const alt = ogImageAlt(["See what's running.", "Know what it belongs to."]);
export const size = ogSize;
export const contentType = ogContentType;

export default function OpengraphImage() {
  return makeOgImage({
    lines: ["See what's running.", "Know what it belongs to."],
    footnote: "One command. Full context.",
  });
}