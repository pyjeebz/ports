import { SiteNav } from "@/components/site/site-nav";
import { Hero } from "@/components/site/hero";
import { Terminal } from "@/components/site/terminal";
import { Capabilities } from "@/components/site/capabilities";
import { CommandReference } from "@/components/site/command-reference";
import { Workflow } from "@/components/site/workflow";
import { Install } from "@/components/site/install";
import { Harbor } from "@/components/site/harbor";
import { SiteFooter } from "@/components/site/site-footer";

export default function Home() {
  return (
    <>
      <SiteNav />
      <div className="frame">
        <div className="guide guide-l" aria-hidden />
        <div className="guide guide-r" aria-hidden />

        <Hero />
        <Terminal />
        <Capabilities />
        <CommandReference />
        <Workflow />
        <Install />

        <section className="harborsec gx">
          <div className="col">
            <div className="sec-head">
              <span className="eyebrow">The harbor</span>
              <h2 className="sec-title">Port of localhost.</h2>
            </div>
            <Harbor />
            <p className="harbor-note">
              No PID was harmed in the making of this harbor. Every ship gets
              SIGTERM before SIGKILL.
            </p>
          </div>
        </section>

        <SiteFooter />
      </div>
    </>
  );
}
