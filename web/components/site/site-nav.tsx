import Link from "next/link";

import { PortsLogo } from "./ports-logo";

export function SiteNav() {
  return (
    <nav className="sticky top-0 z-30 bg-black/80 backdrop-blur-md">
      <div className="mx-auto flex h-[72px] max-w-[1160px] items-center px-8 max-[720px]:h-[60px] max-[720px]:px-6">
        <a href="#" aria-label="ports — home" className="text-foreground">
          <PortsLogo className="h-[34px] w-[34px]" />
        </a>
        <div className="ml-auto flex items-center gap-6 text-sm text-muted">
          <Link href="/docs" className="transition-colors hover:text-foreground">
            Docs
          </Link>
          <Link
            href="/docs/installation"
            className="hidden transition-colors hover:text-foreground sm:inline"
          >
            Installation
          </Link>
          <a
            href="https://github.com/pyjeebz/ports"
            className="transition-colors hover:text-foreground"
          >
            GitHub
          </a>
        </div>
      </div>
    </nav>
  );
}
