"use client";

import { Fragment } from "react";

import { PIPELINES } from "@/lib/site";
import { useMotionAllowed } from "@/lib/motion";
import { useInView } from "@/lib/use-in-view";
import { SectionHead } from "./section-head";
import { Card } from "@/components/ui/card";
import { Arrow } from "@/components/ui/arrow";

const STAGGER = 0.12;

/**
 * One horizontal pipeline: mono nodes on hand-drawn cards joined by
 * pen arrows that appear left to right. The final node — the live
 * measurement — carries the slow pulse. Pure CSS under the section's
 * data-motion arm; the static frame is the fully drawn chain.
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
          <Card
            {...(last ? { "data-pulse": true } : {})}
            rotate={i % 2 ? "1" : "-1"}
            bg="white"
            className="shrink-0 px-4 py-3 text-[13.5px] text-charcoal"
          >
            {node}
          </Card>
        );
        return (
          <Fragment key={node}>
            {i > 0 && (
              <span
                data-appear
                aria-hidden
                className="mx-4 inline-flex shrink-0"
                style={{ animationDelay: delay }}
              >
                <Arrow />
              </span>
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
    <section ref={ref} data-motion={armed ? "on" : undefined} className="py-24">
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

        <p className="mt-12 max-w-[52ch] text-[clamp(1.05rem,1.8vw,1.35rem)] font-medium leading-[1.4] text-charcoal-light">
          A port is the start of the chain, not the answer.{" "}
          <span className="text-charcoal">ports walks it to the end.</span>
        </p>
      </div>
    </section>
  );
}