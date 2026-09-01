"use client";

import { Fragment, useState } from "react";

import { DOCKER_TRACE, SHOWCASE } from "@/lib/site";
import { useMotionAllowed } from "@/lib/motion";
import { SectionHead } from "./section-head";
import { TerminalBlock } from "./terminal-block";

const findOutput = SHOWCASE.find((e) => e.id === "find")!.output;

type View = "os" | "ports";

/**
 * The docker section as a runnable unwrap: the same four slots, two
 * views. The OS view is what lsof sees — docker-proxy, then nothing.
 * The ports view resolves through the daemon to the container and what's
 * actually running. The resting state (and the server frame) is the
 * resolved chain; clicking docker-proxy in the OS view is the product
 * story in one gesture. Motion is armed only after the first toggle, so
 * the server frame sits at rest.
 */
export function DockerTrace() {
  const motion = useMotionAllowed();
  const [view, setView] = useState<View>("ports");
  const [toggled, setToggled] = useState(false);
  const ports = view === "ports";

  const switchTo = (next: View) => {
    setToggled(true);
    setView(next);
  };

  const chip = (i: number) => {
    if (i === 0) return "border-line bg-panel text-foreground";
    if (i === 1) return "border-dashed border-line text-muted";
    if (!ports) return "border-dashed border-line text-faint"; // the invisible slots
    return i === DOCKER_TRACE.length - 1
      ? "border-accent/50 bg-accent-soft text-accent"
      : "border-line bg-panel text-foreground";
  };

  return (
    <section className="border-b border-line py-24">
      <div className="mx-auto max-w-[1160px] px-8">
        <SectionHead
          kicker="docker"
          title="Docker shouldn&rsquo;t make this harder."
          sub="docker-proxy is all the OS reports. Behind it is the container you actually care about — toggle what each tool sees."
        />

        <div className="mt-12 flex flex-wrap items-center gap-3">
          {(
            [
              ["os", "the OS view"],
              ["ports", "the ports view"],
            ] as const
          ).map(([id, label]) => (
            <button
              key={id}
              type="button"
              aria-pressed={view === id}
              onClick={() => switchTo(id)}
              className={`rounded-md border px-4 py-2 font-mono text-[12.5px] transition-colors ${
                view === id
                  ? "border-accent/60 bg-accent-soft text-accent"
                  : "border-line bg-panel text-muted hover:border-strong/40 hover:text-foreground"
              }`}
            >
              {label}
            </button>
          ))}
        </div>

        {/* keyed on the view so a switch remounts the chain and the
            draws replay; the armed attribute only appears after the
            first toggle, keeping the server frame at rest */}
        <div
          key={view}
          data-motion={toggled && motion ? "on" : undefined}
          className="mt-8 overflow-x-auto pb-2"
        >
          <div className="flex min-w-max items-center">
            {DOCKER_TRACE.map((n, i) => (
              <Fragment key={n.node}>
                {i > 0 && (
                  <span
                    aria-hidden
                    data-draw={ports && i > 1 ? true : undefined}
                    className={`mx-5 h-px w-10 shrink-0 ${
                      ports || i === 1 ? "bg-line" : "bg-line-soft"
                    }`}
                  />
                )}
                <div className="shrink-0 text-center">
                  {i === 1 && !ports ? (
                    <button
                      type="button"
                      onClick={() => switchTo("ports")}
                      className="inline-block cursor-pointer rounded-md border border-dashed border-line px-4 py-3 font-mono text-[13.5px] text-muted transition-colors hover:border-accent/60 hover:text-accent"
                    >
                      {n.node} ↠
                    </button>
                  ) : (
                    <span
                      data-appear={ports && i > 1 ? true : undefined}
                      style={
                        ports && i > 1 ? { animationDelay: `${(i - 2) * 0.12}s` } : undefined
                      }
                      className={`inline-block rounded-md border px-4 py-3 font-mono text-[13.5px] ${chip(i)}`}
                    >
                      {ports ? n.node : i > 1 ? "?" : n.node}
                    </span>
                  )}
                  <p className="readout mt-2.5">{ports ? n.label : i === 1 ? "the wall" : i > 1 ? "invisible" : n.label}</p>
                </div>
              </Fragment>
            ))}
          </div>
        </div>

        <p className="readout mt-6">
          {ports
            ? "resolved through the Docker daemon"
            : "everything past docker-proxy is invisible to the OS"}
        </p>

        <div className="mt-10 max-w-[640px]">
          {ports ? (
            <TerminalBlock cmd="ports find postgres" label="resolved">
              {findOutput.map((line, i) => (
                <span key={i} className={i === 0 ? "text-faint" : undefined}>
                  {line}
                  {"\n"}
                </span>
              ))}
            </TerminalBlock>
          ) : (
            <p className="max-w-[46ch] text-[14px] leading-relaxed text-muted">
              lsof stops at docker-proxy. To reach the container you would need to know
              docker is involved, remember the right <code className="font-mono text-[13px] text-foreground">docker ps</code>{" "}
              incantation, and match the socket by hand — every time.
            </p>
          )}
        </div>

        <p className="mt-10 max-w-[46ch] text-[clamp(1.05rem,1.8vw,1.35rem)] font-medium leading-[1.4] text-muted">
          <span className="text-strong">ports follows the connection</span> to the thing actually
          running it.
        </p>
        <p className="readout mt-4">
          containers stop through the Docker daemon — never docker-proxy
        </p>
      </div>
    </section>
  );
}