"use client";

import { useState } from "react";

import { FIND_SHOP, PID_HUNT } from "@/lib/site";
import { useMotionAllowed, useTimedSteps } from "@/lib/motion";
import { SectionHead } from "./section-head";
import { TerminalBlock } from "./terminal-block";

const STEP_MS = 320;

type Way = "ports" | "lsof";

/**
 * The problem section as a runnable comparison: one terminal, two ways to
 * answer the same question. The visitor runs the lsof way — six steps,
 * each one answering less, ending in a dead end — then the ports way:
 * one command, the whole answer, instantly. The resting state (and the
 * server frame) is the ports way; reduced motion keeps the story and
 * drops the pacing.
 */
export function PidHunt() {
  const motion = useMotionAllowed();
  const [way, setWay] = useState<Way>("ports");

  const lsofActive = way === "lsof" && motion;
  const step = useTimedSteps(PID_HUNT.length, STEP_MS, lsofActive);
  // The completed chain is the resting frame — instant for reduced
  // motion, and the state the counter returns to.
  const shown = lsofActive ? step : PID_HUNT.length;

  const tab = (id: Way, label: string) => (
    <button
      key={id}
      type="button"
      aria-pressed={way === id}
      onClick={() => setWay(id)}
      className={`rounded-md border px-4 py-2 font-mono text-[12.5px] transition-colors ${
        way === id
          ? "border-accent/60 bg-accent-soft text-accent"
          : "border-line bg-panel text-muted hover:border-strong/40 hover:text-foreground"
      }`}
    >
      {label}
    </button>
  );

  return (
    <section className="border-b border-line py-24">
      <div className="mx-auto max-w-[1160px] px-8">
        <SectionHead
          kicker="the problem"
          title="Stop hunting PIDs."
          sub="The workflow every developer knows by heart — and none of it tells you which project you are looking at. Run both ways."
        />

        <div className="mt-12 max-w-[760px]">
          <div className="flex flex-wrap items-center gap-3">
            {tab("lsof", "the lsof way")}
            {tab("ports", "ports find shop")}
          </div>

          <div className="mt-5">
            {way === "ports" ? (
              <TerminalBlock
                cmd="ports find shop"
                label="3 services · 1 project"
                preClassName="text-[14px] leading-[2] text-foreground"
              >
                {FIND_SHOP.output.map((line, i) => (
                  <span key={i} className={i === 0 ? "text-faint" : undefined}>
                    {line}
                    {"\n"}
                  </span>
                ))}
              </TerminalBlock>
            ) : (
              <TerminalBlock
                cmd="the lsof way"
                label="six steps · no answers"
                preClassName="text-[14px] leading-[2] text-foreground"
              >
                {PID_HUNT.map((s, i) => (
                  <span
                    key={s.step}
                    className={`block transition-opacity duration-300 ${
                      i < shown ? "opacity-100" : "opacity-0"
                    } ${i === PID_HUNT.length - 1 ? "text-alert" : "text-muted"}`}
                  >
                    {s.step}
                  </span>
                ))}
              </TerminalBlock>
            )}
          </div>
        </div>

        <p className="mt-14 max-w-[46ch] text-[clamp(1.15rem,2vw,1.5rem)] font-semibold leading-[1.3] tracking-[-0.02em] text-strong">
          One command. Context everywhere.
        </p>
      </div>
    </section>
  );
}