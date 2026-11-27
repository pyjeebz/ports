// Single source of truth for the copy the landing page, docs, OG images,
// manifest and sitemap share.

export const SITE = {
  name: "ports",
  domain: "https://ports.tools",
  tagline: "Every dev server on your machine, by name",
  description:
    "ports finds every dev server on your machine and gives it a stable name — http://shop.localhost instead of http://localhost:3000 — however it was started. See what's running, reach it by name, stop it safely. For people and coding agents.",
  links: {
    docs: "/docs/commands",
    commands: "/docs/commands",
    github: "https://github.com/pyjeebz/ports",
    releases: "https://github.com/pyjeebz/ports/releases",
    issues: "https://github.com/pyjeebz/ports/issues",
    install: "/install",
  },
  installCurl: "curl -fsSL https://ports.tools/install | sh",
  installNpm: "npm i -g @pyjeebz/ports",
} as const;
