import { createMDX } from "fumadocs-mdx/next";

/** fumadocs-mdx is ESM-only — this config must stay .mjs. */
export default createMDX()({
  // Getting Started, at "/", is the first page of the docs
  async redirects() {
    return [{ source: "/docs", destination: "/", permanent: false }];
  },
});
