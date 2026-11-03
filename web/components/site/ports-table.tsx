import { projectLabel, type Contact } from "@/lib/site";

/**
 * The ports table, rendered from the CONTACTS roster. Column widths in ch
 * mirror the CLI and scale with the surrounding type — the hero renders
 * it at display size inside its own terminal, the one-binary section at
 * body size. One renderer, so no two renderings can drift.
 *
 * The interactive props are all optional and default to the static table:
 * `variant="machine"` (the hero simulation) makes rows clickable — the
 * open row highlights and its resolution tree expands beneath it. The
 * machine-row/port-cell classes are styling hooks for the dock
 * transition in globals.css; static renderings never carry them.
 */
export function PortsTable({
  rows,
  highlight,
  onHoverPort,
  revealed,
  variant = "static",
  onRowClick,
  openPort = null,
  detail,
}: {
  rows: readonly Contact[];
  highlight?: number | null;
  onHoverPort?: (port: number | null) => void;
  /**
   * Row-reveal cutoff for staggered entrances: row i is faded out until
   * revealed > i. Undefined — the default, and what the server renders —
   * shows every row. Rows are always in the DOM; only opacity changes.
   */
  revealed?: number;
  /** "machine" arms the interactive row hooks; "static" renders plain rows. */
  variant?: "static" | "machine";
  onRowClick?: (port: number) => void;
  /** The port whose tree is expanded, if any. */
  openPort?: number | null;
  /** Resolution trees keyed by port, rendered under the open row. */
  detail?: Record<number, readonly string[]>;
}) {
  const machine = variant === "machine";

  return (
    <>
      <span className="block select-none text-faint">
        <span className="inline-block w-[9ch]">PORT</span>
        <span className="inline-block w-[13ch]">PROCESS</span>
        <span className="inline-block w-[19ch]">PROJECT</span>
        UPTIME
      </span>
      {rows.map((ct, i) => {
        const hot = highlight === ct.port || openPort === ct.port;
        const handlers = onHoverPort
          ? { onMouseEnter: () => onHoverPort(ct.port), onMouseLeave: () => onHoverPort(null) }
          : {};
        const dimmed = revealed !== undefined && revealed <= i;
        const open = openPort === ct.port;
        const tree = open && detail?.[ct.port];

        const className = `-mx-2 block px-2 transition-opacity duration-300 ${
          machine ? "machine-row" : ""
        } ${hot ? "bg-highlighter-yellow/60" : ""} ${dimmed ? "opacity-0" : "opacity-100"} ${
          machine ? "cursor-pointer text-left font-mono" : "cursor-default"
        }`;

        // The interactive row is a button (keyboard focusable, aria-expanded);
        // the static row stays a span. Contents are identical.
        const row = machine ? (
          <button
            key={ct.port}
            type="button"
            aria-expanded={open}
            onClick={() => onRowClick?.(ct.port)}
            className={`w-full ${className}`}
            {...handlers}
          >
            <span className={`inline-block w-[9ch] port-cell ${hot ? "text-strong" : "text-muted"}`}>
              {ct.port}
            </span>
            <span className="inline-block w-[13ch] text-strong">{ct.framework}</span>
            <span className={`inline-block w-[19ch] ${hot ? "text-strong" : "text-muted"}`}>
              {projectLabel(ct)}
            </span>
            <span className={hot ? "text-muted" : "text-faint"}>{ct.uptime}</span>
          </button>
        ) : (
          <span key={ct.port} {...handlers} className={className}>
            <span className={`inline-block w-[9ch] ${hot ? "text-strong" : "text-muted"}`}>
              {ct.port}
            </span>
            <span className="inline-block w-[13ch] text-strong">{ct.framework}</span>
            <span className={`inline-block w-[19ch] ${hot ? "text-strong" : "text-muted"}`}>
              {projectLabel(ct)}
            </span>
            <span className={hot ? "text-muted" : "text-faint"}>{ct.uptime}</span>
          </span>
        );

        return (
          <span key={ct.port} className="block">
            {row}
            {tree && (
              <span className="block px-2 pb-2 pt-0.5 font-mono text-[13px] leading-[1.9] text-muted">
                {detail![ct.port].map((line, j) => (
                  <span key={j} className="block">
                    {j === detail![ct.port].length - 1 ? "└──" : "├──"}
                    {line}
                  </span>
                ))}
              </span>
            )}
          </span>
        );
      })}
    </>
  );
}