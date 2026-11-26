import { SITE } from "@/lib/site";

// The page's interactive parts, as what an agent reading Markdown needs
// from them: the install command, a sentence for each diagram, the code
// inside the tabs, the promises as a list.
const REPLACEMENTS: [RegExp, string][] = [
  [/<Install\s*\/>/g, "```bash\n" + SITE.installCurl + "\n```"],
  [
    /<RouteFlow\s*\/>/g,
    "_Diagram: a request for shop.localhost reaches the router on :80, which matches the Host to the shop route and passes it to :3000; a request for evil.example is refused with 421._",
  ],
  [
    /<PortMove\s*\/>/g,
    "_Diagram: the shop server restarts and comes back on :3001; the browser keeps http://shop.localhost the whole time, and the router logs `~ http://shop.localhost → :3001 Next.js`._",
  ],
  [
    /<Promises\s*\/>/g,
    [
      "- **Listen beyond this machine.** The router binds 127.0.0.1 and ::1, and answers only the .localhost names it knows.",
      "- **Run what you didn't ask for.** Only `ports run` starts a process, and only the command you gave it.",
      "- **Route someone else's server.** It routes the dev servers of the user it runs as, and stops only yours.",
      "- **Phone home.** One static binary. No account, no cloud, no telemetry.",
    ].join("\n"),
  ],
  // tabs: keep each panel's code block under its tab's name
  [/<CodeTabs[^>]*>|<\/CodeTabs>|<\/?CodeTabsList>|<\/CodeTabPanel>/g, ""],
  [/<CodeTab value="[^"]*">\s*([^<]*?)\s*<\/CodeTab>/g, ""],
  [/<CodeTabPanel value="([^"]*)">/g, "**$1**"],
  // fumadocs' heading ids: "## Install [#install]"
  [/^(#{1,6} .*?) \[#[^\]]+\]$/gm, "$1"],
];

export function toPlainMarkdown(text: string): string {
  let out = text;
  for (const [re, to] of REPLACEMENTS) out = out.replace(re, to);
  return dedentPanels(out).replace(/\n{3,}/g, "\n\n").trim() + "\n";
}

// dedentPanels undoes the indentation JSX nesting gave the tab panels: a
// panel runs from its "**name**" label to the next label or heading, and
// loses as much indentation as its code fence had.
function dedentPanels(text: string): string {
  const lines = text.split("\n").map((l) => (l.trim() === "" ? "" : l));
  let indent = -1;
  return lines
    .map((line) => {
      if (/^\s*\*\*[\w-]+\*\*$/.test(line)) {
        indent = -1;
        return line.trim();
      }
      if (/^#{1,6} /.test(line)) indent = -1;
      const fence = line.match(/^(\s*)```/);
      if (fence && indent < 0 && fence[1].length > 0) indent = fence[1].length;
      return indent > 0 && line.startsWith(" ".repeat(indent)) ? line.slice(indent) : line;
    })
    .join("\n");
}
