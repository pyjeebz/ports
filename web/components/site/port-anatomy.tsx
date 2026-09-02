"use client";

import { PORT_ANATOMY } from "@/lib/site";
import { useMotionAllowed, useTimedSteps } from "@/lib/motion";
import { useInView } from "@/lib/use-in-view";
import { SectionHead } from "./section-head";
import { Card } from "@/components/ui/card";

/**
 * The model section: one giant port number, and the three answers that
 * make it interesting — revealed in sequence, accumulating to the final
 * frame the server already rendered. Each answer lands as its own
 * tilted card, none of them hung straight.
 */
const CHIP_ROTATIONS = ["1", "-2", "1", "2"] as const;

export function PortAnatomy() {
  const [ref, inView] = useInView<HTMLElement>({ threshold: 0.35 });
  const motion = useMotionAllowed();
  const active = inView && motion;

  // The first anatomy entry is the numeral itself; the remaining chips
  // reveal one per tick (step counts 0..3, final frame = 3 = all on).
  const chips = PORT_ANATOMY.slice(1);
  const step = useTimedSteps(chips.length, 850, active);

  return (
    <section ref={ref} className="py-24">
      <div className="mx-auto max-w-[1160px] px-8">
        <SectionHead
          kicker="the model"
          title="Your ports aren&rsquo;t the interesting part."
          sub="A port number is just an address. The interesting part is what answers there."
        />

        <div className="mt-14">
          <p className="font-mono text-[clamp(5rem,16vw,11rem)] font-semibold leading-none tracking-[-0.04em] text-charcoal [font-variant-numeric:tabular-nums]">
            3000
          </p>
          <p className="readout mt-4">a port — where a service listens</p>
        </div>

        <div className="mt-12 flex flex-wrap gap-6">
          {chips.map((chip, i) => {
            const visible = step >= i + 1;
            return (
              <Card
                key={chip.label}
                rotate={CHIP_ROTATIONS[i % CHIP_ROTATIONS.length]}
                bg={i % 2 ? "paper" : "white"}
                className={`px-5 py-3 transition-[opacity,transform] duration-500 ${
                  visible ? "translate-y-0 opacity-100" : "translate-y-2 opacity-0"
                }`}
              >
                <span className="readout">{chip.label}</span>
                <span className="mt-1 block font-mono text-[15px] text-charcoal">
                  {chip.value}
                </span>
              </Card>
            );
          })}
        </div>

        <p className="mt-16 max-w-[40ch] font-display text-[clamp(1.3rem,2.4vw,1.8rem)] leading-[1.4] text-charcoal">
          A port is just where a service listens.
          <span className="block text-charcoal-light">
            ports tells you what&rsquo;s actually there.
          </span>
        </p>
      </div>
    </section>
  );
}