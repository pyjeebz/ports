"use client";

import Link from "next/link";
import { useEffect, useLayoutEffect, useRef, useState, type ReactNode } from "react";

import { CONTACTS, CONTACT_PIDS, SHOWCASE, type Contact } from "@/lib/site";
import { useMotionAllowed, useTimedSteps } from "@/lib/motion";
import { SectionHead } from "./section-head";
import { PortsTable } from "./ports-table";
import { TerminalBlock } from "./terminal-block";
import { Tape } from "@/components/ui/tape";

const TYPE_MS = 45;
const LINE_MS = 90;

type Entry =
  | { kind: "table"; cmd: string; rows: Contact[] }
  | { kind: "lines"; cmd: string; lines: readonly string[]; muted?: boolean };

const isContainer = (ct: Contact) => ct.project.startsWith("docker: ");
const containerName = (ct: Contact) => ct.project.replace("docker: ", "");

const freeLine = (ct: Contact) =>
  isContainer(ct)
    ? `✓ stopped ${ct.framework} (container ${containerName(ct)}) — port ${ct.port} is free`
    : `✓ killed ${ct.framework} (pid ${CONTACT_PIDS[ct.port]}) — port ${ct.port} is free`;

// kill names the ports it freed — a container row gets the same line as
// any other, because that's what the binary prints.
const killLine = (ct: Contact) => `✓ stopped ${ct.framework} (:${ct.port})`;

const SURFACE = "the whole surface: ports · find · free · kill · doctor · diff";

function jsonLines(rows: Contact[]): string[] {
  return JSON.stringify(
    rows.map(({ port, framework, project, uptime }) => ({
      port,
      process: framework,
      project,
      uptime,
    })),
    null,
    2,
  ).split("\n");
}

/**
 * The parser — a tiny, honest slice of the real CLI. Each run produces the
 * entry it prints and the roster it leaves behind, so the terminal has
 * state: kill the shop project and the next `ports` shows the silence.
 */
function execute(raw: string, rows: Contact[]): { entry: Entry; rows: Contact[] } | null {
  const tokens = raw.trim().split(/\s+/).filter(Boolean);
  if (tokens.length === 0) return null;
  const flags = tokens.filter((t) => t.startsWith("--"));
  const args = tokens.filter((t) => !t.startsWith("--"));
  const flag = (f: string) => flags.includes(f);

  if (args[0] === "reset")
    return {
      entry: { kind: "lines", cmd: "reset", lines: ["✓ machine restored — six services listening"] },
      rows: [...CONTACTS],
    };
  if (args[0] === "help")
    return {
      entry: { kind: "lines", cmd: "help", lines: [SURFACE] },
      rows,
    };
  if (args[0]?.startsWith("lsof"))
    return {
      entry: {
        kind: "lines",
        cmd: raw,
        lines: ["✗ lsof can't tell you which project this is", "try: ports find shop"],
      },
      rows,
    };
  if (args[0] !== "ports")
    return {
      entry: { kind: "lines", cmd: raw, lines: [`✗ unknown command: ${args[0]}`, SURFACE] },
      rows,
    };

  const sub = args[1];
  const rest = args.slice(2);
  const json = flag("--json");
  const force = flag("--force");

  if (json && !sub)
    return { entry: { kind: "lines", cmd: raw, lines: jsonLines(rows), muted: true }, rows };

  if (!sub) {
    if (rows.length === 0)
      return {
        entry: { kind: "lines", cmd: raw, lines: ["nothing is listening — a quiet machine"] },
        rows,
      };
    return { entry: { kind: "table", cmd: raw, rows }, rows };
  }

  if (sub === "doctor") {
    // Bare doctor is the machine-wide sweep — the simulated roster's
    // containers always resolve, so it reads clean.
    if (!rest[0])
      return {
        entry: {
          kind: "lines",
          cmd: raw,
          lines: [
            "✓ docker engine reachable via /var/run/docker.sock",
            "✓ nothing suspicious — every listener is attributable and unshared",
          ],
        },
        rows,
      };
    const port = Number(rest[0]);
    if (Number.isNaN(port))
      return {
        entry: { kind: "lines", cmd: raw, lines: [`✗ invalid port ${rest[0]}`] },
        rows,
      };
    const hit = rows.find((ct) => ct.port === port);
    // State-aware: doctor a port you freed and it says what the scan
    // says — nothing. That's the point of the re-scan.
    if (!hit)
      return {
        entry: { kind: "lines", cmd: raw, lines: [`✗ nothing is listening on :${port}`] },
        rows,
      };
    // The header follows the real CLI: containers name the container,
    // host processes name the pid and the project.
    const header = isContainer(hit)
      ? `:${hit.port}  ${hit.framework}  container ${containerName(hit)}  up ${hit.uptime}`
      : `:${hit.port}  ${hit.framework}  pid ${CONTACT_PIDS[hit.port]}  ${hit.project}  up ${hit.uptime}`;
    return {
      entry: {
        kind: "lines",
        cmd: raw,
        lines: [
          header,
          `✓ listening on :${hit.port}`,
          "✓ process state R — signals land",
          `✓ pid ${CONTACT_PIDS[hit.port]} still matches the scan — no pid reuse`,
          `✓ nothing else holds :${hit.port} — no stray sockets`,
        ],
      },
      rows,
    };
  }

  if (sub === "find") {
    const q = rest.join(" ").toLowerCase();
    if (!q)
      return { entry: { kind: "lines", cmd: raw, lines: ["✗ find what? — try: ports find postgres"] }, rows };
    const hits = rows.filter((ct) =>
      `${ct.port} ${ct.framework} ${ct.project}`.toLowerCase().includes(q),
    );
    return hits.length
      ? { entry: { kind: "table", cmd: raw, rows: hits }, rows }
      : {
          entry: { kind: "lines", cmd: raw, lines: [`✗ no matches for “${rest.join(" ")}”`] },
          rows,
        };
  }

  if (sub === "free") {
    const port = Number(rest[0]);
    if (!rest[0] || Number.isNaN(port))
      return { entry: { kind: "lines", cmd: raw, lines: ["✗ free which port? — try: ports free 3000"] }, rows };
    const hit = rows.find((ct) => ct.port === port);
    if (!hit)
      return { entry: { kind: "lines", cmd: raw, lines: [`port ${port} is already free`] }, rows };
    const lines = [freeLine(hit)];
    if (force) lines.push("— no SIGTERM, straight to SIGKILL (--force)");
    return {
      entry: { kind: "lines", cmd: raw, lines },
      rows: rows.filter((ct) => ct.port !== port),
    };
  }

  if (sub === "kill") {
    const q = rest.join(" ").toLowerCase();
    if (!q)
      return { entry: { kind: "lines", cmd: raw, lines: ["✗ kill what? — try: ports kill shop"] }, rows };
    const hits = rows.filter((ct) =>
      `${ct.framework} ${ct.project}`.toLowerCase().includes(q),
    );
    if (!hits.length)
      return {
        entry: { kind: "lines", cmd: raw, lines: [`✗ nothing called “${rest.join(" ")}” is running`] },
        rows,
      };
    const lines = hits.map(killLine);
    if (force) lines.push("— no SIGTERM, straight to SIGKILL (--force)");
    return {
      entry: { kind: "lines", cmd: raw, lines },
      rows: rows.filter((ct) => !hits.includes(ct)),
    };
  }

  return {
    entry: { kind: "lines", cmd: raw, lines: [`✗ ports ${sub}?`, SURFACE] },
    rows,
  };
}

