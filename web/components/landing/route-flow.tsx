"use client";

import { ReplayIcon } from "./icons";
import { useSteps } from "./use-steps";

const ROUTES = [
  { name: "shop", port: 3000, framework: "Next.js", dir: "~/shop" },
  { name: "api.shop", port: 8000, framework: "FastAPI", dir: "~/shop/apps/api" },
  { name: "docs.shop", port: 4321, framework: "Astro", dir: "~/shop/apps/docs" },
];

const SCENARIOS = [
  { host: "shop.localhost", route: 0, answer: "200 OK · 12 ms" },
  { host: "api.shop.localhost", route: 1, answer: "200 OK · 4 ms" },
  { host: "evil.example", route: -1, answer: "" },
];

// per request: the request arrives, the router matches its Host, the dev
// server answers, then the diagram clears for the next one
const PHASES = [900, 900, 2000, 300];
const DURATIONS = SCENARIOS.flatMap(() => PHASES);

/**
 * How a request finds its server, told as three requests: two names the
 * router knows, and a Host it refuses. Each step is a short transition —
 * enter on a strong ease-out, under 300ms — and the beats between them
 * are long enough to read. Nothing here moves on its own beyond that.
 */
export function RouteFlow() {
  const { ref, step, replay } = useSteps<HTMLDivElement>(DURATIONS);
  const scenario = SCENARIOS[Math.floor(step / PHASES.length)];
  const phase = step % PHASES.length;
  const clearing = phase === 3;
  const matched = scenario.route >= 0;

  return (
    <figure ref={ref} className="my-6 overflow-hidden rounded-lg border bg-card">
      <figcaption className="flex h-9 items-center justify-between border-b bg-muted/40 pr-1 pl-3.5 text-xs text-muted-foreground">
        <span>A request, through the router</span>
        <ReplayButton onClick={replay} />
      </figcaption>

      <div
        className={`grid grid-cols-[5.5rem_1fr] gap-x-4 px-4 py-4 font-mono text-[13px] transition-opacity duration-300 ease-out sm:grid-cols-[7rem_1fr] ${clearing ? "opacity-0" : "opacity-100"}`}
      >
        <Label>browser</Label>
        <div className="flex min-h-7 items-center gap-2 overflow-hidden whitespace-nowrap">
          <span key={scenario.host} className="enter">
            <span className="text-muted-foreground">GET / · Host: </span>
            <span className={`rounded px-1 ${matched ? "bg-signal/10 text-signal" : "bg-removed-bg text-removed"}`}>
              {scenario.host}
            </span>
          </span>
        </div>

        <span />
        <Wire active={phase === 0} />

        <Label>router :80</Label>
        <div className="grid">
          {ROUTES.map((r, i) => {
            const hit = phase >= 1 && scenario.route === i;
            const dim = phase >= 1 && !hit;
            return (
              <div
                key={r.name}
                className={`flex items-center justify-between rounded-md px-2 py-1 transition-[background-color,opacity] duration-200 ease-out ${hit ? "bg-accent" : ""} ${dim ? "opacity-45" : ""}`}
              >
                <span>
                  <span className={`inline-block w-3 transition-opacity duration-200 ${hit ? "opacity-100" : "opacity-0"}`}>›</span>
                  {r.name}.localhost
                </span>
                <span className="text-muted-foreground">:{r.port}</span>
              </div>
            );
          })}
          <div className="flex min-h-7 items-center px-2 text-xs">
            {phase >= 1 && !matched && (
              <span className="enter text-removed">421 — not a .localhost name this router serves</span>
            )}
          </div>
        </div>

        <span />
        <Wire active={phase === 1 && matched} />

        <Label>dev server</Label>
        <div className="flex min-h-7 items-center justify-between gap-3 overflow-hidden whitespace-nowrap">
          {matched ? (
            <>
              <span className="flex items-center gap-2">
                <span
                  className={`size-1.5 rounded-full transition-colors duration-200 ${phase >= 2 ? "bg-live" : "bg-muted-foreground/40"}`}
                />
                :{ROUTES[scenario.route].port} {ROUTES[scenario.route].framework}
                <span className="hidden text-muted-foreground sm:inline">{ROUTES[scenario.route].dir}</span>
              </span>
              {phase >= 2 && (
                <span key={scenario.host} className="enter text-added">
                  {scenario.answer}
                </span>
              )}
            </>
          ) : (
            <span className="text-muted-foreground">nothing reached</span>
          )}
        </div>
      </div>
    </figure>
  );
}

function Label({ children }: { children: React.ReactNode }) {
  return <span className="flex min-h-7 items-start pt-1 text-xs text-muted-foreground">{children}</span>;
}

/** The hop between two rows: a hairline, and a dot that drops down it
 *  while the request is in flight. */
function Wire({ active }: { active: boolean }) {
  return (
    <div className="relative ml-3.5 h-5 w-px bg-border">
      <span
        className={`absolute -left-[2.5px] size-1.5 rounded-full bg-signal transition-[transform,opacity] ease-out motion-reduce:transition-opacity ${active ? "translate-y-[14px] opacity-100 duration-500" : "translate-y-0 opacity-0 duration-150"}`}
      />
    </div>
  );
}

export function ReplayButton({ onClick }: { onClick: () => void }) {
  return (
    <button
      type="button"
      onClick={onClick}
      aria-label="Replay"
      className="grid size-7 place-items-center rounded-md text-muted-foreground transition-[transform,color,background-color] duration-150 ease-out hover:bg-accent hover:text-foreground active:scale-[0.96]"
    >
      <ReplayIcon className="size-3.5" />
    </button>
  );
}
