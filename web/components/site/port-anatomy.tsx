"use client";

import { PORT_ANATOMY } from "@/lib/site";
import { useMotionAllowed, useTimedSteps } from "@/lib/motion";
import { useInView } from "@/lib/use-in-view";
import { SectionHead } from "./section-head";

/**
 * The model section: one giant port number, and the three answers that
 * make it interesting — revealed in sequence, accumulating to the final
 * frame the server already rendered.
 */
export function PortAnatomy() {
  const [ref, inView] = useInView<HTMLElement>({ threshold: 0.35 });
  const motion = useMotionAllowed();
  const active = inView && motion;

  // The first anatomy entry is the numeral itself; the remaining chips
  // reveal one per tick (step counts 0..3, final frame = 3 = all on).
  const chips = PORT_ANATOMY.slice(1);
  const step = useTimedSteps(chips.length, 850, active);

  return (
    <section ref={ref} className="border-b border-line py-24">
      <div className="mx-auto max-w-[1160px] px-8">
        <SectionHead
          kicker="the model"
          title="Your ports aren&rsquo;t the interesting part."
          sub="A port number is just an address. The interesting part is what answers there."
        />

        <div className="mt-14">
          <p className="font-mono text-[clamp(5rem,16vw,11rem)] font-semibold leading-none tracking-[-0.04em] text-strong [font-variant-numeric:tabular-nums]">
            3000
          </p>
          <p className="readout mt-4">a port — where a service listens</p>
        </div>

        <div className="mt-10 flex flex-wrap gap-3">
          {chips.map((chip, i) => {
            const visible = step >= i + 1;
            return (
              <div
                key={chip.label}
                className={`rounded-lg border border-line bg-panel px-4 py-3 transition-[opacity,transform] duration-500 ${
                  visible ? "translate-y-0 opacity-100" : "translate-y-2 opacity-0"
                }`}
              >
                <span className="readout">{chip.label}</span>
                <span className="mt-1 block font-mono text-[15px] text-strong">{chip.value}</span>
              </div>
            );
          })}
        </div>

        <p className="mt-14 max-w-[40ch] text-[clamp(1.15rem,2vw,1.5rem)] font-semibold leading-[1.3] tracking-[-0.02em] text-strong">
          A port is just where a service listens.
          <span className="block text-muted">ports tells you what&rsquo;s actually there.</span>
        </p>
      </div>
    </section>
  );
}