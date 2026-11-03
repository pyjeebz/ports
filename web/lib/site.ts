// Single source of truth for site copy and the localhost roster the
// landing page, docs, OG images and sitemap all render from.

export const SITE = {
  name: "ports",
  domain: "https://ports.tools",
  tagline: "See what's running. Know what it belongs to.",
  description:
    "A zero-config CLI that shows every listening port with its framework, project, and uptime — and lets you stop things by name.",
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

export type ContactKind = "dev" | "db" | "cache" | "other";

export type Contact = {
  port: number;
  framework: string;
  project: string;
  uptime: string;
  kind: ContactKind;
};

// projectLabel is the PROJECT cell the CLI's table prints: a project
// directory's name (shop), never its path — the path stays in the doctor
// header and the resolution tree, as in the CLI.
export function projectLabel(ct: Contact): string {
  if (ct.project.startsWith("docker: ") || !ct.project.includes("/")) return ct.project;
  return ct.project.slice(ct.project.lastIndexOf("/") + 1);
}

// The localhost roster the landing page, docs snippets and OG images draw.
export const CONTACTS: Contact[] = [
  { port: 3000, framework: "Next.js", project: "~/projects/shop", uptime: "14m", kind: "dev" },
  { port: 5432, framework: "PostgreSQL", project: "docker: shop-db", uptime: "2d", kind: "db" },
  { port: 6379, framework: "Redis", project: "docker: shop-cache", uptime: "2d", kind: "cache" },
  { port: 8000, framework: "FastAPI", project: "~/projects/shop", uptime: "14m", kind: "dev" },
  { port: 5173, framework: "Vite", project: "~/projects/admin", uptime: "3h", kind: "dev" },
  { port: 9229, framework: "Node", project: "~/projects/api", uptime: "26m", kind: "other" },
] as const;

// The trimmed roster the hero renders — the first four listeners, the
// table a real machine greets you with. The full six-row table lives in
// SHOWCASE.list and the one-binary section.
export const HERO_PORTS = [3000, 5432, 8000, 5173] as const;

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
    cmd: "ports stop <port|name>",
    href: "/docs/commands#cmd-stop",
    desc: "Stop what holds a port, or everything a name matches — gracefully.",
  },
  {
    cmd: "ports here",
    href: "/docs/commands#cmd-here",
    desc: "What's running for the project you're in — reuse it instead of starting a second.",
  },
  {
    cmd: "ports wait <port|name>",
    href: "/docs/commands#cmd-wait",
    desc: "Block until it listens, answers HTTP, or is gone.",
  },
  {
    cmd: "ports doctor [port]",
    href: "/docs/commands#cmd-doctor",
    desc: "Diagnose a port — or the whole machine — and say what to do next. Never acts.",
  },
  { cmd: "--json", href: "/docs/commands#flags", desc: "Machine-readable output — the seam for scripts and agents." },
  { cmd: "--sort", href: "/docs/commands#flags", desc: "Order the table by port, uptime, process, or project." },
  { cmd: "--force", href: "/docs/commands#flags", desc: "Skip the SIGTERM grace period and kill immediately." },
] as const;

export const PRINCIPLES = [
  {
    title: "Zero-config",
    body: "One static Go binary. No runtime, no manifest, no daemon — useful sixty seconds after install, on a machine in whatever state it is in.",
  },
  {
    title: "The observer contract",
    body: "ports never starts, supervises, or installs anything. It reports what is already listening — and stops it when you tell it to. Nothing else.",
  },
] as const;

// The PID hunt, as one falling sequence: five commands in, zero answers
// out. Rendered as the left half of the "stop hunting PIDs" section.
export const PID_HUNT = [
  { step: "lsof -i :3000" },
  { step: "PID 48391" },
  { step: "ps -p 48391" },
  { step: "node" },
  { step: "pwd?" },
  { step: "wait... which project?" },
] as const;

