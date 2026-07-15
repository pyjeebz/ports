import type { Metadata } from "next";

import { DocsSidebar } from "@/components/docs/docs-sidebar";

export const metadata: Metadata = {
  title: { template: "%s — ports docs", default: "ports docs" },
};

export default function DocsLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div className="docs-wrap">
      <DocsSidebar />
      {children}
    </div>
  );
}
