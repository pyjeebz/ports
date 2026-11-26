import { defineConfig, defineDocs } from "fumadocs-mdx/config";
import { rehypeCodeDefaultOptions } from "fumadocs-core/mdx-plugins";
import type { ShikiTransformer } from "shiki";

export const docs = defineDocs({
  dir: "content/docs",
  docs: { postprocess: { includeProcessedMarkdown: true } },
});

// The Getting Started page at "/": an MDX document like the docs, so it
// reads like one and its Markdown can be handed to an agent.
export const home = defineDocs({
  dir: "content/home",
  docs: { postprocess: { includeProcessedMarkdown: true } },
});

// In a `console` block a line that starts with "$ " is a command, and
// keeps its highlighting; every other line is what the command printed,
// set in the muted grey (`.line-output` in globals.css) — so the eye
// finds what to type before it reads what came back.
const consoleOutput: ShikiTransformer = {
  name: "console-output",
  line(node) {
    if (this.options.lang !== "console") return;
    const text = node.children
      .map((c) => (c.type === "element" ? c.children.map((t) => ("value" in t ? t.value : "")).join("") : ""))
      .join("");
    if (!text.startsWith("$ ")) this.addClassToHast(node, "line-output");
  },
};

export default defineConfig({
  mdxOptions: {
    rehypeCodeOptions: {
      ...rehypeCodeDefaultOptions,
      transformers: [...(rehypeCodeDefaultOptions.transformers ?? []), consoleOutput],
    },
  },
});
