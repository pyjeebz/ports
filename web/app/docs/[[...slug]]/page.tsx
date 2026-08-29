import type { Metadata } from "next";
import { notFound } from "next/navigation";
import { DocsBody, DocsDescription, DocsPage, DocsTitle } from "fumadocs-ui/layouts/docs/page";

import { source } from "@/lib/source";
import { getMDXComponents } from "@/mdx-components";

export function generateStaticParams() {
  return source.generateParams();
}

export async function generateMetadata(
  props: PageProps<"/docs/[[...slug]]">,
): Promise<Metadata> {
  const params = await props.params;
  const page = source.getPage(params.slug);
  if (!page) return {};

  // Per-doc OG card, rendered by /api/og (a catch-all can't host its own
  // opengraph-image route).
  const og = new URLSearchParams({
    title: page.data.title,
    desc: page.data.description ?? "",
  });

  return {
    title: page.data.title,
    description: page.data.description,
    openGraph: {
      title: `${page.data.title} · ports docs`,
      description: page.data.description,
      images: [`/api/og?${og.toString()}`],
    },
  };
}

export default async function DocPage(props: PageProps<"/docs/[[...slug]]">) {
  const params = await props.params;
  const page = source.getPage(params.slug);
  if (!page) notFound();

  const MDX = page.data.body;

  return (
    <DocsPage toc={page.data.toc}>
      <DocsTitle>{page.data.title}</DocsTitle>
      <DocsDescription>{page.data.description}</DocsDescription>
      <DocsBody>
        <MDX components={getMDXComponents()} />
      </DocsBody>
    </DocsPage>
  );
}