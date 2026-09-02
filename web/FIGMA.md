# Design-system rules for ports.tools (Napkin)

How to build or translate designs into this codebase. Since 2026-09 the site
runs the **Napkin** system (napkin.lol aesthetic: wobbly pen lines on paper,
messy organic UI around rigid technical data). Any Figma import, new section,
or new primitive must follow these rules. Generated reference code is never a
deliverable — adapting it to these patterns is the job.

## Stack (what you're integrating into)

- **Next.js 16.2.10, App Router, React 19.2** — ⚠️ per `web/AGENTS.md`, this
  Next version differs from training data; read `node_modules/next/dist/docs/`
  before writing route/param code. Notably: page props are async
  (`const params = await props.params`).
- **Tailwind CSS v4** — CSS-first config. There is **no `tailwind.config.js`;
  do not create one.** Theme extension happens in `app/globals.css` via
  `@theme inline`. PostCSS plugin: `@tailwindcss/postcss`.
- **Fumadocs** (`fumadocs-core`, `fumadocs-mdx`, `fumadocs-ui`) for the docs
  at `/docs`, MDX in `content/docs/`, generated `.source/` dir read by
  `lib/source.ts`. Its `--color-fd-*` tokens are retinted to the paper
  palette in `globals.css` — adjust there, never per-component.
- **Fonts** via `next/font/google` in `app/layout.tsx`, three families only:
  - **Architects Daughter** (`--font-architects` → `font-display`) — display
    headings, the Sharpie voice.
  - **Kalam** (`--font-kalam` → `font-sans`) — body prose, nav, buttons.
  - **JetBrains Mono** (`--font-jetbrains` → `font-mono`) — STRICTLY code,
    terminal output, tabular data, tags, dates. Never prose.
  Never import other fonts.
- **drawably** — draws the wobbly pen strokes around real HTML controls.
  React wrappers live in `drawably/react` (`DrawablyButton`, `DrawablyCard`,
  `DrawablyBadge`, `DrawablyInput`, `DrawablyCheckbox`, `DrawablyDivider`,
  `DrawablyHighlight`); plain-DOM `drawablyButton(el, opts)` from
  `"drawably"` for anchors (the React wrapper hardcodes `<button>`). Options:
  `seed/roughness/boil/stroke/fill/paper/width`. The wrappers carry no
  `"use client"` — the primitive owns that directive. Never fake sketchy
  borders with CSS `border-radius` hacks, never use Rough.js or canvas.
- **cva + clsx + tailwind-merge** for component variants; `cn()` from
  `lib/utils.ts` is the class-merge helper.

## Design tokens

Everything lives in `app/globals.css`. One file, two layers:

```css
/* 1. Napkin values on :root — the single source of truth */
:root {
  --paper: #f4f1ea;        /* page ground */
  --charcoal: #2c2c2a;     /* ink */
  --charcoal-light: #595955;
  --faint: #8a877d;
  --highlighter-yellow: #ffe27a;  /* also pink/blue/green */
  --accent: #3d63dd;
  /* legacy names (--background, --foreground, --panel, --line, …) are
     remapped onto this palette so untranslated files still render */
}

/* 2. @theme inline exposes them as Tailwind utilities:
   --charcoal → text-charcoal, bg-charcoal, border-charcoal, … */
@theme inline {
  --color-paper: var(--paper);
  --font-display: var(--font-architects), cursive;
  --font-sans: var(--font-kalam), cursive;
  --font-mono: var(--font-jetbrains), ui-monospace, monospace;
}
```

**Rules:**

- Map design colors onto these tokens — a `#2C2C2A` fill is
  `text-charcoal`, not `text-[#2c2c2a]`. New colors become tokens first.
- **The site is light-only.** `color-scheme: light`, `RootProvider
  theme={{ defaultTheme: "light", enableSystem: false }}`. Never emit
  dark-mode variants, `dark:` classes, or theme toggles.
