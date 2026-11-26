import { loader } from "fumadocs-core/source";
import type * as PageTree from "fumadocs-core/page-tree";

// Relative import: the `@/` alias fails to resolve into the dot-prefixed
// `.source` output directory.
import { docs, home } from "../.source/server";

export const source = loader({
  baseUrl: "/docs",
  source: docs.toFumadocsSource(),
});

export const homeSource = loader({
  baseUrl: "/",
  source: home.toFumadocsSource(),
});

// siteTree is the sidebar every page shares: Getting Started, then the docs.
export function siteTree(): PageTree.Root {
  const docsTree = source.getPageTree();
  return {
    ...docsTree,
    children: [{ type: "page", name: "Getting Started", url: "/" }, ...docsTree.children],
  };
}
