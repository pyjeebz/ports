"use client";

import { CONTACTS, INSTALL_METHODS, NEGATIONS } from "@/lib/site";
import { useMotionAllowed, useTimedSteps } from "@/lib/motion";
import { useInView } from "@/lib/use-in-view";
import { PortsTable } from "./ports-table";
import { SectionHead } from "./section-head";
import { TerminalBlock } from "./terminal-block";
import { Stamp } from "@/components/ui/stamp";

const BREW = INSTALL_METHODS[0].lines[0];
const RUN = "ports";

// Each negation lands as a stamp, none of them hung straight, inks
// alternating so the row reads like a page of rubber stamps.
const STAMP_STYLES = [
  { ink: "red", rotate: "-2" },
  { ink: "charcoal", rotate: "1" },
  { ink: "red", rotate: "1" },
  { ink: "charcoal", rotate: "-2" },
] as const;

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
    <section ref={ref} className="py-24">
      <div className="mx-auto max-w-[1160px] px-8">
        <SectionHead
          kicker="the binary"
          title="Nothing running in the background."
          sub="One static Go binary. It scans when you run it and exits when it&rsquo;s done — useful sixty seconds after install, on a machine in whatever state it&rsquo;s in."
        />

        <p className="mt-12 font-display text-[clamp(2.6rem,7.5vw,5.5rem)] leading-none text-charcoal">
          ONE BINARY
        </p>
        <div className="mt-8 flex flex-wrap items-center gap-6">
          {NEGATIONS.map((n, i) => (
            <Stamp key={n} ink={i % 2 ? "charcoal" : "red"} rotate={STAMP_STYLES[i].rotate}>
              {n}
            </Stamp>
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