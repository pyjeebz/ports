"use client";

import { useEffect, useLayoutEffect, useState } from "react";

/** The site's motion gate: reduced-motion users get the static final frame. */
export function prefersReducedMotion(): boolean {
  return window.matchMedia("(prefers-reduced-motion: reduce)").matches;
}

/**
 * True unless the user prefers reduced motion. Starts true so hydration
 * matches the server's final frame, then corrects in a layout effect —
 * before paint, so a reduced-motion user never sees a frame zero.
 */
export function useMotionAllowed() {
  const [allowed, setAllowed] = useState(true);

  useLayoutEffect(() => {
    const mq = window.matchMedia("(prefers-reduced-motion: reduce)");
    const apply = () => setAllowed(!mq.matches);
    apply();
    mq.addEventListener("change", apply);
    return () => mq.removeEventListener("change", apply);
  }, []);

  return allowed;
}

/**
 * A bounded step counter — the only clock the animated sections need.
 * The server renders the final frame (step = count); when `active` turns
 * on, it resets to zero before paint and counts up, one tick per
 * interval, stopping at count. `active` is the caller's
 * in-view-and-motion-allowed flag, so nothing ticks off-screen or under
 * reduced motion.
 */
export function useTimedSteps(count: number, intervalMs: number, active: boolean) {
  const [step, setStep] = useState(count);
  const [prev, setPrev] = useState({ active, count });

  // React's render-time state adjustment: when the section arms, reset to
  // frame zero during this very render — before paint, so the final frame
  // never flashes — and when it disarms, snap back to the final frame. A
  // count change mid-run (the terminal's per-command lengths) resets too.
  if (active !== prev.active || count !== prev.count) {
    setPrev({ active, count });
    setStep(active ? 0 : count);
  }

  useEffect(() => {
    if (!active) return;
    let i = 0;
    const id = setInterval(() => {
      i += 1;
      if (i >= count) {
        clearInterval(id);
        setStep(count);
      } else {
        setStep(i);
      }
    }, intervalMs);
    return () => clearInterval(id);
  }, [active, count, intervalMs]);

  return step;
}