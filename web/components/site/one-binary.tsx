"use client";

import { CONTACTS, INSTALL_METHODS, NEGATIONS } from "@/lib/site";
import { useMotionAllowed, useTimedSteps } from "@/lib/motion";
import { useInView } from "@/lib/use-in-view";
import { PortsTable } from "./ports-table";
import { SectionHead } from "./section-head";
import { TerminalBlock } from "./terminal-block";

const BREW = INSTALL_METHODS[0].lines[0];
const RUN = "ports";

/**
 * The binary section: the zero-footprint statement, then the proof —
 * install, run, and the full table arriving. The brew line is static;
 * `ports` types itself; the roster's rows fade in one per tick. The
 * server renders the completed session.
 */
export function OneBinary() {
  const [ref, inView] = useInView<HTMLElement>({ threshold: 0.3 });
  const motion = useMotionAllowed();
  const active = inView && motion;

  const rows = CONTACTS.length;
  const step = useTimedSteps(RUN.length + rows, 70, active);
  const typed = Math.min(step, RUN.length);
  const revealed = step - RUN.length;
  const typing = active && typed < RUN.length;

  return (
    <section ref={ref} className="border-b border-line py-24">
      <div className="mx-auto max-w-[1160px] px-8">
        <SectionHead
          kicker="the binary"
          title="Nothing running in the background."
          sub="One static Go binary. It scans when you run it and exits when it&rsquo;s done — useful sixty seconds after install, on a machine in whatever state it&rsquo;s in."
        />

        <p className="mt-12 font-mono text-[clamp(2.4rem,7vw,5rem)] font-semibold leading-none tracking-[-0.04em] text-strong">
          ONE BINARY
        </p>
        <div className="mt-6 flex flex-wrap gap-x-8 gap-y-2 font-mono text-[13px] tracking-[0.14em] text-faint">
          {NEGATIONS.map((n) => (
            <span key={n}>{n}</span>
          ))}
        </div>

        <div className="mt-12 max-w-[720px]">
          <TerminalBlock cmd="install and first run" label="sixty seconds">
            <span className="text-faint">$ </span>
            <span className="text-foreground">{BREW}</span>
            {"\n"}
            <span className="text-faint">$ </span>
            <span className="text-foreground">{RUN.slice(0, typed)}</span>
            {typing && <span data-caret className="text-accent">▌</span>}
            {"\n\n"}
            <PortsTable rows={CONTACTS} revealed={active ? revealed : undefined} />
          </TerminalBlock>
        </div>
      </div>
    </section>
  );
}