import Link from "next/link";

import { INSTALL_METHODS, SITE } from "@/lib/site";
import { CopyPill } from "./copy-pill";
import { SectionHead } from "./section-head";
import { ButtonLink } from "@/components/ui/button";
import { Highlighter } from "@/components/ui/highlighter";

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
          title={
            <>
              Sixty seconds from <Highlighter color="yellow">install</Highlighter> to answers.
            </>
          }
          sub="Install it, run it, see your machine. Nothing to configure first."
        />

        <div className="mt-10 flex flex-wrap items-center gap-4">
          <CopyPill text={brew} ariaLabel="Copy the Homebrew install command" />
          <ButtonLink variant="secondary" href={SITE.links.github}>
            GitHub ↗
          </ButtonLink>
          <ButtonLink variant="secondary" href={SITE.links.docs}>
            Documentation
          </ButtonLink>
        </div>

        <p className="mt-8 flex flex-wrap items-center gap-x-6 gap-y-1.5 text-[13px] text-charcoal-light">
          <span className="readout">also</span>
          <Link
            href="/docs/installation"
            className="underline-offset-4 transition-colors hover:text-charcoal hover:underline"
          >
            curl — any POSIX machine
          </Link>
          <Link
            href="/docs/installation"
            className="underline-offset-4 transition-colors hover:text-charcoal hover:underline"
          >
            Scoop — Windows
          </Link>
          <span className="readout ml-auto hidden sm:inline">MIT · one static binary</span>
        </p>
      </div>
    </section>
  );
}