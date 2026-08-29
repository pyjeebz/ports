import { SiteNav } from "@/components/site/site-nav";
import { Hero } from "@/components/site/hero";
import { Flapboard } from "@/components/site/flapboard";
import { CapabilityGrid } from "@/components/site/capability-grid";
import { Principles } from "@/components/site/principles";
import { CommandTable } from "@/components/site/command-table";
import { InstallSection } from "@/components/site/install-section";
import { SiteFooter } from "@/components/site/site-footer";

export default function Home() {
  return (
    <>
      <SiteNav />
      <main>
        <Hero />
        <FlapboardSection />
        <CapabilityGrid />
        <Principles />
        <CommandTable />
        <InstallSection />
      </main>
      <SiteFooter />
    </>
  );
}

function FlapboardSection() {
  return (
    <section className="border-b border-line-soft py-24">
      <div className="mx-auto max-w-[1160px] px-8">
        <p className="eyebrow">▸ the session</p>
        <h2 className="mt-4 max-w-[30ch] text-balance text-[clamp(1.5rem,2.6vw,2rem)] font-semibold leading-[1.2] tracking-[-0.03em] text-strong">
          Watch a project come down, row by row.
        </h2>
        <p className="mt-3 max-w-[56ch] text-[15px] leading-relaxed text-muted">
          Every stop is SIGTERM first, a grace period, then SIGKILL. Containers
          go down through the daemon — never by killing docker-proxy.
        </p>
        <div className="mt-10 max-w-[860px]">
          <Flapboard />
        </div>
      </div>
    </section>
  );
}