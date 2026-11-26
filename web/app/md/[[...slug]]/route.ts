import { notFound } from "next/navigation";
import { toPlainMarkdown } from "@/lib/markdown";
import { homeSource, source } from "@/lib/source";

// Every page as Markdown — /md for Getting Started, /md/docs/<page> for
// the docs — for "Copy page" and for agents reading the site.
export const revalidate = false;

export async function GET(_req: Request, { params }: RouteContext<"/md/[[...slug]]">) {
  const { slug = [] } = await params;
  const page = slug[0] === "docs" ? source.getPage(slug.slice(1)) : slug.length === 0 ? homeSource.getPage([]) : undefined;
  if (!page) notFound();
  const body = await page.data.getText("processed");
  return new Response(toPlainMarkdown(`# ${page.data.title}\n\n${body}`), {
    headers: { "Content-Type": "text/markdown; charset=utf-8" },
  });
}

export function generateStaticParams() {
  return [{ slug: [] }, ...source.getPages().map((p) => ({ slug: ["docs", ...p.slugs] }))];
}
