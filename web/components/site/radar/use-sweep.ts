"use client";

import type { RefObject } from "react";
import { useEffect, useRef } from "react";

/** One sweep revolution, in seconds. */
export const SWEEP_SECONDS = 6;

/**
 * Blip geometry, derived deterministically from the port number so the same
 * roster always lands in the same spots — on the site and in docs snippets.
 */
export function contactAngle(port: number): number {
  // Golden-angle spread keeps neighbours well-separated.
  return (((port * 137.508) % 360) + 360) % 360;
}

export function contactRadius(port: number): number {
  // Stable per-port pseudo-random radius in [0.34, 0.94] of the radar radius.
  const h = Math.abs(Math.sin(port * 12.9898) * 43758.5453) % 1;
  return 0.34 + h * 0.58;
}

/** Nearest contact to a point (in unit-square polar space), or null. */
export function contactAt(
  dx: number,
  dy: number,
  ports: number[],
): number | null {
  const r = Math.hypot(dx, dy);
  if (r > 1.04) return null;
  let best: { port: number; d: number } | null = null;
  for (const port of ports) {
    const a = (contactAngle(port) * Math.PI) / 180;
    const px = Math.cos(a) * contactRadius(port);
    const py = Math.sin(a) * contactRadius(port);
    const d = Math.hypot(px - dx, py - dy);
    if (!best || d < best.d) best = { port, d };
  }
  return best && best.d < 0.055 ? best.port : null;
}

/**
 * Drives the sweep: one RAF loop, DPR-aware sizing via ResizeObserver, and a
 * hard stop (IntersectionObserver + reduced motion) whenever the board is
 * static or off-screen. `onFrame` is held in a ref so re-renders never
 * restart the sweep clock.
 */
export function useSweep(
  canvasRef: RefObject<HTMLCanvasElement | null>,
  onFrame: (ctx: CanvasRenderingContext2D, angleDeg: number, seconds: number) => void,
  run: boolean,
): void {
  const frameRef = useRef(onFrame);
  useEffect(() => {
    frameRef.current = onFrame;
  });

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const ctx = canvas.getContext("2d");
    if (!ctx) return;

    let size = 0;
    const resize = () => {
      const rect = canvas.getBoundingClientRect();
      if (rect.width === size) return;
      size = rect.width;
      const dpr = Math.min(window.devicePixelRatio || 1, 2);
      canvas.width = Math.round(rect.width * dpr);
      canvas.height = Math.round(rect.height * dpr);
      ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
    };
    resize();

    const staticFrame = () => {
      resize();
      frameRef.current(ctx, 292, 0);
    };

    const reduced = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
    if (reduced || !run) {
      staticFrame();
      const ro = new ResizeObserver(staticFrame);
      ro.observe(canvas);
      return () => ro.disconnect();
    }

    let raf = 0;
    let inView = false;
    let start = 0;

    const loop = (now: number) => {
      if (!start) start = now;
      resize();
      frameRef.current(ctx, ((now - start) / (SWEEP_SECONDS * 1000)) % 1 * 360, (now - start) / 1000);
      raf = requestAnimationFrame(loop);
    };

    const io = new IntersectionObserver(([entry]) => {
      inView = entry.isIntersecting;
      const shouldRun = inView && !document.hidden;
      if (shouldRun && !raf) raf = requestAnimationFrame(loop);
      if (!shouldRun && raf) {
        cancelAnimationFrame(raf);
        raf = 0;
      }
    });
    io.observe(canvas);

    const onVisibility = () => {
      const shouldRun = inView && !document.hidden;
      if (shouldRun && !raf) raf = requestAnimationFrame(loop);
      if (!shouldRun && raf) {
        cancelAnimationFrame(raf);
        raf = 0;
      }
    };
    document.addEventListener("visibilitychange", onVisibility);

    return () => {
      if (raf) cancelAnimationFrame(raf);
      io.disconnect();
      document.removeEventListener("visibilitychange", onVisibility);
    };
  }, [canvasRef, run]);
}