function tone(line: string) {
  if (line.startsWith("PORT")) return "text-faint";
  if (line.startsWith("✓")) return "text-accent";
  if (line.startsWith("✗") || line.startsWith("⚠")) return "text-alert";
  return undefined;
}

/** One entry's output — a table, or lines with their tones. */
function EntryBody({ entry, revealed }: { entry: Entry; revealed?: number }) {
  if (entry.kind === "table")
    return <PortsTable rows={entry.rows} revealed={revealed} />;
  return (
    <>
      {entry.lines.map((line, i) => (
        <span
          key={i}
          className={`block transition-opacity duration-300 ${
            revealed === undefined || i < revealed ? "opacity-100" : "opacity-0"
          } ${tone(line) ?? ""} ${entry.muted ? "text-muted" : ""}`}
        >
          {line}
        </span>
      ))}
    </>
  );
}

/**
 * The interface section as the product itself: one live terminal with
 * state. Clicking a command (or typing one) runs it — the command types,
 * the output arrives — and the machine remembers: kill the shop project
 * and the next `ports` shows what's left. The server frame is a session
 * already in progress (one completed `ports`, a live prompt), so SEO and
 * no-JS keep the table in the HTML. Reduced motion types nothing — runs
 * land instantly.
 */
export function FourCommands() {
  const motion = useMotionAllowed();
  const preRef = useRef<HTMLPreElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);
  const [rows, setRows] = useState<Contact[]>([...CONTACTS]);
  const [log, setLog] = useState<Entry[]>([{ kind: "table", cmd: "ports", rows: [...CONTACTS] }]);
  const [live, setLive] = useState<Entry | null>(null);
  const [input, setInput] = useState("");
  const [started, setStarted] = useState(false);

  const run = (raw: string) => {
    const value = raw.trim();
    if (!value) return;
    setStarted(true);
    if (value === "clear") {
      setLog([]);
      setLive(null);
      setInput("");
      return;
    }
    const result = execute(value, rows);
    if (!result) return;
    setRows(result.rows);
    setLog((entries) => (live ? [...entries, live] : entries));
    setLive(result.entry);
    setInput("");
  };

  // The command being run types itself out, then its output arrives line
  // by line — two bounded counters chained on a derived flip.
  const cmdLen = live?.cmd.length ?? 0;
  const typeStep = useTimedSteps(cmdLen, TYPE_MS, live !== null && motion);
  const typed = Math.min(typeStep, cmdLen);
  const typing = live !== null && typed < cmdLen;

  const liveCount = live ? (live.kind === "table" ? live.rows.length : live.lines.length) : 0;
  const revealActive = live !== null && typed >= cmdLen && motion;
  const lineStep = useTimedSteps(liveCount, LINE_MS, revealActive);
  const shown = revealActive ? Math.min(lineStep, liveCount) : liveCount;
  const done = live !== null && typed >= cmdLen && shown >= liveCount;

  // Keep the session pinned to its newest output; focus returns to the
  // prompt only once the visitor has started driving.
  useLayoutEffect(() => {
    const el = preRef.current;
    if (el) el.scrollTop = el.scrollHeight;
  }, [log.length, live, typed, shown]);

  useEffect(() => {
    if (started) inputRef.current?.focus();
  }, [started, live, done]);

  const status = (
    <span className={rows.length > 0 ? "text-accent" : "text-alert"}>
      ● {rows.length} listening
    </span>
  );

  const promptRow = (
    <span className="mt-5 flex items-center gap-2">
      <span className="text-faint">$</span>
      <input
        ref={inputRef}
        value={input}
        onChange={(e) => setInput(e.target.value)}
        onKeyDown={(e) => e.key === "Enter" && run(input)}
        aria-label="type a ports command"
        spellCheck={false}
        autoComplete="off"
        autoCapitalize="off"
        placeholder="ports …  or click a command"
        className="w-full bg-transparent font-mono text-[12.5px] text-foreground caret-[var(--accent)] outline-none placeholder:text-faint"
      />
    </span>
  );

  const liveBlock: ReactNode = live && (
    <span className="block mt-5">
      <span className="text-faint">$ </span>
      <span className="text-foreground">{live.cmd.slice(0, typed)}</span>
      {typing && <span data-caret className="text-accent">▌</span>}
      {typed >= cmdLen && (
        <>
          {"\n"}
          <EntryBody entry={live} revealed={revealActive ? shown : undefined} />
        </>
      )}
    </span>
  );

  return (
    <section data-motion={live !== null && motion ? "on" : undefined} className="py-24">
      <div className="mx-auto max-w-[1160px] px-8">
        <SectionHead
          kicker="the interface"
          title="Four commands and a doctor."
          sub="ports, find, free, kill — and when one of them fails, ports doctor explains what your machine is actually doing. Run them right here; this terminal remembers what you stopped."
        />

        <div className="mt-10 grid items-start gap-12 lg:grid-cols-[1fr_560px] lg:gap-14">
          <div>
            {SHOWCASE.map((entry, i) => (
              <button
                key={entry.id}
                type="button"
                onClick={() => run(entry.cmd)}
                className="group block w-full border-t border-line-soft py-7 text-left last:border-b"
              >
                <p
                  className={`flex flex-wrap items-baseline gap-x-4 font-mono text-[clamp(1.35rem,3vw,2.1rem)] font-semibold leading-[1.15] text-charcoal transition-all group-hover:-rotate-1 group-hover:bg-highlighter-yellow/40 ${
                    i % 2 ? "rotate-[0.4deg]" : "-rotate-[0.4deg]"
                  }`}
                >
                  {entry.cmd}
                  <span className="readout opacity-0 transition-opacity group-hover:opacity-100">
                    run ↵
                  </span>
                </p>
                <p className="mt-2.5 text-[15px] text-charcoal-light">{entry.line}</p>
              </button>
            ))}

            <p className="mt-6 flex flex-wrap items-center gap-x-6 gap-y-1 font-mono text-[12px] text-muted">
              <span>
                <Link
                  href="/docs/commands#flags"
                  className="text-charcoal underline-offset-4 transition-colors hover:text-charcoal hover:underline"
                >
                  --json
                </Link>{" "}
                the same scan, structured
              </span>
              <span>
                <Link
                  href="/docs/commands#flags"
                  className="text-charcoal underline-offset-4 transition-colors hover:text-charcoal hover:underline"
                >
                  --force
                </Link>{" "}
                skip the SIGTERM
              </span>
            </p>
            <p className="readout mt-3">
              both flags work above — this terminal is the whole interface
            </p>
          </div>

          <div className="relative">
            <Tape color="pink" className="-top-3 right-10 rotate-2" />
            <TerminalBlock
              cmd="ports — your machine"
              label={status}
              preRef={preRef}
              preClassName="max-h-[420px] overflow-y-auto text-[12.5px] leading-[1.9] text-foreground"
            >
              {/* append-only history — index keys are stable here */}
              {log.map((entry, i) => (
                <span key={i} className="block mt-5 first:mt-0">
                  <span className="text-faint">$ {entry.cmd}</span>
                  {"\n"}
                  <EntryBody entry={entry} />
                </span>
              ))}
              {liveBlock}
              {(!live || done) && promptRow}
            </TerminalBlock>

            <div className="mt-3 flex items-center justify-between">
              <span className="readout">click a command or type one</span>
              {rows.length < CONTACTS.length && (
                <button
                  type="button"
                  onClick={() => run("reset")}
                  className="readout transition-colors hover:text-charcoal"
                >
                  ↺ reset the machine
                </button>
              )}
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}