import Link from "next/link";

import { COMMANDS } from "@/lib/site";

export function CommandTable() {
  return (
    <section className="border-b border-line-soft py-24">
      <div className="mx-auto max-w-[1160px] px-8">
        <p className="eyebrow">▸ commands</p>
        <h2 className="mt-4 text-balance text-[clamp(1.5rem,2.6vw,2rem)] font-semibold leading-[1.2] tracking-[-0.03em] text-strong">
          Four commands and two flags. That&apos;s the whole tool.
        </h2>
        <div className="mt-10 overflow-hidden rounded-md border border-line-soft">
          {COMMANDS.map((c) => (
            <Link
              key={c.cmd}
              href={c.href}
              className="group grid grid-cols-1 items-baseline gap-1 bg-panel px-5 py-3.5 transition-colors hover:bg-panel-2 sm:grid-cols-[240px_1fr] sm:gap-6 not-first:border-t not-first:border-t-line-soft"
            >
              <code className="font-mono text-[13px] text-accent">
                {c.cmd}
              </code>
              <span className="text-[14px] text-muted group-hover:text-foreground">
                {c.desc}
              </span>
            </Link>
          ))}
        </div>
        <p className="mt-4 text-[12.5px] text-faint">
          <Link href="/docs/commands" className="text-muted underline-offset-4 hover:text-foreground hover:underline">
            Full reference →
          </Link>
        </p>
      </div>
    </section>
  );
}