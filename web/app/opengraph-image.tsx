import { makeOgImage, ogContentType, ogImageAlt, ogSize } from "@/lib/og";

export const alt = ogImageAlt(["See every port.", "Stop it by name."]);
export const size = ogSize;
export const contentType = ogContentType;

export default function OpengraphImage() {
  return makeOgImage({
    lines: ["See every port.", "Stop it by name."],
    footnote: "Zero config. One static binary.",
  });
}