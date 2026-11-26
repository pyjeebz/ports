import { makeOgImage, ogContentType, ogImageAlt, ogSize } from "@/lib/og";

export const alt = ogImageAlt(["Every dev server", "on your machine, by name."]);
export const size = ogSize;
export const contentType = ogContentType;

export default function OpengraphImage() {
  return makeOgImage({
    lines: ["Every dev server", "on your machine, by name."],
    footnote: "http://shop.localhost, not http://localhost:3000",
  });
}