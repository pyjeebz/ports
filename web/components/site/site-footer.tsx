import Link from "next/link";

import { PortsLogo } from "./ports-logo";
import { SITE } from "@/lib/site";

/**
 * Prompt 4: the bottom of a high-quality open-source docs site — a table of
 * contents on the left, project links on the right, small and subdued.
 */
export function SiteFooter() {
  return (
    <footer className="border-t border-line py-16">
      <div className="mx-auto max-w-[1160px] px-8">
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
                <h5 className="text-[12px] font-semibold text-foreground">
                  <Link href={col.href} className="hover:text-strong">
                    {col.title}
                  </Link>
                </h5>
                <ul className="mt-2.5 flex flex-col gap-1.5">
                  {col.items.map((item) => (
                    <li key={item} className="text-[12px] text-faint">
                      {item}
                    </li>
                  ))}
                </ul>
              </div>
            ))}
          </nav>

          {/* project links */}
          <div className="flex flex-col items-start gap-4 sm:items-end">
            <span className="flex items-center gap-2.5 text-[13px] font-semibold text-foreground">
              <PortsLogo className="h-[16px] w-[16px]" />
              ports
            </span>
            <p className="max-w-[320px] text-[12px] leading-relaxed text-faint sm:text-right">
              ports never starts, supervises, or installs anything. It reads
              your machine and — when you ask — stops a thing you name.
            </p>
            <div className="flex gap-4 font-mono text-[11.5px] text-muted">
              <a href={SITE.links.github} className="hover:text-strong">
                GitHub ↗
              </a>
              <a href={SITE.links.releases} className="hover:text-strong">
                Releases ↗
              </a>
            </div>
          </div>
        </div>

        <div className="mt-12 flex items-center justify-between border-t border-line-soft pt-6">
          <span className="readout">MIT LICENSE</span>
          <span className="readout">{SITE.domain.replace("https://", "")}</span>
        </div>
      </div>
    </footer>
  );
}