import Link from "next/link";

import { FAQ_ITEMS } from "@/lib/site";
import { SectionHead } from "./section-head";
import { Card } from "@/components/ui/card";

// Six index cards pinned to the page — alternating tilt, alternating
// paper, like a stack you flipped through and left slightly messy.
const CARD_STYLES = [
  { rotate: "-1", bg: "white" },
  { rotate: "1", bg: "paper" },
  { rotate: "1", bg: "paper" },
  { rotate: "-1", bg: "white" },
  { rotate: "-1", bg: "white" },
  { rotate: "1", bg: "paper" },
] as const;

/**
 * The FAQ close — the borrowed Vercel ending. Every answer is capped
 * with a doc link; the page trusts verifiability over adjectives.
 */
export function Faq() {
  return (
    <section className="py-24">
      <div className="mx-auto max-w-[1160px] px-8">
        <SectionHead kicker="questions" title="Frequently asked questions." />

        <div className="mt-12 grid items-start gap-x-8 gap-y-8 md:grid-cols-2">
          {FAQ_ITEMS.map((item, i) => (
            <Card
              key={item.q}
              rotate={CARD_STYLES[i].rotate}
              bg={CARD_STYLES[i].bg}
              className="p-6"
            >
              <p className="text-[16px] font-bold text-charcoal">{item.q}</p>
              <p className="mt-3 text-[14px] leading-relaxed text-charcoal-light">{item.a}</p>
              <Link
                href={item.href}
                className="mt-4 inline-block font-mono text-[12px] text-charcoal underline-offset-4 transition-colors hover:text-charcoal hover:underline"
              >
                {item.link} →
              </Link>
            </Card>
          ))}
        </div>
      </div>
    </section>
  );
}