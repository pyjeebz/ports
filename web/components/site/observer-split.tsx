import { OBSERVES, OBSERVER_LEDGER, YOU_DECIDE } from "@/lib/site";
import { SectionHead } from "./section-head";
import { Card } from "@/components/ui/card";
import { Tape } from "@/components/ui/tape";

/**
 * The contract section: what the tool does on the left, what you do on
 * the right — the ASCII trees keep it in the CLI's own handwriting. The
 * dashed sheet is the site's idiom for the deliberate actor: you — a
 * torn-off note, not a drawn card.
 */
export function ObserverSplit() {
  return (
    <section className="py-24">
      <div className="mx-auto max-w-[1160px] px-8">
        <SectionHead
          kicker="the contract"
          title="It observes. You decide."
          sub="ports reads the machine and reports. The only two things it will ever do to a process, it does because you said so."
        />

        <div className="mt-12 grid items-start gap-10 md:grid-cols-2">
          {/* what the tool does */}
          <Card rotate="-1" bg="white" className="relative p-6">
            <Tape color="yellow" className="-top-3 left-10 -rotate-3" />
            <p className="font-mono text-[13px] text-charcoal">$ ports</p>
            <pre className="mt-4 font-mono text-[13px] leading-[2] text-charcoal-light">
              {OBSERVES.map((verb, i) => (
                <span key={verb}>
                  {"  "}
                  {i === OBSERVES.length - 1 ? "└──" : "├──"}
                  {verb}
                  {"\n"}
                </span>
              ))}
            </pre>
            <div className="mt-6 border-t border-line-soft pt-5">
              {OBSERVER_LEDGER.map((item) => (
                <p key={item.item} className="font-mono text-[12.5px] leading-[2]">
                  <span className={item.ok ? "text-charcoal" : "text-faint"}>
                    {item.ok ? "✓" : "✗"}
                  </span>{" "}
                  <span className={item.ok ? "text-charcoal" : "text-faint"}>{item.item}</span>
                </p>
              ))}
            </div>
          </Card>

          {/* what you do */}
          <div className="border-[1.5px] border-dashed border-charcoal/50 bg-transparent p-6 md:rotate-1">
            <p className="font-mono text-[13px] text-charcoal">you</p>
            <pre className="mt-4 font-mono text-[15px] leading-[2] text-faint">
              {"  ├── "}
              <span className="text-charcoal">{YOU_DECIDE[0]}</span>
              {"\n  └── "}
              <span className="text-charcoal">{YOU_DECIDE[1]}</span>
            </pre>
            <p className="mt-6 border-t border-line-soft pt-5 text-[14px] leading-relaxed text-charcoal-light">
              The two acts — naming a port to free, or a project to stop. Everything else is
              reading.
            </p>
          </div>
        </div>

        <p className="mt-14 max-w-[52ch] font-display text-[clamp(1.2rem,2.2vw,1.6rem)] leading-[1.4] text-charcoal-light">
          ports never starts your services. It never supervises them. It never installs anything.{" "}
          <span className="text-charcoal">It tells you what&rsquo;s there.</span>
        </p>
      </div>
    </section>
  );
}