- The root carries `.paper-texture` (set once in `app/layout.tsx`'s body) —
  do not re-apply it per section.
- `--drawably-stroke` / `--drawably-fill` on `:root` default the pen ink;
  pass overrides only when a component needs a different pen.
- No Style Dictionary pipeline exists. Tokens are hand-maintained CSS vars —
  treat `globals.css` as the registry.

## Component library (use these, don't reinvent)

All in `components/ui/`, shadcn-shaped (cva + `data-slot` + `cn()`).
Rotation keys are **string unions** (`rotate="1"`, not `rotate={1}`).

- `<Card rotate="-2..2" bg="white|paper|muted">` — drawably-outlined card.
  Vary rotation when placing cards together; identical rotations look
  stamped, not drawn.
- `<Button variant="primary|secondary|ghost">` / `<ButtonLink>` — pen-stroked
  actions; ghost stays bare text.
- `<Badge color="yellow|pink|blue|green|white|transparent" rotate>` — mono
  tags.
- `<Highlighter color="yellow|pink">` — the replacement for bold/emphasis in
  display text.
- `<DrawnLine>` — hand-drawn divider; also replaces `border-t` section rules.
- `<Input>`, `<Checkbox>` — pen-drawn form controls.
- `<Tape color="clear|yellow|pink">` — washi strip, absolutely positioned by
  the caller, always `aria-hidden`.
- `<Stamp ink="red|blue|charcoal" rotate>` — stamped verdict labels.
- `<Arrow>` — hand-drawn connector between pipeline nodes.

`components/site/` holds marketing sections (server components by default;
`"use client"` only where state/effects are genuinely needed). Every
component documents its contract in a JSDoc block — match that comment
density; the codebase explains *why*, not *what*.

## Typography

Display sizes use `font-display` with `clamp()`, e.g. the section template
(`components/site/section-head.tsx`):

```tsx
<p className="eyebrow">{kicker}</p>
<h2 className="mt-4 font-display text-[clamp(1.6rem,3.2vw,2.5rem)] text-charcoal">{title}</h2>
<p className="mt-4 max-w-[62ch] text-[14.5px] leading-relaxed text-charcoal-light">{sub}</p>
```

Reusable micro-typography classes live in `globals.css` (`.eyebrow`,
`.readout`, `.code-shell`) — prefer them over re-deriving stacks.

## Styling approach

- **Tailwind utilities inline**, no CSS Modules, no styled-components.
  Global styles are confined to `app/globals.css` (tokens, micro-typography,
  keyframes, `.drawably-host` font override). Don't add component CSS
  elsewhere.
- Arbitrary values are idiomatic here — `max-w-[1160px]`, `text-[12.5px]`,
  `w-[9ch]` (terminal column widths are in `ch`, mirroring the CLI's table).
- Highlighter washes (`bg-highlighter-yellow/60` etc.) mark interactive or
  emphasized states; hover states may rotate slightly (`group-hover:-rotate-1`).

### Motion — the pattern any prototype must respect

Motion is **armed, not autoplaying**: sections server-render their final
frame; a client island sets `data-motion="on"` (only when in view and motion
is allowed). Keyframes live in `globals.css`, gated as
`[data-motion="on"] [data-appear] { … }`. The gate is `useMotionAllowed()`
from `lib/motion.ts` (respects `prefers-reduced-motion`, plus a global
reduced-motion CSS block as the second belt).

Animate only opacity, color, and small transforms (≤6px translates, small
rotations), never layout properties; express it as keyframes + a `data-*`
arming attribute, not JS-driven frame animation. Interaction-driven changes
(click → state change) are fine client-side — `four-commands.tsx`'s typing
terminal is the model.

### Responsive

- Desktop-first; container is `mx-auto max-w-[1160px] px-8`
  (`max-[720px]:px-6` on the nav).
- Breakpoint via `lg:` grid columns (`lg:grid-cols-[1fr_560px]`), arbitrary
  max-width variants (`max-[720px]:h-[60px]`), and `clamp()` for fluid type.
  No mobile-specific route or separate layout.

## Assets & icons

- `public/` holds only `logo.svg` (favicon). There is no image pipeline in
  active use — **icons are inline SVG React components**, e.g.
  `components/site/ports-logo.tsx`: `viewBox` 0 0 100 100,
  `fill/stroke: currentColor`, `aria-hidden="true"`, sizing via a
  `className` prop, `useId()` for any mask/gradient ids.
- OG images are generated dynamically by `app/api/og/route.tsx` +
  `lib/og.tsx` (paper palette, Geist as the Satori font — the hand-drawn
  fonts ship only as woff2 via next/font, so the OG cards keep Geist). Don't
  export static social images.

## Project structure (where things go)

```
web/
├── app/                  # App Router: page.tsx (landing), docs/, install/,
│   │                     # api/{og,search}, layout.tsx (fonts, RootProvider),
│   │                     # globals.css (tokens + motion + paper texture)
├── components/
│   ├── ui/               # Napkin primitives (Card, Button, Badge, Tape, …)
│   └── site/             # marketing sections (hero, four-commands, …)
├── content/docs/         # fumadocs MDX pages
├── lib/                  # site.ts (shared demo data/contact roster), motion.ts,
│                         # utils.ts (cn), source.ts (fumadocs loader), og.tsx
└── public/               # logo.svg only
```

- `lib/site.ts` is the shared data roster (ports demo data, SITE metadata) —
  content shown in multiple sections becomes a typed structure there, not
  duplicated literals (see `PortsTable` rendering one `Contact[]` at two
  sizes so no two renderings drift).
- Import alias: `@/*` → `web/*`.

## Checklist for every design import

1. Colors → Napkin tokens (`--paper`/`--charcoal`/highlighters); light-only.
2. Fonts → Architects Daughter (display), Kalam (body), JetBrains Mono (code
   and tabular data only).
3. Controls → `components/ui/` primitives with drawably chrome; a new
   primitive goes there in the cva + `data-slot` + `cn()` shape; page
   sections → `components/site/` with a JSDoc contract comment.
4. Rotation on cards/badges varies between neighbors; rotations stay within
   ±3°.
5. Motion → keyframes in `globals.css` armed by `data-motion="on"`;
   reduced-motion safe by construction.
6. Icons → inline `currentColor` SVG components.
7. Layout → `max-w-[1160px] px-8` container, `lg:` grids, `clamp()` type.
8. `"use client"` only when state/effects are genuinely needed.
9. Verify with `npm run build` in `web/` (eslint: `npx eslint <files>`).