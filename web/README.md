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
| `app/page.tsx` | The landing page: prose, terminal output, two diagrams |
| `components/landing/` | Its parts — code blocks, install tabs, the diagrams, nav and TOC |
| `content/docs/*.mdx` | The docs at `/docs`, rendered by Fumadocs |
| `lib/site.ts` | Copy and command data the landing renders |
| `lib/og.tsx`, `app/api/og/` | Open Graph cards (Satori, set in Geist) |
| `app/install/` | `ports.tools/install` — serves `script.sh` |

## Before you change things

- **Design:** `DESIGN.md` — Geist and shadcn's neutral tokens, a page that
  reads like the docs, and the rules for the little motion there is.
- **Next.js:** this is Next 16 — read `AGENTS.md` first; `params`,
  `searchParams`, `cookies`, and `headers` are async.
- **Installer:** `app/install/script.sh` is a byte-identical copy of the
  repo-root `install.sh`. Edit both; CI fails when they drift.
- **Docs follow the binary:** command output, flags, and exit codes in
  `content/docs/` describe what `ports` actually prints — check against a
  real build when either side changes.
