import { DocsLayout } from "fumadocs-ui/layouts/notebook";
import { baseOptions } from "@/lib/layout.shared";
import { siteTree } from "@/lib/source";

// One layout for the landing page and the docs: a top bar across the
// page, the sidebar under it, and the page between it and "On this page".
export default function SiteLayout({ children }: { children: React.ReactNode }) {
  return (
    <DocsLayout {...baseOptions()} tree={siteTree()} nav={{ ...baseOptions().nav, mode: "top" }}>
      {children}
    </DocsLayout>
  );
}
