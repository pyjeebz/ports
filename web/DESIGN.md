# Design

ports.tools is documentation first. The landing page is **Getting
Started**, the first page of the docs, in the same layout: a top bar, the
docs sidebar, one column of prose and real terminal output, and "On this
page" beside it. If a section wouldn't belong in the docs, it doesn't
belong on the page.

## System

- **Layout:** Fumadocs' notebook layout (`app/(site)/layout.tsx`), shared by
  `/` and `/docs/*`. Getting Started is MDX (`content/home/index.mdx`),
  like the docs (`content/docs/*.mdx`).
- **Colour:** Vercel's Geist palette in `app/globals.css`. Light is
  `#fff` / `#171717` with a `#fafafa` top bar; dark is `#000` / `#ededed`.
  Borders are alpha (8% black, 14% white). shadcn's variables carry it,
  and Fumadocs reads them (`fumadocs-ui/css/shadcn.css`), so the docs
  chrome, the page and the components match. Colour only where it
  means something: blue for a request, green for an answer, red for a
  refusal.
- **Type:** Geist Sans and Geist Mono. The h1 is 40/48 and the h2 24/32,
  both weight 450 and tightening as they grow. Body text is 16/28 at 90%
  ink; inline code is 13px.
- **Components:** shadcn (`components/ui/`): Button, Badge, Card, Tabs
  (line variant), Tooltip, Kbd, Separator. Use them before writing new
  ones; `cn` comes from `@/lib/utils`. Page parts live in
  `components/site/` and are registered for MDX in `mdx-components.tsx`.
- **Code:** fenced blocks in MDX, highlighted by Shiki. Use `console` for
  terminal sessions: a line starting with `$ ` is a command and keeps its
  highlighting, and every other line is output, muted by the
  `console-output` transformer in `source.config.ts`.

## Motion

Motion explains; it never decorates. The two diagrams in "How it works"
are the only things that move on their own.

- **Explainers** (`route-flow.tsx`, `port-move.tsx`) step through a story
  with `useSteps`. They play only while on screen and the tab is visible,
  and Replay starts them over.
- **Each change** is a transition of 300ms or less on the strong ease-out
  (`--ease-out: cubic-bezier(0.23, 1, 0.32, 1)`). New values enter with
  `.enter`: 4px up, out of a 2px blur, using `@starting-style`. Keep
  `@starting-style` at the top level: nested under `@media`, it made the
  CSS pipeline drop every rule after it.
- **Feedback:** buttons scale to `0.96` on `:active`. The copy icon
  cross-fades to a check through a 2px blur, and its tooltip changes its
  word.
- **Reduced motion:** movement and blur drop away; opacity stays.
- **Never** use `transition: all`, `ease-in`, or motion on anything
  keyboard-driven or seen many times a minute.

## For agents

Every page has a Markdown twin at `/md` (Getting Started) and
`/md/docs/<page>`, which Copy page uses. `/llms.txt` indexes them.
`lib/markdown.ts` turns the page's components into what an agent needs:
the install command, a sentence per diagram, the code inside the tabs.
When you add a component to the MDX, give it a line there.

## Content

Every command and every line of output is what the CLI really prints.
Check it against a real build when either side changes.
