// Single source of truth for the copy the landing page, docs, OG images,
// manifest and sitemap share.

export const SITE = {
  name: "ports",
  domain: "https://ports.tools",
  tagline: "Every dev server on your machine, by name",
  description:
    "ports gives every dev server a stable name — http://shop.localhost instead of http://localhost:3000 — whether it started it or not. See what's running, reach it by name, stop it safely. For people and coding agents.",
  links: {
    docs: "/docs",
    commands: "/docs/commands",
    github: "https://github.com/pyjeebz/ports",
    releases: "https://github.com/pyjeebz/ports/releases",
    issues: "https://github.com/pyjeebz/ports/issues",
    install: "/install",
  },
  installCurl: "curl -fsSL https://ports.tools/install | sh",
} as const;

export type InstallMethod = {
  id: string;
  label: string;
  lines: string[];
};

export const INSTALL_METHODS: InstallMethod[] = [
  { id: "curl", label: "curl", lines: [SITE.installCurl] },
  { id: "brew", label: "Homebrew", lines: ["brew install pyjeebz/tap/ports"] },
  {
    id: "scoop",
    label: "Scoop",
    lines: [
      "scoop bucket add pyjeebz https://github.com/pyjeebz/scoop-bucket",
      "scoop install ports",
    ],
  },
  { id: "go", label: "Go", lines: ["go install github.com/pyjeebz/ports@latest"] },
];
