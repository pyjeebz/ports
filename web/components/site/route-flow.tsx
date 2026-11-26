"use client";

import { Badge } from "@/components/ui/badge";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { Figure } from "./figure";
import { useSteps } from "./use-steps";

const ROUTES = [
  { name: "shop", port: 3000, framework: "Next.js", dir: "~/shop", pid: 48391, up: "14m" },
  { name: "api.shop", port: 8000, framework: "FastAPI", dir: "~/shop/apps/api", pid: 48412, up: "14m" },
  { name: "docs.shop", port: 4321, framework: "Astro", dir: "~/shop/apps/docs", pid: 51207, up: "3h" },
];

const SCENARIOS = [
  { host: "shop.localhost", route: 0, answer: "200 OK", ms: "12 ms" },
  { host: "api.shop.localhost", route: 1, answer: "200 OK", ms: "4 ms" },
  { host: "evil.example", route: -1, answer: "", ms: "" },
];

// per request: it arrives, the router matches its Host, the dev server
// answers, then the diagram clears for the next one
const PHASES = [900, 900, 2000, 300];
const DURATIONS = SCENARIOS.flatMap(() => PHASES);

/**
 * How a request finds its server, told as three requests: two names the
 * router knows, and a Host it refuses. Each step is one short transition
 * on a strong ease-out; the beats between are long enough to read.
 * Hovering a route shows what `ports` knows about the process behind it.
 */
export function RouteFlow() {
  const { ref, step, replay } = useSteps<HTMLDivElement>(DURATIONS);
  const scenario = SCENARIOS[Math.floor(step / PHASES.length)];
  const phase = step % PHASES.length;
  const matched = scenario.route >= 0;
  const route = matched ? ROUTES[scenario.route] : undefined;

  return (
    <Figure ref={ref} title="A request, through the router" onReplay={replay}>
      <div
        className={`grid grid-cols-[4.25rem_1fr] gap-x-3 px-3 py-5 font-mono text-[12px] sm:grid-cols-[7rem_1fr] sm:gap-x-4 sm:px-4 sm:text-[13px] transition-opacity duration-300 ease-out ${phase === 3 ? "opacity-0" : "opacity-100"}`}
      >
        <Label>browser</Label>
        <div className="flex min-h-7 items-center overflow-hidden whitespace-nowrap">
          <span key={scenario.host} className="enter items-center gap-1.5">
            <span className="text-muted-foreground"><span className="hidden sm:inline">GET / · </span>Host:</span>
            <Badge
              variant="outline"
              className={`rounded-md font-mono text-[12px] font-normal ${matched ? "border-blue/25 bg-blue-soft text-blue" : "border-red/25 bg-red-soft text-red"}`}
            >
              {scenario.host}
            </Badge>
          </span>
        </div>

        <span />
        <Wire active={phase === 0} />

        <Label>router :80</Label>
        <div className="grid gap-0.5">
          {ROUTES.map((r, i) => {
            const hit = phase >= 1 && scenario.route === i;
            const dim = phase >= 1 && !hit;
            return (
              <Tooltip key={r.name}>
                <TooltipTrigger asChild>
                  <div
                    className={`flex cursor-default items-center justify-between rounded-md px-2 py-1 transition-[background-color,opacity] duration-200 ease-out ${hit ? "bg-accent" : "hover:bg-accent/60"} ${dim ? "opacity-40" : ""}`}
                  >
                    <span className="flex items-center">
                      <span
                        className={`mr-1.5 size-1.5 rounded-full bg-blue transition-[opacity,transform] duration-200 ease-out ${hit ? "scale-100 opacity-100" : "scale-50 opacity-0"}`}
                      />
                      {r.name}.localhost
                    </span>
                    <span className="text-muted-foreground">:{r.port}</span>
                  </div>
                </TooltipTrigger>
                <TooltipContent side="right" className="font-mono">
                  {r.framework} · pid {r.pid} · up {r.up} · {r.dir}
                </TooltipContent>
              </Tooltip>
            );
          })}
          <div className="flex min-h-7 items-center px-2">
            {phase >= 1 && !matched && (
              <span className="enter items-center gap-2">
                <Badge variant="outline" className="rounded-md border-red/25 bg-red-soft font-mono font-normal text-red">
                  421
                </Badge>
                <span className="text-xs text-muted-foreground">not a name this router serves</span>
              </span>
            )}
          </div>
        </div>

        <span />
        <Wire active={phase === 1 && matched} />

        <Label>dev server</Label>
        <div className="flex min-h-7 items-center justify-between gap-3 overflow-hidden whitespace-nowrap">
          {route ? (
            <>
              <span className="flex items-center gap-2">
                <span
                  className={`size-1.5 rounded-full transition-colors duration-200 ${phase >= 2 ? "bg-green" : "bg-muted-foreground/40"}`}
                />
                :{route.port} {route.framework}
                <span className="hidden text-muted-foreground sm:inline">{route.dir}</span>
              </span>
              {phase >= 2 && (
                <Badge
                  key={scenario.host}
                  variant="outline"
                  className="enter rounded-md border-green/25 bg-green-soft font-mono font-normal text-green"
                >
                  {scenario.answer}
                  <span className="hidden sm:inline">· {scenario.ms}</span>
                </Badge>
              )}
            </>
          ) : (
            <span className="text-muted-foreground">nothing reached</span>
          )}
        </div>
      </div>
    </Figure>
  );
}

function Label({ children }: { children: React.ReactNode }) {
  return <span className="flex min-h-7 items-start pt-1 text-xs text-muted-foreground">{children}</span>;
}

/** The hop between two rows: a hairline, and a dot that drops down it
 *  while the request is in flight. */
function Wire({ active }: { active: boolean }) {
  return (
    <div className="relative ml-[11px] h-5 w-px bg-border">
      <span
        className={`absolute -left-[2.5px] size-1.5 rounded-full bg-blue transition-[transform,opacity] ease-out motion-reduce:transition-opacity ${active ? "translate-y-[14px] opacity-100 duration-500" : "translate-y-0 opacity-0 duration-150"}`}
      />
    </div>
  );
}
