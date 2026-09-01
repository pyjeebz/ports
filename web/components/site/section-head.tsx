import type { ReactNode } from "react";

/**
 * The module template every section opens with: a domain kicker, a short
 * declarative statement, and one line of subcopy. The borrowed Vercel
 * rhythm — the heading carries the sequence, no section numbers.
 */
export function SectionHead({
  kicker,
  title,
  sub,
}: {
  kicker: string;
  title: ReactNode;
  sub?: string;
}) {
  return (
    <>
      <p className="eyebrow">{kicker}</p>
      <h2 className="mt-4 text-balance text-[clamp(1.5rem,2.6vw,2rem)] font-semibold leading-[1.2] tracking-[-0.03em] text-strong">
        {title}
      </h2>
      {sub && <p className="mt-4 max-w-[62ch] text-[14.5px] leading-relaxed text-muted">{sub}</p>}
    </>
  );
}