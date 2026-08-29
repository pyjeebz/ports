"use client";

import { useEffect, useRef, useState } from "react";

import type { FlapScenario } from "@/lib/site";
import { CONTACTS, FLAP_SCENARIOS } from "@/lib/site";

/**
 * A split-flap session board. It types the command, then the roster rows
 * flip in, one by one, to show how each contact resolves. Replaces the old
 * typed-terminal animation.
 */
export function Flapboard() {
  const [scenarioId, setScenarioId] = useState(FLAP_SCENARIOS[0].id);
  const scenario =
    FLAP_SCENARIOS.find((s) => s.id === scenarioId) ?? FLAP_SCENARIOS[0];

  return (
    <div className="code-shell font-mono text-[13px]">
      {/* scenario tabs — toggle buttons: the "panels" are the shared board
          below, so full tab semantics would be over-engineered here */}
      <div className="flex flex-wrap border-b border-line-soft bg-panel-2">
        {FLAP_SCENARIOS.map((s) => (
          <button
            key={s.id}
            type="button"
            onClick={() => setScenarioId(s.id)}
            aria-pressed={s.id === scenarioId}
            className={`cursor-pointer border-b-2 px-3.5 py-2.5 text-[12px] transition-colors ${
              s.id === scenarioId
                ? "border-accent text-accent"
                : "border-transparent text-muted hover:text-foreground"
            }`}
          >
            {s.typed}
          </button>
        ))}
      </div>

      {/* each scenario remounts its session so the typing starts over */}
      <FlapSession key={scenario.id} scenario={scenario} />
    </div>
  );
}

function FlapSession({ scenario }: { scenario: FlapScenario }) {
  const reduced =
    typeof window !== "undefined" &&
    window.matchMedia("(prefers-reduced-motion: reduce)").matches;

  const [typed, setTyped] = useState(() =>
    reduced ? scenario.typed.length : 0,
  );
  const [running, setRunning] = useState(!reduced);
  const timer = useRef<ReturnType<typeof setInterval> | null>(null);

  useEffect(() => {
    if (reduced) return;
    let i = 0;
    timer.current = setInterval(() => {
      i += 1;
      setTyped(i);
      if (i >= scenario.typed.length) {
        if (timer.current) clearInterval(timer.current);
        // let the last keystroke sit for a beat, then flip the board
        setTimeout(() => setRunning(false), 260);
      }
    }, 42);
    return () => {
      if (timer.current) clearInterval(timer.current);
    };
  }, [reduced, scenario.typed.length]);

  const done = !running;

  return (
    <>
      {/* typed command */}
      <div className="flex items-center border-b border-line-soft px-4 py-3">
        <span className="mr-2 select-none text-faint">▸</span>
        <span className="text-accent-bright">{scenario.typed.slice(0, typed)}</span>
        {running && (
          <span className="ml-0.5 inline-block h-[15px] w-[7px] animate-pulse bg-muted align-middle" />
        )}
      </div>

      {/* roster rows */}
      <div className="flap-scene">
        {CONTACTS.map((ct, i) => {
          const outcome = scenario.outcomes[ct.port];
          return (
            <div
              key={ct.port}
              className="flex items-center justify-between gap-4 px-4 py-2.5 not-last:border-b not-last:border-b-line-soft"
            >
              <span className="min-w-0 shrink-0 tabular-nums text-muted">
                {ct.port}
              </span>
              <span className="min-w-0 flex-1 truncate text-foreground">
                {ct.framework}
              </span>
              <span className="hidden min-w-0 flex-1 truncate text-faint sm:block">
                {ct.project}
              </span>
              <span
                key={`${scenario.id}-${ct.port}`}
                aria-live="polite"
                className="flap-cell shrink-0 basis-[15ch] text-right text-[12px]"
                style={
                  done
                    ? {
                        animation: "flap-in 420ms cubic-bezier(0.2, 0.7, 0.3, 1) both",
                        animationDelay: `${i * 120}ms`,
                      }
                    : undefined
                }
              >
                {done && outcome ? (
                  <span className={outcome.state === "stopped" ? "text-accent" : "text-faint"}>
                    {outcome.label || "— untouched"}
                  </span>
                ) : (
                  <span className="text-faint">· holding</span>
                )}
              </span>
            </div>
          );
        })}
      </div>

      {/* note */}
      <p className="border-t border-line-soft bg-panel px-4 py-3 text-[11.5px] leading-relaxed text-faint">
        {scenario.note}
      </p>
    </>
  );
}