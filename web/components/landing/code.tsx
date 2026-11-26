import { CopyButton } from "./copy-button";

/**
 * A line of a terminal block: a command the reader types (copied by the
 * button), what the command printed, a comment, or a diff line.
 */
export type Line =
  | { cmd: string; note?: string }
  | { out: string; tone?: "ok" | "muted" | "warn" }
  | { add: string; note?: string }
  | { del: string; note?: string };

/** Terminal is a code block the way a doc shows one: a quiet header with
 *  what it is, the lines, and a copy button for the commands in it. */
export function Terminal({
  title = "Terminal",
  lines,
  className = "",
}: {
  title?: string;
  lines: Line[];
  className?: string;
}) {
  const commands = lines.flatMap((l) => ("cmd" in l ? [l.cmd] : "add" in l ? [l.add] : []));
  return (
    <figure className={`my-5 overflow-hidden rounded-lg border bg-card ${className}`}>
      <figcaption className="flex h-9 items-center justify-between border-b bg-muted/40 pr-1 pl-3.5 text-xs text-muted-foreground">
        <span>{title}</span>
        {commands.length > 0 && <CopyButton text={commands.join("\n")} />}
      </figcaption>
      <pre className="overflow-x-auto py-3 font-mono text-[13px] leading-6">
        <code className="grid min-w-max">
          {lines.map((l, i) => (
            <CodeLine key={i} line={l} />
          ))}
        </code>
      </pre>
    </figure>
  );
}

function CodeLine({ line }: { line: Line }) {
  if ("cmd" in line) {
    return (
      <span className="px-4">
        <span className="text-muted-foreground select-none">$ </span>
        <span>{line.cmd}</span>
        {line.note && <span className="text-muted-foreground">{"  "}# {line.note}</span>}
      </span>
    );
  }
  if ("add" in line || "del" in line) {
    const added = "add" in line;
    const text = added ? line.add : line.del;
    return (
      <span className={`px-4 ${added ? "bg-added-bg" : "bg-removed-bg"}`}>
        <span className={`select-none ${added ? "text-added" : "text-removed"}`}>{added ? "+ " : "- "}</span>
        <span>{text}</span>
        {line.note && <span className="text-muted-foreground">{"  "}# {line.note}</span>}
      </span>
    );
  }
  const tone =
    line.tone === "ok"
      ? "text-added"
      : line.tone === "warn"
        ? "text-removed"
        : line.tone === "muted"
          ? "text-muted-foreground"
          : "text-foreground/80";
  return <span className={`px-4 whitespace-pre ${tone}`}>{line.out || " "}</span>;
}

/** Inline code, the way prose in a doc sets it. */
export function C({ children }: { children: React.ReactNode }) {
  return (
    <code className="rounded-md border bg-muted/60 px-1.5 py-0.5 font-mono text-[0.85em] text-foreground">
      {children}
    </code>
  );
}
