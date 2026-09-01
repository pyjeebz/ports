import Link from "next/link";

import { INSTALL_METHODS, SITE } from "@/lib/site";
import { CopyPill } from "./copy-pill";
import { SectionHead } from "./section-head";

const brew = INSTALL_METHODS[0].lines[0];

/**
 * The close: the install line, the two doors — the repo and the docs —
 * and the other ways in, quiet. Ends the way the page argues: one
 * command, no ceremony.
 */
export function Close() {
  return (
    <section className="py-24">
      <div className="mx-auto max-w-[1160px] px-8">
        <SectionHead
          kicker="get started"
          title="Sixty seconds from install to answers."
          sub="Install it, run it, see your machine. Nothing to configure first."
        />

        <div className="mt-8 flex flex-wrap items-center gap-3">
          <CopyPill text={brew} ariaLabel="Copy the Homebrew install command" />
          <a
            href={SITE.links.github}
            className="rounded-md bg-strong px-4 py-2 font-mono text-[12.5px] font-medium text-background transition-opacity hover:opacity-90"
          >
            GitHub ↗
          </a>
          <Link
            href={SITE.links.docs}
            className="rounded-md border border-line bg-panel px-3.5 py-2 font-mono text-[12.5px] text-muted transition-colors hover:border-strong/40 hover:text-foreground"
          >
            Documentation
          </Link>
        </div>

        <p className="mt-6 flex flex-wrap items-center gap-x-6 gap-y-1.5 font-mono text-[12px] text-muted">
          <span className="readout">also</span>
          <Link
            href="/docs/installation"
            className="text-foreground underline-offset-4 transition-colors hover:text-strong hover:underline"
          >
            curl — any POSIX machine
          </Link>
          <Link
            href="/docs/installation"
            className="text-foreground underline-offset-4 transition-colors hover:text-strong hover:underline"
          >
            Scoop — Windows
          </Link>
          <span className="readout ml-auto hidden sm:inline">MIT · one static binary</span>
        </p>
      </div>
    </section>
  );
}