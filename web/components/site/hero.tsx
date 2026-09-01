import Link from "next/link";

import { SITE } from "@/lib/site";
import { PortsMachine } from "@/components/site/ports-machine";

/**
 * The hero: the question, then the product as a machine the visitor
 * wakes up — empty, a scan, rows arriving at their berths, rows that
 * open into what ports resolved. Not an illustration, not a video: the
 * site behaves like ports.
 */
export function Hero() {
  return (
    <section className="relative overflow-hidden border-b border-line">
      {/* faint top glow — the one Vercel-style gradient on the page */}
      <div
        aria-hidden="true"
        className="pointer-events-none absolute inset-x-0 top-0 h-[400px]"
        style={{
          background:
            "radial-gradient(ellipse 60% 100% at 50% -20%, rgba(0, 112, 243, 0.08), transparent 70%)",
        }}
      />
      <div className="relative mx-auto max-w-[1160px] px-8 pb-24 pt-20">
        <p className="eyebrow">ports.tools</p>
        <h1 className="mt-4 max-w-[24ch] text-balance text-[clamp(2.2rem,5.4vw,3.6rem)] font-semibold leading-[1.06] tracking-[-0.045em] text-strong">
          What&rsquo;s running on your machine?
        </h1>
        <p className="mt-5 max-w-[54ch] text-[15px] leading-relaxed text-muted">
          <code className="font-mono text-[13.5px] text-foreground">ports</code> shows everything
          listening — which framework is running, which project owns it, which container it lives
          in, and how long it&rsquo;s been up. Run it below. One static binary. One command.
        </p>
        <div className="mt-7 flex flex-wrap items-center gap-3">
          <Link
            href="/docs/quickstart"
            className="rounded-md bg-strong px-4 py-2 font-mono text-[12.5px] font-medium text-background transition-opacity hover:opacity-90"
          >
            Get started
          </Link>
          <a
            href={SITE.links.github}
            className="rounded-md border border-line bg-panel px-3.5 py-2 font-mono text-[12.5px] text-muted transition-colors hover:border-strong/40 hover:text-foreground"
          >
            GitHub ↗
          </a>
          <span className="readout hidden sm:inline">zero config · one static binary</span>
        </div>

        {/* the artwork: the machine itself, at display size */}
        <div className="mt-12">
          <PortsMachine />
        </div>
      </div>
    </section>
  );
}