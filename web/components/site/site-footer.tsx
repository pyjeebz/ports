import Link from "next/link";

import { PortsLogo } from "./ports-logo";
import { SITE } from "@/lib/site";

export function SiteFooter() {
  return (
    <footer className="py-14">
      <div className="mx-auto max-w-[1160px] px-8">
        <div className="grid grid-cols-1 gap-10 sm:grid-cols-[2fr_1fr_1fr]">
          <div className="max-w-[320px]">
            <span className="flex items-center gap-2.5 text-[15px] font-semibold">
              <PortsLogo className="h-[18px] w-[18px]" />
              ports
            </span>
            <p className="mt-3 text-[13.5px] leading-relaxed text-muted">
              ports never starts, supervises, or installs anything. It reads
              your machine and — when you ask — stops a thing you name.
            </p>
            <p className="mt-3 font-mono text-[11.5px] tracking-[0.08em] text-faint">
              MIT LICENSE · {SITE.domain.replace("https://", "")}
            </p>
          </div>
          <div className="flex flex-col gap-2.5">
            <h5 className="text-[12.5px] font-semibold text-foreground">Docs</h5>
            <Link href="/docs" className="text-[13.5px] text-muted hover:text-foreground">
              Introduction
            </Link>
            <Link href="/docs/quickstart" className="text-[13.5px] text-muted hover:text-foreground">
              Quickstart
            </Link>
            <Link href="/docs/commands" className="text-[13.5px] text-muted hover:text-foreground">
              Commands
            </Link>
          </div>
          <div className="flex flex-col gap-2.5">
            <h5 className="text-[12.5px] font-semibold text-foreground">Project</h5>
            <a href={SITE.links.github} className="text-[13.5px] text-muted hover:text-foreground">
              GitHub ↗
            </a>
            <a href={SITE.links.releases} className="text-[13.5px] text-muted hover:text-foreground">
              Releases ↗
            </a>
          </div>
        </div>
      </div>
    </footer>
  );
}