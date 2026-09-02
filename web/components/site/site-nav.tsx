import Link from "next/link";

import { PortsLogo } from "./ports-logo";
import { DrawnLine } from "@/components/ui/drawn-line";

export function SiteNav() {
  return (
    <nav className="sticky top-0 z-30 bg-paper/90">
      <div className="mx-auto flex h-[72px] max-w-[1160px] items-center px-8 max-[720px]:h-[60px] max-[720px]:px-6">
        <a href="#" aria-label="ports — home" className="text-charcoal">
          <PortsLogo className="h-[34px] w-[34px]" />
        </a>
        <div className="ml-auto flex items-center gap-6 font-bold text-[15px] text-charcoal-light">
          <Link href="/docs" className="transition-colors hover:text-charcoal">
            Docs
          </Link>
          <Link
            href="/docs/installation"
            className="hidden transition-colors hover:text-charcoal sm:inline"
          >
            Installation
          </Link>
          <a
            href="https://github.com/pyjeebz/ports"
            className="transition-colors hover:text-charcoal"
          >
            GitHub
          </a>
        </div>
      </div>
      {/* the nav's bottom rule: drawn, not ruled */}
      <DrawnLine className="absolute inset-x-0 bottom-0 mx-auto max-w-[1160px]" />
    </nav>
  );
}