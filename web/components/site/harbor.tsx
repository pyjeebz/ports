"use client";

import { useEffect, useRef } from "react";

const ARIA =
  "A container ship named LOCALHOST sails in from the left and docks at a night harbor. A crane lifts framework containers off its deck and sets each one on a berth numbered with its TCP port.";

export function Harbor() {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const tickerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const cv = canvasRef.current;
    const tickerEl = tickerRef.current;
    if (!cv || !tickerEl) return;
    const ctx = cv.getContext("2d");
    if (!ctx) return;

    const reduced = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
    const W = 1200,
      H = 480,
      HORIZON = 300;

    const COL = {
      line: "#232327",
      soft: "#1a1a1e",
      hull: "#141417",
      hullEdge: "#33333a",
      box: "#191a1e",
      boxEdge: "#3a3a42",
      ink: "#ededed",
      muted: "#9d9da6",
      faint: "#62626b",
      green: "#4ade80",
      water: "#0c0c0f",
      sky: "#0a0a0a",
    };

    const TIER1 = [
      { t: "VITE", x: 244 },
      { t: "DJANGO", x: 319 },
      { t: "NODE", x: 394 },
      { t: "OLLAMA", x: 469 },
    ];
    const TIER2 = [
      { t: "DOCKER", x: 282 },
      { t: "GO", x: 357 },
      { t: "NGINX", x: 432 },
    ];
    const STACK_X = 560,
      BOXW = 70,
      BOXH = 26;
    const CARGO = [
      { name: "PostgreSQL", tag: "POSTGRES", port: 5432, dropX: 920 },
      { name: "Redis", tag: "REDIS", port: 6379, dropX: 995 },
      { name: "Next.js", tag: "NEXT.JS", port: 3000, dropX: 1070 },
      { name: "FastAPI", tag: "FASTAPI", port: 8000, dropX: 1145 },
    ];
    const stackTopY = (i: number) => 180 + i * 28;

    const QUAY_Y = 268,
      BEAM_Y = 118,
      PICK_X = STACK_X + BOXW / 2,
      CARRY_Y = 148,
      REST_Y = QUAY_Y - 24;

    const ARRIVE = 5.5,
      PER = 7.0,
      HOLD = 4.0,
      FADE = 1.2;
    const TOTAL = ARRIVE + CARGO.length * PER + HOLD + FADE;
    const ez = (x: number) =>
      x <= 0 ? 0 : x >= 1 ? 1 : x < 0.5 ? 2 * x * x : 1 - Math.pow(-2 * x + 2, 2) / 2;
    const seg = (t: number, a: number, d: number) => ez((t - a) / d);
    const PH: Record<string, number> = {
      toPick: 1.3,
      down: 1.0,
      latch: 0.3,
      up: 1.0,
      move: 1.5,
      place: 0.9,
      rel: 0.3,
      home: 0.7,
    };

    const shipOffset = (t: number) => (t >= ARRIVE ? 0 : -900 * (1 - ez(t / ARRIVE)));

    type CraneState = {
      tx: number;
      hook: number;
      carrying: boolean;
      picked: number;
      placed: number;
    };

    function craneState(t: number): CraneState {
      if (t < ARRIVE)
        return { tx: 985, hook: CARRY_Y, carrying: false, picked: 0, placed: 0 };
      let u = t - ARRIVE;
      if (u >= CARGO.length * PER)
        return { tx: CARGO[3].dropX, hook: CARRY_Y, carrying: false, picked: 4, placed: 4 };
      const i = Math.floor(u / PER);
      u -= i * PER;
      const c = CARGO[i];
      const tx0 = i === 0 ? 985 : CARGO[i - 1].dropX;
      const pickY = stackTopY(i);
      const st: CraneState = {
        tx: tx0,
        hook: CARRY_Y,
        carrying: false,
        picked: i,
        placed: i,
      };
      let a = 0;
      for (const k of Object.keys(PH)) {
        const d = PH[k];
        const p = seg(u, a, d);
        switch (k) {
          case "toPick":
            st.tx = tx0 + (PICK_X - tx0) * p;
            break;
          case "down":
            st.hook = CARRY_Y + (pickY - CARRY_Y) * p;
            break;
          case "latch":
            if (u >= a + d * 0.5) {
              st.carrying = true;
              st.picked = i + 1;
            }
            break;
          case "up":
            st.carrying = true;
            st.picked = i + 1;
            st.hook = pickY + (CARRY_Y - pickY) * p;
            break;
          case "move":
            st.tx = PICK_X + (c.dropX - PICK_X) * p;
            break;
          case "place":
            st.hook = CARRY_Y + (REST_Y - CARRY_Y) * p;
            break;
          case "rel":
            if (u >= a + d * 0.5) {
              st.carrying = false;
              st.placed = i + 1;
            }
            break;
          case "home":
            st.hook = REST_Y + (CARRY_Y - REST_Y) * p;
            break;
        }
        a += d;
        if (u < a) break;
      }
      return st;
    }

    let seed = 7;
    const rnd = () => (seed = (seed * 16807) % 2147483647) / 2147483647;
    const STARS = Array.from({ length: 42 }, () => ({
      x: rnd() * W,
      y: rnd() * (HORIZON - 90),
      a: 0.25 + rnd() * 0.5,
      p: rnd() * 6.28,
    }));

    function drawBoxAt(x: number, y: number, w: number, label: string, labelCol?: string) {
      const c = ctx!;
      c.fillStyle = COL.box;
      c.strokeStyle = COL.boxEdge;
      c.fillRect(x, y, w, BOXH);
      c.strokeRect(x + 0.5, y + 0.5, w, BOXH);
      c.strokeStyle = COL.soft;
      c.beginPath();
      c.moveTo(x + 4, y + 3);
      c.lineTo(x + 4, y + BOXH - 3);
      c.moveTo(x + w - 4, y + 3);
      c.lineTo(x + w - 4, y + BOXH - 3);
      c.stroke();
      c.fillStyle = labelCol || COL.muted;
      c.font = "500 8px 'Geist Mono', monospace";
      c.textAlign = "center";
      c.fillText(label, x + w / 2, y + 17);
    }
    function line(a: number, b: number, cc: number, d: number) {
      const c = ctx!;
      c.beginPath();
      c.moveTo(a, b);
      c.lineTo(cc, d);
      c.stroke();
    }
    function streak(
      x: number,
      top: number,
      h: number,
      w: number,
      color: string,
      alpha: number,
      t: number,
    ) {
      const c = ctx!;
      const wob = Math.sin(t * 1.3 + x) * 2;
      const g = c.createLinearGradient(0, top, 0, top + h);
      g.addColorStop(0, color);
      g.addColorStop(1, "transparent");
      c.globalAlpha = alpha;
      c.fillStyle = g;
      c.fillRect(x - w / 2 + wob, top, w, h);
      c.globalAlpha = 1;
    }

    function drawScene(t: number) {
      const c = ctx!;
      const loopT = t % TOTAL;
      const fade =
        loopT > TOTAL - FADE
          ? 1 - (loopT - (TOTAL - FADE)) / FADE
          : loopT < 0.8
            ? loopT / 0.8
            : 1;
      c.clearRect(0, 0, W, H);

      c.fillStyle = COL.sky;
      c.fillRect(0, 0, W, HORIZON);
      for (const s of STARS) {
        c.globalAlpha = s.a * (0.7 + 0.3 * Math.sin(t * 0.8 + s.p));
        c.fillStyle = COL.muted;
        c.fillRect(s.x, s.y, 1.3, 1.3);
      }
      c.globalAlpha = 1;
      c.fillStyle = "#c9c9d1";
      c.globalAlpha = 0.8;
      c.beginPath();
      c.arc(1058, 66, 20, 0, 6.29);
      c.fill();
      c.fillStyle = COL.sky;
      c.globalAlpha = 1;
      c.beginPath();
      c.arc(1050, 60, 18, 0, 6.29);
      c.fill();

      c.fillStyle = COL.water;
      c.fillRect(0, HORIZON, W, H - HORIZON);
      c.strokeStyle = COL.line;
      c.lineWidth = 1;
      c.beginPath();
      c.moveTo(0, HORIZON + 0.5);
      c.lineTo(W, HORIZON + 0.5);
      c.stroke();
      for (let y = HORIZON + 18; y < H - 6; y += 14) {
        const off = (t * 12 * (y % 28 ? 1 : -1)) % 34;
        c.globalAlpha = 0.05 + 0.05 * ((y - HORIZON) / (H - HORIZON));
        c.strokeStyle = COL.muted;
        c.setLineDash([16, 18]);
        c.lineDashOffset = off;
        c.beginPath();
        c.moveTo(0, y + Math.sin(t * 0.7 + y) * 1.2);
        c.lineTo(W, y + Math.sin(t * 0.7 + y) * 1.2);
        c.stroke();
      }
      c.setLineDash([]);
      c.globalAlpha = 1;

      const by = HORIZON + 26 + Math.sin(t * 1.4) * 2.5;
      c.strokeStyle = COL.hullEdge;
      c.fillStyle = COL.hull;
      c.beginPath();
      c.moveTo(62, by);
      c.lineTo(78, by);
      c.lineTo(74, by - 14);
      c.lineTo(66, by - 14);
      c.closePath();
      c.fill();
      c.stroke();
      c.beginPath();
      c.moveTo(70, by - 14);
      c.lineTo(70, by - 22);
      c.stroke();
      if (Math.sin(t * 2.1) > 0.55) {
        c.fillStyle = COL.green;
        c.beginPath();
        c.arc(70, by - 24, 2.4, 0, 6.29);
        c.fill();
        streak(70, by + 6, 26, 3, COL.green, 0.18, t);
      }

      c.globalAlpha = fade;
      const st = craneState(loopT);
      const arriving = loopT < ARRIVE;
      const ax = shipOffset(loopT);
      const sx = ax + (arriving ? 0 : Math.sin(t * 0.22) * 2.5);
      const bob = Math.sin(t * 0.9) * 2.2;

      c.save();
      c.translate(sx, bob);
      c.fillStyle = COL.hull;
      c.strokeStyle = COL.hullEdge;
      c.beginPath();
      c.moveTo(170, 292);
      c.lineTo(700, 292);
      c.lineTo(668, 348);
      c.lineTo(196, 348);
      c.closePath();
      c.fill();
      c.stroke();
      c.fillStyle = "#0b0b0d";
      c.fillRect(186, 336, 496, 12);
      c.fillStyle = COL.faint;
      c.font = "600 10px 'Geist Mono', monospace";
      c.textAlign = "left";
      c.fillText("LOCALHOST", 210, 316);
      c.font = "400 7px 'Geist Mono', monospace";
      c.fillText("127.0.0.1", 210, 328);
      if (Math.sin(t * 1.6) > 0) {
        c.fillStyle = COL.green;
        c.beginPath();
        c.arc(694, 288, 2, 0, 6.29);
        c.fill();
      }

      c.fillStyle = COL.hull;
      c.strokeStyle = COL.hullEdge;
      c.fillRect(178, 224, 54, 68);
      c.strokeRect(178.5, 224.5, 54, 68);
      c.fillRect(190, 208, 30, 16);
      c.strokeRect(190.5, 208.5, 30, 16);
      c.beginPath();
      c.moveTo(205, 208);
      c.lineTo(205, 192);
      c.stroke();
      if (Math.sin(t * 1.1 + 2) > -0.2) {
        c.fillStyle = COL.muted;
        c.beginPath();
        c.arc(205, 190, 1.6, 0, 6.29);
        c.fill();
      }
      for (let r = 0; r < 2; r++)
        for (let cc = 0; cc < 4; cc++) {
          c.globalAlpha = fade * ((r + cc) % 3 ? 0.55 : 0.2);
          c.fillStyle = COL.muted;
          c.fillRect(188 + cc * 10, 238 + r * 14, 5, 4);
        }
      c.globalAlpha = fade;

      for (const b of TIER1) drawBoxAt(b.x, 264, BOXW, b.t);
      for (const b of TIER2) drawBoxAt(b.x, 236, BOXW, b.t);
      CARGO.forEach((cg, i) => {
        if (i < st.picked) return;
        drawBoxAt(STACK_X, stackTopY(i), BOXW, cg.tag);
      });

      c.strokeStyle = COL.muted;
      if (arriving) {
        const speed = 1 - loopT / ARRIVE;
        for (let i = 0; i < 4; i++) {
          const ph = (t * 1.2 + i * 0.25) % 1;
          c.globalAlpha = fade * 0.35 * (0.3 + speed) * (1 - ph);
          c.beginPath();
          c.arc(176 - ph * 55, 348, 4 + ph * 13, Math.PI * 1.2, Math.PI * 1.9);
          c.stroke();
        }
      } else {
        for (let i = 0; i < 3; i++) {
          const ph = (t * 0.5 + i * 0.33) % 1;
          c.globalAlpha = fade * 0.22 * (1 - ph);
          c.beginPath();
          c.arc(700 + ph * 26, 348, 6 + ph * 14, Math.PI * 1.15, Math.PI * 1.85);
          c.stroke();
        }
      }
      c.globalAlpha = fade;
      c.restore();

      streak(205 + sx, 352, 40, 4, COL.muted, 0.1 * fade, t);
      streak(430 + sx, 352, 60, 5, COL.muted, 0.07 * fade, t);
      streak(694 + sx, 352, 30, 3, COL.green, 0.1 * fade, t);

      c.fillStyle = "#0e0e11";
      c.strokeStyle = COL.line;
      c.fillRect(870, QUAY_Y, 330, H - QUAY_Y);
      c.beginPath();
      c.moveTo(870, QUAY_Y + 0.5);
      c.lineTo(1200, QUAY_Y + 0.5);
      c.stroke();
      c.beginPath();
      c.moveTo(870.5, QUAY_Y);
      c.lineTo(870.5, H);
      c.stroke();
      c.fillStyle = COL.faint;
      c.font = "600 11px 'Geist Mono', monospace";
      c.textAlign = "center";
      c.globalAlpha = fade * 0.85;
      for (const cg of CARGO) c.fillText(":" + cg.port, cg.dropX, 296);
      c.globalAlpha = fade;
      for (const bxx of [886, 961, 1036, 1111, 1186]) {
        c.fillStyle = COL.hullEdge;
        c.fillRect(bxx, QUAY_Y - 7, 8, 7);
      }
      c.strokeStyle = COL.soft;
      c.setLineDash([10, 8]);
      c.beginPath();
      c.moveTo(880, QUAY_Y + 14);
      c.lineTo(1195, QUAY_Y + 14);
      c.stroke();
      c.setLineDash([]);

      CARGO.forEach((cg, i) => {
        if (i >= st.placed) return;
        drawBoxAt(cg.dropX - BOXW / 2, REST_Y, 68, cg.tag, COL.ink);
        if (Math.sin(t * 3 + i) > 0) {
          c.fillStyle = COL.green;
          c.beginPath();
          c.arc(cg.dropX + 26, REST_Y + 6, 1.6, 0, 6.29);
          c.fill();
        }
      });

      c.strokeStyle = COL.hullEdge;
      c.lineWidth = 2;
      line(920, QUAY_Y, 920, BEAM_Y);
      line(1150, QUAY_Y, 1150, BEAM_Y);
      line(908, QUAY_Y, 932, QUAY_Y);
      line(1138, QUAY_Y, 1162, QUAY_Y);
      line(540, BEAM_Y, 1170, BEAM_Y);
      c.lineWidth = 1;
      c.strokeStyle = COL.line;
      line(920, BEAM_Y, 1000, BEAM_Y - 26);
      line(1000, BEAM_Y - 26, 1150, BEAM_Y);
      line(920, BEAM_Y - 26 + 4, 700, BEAM_Y - 2);
      line(920, 200, 1150, 200);
      if (Math.sin(t * 1.8 + 1) > 0.3) {
        c.fillStyle = COL.muted;
        c.beginPath();
        c.arc(542, BEAM_Y - 4, 1.8, 0, 6.29);
        c.fill();
      }

      c.fillStyle = COL.hullEdge;
      c.fillRect(st.tx - 12, BEAM_Y - 3, 24, 8);
      c.strokeStyle = COL.line;
      line(st.tx, BEAM_Y + 4, st.tx, st.hook - 2);
      c.fillStyle = COL.muted;
      c.fillRect(st.tx - 10, st.hook - 3, 20, 3);
      if (st.carrying) {
        const cg = CARGO[Math.min(st.picked - 1, 3)];
        drawBoxAt(st.tx - BOXW / 2, st.hook, BOXW, cg.tag, COL.ink);
      }

      streak(1054, HORIZON + 8, 90, 5, "#c9c9d1", 0.08, t);
      c.globalAlpha = 1;
    }

    let lastMsg = "";
    function ticker(loopT: number) {
      let msg: string;
      const ok = (s: string) => `<span style="color:${COL.green}">${s}</span>`;
      if (loopT < ARRIVE - 1.2) msg = "m/v localhost · inbound";
      else if (loopT < ARRIVE) msg = "m/v localhost · moored — commencing discharge";
      else if (loopT < ARRIVE + CARGO.length * PER) {
        const i = Math.min(Math.floor((loopT - ARRIVE) / PER), 3);
        const u = loopT - ARRIVE - i * PER;
        const cg = CARGO[i];
        const placeAt =
          PH.toPick + PH.down + PH.latch + PH.up + PH.move + PH.place + PH.rel * 0.5;
        msg =
          u < placeAt
            ? `unloading ${cg.name.toLowerCase()} — berth :${cg.port}`
            : `${ok("✓")} ${cg.name} berthed at ${ok(":" + cg.port)}`;
      } else msg = `${ok("✓")} manifest complete — 4 services berthed`;
      if (msg !== lastMsg) {
        tickerEl!.innerHTML = msg;
        lastMsg = msg;
      }
    }

    function resize() {
      const dpr = Math.min(window.devicePixelRatio || 1, 2);
      const w = cv!.clientWidth;
      if (!w) return;
      cv!.width = w * dpr;
      cv!.height = ((w * H) / W) * dpr;
      ctx!.setTransform((w * dpr) / W, 0, 0, (w * dpr) / W, 0, 0);
    }

    let raf = 0;
    let visible = false;
    let t0: number | null = null;
    const io = new IntersectionObserver(
      (es) => {
        visible = es[0].isIntersecting;
        if (visible && !cv.width) resize();
      },
      { threshold: 0.2 },
    );
    io.observe(cv);
    const onResize = () => {
      resize();
      if (reduced) drawScene(ARRIVE + PER * 2 + 3);
    };
    window.addEventListener("resize", onResize);

    resize();
    if (reduced) {
      const t = ARRIVE + PER * 2 + 3;
      drawScene(t);
      ticker(t);
    } else {
      const frame = (now: number) => {
        if (visible) {
          if (t0 === null) t0 = now;
          const t = (now - t0) / 1000;
          drawScene(t);
          ticker(t % TOTAL);
        }
        raf = requestAnimationFrame(frame);
      };
      raf = requestAnimationFrame(frame);
    }

    return () => {
      cancelAnimationFrame(raf);
      io.disconnect();
      window.removeEventListener("resize", onResize);
    };
  }, []);

  return (
    <div className="overflow-hidden rounded-md border border-line-soft bg-panel">
      <div className="flex items-center justify-between border-b border-line-soft px-4 py-[9px] font-mono text-[11px] uppercase tracking-[0.1em] text-faint">
        <span>127.0.0.1</span>
        <span className="flex items-center gap-1.5 text-accent">
          <span className="h-[5px] w-[5px] rounded-full bg-accent" />
          live
        </span>
      </div>
      <canvas
        ref={canvasRef}
        width={1200}
        height={480}
        role="img"
        aria-label={ARIA}
        className="block h-auto w-full"
      />
      <div
        ref={tickerRef}
        aria-live="polite"
        className="min-h-[44px] border-t border-line-soft py-[13px] text-center font-mono text-[12px] text-faint"
      >
        m/v localhost · inbound
      </div>
    </div>
  );
}
