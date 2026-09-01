"use client";

import { useLayoutEffect, useState } from "react";

import { CONTACTS, CONTACT_DETAILS, HERO_PORTS, SCAN_STEPS } from "@/lib/site";
import { useMotionAllowed, useTimedSteps } from "@/lib/motion";
import { PortsTable } from "./ports-table";
import { TerminalBlock } from "./terminal-block";

const heroRows = CONTACTS.filter((ct) =>
  (HERO_PORTS as readonly number[]).includes(ct.port),
);
const SCAN_MS = 600;
const ROW_MS = 140;

type Phase = "live" | "idle" | "scanning";

/**
 * The hero as a product simulation, not an illustration: a machine the
 * visitor wakes up. The server renders the completed state (SEO and
 * no-JS keep the table in the HTML); with JavaScript the machine resets
 * to its invitation before paint — `No ports detected`, a Run button —
 * and the click drives the story: scan, discovery, rows docking into
 * their berths, and rows that expand into what ports resolved. Reduced
 * motion keeps the story and drops the choreography — one click, the
 * completed machine, no steps or staggers.
 */
export function PortsMachine() {
  const motion = useMotionAllowed();
  // "live" is the server's frame; the layout effect below resets it.
  const [phase, setPhase] = useState<Phase>("live");
  const [openPort, setOpenPort] = useState<number | null>(null);

  // The machine wakes to its invitation. A pre-paint layout reset is
  // React's adjust-on-mount pattern: no flash of the completed frame,
  // one extra render. The compiler lint flags any setState in an effect;
  // this is the sanctioned exception, and the only one on the site.
  useLayoutEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect
    setPhase("idle");
  }, []);

  const scanStep = useTimedSteps(SCAN_STEPS.length, SCAN_MS, phase === "scanning");
  // The scan's last tick completes the machine without another effect —
  // the live body simply takes over when every step is done.
  const live = phase === "live" || (phase === "scanning" && scanStep >= SCAN_STEPS.length);

  const revealActive = live && motion;
  const rowStep = useTimedSteps(heroRows.length, ROW_MS, revealActive);
  const revealed = revealActive ? rowStep : undefined;

  const run = () => {
    setOpenPort(null);
    setPhase(motion ? "scanning" : "live");
  };
  const togglePort = (port: number) =>
    setOpenPort((current) => (current === port ? null : port));

  const status =
    phase === "idle" ? (
      "not running"
    ) : live ? (
      <span className="text-accent">● live — {heroRows.length} listening</span>
    ) : (
      <span className="text-accent">scanning</span>
    );

  return (
    <div data-motion={revealActive ? "on" : undefined}>
      <TerminalBlock
        cmd="localhost"
        label={status}
        preClassName="text-[15px] leading-[2.05] text-foreground"
      >
        {phase === "idle" ? (
          <span className="block py-10 text-center">
            <span className="block text-faint">No ports detected</span>
            <button
              type="button"
              onClick={run}
              className="mt-5 rounded-md border border-line bg-panel px-4 py-2 font-mono text-[12.5px] text-foreground transition-colors hover:border-accent/60 hover:text-accent"
            >
              Run ports
            </button>
          </span>
        ) : live ? (
          <>
            <PortsTable
              rows={heroRows}
              variant="machine"
              revealed={revealed}
              onRowClick={togglePort}
              openPort={openPort}
              detail={CONTACT_DETAILS}
            />
            <span className="readout mt-3 block">last scan: just now · click a row</span>
          </>
        ) : (
          <span className="block py-6">
            {SCAN_STEPS.map((step, i) => {
              const done = i < scanStep;
              const current = i === scanStep;
              return (
                <span
                  key={step}
                  className={`block ${
                    done ? "text-faint" : current ? "text-foreground" : "text-faint opacity-40"
                  }`}
                >
                  {done ? "✓" : "◌"} {step}
                </span>
              );
            })}
          </span>
        )}
      </TerminalBlock>
    </div>
  );
}