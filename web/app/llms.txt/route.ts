import { SITE } from "@/lib/site";
import { source } from "@/lib/source";

// llms.txt: what ports is, and where each page lives as Markdown.
export const revalidate = false;

export function GET() {
  const docs = source
    .getPages()
    .map((p) => `- [${p.data.title}](${SITE.domain}/md/docs/${p.slugs.join("/")}): ${p.data.description ?? ""}`)
    .join("\n");
  const text = `# ports

> ${SITE.description}

- [Getting Started](${SITE.domain}/md): install, names, how the router works, ports run, guarded stops, MCP

## Docs

${docs}
`;
  return new Response(text, { headers: { "Content-Type": "text/plain; charset=utf-8" } });
}
