import { notFound } from "next/navigation";
import {
  DocsBody,
  DocsPage,
  DocsTitle,
  MarkdownCopyButton,
  ViewOptionsPopover,
} from "fumadocs-ui/layouts/notebook/page";
import { homeSource } from "@/lib/source";
import { SITE } from "@/lib/site";
import { getMDXComponents } from "@/mdx-components";

/**
 * Getting Started — the landing page, written as the first page of the
 * docs: the same layout, the same type, an "On this page" list, and a
 * Copy page button that hands the whole thing to an agent as Markdown.
 */
export default function Home() {
  const page = homeSource.getPage([]);
  if (!page) notFound();
  const MDX = page.data.body;

  return (
    <DocsPage toc={page.data.toc} breadcrumb={{ enabled: false }} footer={{ enabled: false }}>
      <div className="flex flex-wrap items-start justify-between gap-4">
        <DocsTitle>{page.data.title}</DocsTitle>
        <div className="flex items-center gap-1.5 pt-1.5">
          <MarkdownCopyButton markdownUrl="/md" />
          <ViewOptionsPopover markdownUrl="/md" githubUrl={`${SITE.links.github}/blob/main/web/content/home/index.mdx`} />
        </div>
      </div>
      <DocsBody>
        <MDX components={getMDXComponents()} />
      </DocsBody>
    </DocsPage>
  );
}
