import Link from "next/link";

import { PortsLogo } from "./ports-logo";
import { SITE } from "@/lib/site";
import { DrawnLine } from "@/components/ui/drawn-line";

/**
 * The bottom of the page — a table of contents on the left, project
 * links on the right, small and subdued, above one final drawn rule.
 */
export function SiteFooter() {
  return (
    <footer className="py-16">
      <div className="mx-auto max-w-[1160px] px-8">
        <DrawnLine className="mb-14" />
        <div className="grid grid-cols-1 gap-10 sm:grid-cols-[1fr_auto] sm:items-end">
          {/* table-of-contents columns */}
          <nav
            aria-label="Documentation contents"
            className="grid grid-cols-2 gap-x-10 gap-y-8 sm:grid-cols-4"
          >
            {[
              { title: "Installation", href: "/docs/installation", items: ["Homebrew", "curl", "Scoop", "Verify"] },
              { title: "Usage", href: "/docs/quickstart", items: ["The table", "Stop by name", "Grace period", "Uptime"] },
              { title: "Filtering", href: "/docs/commands#cmd-find", items: ["ports find", "By project", "By container", "By port"] },
              { title: "The API", href: "/docs/commands#flags", items: ["--json", "--force", "Piping", "Agents"] },
            ].map((col) => (
              <div key={col.title}>
                <h5 className="text-[13px] font-bold text-charcoal">
                  <Link href={col.href} className="hover:underline">
                    {col.title}
                  </Link>
                </h5>
                <ul className="mt-2.5 flex flex-col gap-1.5">
                  {col.items.map((item) => (
                    <li key={item} className="text-[12.5px] text-faint">
                      {item}
                    </li>
                  ))}
                </ul>
              </div>
            ))}
          </nav>

          {/* project links */}
          <div className="flex flex-col items-start gap-4 sm:items-end">
            <span className="flex items-center gap-2.5 text-[14px] font-bold text-charcoal">
              <PortsLogo className="h-[16px] w-[16px]" />
              ports
            </span>
            <p className="max-w-[320px] text-[12.5px] leading-relaxed text-faint sm:text-right">
              ports never starts, supervises, or installs anything. It reads
              your machine and — when you ask — stops a thing you name.
            </p>
            <div className="flex gap-4 font-mono text-[11.5px] text-charcoal-light">
              <a href={SITE.links.github} className="hover:text-charcoal">
                GitHub ↗
              </a>
              <a href={SITE.links.releases} className="hover:text-charcoal">
                Releases ↗
              </a>
            </div>
          </div>
        </div>

        <div className="mt-12 flex items-center justify-between">
          <span className="readout">MIT LICENSE</span>
          <span className="readout">{SITE.domain.replace("https://", "")}</span>
        </div>
      </div>
    </footer>
  );
}