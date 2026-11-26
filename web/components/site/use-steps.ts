"use client";

import { useCallback, useEffect, useRef, useState } from "react";

/**
 * Plays a sequence of steps — each held for its duration — while the
 * element is on screen and the tab is visible, looping until it isn't.
 * An explainer that runs off-screen is motion nobody asked for; one that
 * waits until you're looking starts at the beginning of its story.
 */
export function useSteps<T extends HTMLElement>(durations: number[], loops = Infinity) {
  const ref = useRef<T>(null);
  const [step, setStep] = useState(0);
  const [visible, setVisible] = useState(false);
  const [round, setRound] = useState(0);
  const done = round >= loops;

  useEffect(() => {
    const el = ref.current;
    if (!el) return;
    const io = new IntersectionObserver(([e]) => setVisible(e.isIntersecting), { threshold: 0.35 });
    io.observe(el);
    const onVis = () => setVisible((v) => v && document.visibilityState === "visible");
    document.addEventListener("visibilitychange", onVis);
    return () => {
      io.disconnect();
      document.removeEventListener("visibilitychange", onVis);
    };
  }, []);

  useEffect(() => {
    if (!visible || done) return;
    const t = setTimeout(() => {
      setStep((s) => {
        const next = (s + 1) % durations.length;
        if (next === 0) setRound((r) => r + 1);
        return next;
      });
    }, durations[step]);
    return () => clearTimeout(t);
  }, [visible, step, done, durations]);

  const replay = useCallback(() => {
    setRound(0);
    setStep(0);
  }, []);

  return { ref, step, round, replay };
}