// What the hero machine walks when it scans — the four things ports
// actually does, in order, before the first row appears.
export const SCAN_STEPS = [
  "discovering listeners",
  "identifying processes",
  "resolving projects",
  "inspecting containers",
] as const;

// What a hero row expands into when the visitor clicks it — the resolution
// tree beneath the port, in reading order. PID matches the roster's 48391.
export const CONTACT_DETAILS: Record<number, readonly string[]> = {
  3000: ["Next.js", "~/projects/shop", "PID 48391", "listening for 14m"],
  5432: ["PostgreSQL", "docker: shop-db", "container: 8f3a91c", "listening for 2d"],
  8000: ["FastAPI", "~/projects/shop", "PID 48407", "listening for 14m"],
  5173: ["Vite", "~/projects/admin", "PID 48512", "listening for 3h"],
};

// Every listener's PID — what `ports free` reports when it stops a
// process by port. Matches CONTACT_DETAILS where the two overlap.
export const CONTACT_PIDS: Record<number, number> = {
  3000: 48391,
  5432: 48406,
  6379: 48419,
  8000: 48407,
  5173: 48512,
  9229: 48620,
};

// The answer beside it: one command, three services, full context.
// Output mirrors the SHOWCASE column widths so the two read as one CLI.
export const FIND_SHOP = {
  cmd: "ports find shop",
  output: [
    "PORT     PROCESS      PROJECT             UPTIME",
    "3000     Next.js      shop                14m",
    "8000     FastAPI      shop                14m",
    "5432     PostgreSQL   docker: shop-db     2d",
  ],
} as const;

// One port, four answers — the anatomy the giant numeral reveals in stages.
export const PORT_ANATOMY = [
  { label: "a port", value: "3000" },
  { label: "a process", value: "Next.js" },
  { label: "a project", value: "~/projects/shop" },
  { label: "uptime", value: "14m" },
] as const;

// The socket-to-service pipelines, drawn left to right. The second is the
// docker case — its resolved endpoint is a container, not a directory.
export const PIPELINES = [
  { nodes: [":3000", "Next.js", "~/projects/shop", "14 minutes"] },
  { nodes: [":5432", "PostgreSQL", "docker: shop-db", "2d"], docker: true },
] as const;

// The zero-footprint negations, one tracked row.
export const NEGATIONS = ["NO DAEMON", "NO CONFIG", "NO AGENT", "NO SUPERVISOR"] as const;

// The observer split: what the tool does on the left, what you do on the right.
export const OBSERVES = ["sees", "identifies", "maps", "reports"] as const;
export const YOU_DECIDE = ["free", "kill"] as const;

// The docker unwrap, as one resolution chain: what the OS reports vs.
// what ports resolves. The terminal beside it renders the real row.
export const DOCKER_TRACE = [
  { node: ":5432", label: "the port" },
  { node: "docker-proxy", label: "what the OS reports" },
  { node: "shop-db", label: "the container" },
  { node: "PostgreSQL", label: "what's actually running" },
] as const;

export type LedgerItem = { item: string; ok: boolean };

export const OBSERVER_LEDGER: LedgerItem[] = [
  { item: "starts anything", ok: false },
  { item: "supervises processes", ok: false },
  { item: "installs software", ok: false },
  { item: "stops only what you name", ok: true },
];

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

// Command showcase: one terminal loop per command. Kept beside the table
// copy so the rendered output cannot drift from the docs.
export type ShowcaseEntry = {
  id: string;
  cmd: string;
  title: string;
  desc: string;
  line: string;
  note?: string;
  output: string[];
};

