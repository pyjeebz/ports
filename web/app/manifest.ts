import type { MetadataRoute } from "next";

import { SITE } from "@/lib/site";

export default function manifest(): MetadataRoute.Manifest {
  return {
    name: "ports",
    short_name: "ports",
    description: SITE.description,
    start_url: "/",
    display: "standalone",
    background_color: "#05080a",
    theme_color: "#05080a",
    icons: [{ src: "/icon.svg", sizes: "any", type: "image/svg+xml" }],
  };
}