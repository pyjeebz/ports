import Link from "next/link";

import { FAQ_ITEMS } from "@/lib/site";
import { SectionHead } from "./section-head";

/**
 * The FAQ close — the borrowed Vercel ending. Every answer is capped
 * with a doc link; the page trusts verifiability over adjectives.
 */
export function Faq() {
  return (
    <section className="border-b border-line py-24">
      <div className="mx-auto max-w-[1160px] px-8">
        <SectionHead kicker="questions" title="Frequently asked questions." />

        <div className="mt-10 grid gap-x-12 md:grid-cols-2">
          {FAQ_ITEMS.map((item) => (
            <div key={item.q} className="border-t border-line-soft py-6">
              <p className="text-[15px] font-medium text-strong">{item.q}</p>
              <p className="mt-2.5 text-[13.5px] leading-relaxed text-muted">{item.a}</p>
              <Link
                href={item.href}
                className="mt-3 inline-block font-mono text-[12px] text-foreground underline-offset-4 transition-colors hover:text-strong hover:underline"
              >
                {item.link} →
              </Link>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}