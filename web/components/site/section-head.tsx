import type { ReactNode } from "react";

import { cn } from "@/lib/utils";

/**
 * The module template every section opens with: a domain kicker in mono
 * (the technical voice), a Sharpie statement in Architects Daughter, and
 * one line of subcopy in the ballpoint. The borrowed Vercel rhythm —
 * the heading carries the sequence, no section numbers.
 */
export function SectionHead({
  kicker,
  title,
  sub,
  className,
}: {
  kicker: string;
  title: ReactNode;
  sub?: string;
  className?: string;
}) {
  return (
    <>
      <p className="eyebrow">{kicker}</p>
      <h2
        className={cn(
          "mt-4 max-w-[26ch] text-balance font-display text-[clamp(1.6rem,3.2vw,2.5rem)] leading-[1.25] text-charcoal",
          className,
        )}
      >
        {title}
      </h2>
      {sub && (
        <p className="mt-4 max-w-[62ch] text-[15px] leading-relaxed text-charcoal-light">
          {sub}
        </p>
      )}
    </>
  );
}