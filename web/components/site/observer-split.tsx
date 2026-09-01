import { OBSERVES, OBSERVER_LEDGER, YOU_DECIDE } from "@/lib/site";
import { SectionHead } from "./section-head";

/**
 * The contract section: what the tool does on the left, what you do on
 * the right — the ASCII trees keep it in the CLI's own handwriting. The
 * dashed panel is the site's idiom for the deliberate actor: you.
 */
export function ObserverSplit() {
  return (
    <section className="border-b border-line py-24">
      <div className="mx-auto max-w-[1160px] px-8">
        <SectionHead
          kicker="the contract"
          title="It observes. You decide."
          sub="ports reads the machine and reports. The only two things it will ever do to a process, it does because you said so."
        />

        <div className="mt-12 grid gap-6 md:grid-cols-2">
          {/* what the tool does */}
          <div className="rounded-xl border border-line bg-panel p-6">
            <p className="font-mono text-[13px] text-strong">$ ports</p>
            <pre className="mt-4 font-mono text-[13px] leading-[2] text-muted">
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
                  <span className={item.ok ? "text-accent" : "text-faint"}>{item.ok ? "✓" : "✗"}</span>{" "}
                  <span className={item.ok ? "text-foreground" : "text-faint"}>{item.item}</span>
                </p>
              ))}
            </div>
          </div>

          {/* what you do */}
          <div className="rounded-xl border border-dashed border-strong/30 p-6">
            <p className="font-mono text-[13px] text-strong">you</p>
            <pre className="mt-4 font-mono text-[15px] leading-[2] text-faint">
              {"  ├── "}
              <span className="text-strong">{YOU_DECIDE[0]}</span>
              {"\n  └── "}
              <span className="text-strong">{YOU_DECIDE[1]}</span>
            </pre>
            <p className="mt-6 border-t border-line-soft pt-5 text-[13px] leading-relaxed text-muted">
              The two acts — naming a port to free, or a project to stop. Everything else is
              reading.
            </p>
          </div>
        </div>

        <p className="mt-12 max-w-[52ch] text-[clamp(1.05rem,1.8vw,1.35rem)] font-medium leading-[1.4] text-muted">
          ports never starts your services. It never supervises them. It never installs anything.{" "}
          <span className="text-strong">It tells you what&rsquo;s there.</span>
        </p>
      </div>
    </section>
  );
}