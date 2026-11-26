# ports.tools

The marketing site and docs for [ports](../README.md) — Next.js 16 (App
Router), Tailwind v4, and Fumadocs, deployed on Vercel.

```sh
npm install
ports run -- npm run dev   # http://ports.localhost, with `ports proxy start` running
npm run build              # fumadocs-mdx, then next build
npm run lint
```

## Where things live

| Path | What |
| ---- | ---- |
| `content/home/index.mdx` | Getting Started — the landing page at `/`, written as a doc |
| `content/docs/*.mdx` | The docs at `/docs` |
| `app/(site)/` | The shared notebook layout, and the pages that render the MDX |
| `components/site/` | Page parts for MDX — install, the two diagrams, code tabs, promises |
| `components/ui/` | shadcn components |
| `app/md/`, `app/llms.txt/` | Every page as Markdown, for Copy page and agents |
| `lib/site.ts` | Copy the site shares — tagline, links, the install command |
| `lib/og.tsx`, `app/api/og/` | Open Graph cards (Satori, set in Geist) |
| `app/install/` | `ports.tools/install` — serves `script.sh` |

## Before you change things

- **Design:** `DESIGN.md` — Vercel's Geist palette and type, shadcn, a
  landing page that is the first page of the docs, and the rules for the
  little motion there is.
- **Next.js:** this is Next 16 — read `AGENTS.md` first; `params`,
  `searchParams`, `cookies`, and `headers` are async.
- **Installer:** `app/install/script.sh` is a byte-identical copy of the
  repo-root `install.sh`. Edit both; CI fails when they drift.
- **Docs follow the binary:** command output, flags, and exit codes in
  `content/docs/` describe what `ports` actually prints — check against a
  real build when either side changes.
