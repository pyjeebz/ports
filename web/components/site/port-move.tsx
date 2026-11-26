"use client";

import { Badge } from "@/components/ui/badge";
import { Figure } from "./figure";
import { useSteps } from "./use-steps";

// a dev server that restarts and comes back on another port, twice
const STATES = [
  { port: 3000, up: true },
  { port: 3000, up: false },
  { port: 3001, up: true },
  { port: 3001, up: false },
] as const;
const DURATIONS = [1900, 1300, 2300, 1300];

/**
 * The name doesn't move when the server does. The browser on the left
 * keeps one URL the whole time — while the server is down it shows the
 * router's own "isn't running" page, which reloads itself — and the
 * router's log on the right says where the name went. The port number
 * is the only thing that changes, and it changes in place.
 */
export function PortMove() {
  const { ref, step, round, replay } = useSteps<HTMLDivElement>(DURATIONS);
  const s = STATES[step];
  const moves = movesSoFar(step, round);

  return (
    <Figure ref={ref} title="A restart on another port" onReplay={replay}>
      <div className="grid font-mono sm:grid-cols-2">
        {/* the browser: one URL, start to finish */}
        <div className="border-b p-4 sm:border-r sm:border-b-0">
          <div className="flex h-8 items-center justify-between gap-2 rounded-md border bg-chrome pr-1 pl-2.5 font-mono text-xs">
            <span>shop.localhost</span>
            <Badge variant="secondary" className="rounded-[5px] px-1.5 font-sans text-[11px] font-normal text-muted-foreground">
              same URL throughout
            </Badge>
          </div>
          <div className="relative mt-3 h-[116px]">
            <div
              className={`absolute inset-0 transition-[opacity,filter] duration-300 ease-out ${s.up ? "opacity-100" : "opacity-0 blur-[2px]"}`}
              aria-hidden={!s.up}
            >
              <p className="font-sans text-sm font-medium">Shop</p>
              <div className="mt-3 grid gap-2">
                <span className="h-2 w-3/4 rounded-full bg-muted" />
                <span className="h-2 w-1/2 rounded-full bg-muted" />
                <span className="h-2 w-2/3 rounded-full bg-muted" />
              </div>
              <div className="mt-4 flex gap-2">
                <span className="h-6 w-16 rounded-md bg-muted" />
                <span className="h-6 w-12 rounded-md border" />
              </div>
            </div>
            <div
              className={`absolute inset-0 font-mono text-xs leading-5 transition-[opacity,filter] duration-300 ease-out ${s.up ? "opacity-0 blur-[2px]" : "opacity-100"}`}
              aria-hidden={s.up}
            >
              <p className="font-sans text-sm font-medium">shop isn&apos;t running</p>
              <p className="mt-1.5 text-muted-foreground">
                It was Next.js on :{s.port}, in ~/shop.
                <br />
                Start it again and this page reloads into it.
              </p>
            </div>
          </div>
        </div>

        {/* the router's log, and the server */}
        <div className="flex flex-col p-4 font-mono text-xs leading-6">
          <span className="text-muted-foreground">$ ports proxy</span>
          <span>
            <span className="text-green">✓</span> routing *.localhost on :80
          </span>
          {moves.map((m) => (
            <span key={m.key} className="enter text-blue">
              ~ http://shop.localhost → :{m.port} Next.js
            </span>
          ))}
          <span className="mt-auto flex items-center gap-2 pt-3">
            <span
              className={`size-1.5 rounded-full transition-colors duration-200 ${s.up ? "bg-green" : "bg-muted-foreground/40"}`}
            />
            <span className="text-muted-foreground">dev server</span>
            {s.up ? (
              <span key={`up-${s.port}`} className="enter">
                :{s.port} Next.js
              </span>
            ) : (
              <span key={`down-${s.port}`} className="enter text-muted-foreground">
                restarting…
              </span>
            )}
          </span>
        </div>
      </div>
    </Figure>
  );
}

// movesSoFar is the router's log for this pass: a "~" line each time the
// server came back on a different port, the newest last, at most two.
function movesSoFar(step: number, round: number) {
  const out: { key: string; port: number }[] = [];
  if (round > 0 && step < 2) out.push({ key: `${round}-0`, port: 3000 });
  if (step >= 2) out.push({ key: `${round}-2`, port: 3001 });
  return out;
}
