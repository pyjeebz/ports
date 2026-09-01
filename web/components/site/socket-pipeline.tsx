"use client";

import { Fragment } from "react";

import { PIPELINES } from "@/lib/site";
import { useMotionAllowed } from "@/lib/motion";
import { useInView } from "@/lib/use-in-view";
import { SectionHead } from "./section-head";

const STAGGER = 0.12;

/**
 * One horizontal pipeline: bordered mono nodes joined by hairlines that
 * draw left to right. The final node — the live measurement — carries
 * the slow accent pulse. Pure CSS under the section's data-motion arm;
 * the static frame is the fully drawn chain.
 */
function Pipeline({
  nodes,
  baseDelay,
}: {
  nodes: readonly string[];
  baseDelay: number;
}) {
  return (
    <div className="flex items-center font-mono">
      {nodes.map((node, i) => {
        const delay = `${baseDelay + i * STAGGER}s`;
        const last = i === nodes.length - 1;
        const box = (
          <span
            {...(last ? { "data-pulse": true } : {})}
            className={`shrink-0 rounded-md border border-line bg-panel px-4 py-3 text-[13.5px] text-foreground`}
          >
            {node}
          </span>
        );
        return (
          <Fragment key={node}>
            {i > 0 && (
              <span
                data-draw
                aria-hidden
                className="mx-4 h-px w-8 shrink-0 bg-line"
                style={{ animationDelay: delay }}
              />
            )}
            {/* wrapper: data-appear handles the entrance, the inner box
                carries data-pulse — two animations, two elements */}
            <span data-appear className="inline-flex" style={{ animationDelay: delay }}>
              {box}
            </span>
          </Fragment>
        );
      })}
    </div>
  );
}

/**
 * The system section — the page's one animated diagram. Two chains:
 * a dev server and a database, each drawn socket → process → project →
 * time when the section enters view, and paused when it leaves.
 */
export function SocketPipeline() {
  const [ref, inView] = useInView<HTMLElement>({ threshold: 0.3, once: false });
  const motion = useMotionAllowed();
  const armed = inView && motion;

  return (
    <section
      ref={ref}
      data-motion={armed ? "on" : undefined}
      className="border-b border-line py-24"
    >
      <div className="mx-auto max-w-[1160px] px-8">
        <SectionHead
          kicker="the system"
          title="From socket to service."
          sub="Every row in the table is one of these chains: where it listens, what it is, who owns it, how long it&rsquo;s been up."
        />

        <div className="mt-12 space-y-6 overflow-x-auto pb-2">
          <div className="flex min-w-max items-center gap-6">
            <p className="readout w-[7.5rem] shrink-0">a dev server</p>
            <Pipeline nodes={PIPELINES[0].nodes} baseDelay={0} />
          </div>
          <div className="flex min-w-max items-center gap-6">
            <p className="readout w-[7.5rem] shrink-0">a database</p>
            <Pipeline nodes={PIPELINES[1].nodes} baseDelay={0.9} />
          </div>
        </div>

        <p className="mt-12 max-w-[52ch] text-[clamp(1.05rem,1.8vw,1.35rem)] font-medium leading-[1.4] text-muted">
          A port is the start of the chain, not the answer.{" "}
          <span className="text-strong">ports walks it to the end.</span>
        </p>
      </div>
    </section>
  );
}