export const SHOWCASE: ShowcaseEntry[] = [
  {
    id: "list",
    cmd: "ports",
    title: "The whole table",
    line: "See everything.",
    desc: "Every listening port with the context lsof never gives you.",
    output: [
      "PORT     PROCESS      PROJECT             UPTIME",
      "3000     Next.js      shop                14m",
      "5432     PostgreSQL   docker: shop-db     2d",
      "6379     Redis        docker: shop-cache  2d",
      "8000     FastAPI      shop                14m",
      "5173     Vite         admin               3h",
      "9229     Node         api                 26m",
    ],
  },
  {
    id: "find",
    cmd: "ports find postgres",
    title: "Find by name",
    line: "Find anything.",
    desc: "Case-insensitive substring match across framework, process, project, and container.",
    output: [
      "PORT     PROCESS      PROJECT             UPTIME",
      "5432     PostgreSQL   docker: shop-db     2d",
    ],
  },
  {
    id: "free",
    cmd: "ports stop 3000",
    title: "Free by port",
    line: "Free a port.",
    desc: "The process holding a port, identified and stopped gracefully.",
    output: [
      "✓ killed Next.js (pid 48391) — port 3000 is free",
    ],
  },
  {
    id: "kill",
    cmd: "ports stop shop",
    title: "Kill by project",
    line: "Stop the project.",
    desc: "Stops every process that belongs to a named project — containers through the daemon, never docker-proxy.",
    note: "Add --force to skip the SIGTERM grace period and kill immediately.",
    output: [
      "✓ stopped Next.js (:3000)",
      "✓ stopped FastAPI (:8000)",
      "✓ stopped PostgreSQL (:5432)",
      "✓ stopped Redis (:6379)",
    ],
  },
  {
    id: "doctor",
    cmd: "ports doctor 3000",
    title: "The doctor",
    line: "Explain the row.",
    desc: "Why a stop failed or a row looks odd — findings and the way out, never an action taken on its own.",
    output: [
      ":3000  Next.js  pid 48391  ~/projects/shop  up 14m",
      "✓ listening on :3000",
      "✓ responds to HTTP → http://localhost:3000",
      "✓ process state R — signals land",
      "✓ pid 48391 still matches the scan — no pid reuse",
      "✓ nothing else holds :3000 — no stray sockets",
    ],
  },
] as const;

// The FAQ close — every answer capped with a doc link, the way the rest
// of the page trusts verifiability over adjectives.
export const FAQ_ITEMS = [
  {
    q: "How is this different from lsof?",
    a: "lsof gives you a PID and a process name. ports replaces both with what you actually wanted: the framework, the project directory, the container behind docker-proxy, and the uptime.",
    href: "/docs",
    link: "read the introduction",
  },
  {
    q: "Does it work with Docker?",
    a: "Yes. docker-proxy rows are unwrapped over the Docker socket to the real container and Compose service behind them — and containers stop through the daemon, never the proxy.",
    href: "/docs/commands",
    link: "see the commands",
  },
  {
    q: "How does it know which project a port belongs to?",
    a: "It walks the process working directory to the nearest repository or marker file. No configuration, no manifest — the context is inferred from the machine itself.",
    href: "/docs",
    link: "how it works",
  },
  {
    q: "A port won't free. Now what?",
    a: "ports doctor re-examines the port and tells you what the machine is actually doing — an uninterruptible process, a zombie, TIME_WAIT sockets, a Windows-side listener — each with the command that resolves it. It never acts; every fix is yours to run.",
    href: "/docs/commands#cmd-doctor",
    link: "meet the doctor",
  },
  {
    q: "Which platforms does it run on?",
    a: "macOS, Linux, and Windows via Scoop. On WSL2 it distinguishes Windows-side listeners from WSL processes in the same table — and never offers the Windows side to the kill path.",
    href: "/docs/installation",
    link: "installation",
  },
  {
    q: "Does anything run in the background?",
    a: "No. ports is one static Go binary with no daemon, no agent, and nothing you have to configure — a config file is never required. The optional ports-config.yaml is an extension layer for custom framework names and project markers, not something the tool needs to work. It scans when you run it and exits when it's done — `--watch` is the one exception, and it's still just a foreground loop that redraws the table until you interrupt it. It dies with its terminal.",
    href: "/docs/quickstart",
    link: "the quickstart",
  },
] as const;