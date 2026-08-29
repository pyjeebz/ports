"use client";

import { useCallback, useEffect, useRef } from "react";

import type { Contact } from "@/lib/site";
import { contactAngle, contactRadius, contactAt, useSweep } from "./use-sweep";

const RING_COLOR = "#1e2a24";
const TICK_COLOR = "#5f7269";

export function RadarBoard({
  contacts,
  selected,
  onSelect,
}: {
  contacts: readonly Contact[];
  selected: number | null;
  onSelect: (port: number | null) => void;
}) {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const selectedRef = useRef<number | null>(null);
  useEffect(() => {
    selectedRef.current = selected;
  }, [selected]);

  const draw = useCallback(
    (ctx: CanvasRenderingContext2D, angleDeg: number) => {
      const canvas = ctx.canvas;
      const dpr = Math.min(window.devicePixelRatio || 1, 2) || 1;
      const s = canvas.width / dpr; // logical (square) size
      const c = s / 2;
      const R = c - 14;

      ctx.clearRect(0, 0, s, s);

      // plotting rings + crosshairs
      ctx.strokeStyle = RING_COLOR;
      ctx.lineWidth = 1;
      for (const f of [0.34, 0.67, 1]) {
        ctx.beginPath();
        ctx.arc(c, c, R * f, 0, Math.PI * 2);
        ctx.stroke();
      }
      ctx.beginPath();
      ctx.moveTo(c - R, c);
      ctx.lineTo(c + R, c);
      ctx.moveTo(c, c - R);
      ctx.lineTo(c, c + R);
      ctx.stroke();

      // rim ticks, every 30°
      ctx.strokeStyle = TICK_COLOR;
      for (let d = 0; d < 360; d += 30) {
        const rad = (d * Math.PI) / 180;
        ctx.beginPath();
        ctx.moveTo(c + Math.cos(rad) * (R - 5), c + Math.sin(rad) * (R - 5));
        ctx.lineTo(c + Math.cos(rad) * R, c + Math.sin(rad) * R);
        ctx.stroke();
      }

      // sweep trail — 54 segments fading behind the beam
      const a = (angleDeg * Math.PI) / 180;
      for (let i = 0; i < 54; i++) {
        const ta = a - (i * 1.1 * Math.PI) / 180;
        ctx.strokeStyle =
          i === 0
            ? "rgba(169, 255, 201, 0.85)"
            : `rgba(51, 229, 129, ${((1 - i / 54) * 0.055).toFixed(4)})`;
        ctx.beginPath();
        ctx.moveTo(c, c);
        ctx.lineTo(c + Math.cos(ta) * R, c + Math.sin(ta) * R);
        ctx.stroke();
      }

      // blips
      ctx.textBaseline = "middle";
      for (const contact of contacts) {
        const ca = (contactAngle(contact.port) * Math.PI) / 180;
        const cr = contactRadius(contact.port) * R;
        const x = c + Math.cos(ca) * cr;
        const y = c + Math.sin(ca) * cr;

        // degrees since the beam last passed this blip
        const passed = (((angleDeg - contactAngle(contact.port)) % 360) + 360) % 360;
        const flash = passed < 90 ? 1 - passed / 90 : 0;
        const isSel = selectedRef.current === contact.port;

        // expanding ping ring right after the beam crosses
        if (passed < 55) {
          const t = passed / 55;
          ctx.strokeStyle = `rgba(51, 229, 129, ${((1 - t) * 0.45).toFixed(3)})`;
          ctx.beginPath();
          ctx.arc(x, y, 5 + t * 22, 0, Math.PI * 2);
          ctx.stroke();
        }

        ctx.fillStyle = isSel
          ? "#a9ffc9"
          : `rgba(51, 229, 129, ${(0.38 + flash * 0.62).toFixed(3)})`;
        ctx.beginPath();
        ctx.arc(x, y, isSel ? 5 : 3.5, 0, Math.PI * 2);
        ctx.fill();

        if (flash > 0 || isSel) {
          ctx.fillStyle = `rgba(51, 229, 129, ${(flash * 0.3 + (isSel ? 0.15 : 0)).toFixed(3)})`;
          ctx.beginPath();
          ctx.arc(x, y, 11, 0, Math.PI * 2);
          ctx.fill();
        }

        // decorative labels — the visible legend carries the real data
        ctx.font = "500 10.5px ui-monospace, monospace";
        ctx.textAlign = Math.cos(ca) < -0.2 ? "right" : "left";
        ctx.fillStyle =
          isSel
            ? "rgba(169, 255, 201, 0.95)"
            : flash > 0.45
              ? "rgba(169, 255, 201, 0.8)"
              : `rgba(157, 179, 167, ${(0.28 + flash * 0.42).toFixed(3)})`;
        ctx.fillText(
          contact.framework,
          x + Math.cos(ca) * 13,
          y + Math.sin(ca) * 13,
        );
      }
    },
    [contacts],
  );

  useSweep(canvasRef, draw, true);

  function onMouseMove(e: React.MouseEvent<HTMLDivElement>) {
    const rect = e.currentTarget.getBoundingClientRect();
    const dx = ((e.clientX - rect.left) / rect.width) * 2 - 1;
    const dy = ((e.clientY - rect.top) / rect.height) * 2 - 1;
    onSelect(contactAt(dx, dy, contacts.map((ct) => ct.port)));
  }

  // The canvas is decorative; the contact legend is the real, keyboardable UI.
  return (
    <div
      className="relative aspect-square w-full max-w-[480px]"
      aria-hidden="true"
      onMouseMove={onMouseMove}
      onMouseLeave={() => onSelect(null)}
    >
      <canvas ref={canvasRef} className="block h-full w-full" />
    </div>
  );
}