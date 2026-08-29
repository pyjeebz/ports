import Link from "next/link";

import { CopyButton } from "@/components/site/copy-button";
import { PortsLogo } from "@/components/site/ports-logo";

export const metadata = { title: "ports — redesign draft" };

const TERMINAL = `<span style="color:var(--faint)">$</span> ports
<span style="color:var(--faint)">PORT     PROCESS       PROJECT                UPTIME</span>
3000     Next.js       ~/projects/shop        14m
5432     PostgreSQL    docker: shop-db        2d
8000     FastAPI       ~/projects/shop        14m
5173     Vite          ~/projects/admin       3h
11434    Ollama        windows                ?

<span style="color:var(--faint)">$</span> ports kill shop
<span style="color:var(--accent)">✓ stopped Next.js (:3000)</span>
<span style="color:var(--accent)">✓ stopped FastAPI (:8000)</span>
<span style="color:var(--accent)">✓ stopped PostgreSQL (container shop-db)</span>`;

const COMMANDS: [string, string][] = [
  ["ports", "List every listening port with its framework, project, and uptime."],
  ["ports find <name>", "Filter by framework, process, project, or container. Case insensitive."],
  ["ports free <port>", "Kill whatever holds a port."],
  ["ports kill <name>", "Stop every process in a project or service, containers included."],
  ["--json", "Machine readable output for scripts and coding agents."],
  ["--force", "Skip the grace period and kill immediately."],
];

const INSTALL: [string, string][] = [
  ["macOS", "brew install pyjeebz/tap/ports"],
  ["Linux, macOS", "curl -fsSL https://ports.tools/install | sh"],
  ["Windows", "scoop install ports"],
];

