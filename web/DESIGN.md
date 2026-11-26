# Design

ports.tools reads like documentation, not a marketing page: one column of
prose, real terminal output, and a table of contents beside it. If a
section wouldn't belong in the docs, it doesn't belong on the page.

## System

- **Type:** Geist Sans for prose, Geist Mono for anything typed or printed
  (`geist` package). Headings tighten as they grow (`-0.025em` on the
  h1, `-0.015em` on h2); body text stays at 15px/28px.
- **Colour:** shadcn's neutral tokens in `app/globals.css`, light and dark.
  Ink on white, one muted grey for secondary text, hairline borders.
  Colour appears only where it means something: `--added` / `--removed`
  for diffs and outcomes, `--live` for a running server, `--signal` for
  the request in a diagram. Fumadocs reads the same tokens
  (`fumadocs-ui/css/shadcn.css`), so the docs match the landing page.
- **Surfaces:** code blocks and diagrams are bordered cards with a quiet
  header (`bg-muted/40`) naming what they show, plus a copy or replay
  button. The nav is translucent (`backdrop-blur`) so the page scrolls
  under it.

## Motion

Motion explains; it never decorates. The two diagrams in "How it works"
are the only things that move on their own.

- **Explainers** (`route-flow.tsx`, `port-move.tsx`) step through a story
  with `useSteps`. They play only while on screen and the tab is visible,
  and Replay starts them over.
- **Each change** is a transition of 300ms or less on the strong
  ease-out (`--ease-out: cubic-bezier(0.23, 1, 0.32, 1)`). New values
  enter with `.enter`: 4px up, out of a 2px blur, using `@starting-style`.
- **Feedback:** pressable things scale to `0.96`–`0.97` on `:active`. The
  copy icon cross-fades to a check through a 2px blur. The install tabs'
  underline slides to the chosen tab.
- **Reduced motion:** movement and blur drop away; opacity changes stay.
- **Never** use `transition: all`, `ease-in`, animation on anything
  keyboard-driven, or motion on things a reader sees many times a
  minute.

## Content

Every command and every line of output on the page is what the CLI
really prints. Check it against a real build when either side changes.
