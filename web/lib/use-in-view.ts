"use client";

import { useEffect, useRef, useState } from "react";

/**
 * IntersectionObserver wrapper. The server renders the final frame of
 * every animated section; the client only needs to know when that frame
 * enters the viewport so the animation can play once, from the top.
 *
 * `once` (the default) fires a single time and stops observing — the
 * reveal sections. `once: false` tracks visibility continuously — the
 * looping terminals, so their loops pause when scrolled away.
 */
export function useInView<T extends Element>({
  threshold = 0.25,
  once = true,
}: { threshold?: number; once?: boolean } = {}) {
  const ref = useRef<T | null>(null);
  const [inView, setInView] = useState(false);

  useEffect(() => {
    const el = ref.current;
    if (!el || (once && inView)) return;
    if (typeof IntersectionObserver === "undefined") {
      // No observer (ancient browsers, test environments): don't animate.
      return;
    }
    const observer = new IntersectionObserver(
      (entries) => {
        const hit = entries.some((e) => e.isIntersecting);
        if (hit) setInView(true);
        else if (!once) setInView(false);
        if (once && hit) observer.disconnect();
      },
      { threshold },
    );
    observer.observe(el);
    return () => observer.disconnect();
  }, [inView, once, threshold]);

  return [ref, inView] as const;
}