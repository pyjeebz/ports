import type { ReactNode, Ref } from "react";

/**
 * The shared terminal shell: a chrome header (command on the left, a
 * readout on the right) over a code-shell pre. The one shape every
 * section's visual takes — the CLI output is the artwork, so the frame
 * around it stays identical everywhere.
 */
export function TerminalBlock({
  cmd,
  label = "output",
  children,
  preClassName = "text-[12.5px] leading-[1.85] text-foreground",
  className = "",
  preRef,
}: {
  cmd: string;
  /** The right-hand readout; a node so callers can color it (the hero
      machine's live state). */
  label?: ReactNode;
  children: ReactNode;
  /** Classes for the pre — sections override to scale the type. */
  preClassName?: string;
  className?: string;
  /** Lets scrolling callers drive their pre (the terminal playground's
      auto-scroll). */
  preRef?: Ref<HTMLPreElement>;
}) {
  return (
    <div className={`code-shell ${className}`}>
      <div className="flex items-center justify-between border-b border-line-soft bg-panel-2 px-4 py-2.5">
        <span className="truncate font-mono text-[11.5px] text-foreground">{cmd}</span>
        <span className="readout shrink-0">{label}</span>
      </div>
      <pre ref={preRef} className={preClassName}>{children}</pre>
    </div>
  );
}