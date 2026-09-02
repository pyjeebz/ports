import { SITE } from "@/lib/site";
import { PortsMachine } from "@/components/site/ports-machine";
import { ButtonLink } from "@/components/ui/button";
import { Highlighter } from "@/components/ui/highlighter";
import { Card } from "@/components/ui/card";
import { Tape } from "@/components/ui/tape";

/**
 * The hero: the question, then the product as a machine the visitor
 * wakes up — empty, a scan, rows arriving at their berths, rows that
 * open into what ports resolved. Not an illustration, not a video: the
 * site behaves like ports. On paper, the machine is a taped-down page
 * of the sketchbook.
 */
export function Hero() {
  return (
    <section className="relative overflow-hidden">
      <div className="relative mx-auto max-w-[1160px] px-8 pb-24 pt-16">
        <p className="eyebrow">ports.tools</p>
        <h1 className="mt-4 max-w-[24ch] text-balance font-display text-[clamp(2.4rem,5.8vw,4rem)] leading-[1.15] text-charcoal">
          What&rsquo;s <Highlighter color="yellow">running</Highlighter> on your machine?
        </h1>
        <p className="mt-5 max-w-[54ch] text-[16px] leading-relaxed text-charcoal-light">
          <code className="font-mono text-[13.5px] text-charcoal">ports</code> shows everything
          listening — which framework is running, which project owns it, which container it lives
          in, and how long it&rsquo;s been up. Run it below.{" "}
          <Highlighter color="pink">One static binary. One command.</Highlighter>
        </p>
        <div className="mt-7 flex flex-wrap items-center gap-4">
          <ButtonLink href="/docs/quickstart">Get started</ButtonLink>
          <ButtonLink variant="secondary" href={SITE.links.github}>
            GitHub ↗
          </ButtonLink>
          <span className="readout hidden sm:inline">zero config · one static binary</span>
        </div>

        {/* the artwork: the machine itself, taped onto the page */}
        <div className="relative mt-14">
          <Tape color="yellow" className="-top-4 left-1/2 -translate-x-1/2 -rotate-2" />
          <Card rotate="-1" bg="white" className="mx-auto max-w-[760px] p-2 sm:p-4">
            <PortsMachine />
          </Card>
          <Tape color="clear" className="-bottom-3 left-16 -rotate-6" />
        </div>
      </div>
    </section>
  );
}