export default function Redesign() {
  return (
    <div className="mx-auto max-w-[1080px] px-6">
      <header className="flex items-center justify-between py-6">
        <span className="flex items-center gap-2.5 text-[15px] font-semibold">
          <PortsLogo className="h-5 w-5" /> ports
        </span>
        <nav className="flex gap-6 text-sm text-muted">
          <Link href="/docs" className="transition-colors hover:text-foreground">
            Docs
          </Link>
          <a
            href="https://github.com/pyjeebz/ports"
            className="transition-colors hover:text-foreground"
          >
            GitHub
          </a>
        </nav>
      </header>

      {/* opening: the claim and the evidence, co-primary */}
      <section className="pb-28 pt-16">
        <h1 className="max-w-[19ch] font-semibold tracking-[-0.03em] text-strong leading-[1.08] text-[clamp(2rem,4.4vw,3.25rem)]">
          See what&apos;s running on your machine, by project.
        </h1>
        <p className="mt-5 max-w-[60ch] leading-relaxed text-muted text-[18px]">
          ports lists every listening port with the framework, project, and
          uptime behind it, then stops things by name. No configuration, one
          static binary.
        </p>

        <div className="mt-10 overflow-x-auto rounded-md border border-line-soft bg-panel">
          <pre
            className="p-6 font-mono text-[13.5px] leading-[1.9] text-foreground [font-variant-numeric:tabular-nums]"
            dangerouslySetInnerHTML={{ __html: TERMINAL }}
          />
        </div>

        <div className="mt-6 flex flex-wrap items-center gap-4">
          <div className="flex items-center gap-3 rounded-md border border-line bg-panel px-4 py-2.5 font-mono text-[13px]">
            <span style={{ color: "var(--faint)" }}>$</span>
            <span>curl -fsSL https://ports.tools/install | sh</span>
            <CopyButton text="curl -fsSL https://ports.tools/install | sh" />
          </div>
          <Link
            href="/docs/installation"
            className="text-sm text-muted transition-colors hover:text-foreground"
          >
            Homebrew and Scoop
          </Link>
        </div>
      </section>

      {/* why */}
      <section className="pb-28">
        <h2 className="max-w-[26ch] font-semibold leading-tight tracking-[-0.02em] text-strong text-[clamp(1.4rem,2.6vw,1.9rem)]">
          Ports and PIDs are how the OS sees your machine. You think in
          projects.
        </h2>
        <div className="mt-6 max-w-[64ch] space-y-4 leading-relaxed text-muted text-[16px]">
          <p>
            The dance is familiar. Port 3000 is in use, so you run lsof, get
            back a PID, and kill it. You never really know what you killed.
            Yesterday&apos;s dev server, or the database you still needed.
          </p>
          <p>
            A PID is a number. A project is a frontend, an API, and a database
            that belong together. ports adds the context the process table
            leaves out, so you stop yesterday&apos;s server and keep the
            database.
          </p>
        </div>
      </section>

      {/* what it reads, ranked */}
      <section className="pb-28">
        <h2 className="font-semibold tracking-[-0.02em] text-strong text-[clamp(1.4rem,2.6vw,1.9rem)]">
          Context the process table hides.
        </h2>
        <div className="mt-10 grid gap-x-16 gap-y-10 md:grid-cols-2">
          <div>
            <h3 className="text-[17px] font-semibold text-strong">
              Framework, not process name
            </h3>
            <p className="mt-2 max-w-[46ch] leading-relaxed text-muted text-[15px]">
              Next.js, Vite, FastAPI, Postgres, Redis, named the way you know
              them. Framework detection first, then the container image, then
              the raw process.
            </p>
          </div>
          <div>
            <h3 className="text-[17px] font-semibold text-strong">
              Project, not PID
            </h3>
            <p className="mt-2 max-w-[46ch] leading-relaxed text-muted text-[15px]">
              Each process is traced from its working directory up to the git
              root, so a frontend, API, and database read and stop as one
              application.
            </p>
          </div>
        </div>
        <dl className="mt-12 grid gap-x-16 gap-y-4 pt-2 text-[14px] sm:grid-cols-2">
          {[
            ["Docker", "Real container and Compose service names, not docker-proxy."],
            ["WSL2", "Windows-side listeners kept distinct from WSL processes."],
            ["JSON", "--json on any command, for scripts and agents."],
            ["Safe stops", "SIGTERM first, SIGKILL only if it will not exit."],
          ].map(([k, v]) => (
            <div key={k} className="flex gap-4">
              <dt className="w-24 shrink-0 text-foreground">{k}</dt>
              <dd className="text-muted">{v}</dd>
            </div>
          ))}
        </dl>
      </section>

      {/* commands as evidence */}
      <section className="pb-28">
        <h2 className="font-semibold tracking-[-0.02em] text-strong text-[clamp(1.4rem,2.6vw,1.9rem)]">
          Four commands and two flags.
        </h2>
        <table className="mt-8 w-full border-collapse text-left text-[14px]">
          <thead>
            <tr className="border-b border-line-soft text-[13px] text-faint">
              <th className="py-2.5 pr-8 font-medium">Command</th>
              <th className="py-2.5 font-medium">What it does</th>
            </tr>
          </thead>
          <tbody>
            {COMMANDS.map(([cmd, desc]) => (
              <tr key={cmd} className="border-b border-line-soft/60 align-baseline">
                <td className="py-3 pr-8 font-mono text-foreground">{cmd}</td>
                <td className="py-3 text-muted">{desc}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>

      {/* install and the honest limit */}
      <section className="pb-24">
        <h2 className="font-semibold tracking-[-0.02em] text-strong text-[clamp(1.4rem,2.6vw,1.9rem)]">
          Install
        </h2>
        <div className="mt-8 space-y-3">
          {INSTALL.map(([label, cmd]) => (
            <div key={label} className="flex flex-wrap items-center gap-4">
              <span className="w-28 shrink-0 text-[13px] text-faint">{label}</span>
              <div className="flex items-center gap-3 rounded-md border border-line-soft bg-panel px-4 py-2 font-mono text-[13px]">
                <span style={{ color: "var(--faint)" }}>$</span>
                <span>{cmd}</span>
                <CopyButton text={cmd} />
              </div>
            </div>
          ))}
        </div>
        <p className="mt-10 max-w-[62ch] text-[14px] text-faint">
          ports is an observer with a targeted kill. It never starts, restarts,
          or supervises anything.
        </p>
      </section>

      <footer className="flex flex-wrap items-center justify-between gap-4 py-10 text-[13px] text-faint">
        <span className="flex items-center gap-2">
          <PortsLogo className="h-4 w-4" /> ports
        </span>
        <div className="flex gap-6">
          <a
            href="https://github.com/pyjeebz/ports"
            className="transition-colors hover:text-foreground"
          >
            GitHub
          </a>
          <Link href="/docs" className="transition-colors hover:text-foreground">
            Docs
          </Link>
          <a
            href="https://github.com/pyjeebz/ports/releases"
            className="transition-colors hover:text-foreground"
          >
            Releases
          </a>
        </div>
      </footer>
    </div>
  );
}
