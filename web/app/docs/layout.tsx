import { DocsLayout } from "fumadocs-ui/layouts/docs";

import { source } from "@/lib/source";

export default function RootDocLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <DocsLayout
      tree={source.getPageTree()}
      sidebar={{
        enabled: true,
      }}
    >
      {children}
    </DocsLayout>
  );
}