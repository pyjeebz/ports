// Single source of truth for site copy and the localhost roster the
// landing page, docs, OG images and sitemap all render from.

export const SITE = {
  name: "ports",
  domain: "https://ports.tools",
  tagline: "See what's running on your machine, by project.",
  description:
    "A zero-config CLI that shows every listening port with its framework, project, and uptime — and lets you stop things by name.",
  links: {
    docs: "/docs",
    commands: "/docs/commands",
    github: "https://github.com/pyjeebz/ports",
    releases: "https://github.com/pyjeebz/ports/releases",
    install: "/install",
  },
  installCurl: "curl -fsSL https://ports.tools/install | sh",
} as const;

export type ContactKind = "dev" | "db" | "cache" | "other";

export type Contact = {
  port: number;
  framework: string;
  project: string;
  uptime: string;
  kind: ContactKind;
};

// The localhost roster the radar, flap board and docs snippets draw.
// Positions derive deterministically from the port (see radar/use-sweep).
export const CONTACTS: Contact[] = [
  { port: 3000, framework: "Next.js", project: "~/projects/shop", uptime: "14m", kind: "dev" },
  { port: 5432, framework: "PostgreSQL", project: "docker: shop-db", uptime: "2d", kind: "db" },
  { port: 6379, framework: "Redis", project: "docker: shop-cache", uptime: "2d", kind: "cache" },
  { port: 8000, framework: "FastAPI", project: "~/projects/shop", uptime: "14m", kind: "dev" },
  { port: 5173, framework: "Vite", project: "~/projects/admin", uptime: "3h", kind: "dev" },
  { port: 9229, framework: "Node", project: "~/projects/api", uptime: "26m", kind: "other" },
] as const;

export const COMMANDS = [
  {
    cmd: "ports",
    href: "/docs/commands#cmd-list",
    desc: "Every listening port with its framework, project, and uptime.",
  },
  {
    cmd: "ports find <name>",
    href: "/docs/commands#cmd-find",
    desc: "Locate a service by name, port, or project.",
  },
  {
    cmd: "ports free <port>",
    href: "/docs/commands#cmd-free",
    desc: "Identify what holds a port and stop it, gracefully.",
  },
  {
    cmd: "ports kill <name>",
    href: "/docs/commands#cmd-kill",
    desc: "Stop a project or service by name.",
  },
  { cmd: "--json", href: "/docs/commands#flags", desc: "Machine-readable output — the seam for scripts and agents." },
  { cmd: "--force", href: "/docs/commands#flags", desc: "Skip the SIGTERM grace period and kill immediately." },
] as const;

export const CAPABILITIES = [
  {
    title: "Framework detection",
    body: "Next.js, Vite, FastAPI, PostgreSQL, Redis — the process name is replaced by what the process is.",
  },
  {
    title: "Project grouping",
    body: "Ports roll up under their project, walked from the process cwd to the nearest repo or marker file.",
  },
  {
    title: "Docker unwrap",
    body: "docker-proxy rows resolve to the real container and compose service behind them, over the socket.",
  },
  {
    title: "WSL2 awareness",
    body: "Windows-side listeners are named as such — never offered to the kill path.",
  },
  {
    title: "JSON output",
    body: "`ports --json` is the same scan, structured. Build dashboards on it; agents read it.",
  },
  {
    title: "Safe stops",
    body: "SIGTERM first, a grace period, then SIGKILL. Containers stop through the daemon, never the proxy.",
  },
] as const;

export const PRINCIPLES = [
  {
    title: "Observer, never a runner",
    body: "ports never starts, restarts, supervises, or installs anything. Read-only plus a targeted kill — nothing else.",
  },
  {
    title: "Zero config",
    body: "No manifest, no daemon, no declarations. Useful sixty seconds after install, on a machine in whatever state it is in.",
  },
  {
    title: "Local-first",
    body: "One static binary. No cloud, no accounts, no telemetry. Everything is inferred on your machine, from your machine.",
  },
] as const;

export type InstallMethod = {
  id: string;
  label: string;
  os: string;
  lines: string[];
};

export const INSTALL_METHODS: InstallMethod[] = [
  {
    id: "brew",
    label: "Homebrew",
    os: "macOS / Linux",
    lines: ["brew install pyjeebz/tap/ports"],
  },
  {
    id: "curl",
    label: "curl",
    os: "macOS / Linux",
    lines: [SITE.installCurl],
  },
  {
    id: "scoop",
    label: "Scoop",
    os: "Windows",
    lines: [
      "scoop bucket add pyjeebz https://github.com/pyjeebz/scoop-bucket",
      "scoop install ports",
    ],
  },
] as const;

// Flap-board scenarios: the command, the line that types it, and how each
// contact's row resolves when it runs. Kept here so the board data cannot
// drift from CONTACTS.
export type FlapScenario = {
  id: string;
  typed: string;
  note: string;
  outcomes: Record<number, { state: "stopped" | "held"; label: string }>;
};

export const FLAP_SCENARIOS: FlapScenario[] = [
  {
    id: "kill",
    typed: "ports kill shop",
    note: "term first, 2s grace, then SIGKILL — the docker row stops through the daemon",
    outcomes: {
      3000: { state: "stopped", label: "✓ stopped" },
      8000: { state: "stopped", label: "✓ stopped" },
      5432: { state: "stopped", label: "✓ stopped · container shop-db" },
      6379: { state: "stopped", label: "✓ stopped · container shop-cache" },
      5173: { state: "held", label: "— untouched · different project" },
      9229: { state: "held", label: "— untouched · different project" },
    },
  },
  {
    id: "find",
    typed: "ports find postgres",
    note: "search by name, port, or project across the whole scan",
    outcomes: {
      5432: { state: "stopped", label: "★ match · docker: shop-db" },
      3000: { state: "held", label: "" },
      8000: { state: "held", label: "" },
      6379: { state: "held", label: "" },
      5173: { state: "held", label: "" },
      9229: { state: "held", label: "" },
    },
  },
  {
    id: "free",
    typed: "ports free 3000",
    note: "one port, the process holding it, and a graceful stop",
    outcomes: {
      3000: { state: "stopped", label: "✓ stopped · next-server" },
      5432: { state: "held", label: "" },
      6379: { state: "held", label: "" },
      8000: { state: "held", label: "" },
      5173: { state: "held", label: "" },
      9229: { state: "held", label: "" },
    },
  },
] as const;