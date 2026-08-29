import type { MetadataRoute } from "next";

import { SITE } from "@/lib/site";
import { source } from "@/lib/source";

export default function sitemap(): MetadataRoute.Sitemap {
  const lastModified = new Date();
  return [
    { url: SITE.domain, lastModified, priority: 1 },
    ...source.getPages().map((page) => ({
      url: `${SITE.domain}${page.url}`,
      lastModified,
    })),
  ];
}